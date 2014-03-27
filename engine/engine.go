package engine

import (
	"container/list"
	"net/http"
	"strconv"
	"time"
)

type httpActionResult struct {
	is_client_error bool           // wether server responded with a 4xx
	is_server_error bool           // wether server responded with a 5xx code
	has_timed_out   bool           // wether the requests timed out
	response        *http.Response // the response as received from the server
	response_time   time.Duration  // the round trip time
	// the number of concurrent users at the moment of performing this request
	concurrent_users int
}

type ProgressEvent struct {
	Time_elapsed             time.Duration
	Total_users              int
	Average_concurrent_users float64
	Max_concurrent_users     int
	Total_hits               int
	Total_client_errors      int
	Total_server_errors      int
	Total_timeouts           int
	Average_response_time    float64
}

type Engine struct {
	External_ch              chan *ProgressEvent
	Quit_test                chan *ProgressEvent
	users_spawned            int
	current_concurrent_users int
	hits                     int // number of hits
	client_errors            int // number of client error responses
	server_errors            int // number of server error responses
	timeouts                 int // number of timeouts
	// list of responses and some stats about them
	execution_results *list.List
	// frequency at which the engine gives info to the client
	progress_update_frequency time.Duration
	Scenario                  *Scenario
	// channel to communicate with goroutines
	ch chan *httpActionResult
	// channel used by the goroutines to signal the job's done
	quit chan int
	// Tickers for repetead actions
	spawn_ticker          *time.Ticker
	test_progress_ticker  *time.Ticker
	test_completed_ticker *time.Ticker
	// Test starts at
	start_time time.Time
}

// an update is sent to the client with this frequency in seconds
const CLIENT_UPDATE_FREQUENCY = 5 * time.Second

func New() *Engine {
	engine := new(Engine)
	engine.External_ch = make(chan *ProgressEvent)
	engine.Quit_test = make(chan *ProgressEvent)
	engine.execution_results = list.New()
	engine.progress_update_frequency = CLIENT_UPDATE_FREQUENCY
	engine.ch = make(chan *httpActionResult)
	engine.quit = make(chan int)
	return engine
}

func (e *Engine) CreateScenario(users int, url string, testDuration time.Duration,
	pauseDuration time.Duration, timeoutDuration time.Duration) {
	if e.Scenario == nil {
		e.Scenario = newScenario(users, url, testDuration, pauseDuration,
			timeoutDuration)
	}
}

func (e *Engine) Run() {
	if e.Scenario == nil {
		panic("Error: a scenario has not been configured for this engine, try calling engine.CreateScenario first. Aborting...")
	}
	test_completed := false
	e.start_time = time.Now()
	e.initializeTickers()
	go func() {
		for {
			select {
			case <-e.spawn_ticker.C:
				if e.users_spawned < e.Scenario.total_users {
					e.users_spawned++
					e.current_concurrent_users++
					// spawn a new user
					go executeScenario(e.Scenario, e.ch, e.quit, e.current_concurrent_users)
				} else {
					e.spawn_ticker.Stop()
				}
			case <-e.test_progress_ticker.C:
				evt := e.createProgressEvent()
				e.External_ch <- evt
			case <-e.test_completed_ticker.C:
				test_completed = true
				e.test_progress_ticker.Stop()
				e.test_completed_ticker.Stop()
				e.spawn_ticker.Stop()
			case result := <-e.ch:
				e.execution_results.PushBack(result)
				e.updateStats(result)
			case <-e.quit:
				e.current_concurrent_users--
				if e.current_concurrent_users == 0 && test_completed == true {
					evt := e.createProgressEvent()
					e.Quit_test <- evt
					close(e.ch)
					close(e.quit)
					close(e.Quit_test)
					close(e.External_ch)
				}
			default:
				// continue
			}
		}
	}()
}

func (e *Engine) initializeTickers() {
	freq := computeSpawnFrequency(e.Scenario.test_duration.Seconds(),
		float64(e.Scenario.total_users))
	e.spawn_ticker = time.NewTicker(freq)
	e.test_progress_ticker = time.NewTicker(e.progress_update_frequency)
	e.test_completed_ticker = time.NewTicker(e.Scenario.test_duration *
		time.Second)
}

func computeSpawnFrequency(duration float64, users float64) time.Duration {
	res := (duration / users) * 1000
	str := strconv.FormatFloat(res, 'f', 5, 64)
	str = str + "ms"
	d, _ := time.ParseDuration(str)
	return d
}

func (e *Engine) updateStats(result *httpActionResult) {
	e.hits++
	if result.is_client_error == true {
		e.client_errors++
	} else if result.is_server_error == true {
		e.server_errors++
	} else if result.has_timed_out == true {
		e.timeouts++
	}
}

func executeScenario(scenario *Scenario, ch chan<- *httpActionResult, quit chan<- int, concurrent_users int) {
	// Set timeout
	var myTransport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		ResponseHeaderTimeout: scenario.pause_duration * time.Millisecond,
	}
	client := &http.Client{Transport: myTransport}
	for e := scenario.steps.Front(); e != nil; e = e.Next() {
		req := e.Value.(*http.Request)
		internal_chan := make(chan *httpActionResult)
		go executeAction(internal_chan, client, req)
		result := <-internal_chan
		result.concurrent_users = concurrent_users
		ch <- result
	}
	// Send a message over the quit channel
	quit <- 1
}

func executeAction(channel chan<- *httpActionResult, client *http.Client, req *http.Request) {
	isTimeout := false
	client_err := false
	serv_error := false
	t0 := time.Now()
	resp, err := client.Do(req)
	t1 := time.Now()

	if err != nil {
		isTimeout = true
	} else if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
		client_err = true
	} else if resp != nil && resp.StatusCode >= 500 {
		serv_error = true
	}
	result := &httpActionResult{
		is_client_error: client_err,
		is_server_error: serv_error,
		has_timed_out:   isTimeout,
		response:        resp,
		response_time:   t1.Sub(t0),
	}
	channel <- result
}

func (e *Engine) createProgressEvent() *ProgressEvent {
	time_now := time.Now()
	avg_cu, avg_rt, max_cu := e.computeAverages()
	event := &ProgressEvent{
		Time_elapsed:             time_now.Sub(e.start_time),
		Total_users:              e.users_spawned,
		Average_concurrent_users: avg_cu,
		Max_concurrent_users:     max_cu,
		Total_hits:               e.hits,
		Total_client_errors:      e.client_errors,
		Total_server_errors:      e.server_errors,
		Total_timeouts:           e.timeouts,
		Average_response_time:    avg_rt,
	}
	return event
}

func (e *Engine) computeAverages() (float64, float64, int) {
	average_concurrent_users, average_response_time := 0.0, 0.0
	var result *httpActionResult
	var avg_rt_counter time.Duration
	max_concurrent_users := 0
	avg_cu_counter := 0.0
	counter := 0.0
	if e.execution_results.Len() > 0 {
		for elem := e.execution_results.Front(); elem != nil; elem = elem.Next() {
			result = elem.Value.(*httpActionResult)
			avg_cu_counter += float64(result.concurrent_users)
			if result.concurrent_users > max_concurrent_users {
				max_concurrent_users = result.concurrent_users
			}
			if result.is_client_error == false &&
				result.is_server_error == false &&
				result.has_timed_out == false {
				avg_rt_counter += result.response_time
				counter++
			}
		}
		average_concurrent_users = avg_cu_counter / float64(e.execution_results.Len())
		average_response_time = float64(avg_rt_counter*time.Millisecond) / counter
	}
	return average_concurrent_users, average_response_time,
		max_concurrent_users
}

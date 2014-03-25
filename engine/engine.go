package engine

import (
	"http"
	"container/list"
	"time"
)

type httpActionResult struct {
	is_client_error		bool, // wether server responded with a 4xx
	is_server_error		bool, // wether server responded with a 5xx code
	has_timed_out 		bool, // wether the requests timed out
	response 			*Response, // the response as received from the server
	response_time 		time.Duration, // the round trip time
	// the number of concurrent users at the moment of performing this request
	concurrent_users 	int 
}

type ProgressEvent struct {
	Time_elapsed 				float64
	Total_users 				int
	Average_concurrent_users 	float64
	Max_concurrent_users		int
	Total_hits 					int
	Total_client_errors 		int
	Total_server_errors 		int
	Total_timeouts 				int
	Average_response_time 		float64
}

type Engine struct {
	External_ch 				chan ProgressEvent
	Quit_test					chan ProgressEvent
	users_spawned 				int
	current_concurrent_users 	int
	hits 						int // number of hits
	client_errors 				int // number of client error responses 
	server_errors 				int // number of server error responses 
	timeouts 					int // number of timeouts
	// list of responses and some stats about them
	execution_results 			*List 
	// frequency at which the engine gives info to the client
	progress_update_frequency 	float64 
	scenario 					*Scenario
	// channel to communicate with goroutines
	ch 							chan *httpActionResult 
	// channel used by the goroutines to signal the job's done
	quit 						chan int 
	// Tickers for repetead actions	
	spawn_ticker 				*Ticker 
	test_progress_ticker 		*Ticker
	test_completed_ticker 		*Ticker
	// Test starts at
	start_time 					Time
}

// an update is sent to the client with this frequency in seconds
const CLIENT_UPDATE_FREQUENCY = 5 * time.Second 


func New() *Engine {
	engine := new(Engine)	
	engine.External_ch = make(chan ProgressEvent)
	engine.Quit_test = make(chan ProgressEvent)
	engine.execution_results = list.New()
	engine.progress_update_frequency = CLIENT_UPDATE_FREQUENCY
	engine.ch = make(chan *httpActionResult)
	engine.quit = make(chan int)
	return engine
}

func (e *Engine) CreateScenario(users int, url string, testDuration int, 
	pauseDuration int, timeoutDuration float64, stepsCount int) *Scenario {
	if (e.scenario != nil) {
		return e.scenario
	}
	else {
		s := NewScenario(users, url, testDuration, pauseDuration, 
			timeoutDuration, stepsCount)
		e.scenario = s
		return s
	}	
}

func (e *Engine) Run() {
	if e.scenario == nil {
		panic("Error: a scenario has not been configured for this engine, 
			try calling engine.CreateScenario first. Aborting...")
	}
	test_completed := false
	e.start_time = time.Now()
	e.initializeTickers()
	go func() {
		for {
			select {
			case x := <-e.spawn_ticker.C:
				if e.users_spawned < e.scenario.total_users {
					e.users_spawned++
					e.current_concurrent_users++
					// spawn a new user
					go executeScenario(e.scenario, e.ch<-, e.quit<-, 
										e.current_concurrent_users)
				}
				else {
					e.spawn_ticker.Stop()
				}
			case x := <-e.test_progress_ticker.C:
				evt := e.createProgressEvent()				
				external_ch<- evt
			case x := <-e.test_completed_ticker.C:
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
					quit_test<- evt
				}
			default: 
				// continue
			}
		}
	}()
}


func (e *Engine) initializeTickers() {
	freq := computeSpawnFrequency(float64(e.scenario.test_duration), 
		float64(e.scenario.total_users)) * 1000000
	e.spawn_ticker = time.NewTicker()
	e.test_progress_ticker = time.NewTicker(CLIENT_UPDATE_FREQUENCY)
	e.test_completed_ticker = time.NewTicker(e.scenario.test_duration * 
		time.Second)
}

func computeSpawnFrequency(float64 duration, float64 users) float64 {
	return (duration / users) * 1000
}

func (e *Engine) updateStats(result *httpActionResult) {
	e.hits++
	if result.is_client_error == true {
		e.client_errors++
	}
	else if result.is_server_error == true {
		e.server_errors++
	}
	else if result.has_timed_out == true {
		e.timeouts++
	}
}

// TODO Move this into scenario and make it an instance method
func executeScenario(scenario *Scenario, ch chan<- *httpActionResult, 
	quit chan<- int, concurrent_users int) {
	// Set timeout
	var myTransport http.RoundTripper = &http.Transport{
        Proxy:                 http.ProxyFromEnvironment,
        ResponseHeaderTimeout: scenario.pause_duration * time.Millisecond,
	}
	client := &http.Client{Transport: myTransport}
	for _, req := range scenario.steps {
		internal_chan := make(chan *httpActionResult)
		go executeAction(internal_chan, client, req)
		result := <-internal_chan
		result.concurrent_users = concurrent_users
		ch<- result
	}
	// Send a message over the quit channel
	quit<- 1
}
// TODO Move it into scenario
func executeAction(channel chan<- *httpActionResult, client *Client, 
	req *http.Request)  {
	isTimeout := false
	client_err := false
	serv_error := false
	t0 := time.Now()
	resp, err := client.Do(req)
	t1 := time.Now()	
	
	if err != nil {
		isTimeout = true
	}	
	else if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
		client_err = true
	}	
	else if resp != nil && resp.StatusCode >= 500 {
		serv_error = true
	}
	result := &httpActionResult{
		is_client_error: client_err
		is_server_error: serv_error
		has_timed_out: isTimeout,
		response: resp,
		response_time: t1.Sub(t0)
	}
	channel<- result
}

func (e *Engine) createProgressEvent() *ProgressEvent {
	time_now := time.Now()
	avg_cu, avg_rt, max_cu = e.computeAverages()
	event := &ProgressEvent{
		Time_elapsed: time_now.Sub(e.start_time),
		Total_users: e.users_spawned,
		Average_concurrent_users: avg_cu,
		Max_concurrent_users: max_cu,
		Total_hits: e.hits,
		Total_client_errors: e.client_errors,
		Total_server_errors: e.server_errors,
		Total_timeouts: e.timeouts,
		Average_response_time: avg_rt
	}
}

func (e *Engine) computeAverages() (float64, float64, int) {
	average_concurrent_users, average_response_time := 0.0, 0.0
	var result *httpActionResult
	var avg_rt_counter time.Duration
	max_concurrent_users = 0
	avg_cu_counter := 0.0
	counter := 0.0
	if e.execution_results.len() > 0 {
		for e := l.Front(); e != nil; e = e.Next() {
			result = e.Value
			avg_cu_counter += float64(result.concurrent_users)
			if result.concurrent_users > max_concurrent_users {
				max_concurrent_users = result.concurrent_users
			}
			if e.is_client_error == false && 
			   		e.is_server_error == false &&
						e.has_timed_out == false {				
				avg_rt_counter += result.response_time
				counter++
			}			
		}
		average_concurrent_users = avg_cu_counter/e.execution_results.len()
		average_response_time = float64(avg_rt_counter)/counter
		return average_concurrent_users, average_response_time, 
				max_concurrent_users
	}
}
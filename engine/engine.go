package engine

import (
	"http"
	"container/list"
)

type httpActionResult struct {
	has_timed_out 		bool, // wether the requests timed out
	response 			*Response, // the response as received from the server
	response_time 		float64, // the round trip time
	concurrent_users 	int // the number of concurrent users at the moment of performing this request
}

// type of the message that is sent over the channel ch
type messageWrapper struct {
	goroutine_id 	int,
	action_result 	*httpActionResult
}

type ProgressEvent struct {
	time_elapsed 				float64
	total_users 				int
	average_concurrent_users 	float64
	total_hits 					int
	total_errors 				int
	total_timeouts 				int
	average_response_time 		float64
}

type CompletedEvent struct {
	time_elapsed 				float64
	total_users 				int
	average_concurrent_users 	float64
	max_concurrent_users		int
	total_hits 					int
	total_errors 				int
	total_timeouts 				int
	average_response_time 		float64
}

type Engine struct {
	external_ch chan ProgressEvent
	quit_test chan CompletedEvent
	hits int // number of hits
	errors int // number of error responses 
	timeouts int // number of timeouts
	execution_results *List // list of responses and some stats about them
	progress_update_frequency float64 // frequency at which the engine gives info to the client
	scenario *Scenario
}

const CLIENT_UPDATE_FREQUENCY = 5000 // an update is sent to the client with this frequency in milliseconds
var test_scenario Scenario // the current scenario
var ch chan httpActionResult // channel to communicate with goroutines
var quit chan int // channel used by the goroutines to signal the job's done
var next_goroutine_id int = 0 // identifier for the next goroutine spawned 

func New() *Engine {
	external_ch := make(chan ProgressEvent)
	quit_test := make(chan CompletedEvent)
	execution_results := list.New()
	progress_update_frequency = CLIENT_UPDATE_FREQUENCY	
}

func (e *Engine) CreateScenario(users int, url string, testDuration int, pauseDuration float64, timeoutDuration float64, stepsCount int) *Scenario {
	if (e.scenario != nil) {
		return e.scenario
	}
	else {
		return NewScenario(users, url, testDuration, pauseDuration, timeoutDuration, stepsCount)	
	}	
}



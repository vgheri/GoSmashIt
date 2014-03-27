package engine

import (
	"testing"
	"time"
	"container/list"
)

type testpair struct {
	execution_results *list.List
	average_concurrent_users float64
	average_response_time float64
	max_concurrent_users int
}

func get_duration(d string) time.Duration {
	duration, _ := time.ParseDuration(d)
	return duration
}

var tests []testpair 

func TestComputeAverages(t *testing.T) {	
	createTestData()
	for _, pair := range tests {
		avg_cu, avg_rt, max_cu := computeAverages(pair.execution_results)
		if avg_cu != 5.25 || avg_rt != 58 || max_cu != 9 {
			t.Error(
				"For TestComputeAverages",
				"expected", pair.average_concurrent_users, pair.average_response_time, pair.max_concurrent_users,
				"got", avg_cu, avg_rt, max_cu,
			)
		}
	} 
}

func createTestData() {
	l1 := list.New()
	l1.PushBack(&httpActionResult{
	is_client_error: false,
	is_server_error: false,
	has_timed_out: false,			
	response_time: get_duration("74ms"),				
	concurrent_users: 2,
	})
	l1.PushBack(&httpActionResult{
	is_client_error: false,
	is_server_error: false,
	has_timed_out: false,			
	response_time: get_duration("47ms"),				
	concurrent_users: 4,
	})
	l1.PushBack(&httpActionResult{
	is_client_error: false,
	is_server_error: false,
	has_timed_out: false,			
	response_time: get_duration("51ms"),				
	concurrent_users: 6,
	})
	l1.PushBack(&httpActionResult{
	is_client_error: false,
	is_server_error: false,
	has_timed_out: false,			
	response_time: get_duration("60ms"),				
	concurrent_users: 9,
	})

	tests = []testpair{
		{
			execution_results: l1,
			average_concurrent_users: 5.25,
			average_response_time: 58,
			max_concurrent_users: 9,
		},
	}
}


package engine

import (
	"bytes"
	"encoding/base64"
	"http"
	"strings"	
)

type Scenario struct {
	total_users int,
	base_address string,
	test_duration int, // test duration expressed in seconds
	pause_duration float64, //length of pause between steps, expressed in seconds (e.g. 2.5)
	timeout float64, // amount of milliseconds to wait before considering the request as timed out
	steps []*Request // the load test execution plan
}

func newScenario(users int, url string, testDuration int, pauseDuration float64, timeoutDuration float64, stepsCount int) *Scenario {
	s = &Scenario{
		total_users: users,
		url: base_address,
		test_duration: testDuration,
		pause_duration: pauseDuration,
		timeout: timeoutDuration,
		steps: make([]*Request, 0, stepsCount)
	}
	return s
}

func (s *Scenario) AddStep(headers map[string][string], method string, endpoint string, content string) (*Scenario, error) {
	method = ToUpper(method)
	url = s.base_address + endpoint
	var req *Request
	var err error
	if content == nil {
		req, err = http.NewRequest(method, url, nil)
	}
	else {
		buf := bytes.NewBufferString(content)
		dec := base64.NewDecoder(base64.StdEncoding, buf)
		req, err =  http.NewRequest(method, url, dec)
	}
	if err != nil {
		return nil, err
	}
	// For each header (if any)
	if (headers != nil && len(headers) > 0) {
		//req.Header.Set("Content-Type", bodyType)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	return s, nil	
}
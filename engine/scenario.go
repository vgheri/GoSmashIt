package engine

import (
	"bytes"
	"container/list"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

type Scenario struct {
	total_users    int
	base_address   string
	test_duration  time.Duration // test duration
	pause_duration time.Duration //length of pause between steps,
	timeout        time.Duration // amount of time  to wait before considering the request as timed out
	steps          *list.List    // the load test execution plan
}

func newScenario(users int, url string, testDuration time.Duration, pauseDuration time.Duration, timeoutDuration time.Duration) *Scenario {
	s := &Scenario{
		total_users:    users,
		base_address:   url,
		test_duration:  testDuration,
		pause_duration: pauseDuration,
		timeout:        timeoutDuration,
		steps:          list.New(),
	}
	return s
}

func (s *Scenario) AddStep(headers map[string]string, method string, endpoint string, content string) error {
	method = strings.ToUpper(method)
	url := s.base_address + endpoint
	var req *http.Request
	var err error
	if len(content) == 0 {
		req, err = http.NewRequest(method, url, nil)
	} else {
		buf := bytes.NewBufferString(content)
		dec := base64.NewDecoder(base64.StdEncoding, buf)
		req, err = http.NewRequest(method, url, dec)
	}
	if err != nil {
		return err
	}
	// For each header (if any)
	if headers != nil && len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	s.steps.PushBack(req)
	return nil
}

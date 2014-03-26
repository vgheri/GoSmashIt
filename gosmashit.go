package main

import (
	"fmt"
	"github.com/vgheri/GoSmashIt/engine"
	"time"
)

func main() {
	users := 200
	address := "http://localhost/"
	test_duration := 60 * time.Second
	timeout := 3000 * time.Millisecond
	pause_duration := 3000 * time.Millisecond
	engine := engine.New()
	engine.CreateScenario(users, address, test_duration,
		pause_duration, timeout)
	engine.Scenario.AddStep(nil, "GET", "", "")
	engine.Run()
	for {
		select {
		case update := <-engine.External_ch:
			printEvent(update, false)
		case complete := <-engine.Quit_test:
			printEvent(complete, true)
			return
		}
	}
}

func printEvent(event *engine.ProgressEvent, test_completed bool) {
	if test_completed == false {
		fmt.Println("Simulation progress:")
	} else {
		fmt.Println("Simulation completed:")
	}
	fmt.Printf("Total number of users: %d\n", event.Total_users)
	fmt.Printf("Test duration: %v\n", event.Time_elapsed)
	fmt.Printf("Max number of concurrent users: %d\n",
		event.Max_concurrent_users)
	fmt.Printf("Average number of concurrent users: %v\n",
		event.Average_concurrent_users)
	fmt.Printf("Average response time: %v ms\n", event.Average_response_time)
	fmt.Printf("Total hits: %d\n", event.Total_hits)
	fmt.Printf("Total client errors: %d\n", event.Total_client_errors)
	fmt.Printf("Total server errors: %d\n", event.Total_server_errors)
	fmt.Printf("Total timeouts: %d\n", event.Total_timeouts)
}

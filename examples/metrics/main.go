package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ambiyansyah-risyal/klayengo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Create a client with metrics enabled
	client := klayengo.New(
		klayengo.WithMetrics(),
		klayengo.WithMaxRetries(3),
		klayengo.WithCache(5*time.Minute),
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  10 * time.Second,
			SuccessThreshold: 2,
		}),
	)

	// Start metrics server in background
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Metrics server starting on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Create a test server that sometimes fails
	testServer := &http.Server{
		Addr: ":8081",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate occasional failures
			if time.Now().Unix()%5 == 0 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}),
	}

	go func() {
		log.Println("Test server starting on :8081")
		log.Fatal(testServer.ListenAndServe())
	}()

	// Wait for servers to start
	time.Sleep(2 * time.Second)

	// Make some requests to generate metrics
	fmt.Println("Making requests to generate metrics...")

	for i := 0; i < 10; i++ {
		resp, err := client.Get(context.Background(), "http://localhost:8081/test")
		if err != nil {
			fmt.Printf("Request %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("Request %d succeeded with status: %s\n", i+1, resp.Status)
			resp.Body.Close()
		}

		// Wait a bit between requests
		time.Sleep(1 * time.Second)
	}

	// Make some cached requests
	fmt.Println("\nMaking cached requests...")
	for i := 0; i < 3; i++ {
		resp, err := client.Get(context.Background(), "http://localhost:8081/test")
		if err != nil {
			fmt.Printf("Cached request %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("Cached request %d succeeded with status: %s\n", i+1, resp.Status)
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\nMetrics are now available at http://localhost:8080/metrics")
	fmt.Println("You can view them with: curl http://localhost:8080/metrics")

	// Keep the program running to allow metrics inspection
	fmt.Println("Press Ctrl+C to exit")
	select {}
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ambiyansyah-risyal/klayengo"
)

func main() {
	// Display version information
	fmt.Printf("Klayengo Version: %s\n\n", klayengo.GetVersion())

	// Create a retry client with enhanced error handling and debugging
	client := klayengo.New(
		klayengo.WithMaxRetries(3),
		klayengo.WithInitialBackoff(200*time.Millisecond),
		klayengo.WithMaxBackoff(2*time.Second),
		klayengo.WithJitter(0.1),                   // Add 10% jitter
		klayengo.WithRateLimiter(5, 1*time.Second), // 5 requests per second
		klayengo.WithCache(1*time.Minute),          // Cache responses for 1 minute
		klayengo.WithCacheCondition(func(req *http.Request) bool {
			// Only cache GET requests to httpbin.org
			return req.Method == "GET" && req.URL.Host == "httpbin.org"
		}),
		klayengo.WithTimeout(10*time.Second),
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  5 * time.Second,
			SuccessThreshold: 1,
		}),
		klayengo.WithSimpleLogger(), // Enable debug logging
		klayengo.WithMiddleware(loggingMiddleware),
	)

	// Example 1: Simple GET request
	fmt.Println("=== Example 1: Simple GET request ===")
	resp, err := client.Get(context.Background(), "https://httpbin.org/get")
	if err != nil {
		log.Printf("GET request failed: %v", err)
	} else {
		fmt.Printf("GET request succeeded with status: %s\n", resp.Status)
		resp.Body.Close()
	}

	// Example 2: POST request
	fmt.Println("\n=== Example 2: POST request ===")
	resp, err = client.Post(context.Background(), "https://httpbin.org/post", "application/json", nil)
	if err != nil {
		log.Printf("POST request failed: %v", err)
	} else {
		fmt.Printf("POST request succeeded with status: %s\n", resp.Status)
		resp.Body.Close()
	}

	// Example 3: Custom request with context timeout
	fmt.Println("\n=== Example 3: Custom request with context timeout ===")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/delay/1", nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return
	}

	resp, err = client.Do(req)
	if err != nil {
		log.Printf("Custom request failed: %v", err)
	} else {
		fmt.Printf("Custom request succeeded with status: %s\n", resp.Status)
		resp.Body.Close()
	}

	fmt.Println("\n=== Example completed ===")
}

// loggingMiddleware is an example middleware that logs requests
func loggingMiddleware(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
	start := time.Now()
	fmt.Printf("Starting request to %s %s\n", req.Method, req.URL)

	resp, err := next.RoundTrip(req)

	duration := time.Since(start)
	if err != nil {
		fmt.Printf("Request failed after %v: %v\n", duration, err)
	} else {
		fmt.Printf("Request completed in %v with status %s\n", duration, resp.Status)
	}

	return resp, err
}

package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	klayengo "github.com/ambiyansyah-risyal/klayengo"
)

func main() {
	// Create a client with deduplication enabled
	client := klayengo.New(
		klayengo.WithDeduplication(),
		klayengo.WithDebug(),
		klayengo.WithSimpleLogger(),
	)

	// Create a test server that simulates slow responses
	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("Server: Handling request at %s\n", time.Now().Format("15:04:05.000"))
			time.Sleep(2 * time.Second) // Simulate slow response
			w.WriteHeader(200)
			w.Write([]byte(fmt.Sprintf("Response at %s", time.Now().Format("15:04:05.000"))))
		}),
	}

	// Start server in background
	go server.ListenAndServe()
	defer server.Close()

	time.Sleep(100 * time.Millisecond) // Let server start

	var wg sync.WaitGroup
	results := make([]string, 5)

	// Make 5 concurrent requests to the same endpoint
	fmt.Println("Making 5 concurrent requests...")
	start := time.Now()

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://localhost:8080/test", nil)
			resp, err := client.Do(req)

			if err != nil {
				results[index] = fmt.Sprintf("Error: %v", err)
				return
			}

			body := make([]byte, 100)
			n, _ := resp.Body.Read(body)
			results[index] = string(body[:n])
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(start)

	fmt.Printf("All requests completed in %v\n", totalTime)
	fmt.Println("Results:")
	for i, result := range results {
		fmt.Printf("  Request %d: %s\n", i+1, result)
	}

	// The deduplication should ensure that only 1 actual request was made to the server
	// while all 5 goroutines got the same response
	fmt.Println("\nWith deduplication enabled:")
	fmt.Println("- Only 1 request actually hits the server")
	fmt.Println("- The other 4 requests wait for the first one to complete")
	fmt.Println("- All requests return the same response")
}

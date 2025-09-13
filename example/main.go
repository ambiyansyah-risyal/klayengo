// Example demonstrating klayengo HTTP client features including
// retry, rate limiting, caching, circuit breaker, deduplication, metrics, and middleware.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ambiyansyah-risyal/klayengo"
)

// API endpoints used in examples
const (
	httpbinJSON      = "https://httpbin.org/json"
	httpbinUUID      = "https://httpbin.org/uuid"
	httpbinPost      = "https://httpbin.org/post"
	httpbinUserAgent = "https://httpbin.org/user-agent"
	httpbinStatus200 = "https://httpbin.org/status/200"
	httpbinStatus201 = "https://httpbin.org/status/201"
	httpbinStatus500 = "https://httpbin.org/status/500"
)

func main() {
	fmt.Println("🚀 klayengo - Resilient HTTP Client Example")
	fmt.Println(strings.Repeat("=", 50))

	// Example 1: Basic client with all features enabled
	basicExample()

	// Example 2: Advanced client with custom middleware and logging
	advancedExample()

	// Example 3: Production-ready client with metrics
	productionExample()

	// Example 4: Specialized clients for different scenarios
	specializedClientsExample()
}

// basicExample demonstrates a client with all core features enabled
func basicExample() {
	fmt.Println("\n📝 Example 1: Basic Client with All Features")
	fmt.Println(strings.Repeat("-", 40))

	client := klayengo.New(
		klayengo.WithMaxRetries(3),
		klayengo.WithInitialBackoff(100*time.Millisecond),
		klayengo.WithMaxBackoff(5*time.Second),
		klayengo.WithBackoffMultiplier(2.0),
		klayengo.WithJitter(0.1),
		klayengo.WithRateLimiter(10, time.Second),
		klayengo.WithCache(5*time.Minute),
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 3,
		}),
		klayengo.WithDeduplication(),
		klayengo.WithSimpleLogger(),
		klayengo.WithDebug(),
		klayengo.WithTimeout(30*time.Second),
	)

	if !client.IsValid() {
		log.Fatalf("Client configuration is invalid: %v", client.ValidationError())
	}
	ctx := context.Background()
	resp, err := client.Get(ctx, httpbinJSON)
	if err != nil {
		fmt.Printf("❌ GET request failed: %v\n", err)
	} else {
		fmt.Printf("✅ GET request successful: Status %d\n", resp.StatusCode)
		resp.Body.Close()
	}

	payload := map[string]interface{}{
		"message":   "Hello from klayengo!",
		"timestamp": time.Now().Unix(),
	}
	jsonPayload, _ := json.Marshal(payload)

	postResp, err := client.Post(ctx, httpbinPost,
		"application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("❌ POST request failed: %v\n", err)
	} else {
		fmt.Printf("✅ POST request successful: Status %d\n", postResp.StatusCode)
		postResp.Body.Close()
	}
}

// advancedExample demonstrates middleware, custom configurations, and advanced logging
func advancedExample() {
	fmt.Println("\n🔧 Example 2: Advanced Client with Middleware & Custom Configs")
	fmt.Println(strings.Repeat("-", 55))

	headerMiddleware := func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
		req.Header.Set("User-Agent", "klayengo-example/1.0")
		req.Header.Set("X-Client-Version", "advanced")

		start := time.Now()
		resp, err := next.RoundTrip(req)
		duration := time.Since(start)

		fmt.Printf("🕒 Request to %s took %v\n", req.URL.Host, duration)
		return resp, err
	}

	authMiddleware := func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
		req.Header.Set("Authorization", "Bearer demo-token-12345")
		return next.RoundTrip(req)
	}
	client := klayengo.New(
		klayengo.WithMaxRetries(5),
		klayengo.WithInitialBackoff(200*time.Millisecond),
		klayengo.WithMaxBackoff(10*time.Second),
		klayengo.WithBackoffMultiplier(1.5),
		klayengo.WithJitter(0.25),
		klayengo.WithRetryCondition(func(resp *http.Response, err error) bool {
			if err != nil {
				return true
			}
			return resp.StatusCode >= 500
		}),
		klayengo.WithRateLimiter(5, 2*time.Second),
		klayengo.WithCache(2*time.Minute),
		klayengo.WithCacheCondition(func(req *http.Request) bool {
			return req.Method == "GET" && req.Header.Get("Authorization") == ""
		}),
		klayengo.WithCacheKeyFunc(func(req *http.Request) string {
			return fmt.Sprintf("%s:%s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
		}),
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  10 * time.Second,
			SuccessThreshold: 2,
		}),
		klayengo.WithMiddleware(headerMiddleware, authMiddleware),
		klayengo.WithDeduplicationCondition(func(req *http.Request) bool {
			return req.Method == "GET" || req.Method == "HEAD" ||
				req.Method == "OPTIONS" || req.Method == "POST"
		}),

		// Enable detailed debugging
		klayengo.WithDebugConfig(&klayengo.DebugConfig{
			Enabled:      true,
			LogRequests:  true,
			LogRetries:   true,
			LogCache:     true,
			LogRateLimit: true,
			LogCircuit:   true,
			RequestIDGen: func() string {
				return fmt.Sprintf("adv_%d", time.Now().UnixNano())
			},
		}),
		klayengo.WithSimpleLogger(),

		// Custom timeout
		klayengo.WithTimeout(45*time.Second),
	)

	ctx := context.Background()

	// Test with multiple requests to demonstrate caching and deduplication
	fmt.Println("\n🔄 Testing caching and deduplication:")
	for i := 0; i < 3; i++ {
		resp, err := client.Get(ctx, httpbinUUID)
		if err != nil {
			fmt.Printf("❌ Request %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("✅ Request %d successful: Status %d\n", i+1, resp.StatusCode)
			resp.Body.Close()
		}
	}

	// Test POST with authentication middleware
	fmt.Println("\n🔐 Testing POST with authentication:")
	postData := map[string]interface{}{
		"user_id": 123,
		"action":  "advanced_example",
		"data": map[string]string{
			"feature": "middleware",
			"version": "v1.0",
		},
	}
	jsonData, _ := json.Marshal(postData)

	resp, err := client.Post(ctx, httpbinPost,
		"application/json", bytes.NewReader(jsonData))
	if err != nil {
		fmt.Printf("❌ Authenticated POST failed: %v\n", err)
	} else {
		fmt.Printf("✅ Authenticated POST successful: Status %d\n", resp.StatusCode)
		resp.Body.Close()
	}
}

// productionExample demonstrates a production-ready client with metrics
func productionExample() {
	fmt.Println("\n📊 Example 3: Production Client with Metrics")
	fmt.Println(strings.Repeat("-", 45))

	// Create metrics collector
	metricsCollector := klayengo.NewMetricsCollector()

	// Production-ready client configuration
	client := klayengo.New(
		// Conservative retry settings for production
		klayengo.WithMaxRetries(3),
		klayengo.WithInitialBackoff(100*time.Millisecond),
		klayengo.WithMaxBackoff(30*time.Second),
		klayengo.WithBackoffMultiplier(2.0),
		klayengo.WithJitter(0.1),

		// Rate limiting based on API limits
		klayengo.WithRateLimiter(100, time.Minute), // 100 requests per minute

		// Longer cache TTL for production
		klayengo.WithCache(15*time.Minute),

		// Circuit breaker tuned for stability
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 10,               // Higher threshold for production
			RecoveryTimeout:  60 * time.Second, // Longer recovery time
			SuccessThreshold: 5,                // More successes needed to close
		}),

		// Enable deduplication for efficiency
		klayengo.WithDeduplication(),

		// Add metrics collection
		klayengo.WithMetricsCollector(metricsCollector),

		// Production timeout
		klayengo.WithTimeout(60*time.Second),

		// Minimal logging for production (you might use structured logging)
		klayengo.WithLogger(&ProductionLogger{}),
	)

	// Simulate production usage
	ctx := context.Background()

	fmt.Println("\n📈 Simulating production traffic:")

	// Multiple API calls to generate metrics
	endpoints := []string{
		httpbinStatus200,
		httpbinStatus201,
		httpbinJSON,
		httpbinUUID,
		httpbinUserAgent,
	}

	for i, endpoint := range endpoints {
		resp, err := client.Get(ctx, endpoint)
		if err != nil {
			fmt.Printf("❌ Request %d to %s failed: %v\n", i+1, endpoint, err)
		} else {
			fmt.Printf("✅ Request %d to %s: Status %d\n", i+1, endpoint, resp.StatusCode)
			resp.Body.Close()
		}

		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	// Test error handling with metrics
	fmt.Println("\n🚨 Testing error scenarios:")
	_, err := client.Get(ctx, httpbinStatus500)
	if err != nil {
		fmt.Printf("⚠️  Expected error for 500 status: %v\n", err)
	}

	fmt.Println("\n📊 Metrics would be available at /metrics endpoint in a real application")
	fmt.Println("   Registry contains counters for requests, errors, cache hits/misses, etc.")
}

// specializedClientsExample shows different client configurations for different use cases
func specializedClientsExample() {
	fmt.Println("\n🎯 Example 4: Specialized Clients for Different Use Cases")
	fmt.Println(strings.Repeat("-", 58))

	ctx := context.Background()

	// High-frequency trading client - optimized for speed and low latency
	fmt.Println("\n⚡ High-Performance Client (optimized for speed):")
	fastClient := klayengo.New(
		klayengo.WithMaxRetries(1),                       // Fail fast
		klayengo.WithInitialBackoff(10*time.Millisecond), // Very short backoff
		klayengo.WithMaxBackoff(100*time.Millisecond),
		klayengo.WithRateLimiter(1000, time.Second), // High rate limit
		klayengo.WithTimeout(2*time.Second),         // Short timeout
		klayengo.WithCache(30*time.Second),          // Short-lived cache
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 2,               // Open quickly
			RecoveryTimeout:  5 * time.Second, // Recover quickly
		}),
		klayengo.WithDeduplication(), // Still deduplicate for efficiency
	)

	resp, err := fastClient.Get(ctx, httpbinUUID)
	if err != nil {
		fmt.Printf("❌ Fast client request failed: %v\n", err)
	} else {
		fmt.Printf("✅ Fast client request successful: Status %d\n", resp.StatusCode)
		resp.Body.Close()
	}

	// Batch processing client - optimized for reliability and throughput
	fmt.Println("\n🔄 Batch Processing Client (optimized for reliability):")
	batchClient := klayengo.New(
		klayengo.WithMaxRetries(10),                       // Many retries
		klayengo.WithInitialBackoff(500*time.Millisecond), // Longer backoff
		klayengo.WithMaxBackoff(60*time.Second),           // Very long max backoff
		klayengo.WithBackoffMultiplier(1.5),               // Gentle exponential growth
		klayengo.WithJitter(0.2),                          // More jitter to spread load
		klayengo.WithRateLimiter(10, 10*time.Second),      // Conservative rate limiting
		klayengo.WithTimeout(300*time.Second),             // Long timeout for large operations
		klayengo.WithCache(60*time.Minute),                // Long-lived cache
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 20,                // Very tolerant
			RecoveryTimeout:  120 * time.Second, // Long recovery
			SuccessThreshold: 10,                // Many successes needed
		}),
		klayengo.WithDeduplication(),
		klayengo.WithSimpleLogger(), // Log for monitoring batch operations
	)

	resp, err = batchClient.Get(ctx, httpbinJSON)
	if err != nil {
		fmt.Printf("❌ Batch client request failed: %v\n", err)
	} else {
		fmt.Printf("✅ Batch client request successful: Status %d\n", resp.StatusCode)
		resp.Body.Close()
	}

	// External API integration client - balanced for third-party APIs
	fmt.Println("\n🌐 External API Client (balanced for third-party integrations):")
	apiClient := klayengo.New(
		klayengo.WithMaxRetries(5),
		klayengo.WithInitialBackoff(250*time.Millisecond),
		klayengo.WithMaxBackoff(15*time.Second),
		klayengo.WithBackoffMultiplier(2.0),
		klayengo.WithJitter(0.15),
		klayengo.WithRateLimiter(50, time.Minute), // Respect API rate limits
		klayengo.WithTimeout(30*time.Second),
		klayengo.WithCache(5*time.Minute), // Balance freshness vs performance
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 3,
		}),
		klayengo.WithDeduplication(),

		// Custom retry condition for API integration
		klayengo.WithRetryCondition(func(resp *http.Response, err error) bool {
			if err != nil {
				return true
			}
			// Retry on 5xx, 429 (rate limited), and 502-504 (bad gateway, etc.)
			return resp.StatusCode >= 500 || resp.StatusCode == 429 ||
				resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504
		}),

		// Add User-Agent middleware
		klayengo.WithMiddleware(func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
			req.Header.Set("User-Agent", "MyApp-Integration/1.0 (klayengo)")
			return next.RoundTrip(req)
		}),
	)

	resp, err = apiClient.Get(ctx, httpbinUserAgent)
	if err != nil {
		fmt.Printf("❌ API client request failed: %v\n", err)
	} else {
		fmt.Printf("✅ API client request successful: Status %d\n", resp.StatusCode)
		resp.Body.Close()
	}

	fmt.Println("\n🎉 All examples completed! klayengo provides flexibility for any HTTP client needs.")
}

// ProductionLogger is a simple logger for production use
type ProductionLogger struct{}

func (l *ProductionLogger) Debug(msg string, args ...interface{}) {
	// In production, you might not want debug logs, or send them to a different destination
}

func (l *ProductionLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg+"\n", args...)
}

func (l *ProductionLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] "+msg+"\n", args...)
}

func (l *ProductionLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg+"\n", args...)
}

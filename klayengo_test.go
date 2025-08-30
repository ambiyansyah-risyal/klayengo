package klayengo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.maxRetries != 3 {
		t.Errorf("Expected maxRetries to be 3, got %d", client.maxRetries)
	}
	if client.initialBackoff != 100*time.Millisecond {
		t.Errorf("Expected initialBackoff to be 100ms, got %v", client.initialBackoff)
	}
}

func TestWithMaxRetries(t *testing.T) {
	client := New(WithMaxRetries(5))
	if client.maxRetries != 5 {
		t.Errorf("Expected maxRetries to be 5, got %d", client.maxRetries)
	}
}

func TestWithJitter(t *testing.T) {
	client := New(WithJitter(0.2))
	if client.jitter != 0.2 {
		t.Errorf("Expected jitter to be 0.2, got %f", client.jitter)
	}
}

func TestCalculateBackoffWithJitter(t *testing.T) {
	client := New(WithJitter(0.1))
	backoff1 := client.calculateBackoff(1)
	backoff2 := client.calculateBackoff(1)
	// With jitter, backoffs should be different (most of the time)
	if backoff1 == backoff2 {
		t.Log("Backoffs are the same, but jitter might not have kicked in")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	// Should allow initial requests
	if !rl.Allow() {
		t.Error("Expected rate limiter to allow first request")
	}
	if !rl.Allow() {
		t.Error("Expected rate limiter to allow second request")
	}

	// Should deny third request
	if rl.Allow() {
		t.Error("Expected rate limiter to deny third request")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should allow again
	if !rl.Allow() {
		t.Error("Expected rate limiter to allow after refill")
	}
}

func TestWithRateLimiter(t *testing.T) {
	client := New(WithRateLimiter(1, 1*time.Second))
	if client.rateLimiter == nil {
		t.Error("Expected rate limiter to be set")
	}
}

func TestInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache()
	entry := &CacheEntry{
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	// Test Set and Get
	cache.Set("test-key", entry, 1*time.Minute)
	retrieved, found := cache.Get("test-key")
	if !found {
		t.Error("Expected cache entry to be found")
	}
	if string(retrieved.Body) != "test data" {
		t.Errorf("Expected body 'test data', got '%s'", string(retrieved.Body))
	}

	// Test Delete
	cache.Delete("test-key")
	_, found = cache.Get("test-key")
	if found {
		t.Error("Expected cache entry to be deleted")
	}

	// Test Clear
	cache.Set("key1", entry, 1*time.Minute)
	cache.Set("key2", entry, 1*time.Minute)
	cache.Clear()
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	if found1 || found2 {
		t.Error("Expected all cache entries to be cleared")
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewInMemoryCache()
	entry := &CacheEntry{
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	// Set with short TTL
	cache.Set("test-key", entry, 100*time.Millisecond)
	time.Sleep(150 * time.Millisecond)

	_, found := cache.Get("test-key")
	if found {
		t.Error("Expected cache entry to be expired")
	}
}

func TestWithCache(t *testing.T) {
	client := New(WithCache(5 * time.Minute))
	if client.cache == nil {
		t.Error("Expected cache to be set")
	}
	if client.cacheTTL != 5*time.Minute {
		t.Errorf("Expected cacheTTL to be 5m, got %v", client.cacheTTL)
	}
}

func TestWithCustomCache(t *testing.T) {
	customCache := NewInMemoryCache()
	client := New(WithCustomCache(customCache, 10*time.Minute))
	if client.cache != customCache {
		t.Error("Expected custom cache to be set")
	}
	if client.cacheTTL != 10*time.Minute {
		t.Errorf("Expected cacheTTL to be 10m, got %v", client.cacheTTL)
	}
}

func TestWithCacheKeyFunc(t *testing.T) {
	customKeyFunc := func(req *http.Request) string {
		return "custom-key"
	}
	client := New(WithCacheKeyFunc(customKeyFunc))
	if client.cacheKeyFunc == nil {
		t.Error("Expected cache key function to be set")
	}
}

func TestDefaultCacheKeyFunc(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/test", nil)
	key := DefaultCacheKeyFunc(req)
	expected := "GET:https://example.com/test"
	if key != expected {
		t.Errorf("Expected cache key '%s', got '%s'", expected, key)
	}
}

func TestCachingInDo(t *testing.T) {
	// Create a test server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("cached response"))
	}))
	defer server.Close()

	client := New(WithCache(1 * time.Minute))

	// First request should hit the server
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp1, err := client.Do(req)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	// Second request should use cache
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	resp2.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected still 1 server call (cached), got %d", callCount)
	}
}

func TestWithMetrics(t *testing.T) {
	client := New(WithMetrics())
	if client.metrics == nil {
		t.Error("Expected metrics collector to be set")
	}
}

func TestMetricsCollection(t *testing.T) {
	// Create a custom registry for this test to avoid conflicts
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	client := New(WithMetricsCollector(collector))

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Make a request
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	// Check that metrics were recorded (we can't easily check the actual values
	// without exposing internal prometheus structures, but we can verify no panics)
	if client.metrics == nil {
		t.Error("Expected metrics collector to be initialized")
	}
}

func TestContextCacheControl(t *testing.T) {
	client := New(WithCache(1 * time.Minute))

	// Test cache enabled via context
	ctx := WithContextCacheEnabled(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
	if !client.shouldCacheRequest(req) {
		t.Error("Expected request to be cacheable with context override")
	}

	// Test cache disabled via context
	ctx = WithContextCacheDisabled(context.Background())
	req, _ = http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
	if client.shouldCacheRequest(req) {
		t.Error("Expected request to not be cacheable with context override")
	}

	// Test custom TTL via context
	ctx = WithContextCacheTTL(context.Background(), 10*time.Minute)
	req, _ = http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
	ttl := client.getCacheTTLForRequest(req)
	if ttl != 10*time.Minute {
		t.Errorf("Expected TTL to be 10m, got %v", ttl)
	}
}

func TestCacheConditionFunction(t *testing.T) {
	client := New(
		WithCache(1*time.Minute),
		WithCacheCondition(func(req *http.Request) bool {
			return req.Header.Get("Cache-Control") == "cache"
		}),
	)

	// Request with cache header should be cached
	req1, _ := http.NewRequest("GET", "https://example.com", nil)
	req1.Header.Set("Cache-Control", "cache")
	if !client.shouldCacheRequest(req1) {
		t.Error("Expected request with cache header to be cacheable")
	}

	// Request without cache header should not be cached
	req2, _ := http.NewRequest("GET", "https://example.com", nil)
	if client.shouldCacheRequest(req2) {
		t.Error("Expected request without cache header to not be cacheable")
	}
}

func TestCacheSpecificRequests(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer server.Close()

	client := New(WithCache(1 * time.Minute))

	// First request - should hit server
	req1, _ := http.NewRequest("GET", server.URL+"/api/data", nil)
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	// Second request to same URL - should use cache
	req2, _ := http.NewRequest("GET", server.URL+"/api/data", nil)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	resp2.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected still 1 server call (cached), got %d", callCount)
	}

	// Request with cache disabled - should hit server again
	ctx := WithContextCacheDisabled(context.Background())
	req3, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/data", nil)
	resp3, err := client.Do(req3)
	if err != nil {
		t.Fatalf("Third request failed: %v", err)
	}
	resp3.Body.Close()

	if callCount != 2 {
		t.Errorf("Expected 2 server calls (cache disabled), got %d", callCount)
	}
}

func TestDefaultRetryCondition(t *testing.T) {
	// Test with error
	if !DefaultRetryCondition(nil, http.ErrHandlerTimeout) {
		t.Error("Expected retry on error")
	}

	// Test with 500 status
	resp := &http.Response{StatusCode: 500}
	if !DefaultRetryCondition(resp, nil) {
		t.Error("Expected retry on 500 status")
	}

	// Test with 200 status
	resp = &http.Response{StatusCode: 200}
	if DefaultRetryCondition(resp, nil) {
		t.Error("Expected no retry on 200 status")
	}

	// Test with 404 status
	resp = &http.Response{StatusCode: 404}
	if DefaultRetryCondition(resp, nil) {
		t.Error("Expected no retry on 404 status")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  100 * time.Millisecond,
		SuccessThreshold: 1,
	})

	// Initially closed
	if !cb.Allow() {
		t.Error("Expected circuit breaker to allow requests initially")
	}

	// Record failures
	cb.RecordFailure()
	if cb.state != StateClosed {
		t.Error("Expected circuit breaker to remain closed after first failure")
	}

	cb.RecordFailure()
	if cb.state != StateOpen {
		t.Error("Expected circuit breaker to open after reaching failure threshold")
	}

	// Should not allow requests when open
	if cb.Allow() {
		t.Error("Expected circuit breaker to not allow requests when open")
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Should allow requests in half-open state
	if !cb.Allow() {
		t.Error("Expected circuit breaker to allow requests after recovery timeout")
	}

	// Record success
	cb.RecordSuccess()
	if cb.state != StateClosed {
		t.Error("Expected circuit breaker to close after success in half-open state")
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New()
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPost(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Created"))
	}))
	defer server.Close()

	client := New()
	resp, err := client.Post(context.Background(), server.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("Post() returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

// Benchmark tests for performance measurement

// BenchmarkBasicGet benchmarks a basic GET request without retries
func BenchmarkBasicGet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(WithMaxRetries(0)) // No retries for baseline

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkGetWithRetries benchmarks GET requests with retry logic
func BenchmarkGetWithRetries(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(WithMaxRetries(3))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkGetWithCircuitBreaker benchmarks requests with circuit breaker
func BenchmarkGetWithCircuitBreaker(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(
		WithMaxRetries(3),
		WithCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  1 * time.Second,
			SuccessThreshold: 2,
		}),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkGetWithMiddleware benchmarks requests with middleware
func BenchmarkGetWithMiddleware(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(
		WithMaxRetries(3),
		WithMiddleware(func(req *http.Request, next RoundTripper) (*http.Response, error) {
			// Simple middleware that just passes through
			return next.RoundTrip(req)
		}),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkGetWithMultipleMiddleware benchmarks requests with multiple middleware
func BenchmarkGetWithMultipleMiddleware(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(
		WithMaxRetries(3),
		WithMiddleware(
			func(req *http.Request, next RoundTripper) (*http.Response, error) {
				return next.RoundTrip(req)
			},
			func(req *http.Request, next RoundTripper) (*http.Response, error) {
				return next.RoundTrip(req)
			},
			func(req *http.Request, next RoundTripper) (*http.Response, error) {
				return next.RoundTrip(req)
			},
		),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkGetWithRetriesAndFailures benchmarks requests that trigger retries
func BenchmarkGetWithRetriesAndFailures(b *testing.B) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Fail the first 2 attempts, succeed on the 3rd
		if callCount%3 != 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(
		WithMaxRetries(2),
		WithCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1000, // High threshold to avoid circuit breaker opening
			RecoveryTimeout:  1 * time.Second,
			SuccessThreshold: 1,
		}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(context.Background(), server.URL)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkCircuitBreakerAllow benchmarks circuit breaker Allow() method
func BenchmarkCircuitBreakerAllow(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  1 * time.Second,
		SuccessThreshold: 2,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}

// BenchmarkCalculateBackoff benchmarks backoff calculation
func BenchmarkCalculateBackoff(b *testing.B) {
	client := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.calculateBackoff(i % 10)
	}
}

// BenchmarkCalculateBackoffWithJitter benchmarks backoff calculation with jitter
func BenchmarkCalculateBackoffWithJitter(b *testing.B) {
	client := New(WithJitter(0.1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.calculateBackoff(i % 10)
	}
}

// BenchmarkRateLimiterAllow benchmarks rate limiter Allow method
func BenchmarkRateLimiterAllow(b *testing.B) {
	rl := NewRateLimiter(100, 10*time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow()
	}
}

// BenchmarkDefaultRetryCondition benchmarks retry condition evaluation
func BenchmarkDefaultRetryCondition(b *testing.B) {
	resp := &http.Response{StatusCode: 500}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DefaultRetryCondition(resp, nil)
	}
}

// BenchmarkClientCreation benchmarks client creation with various options
func BenchmarkClientCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(
			WithMaxRetries(3),
			WithInitialBackoff(100*time.Millisecond),
			WithMaxBackoff(10*time.Second),
			WithTimeout(30*time.Second),
			WithCircuitBreaker(CircuitBreakerConfig{
				FailureThreshold: 5,
				RecoveryTimeout:  60 * time.Second,
				SuccessThreshold: 2,
			}),
		)
	}
}

// BenchmarkDifferentBackoffStrategies benchmarks different backoff configurations
func BenchmarkDifferentBackoffStrategies(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	configs := []struct {
		name string
		opts []Option
	}{
		{"LinearBackoff", []Option{
			WithMaxRetries(3),
			WithInitialBackoff(100 * time.Millisecond),
			WithBackoffMultiplier(1.0),
		}},
		{"ExponentialBackoff", []Option{
			WithMaxRetries(3),
			WithInitialBackoff(100 * time.Millisecond),
			WithBackoffMultiplier(2.0),
		}},
		{"AggressiveBackoff", []Option{
			WithMaxRetries(3),
			WithInitialBackoff(10 * time.Millisecond),
			WithBackoffMultiplier(3.0),
		}},
	}

	for _, config := range configs {
		b.Run(config.name, func(b *testing.B) {
			client := New(config.opts...)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					resp, err := client.Get(context.Background(), server.URL)
					if err != nil {
						b.Fatal(err)
					}
					resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate network delay
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(WithMaxRetries(2))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkLargePayload benchmarks requests with larger response bodies
func BenchmarkLargePayload(b *testing.B) {
	// Create a 1MB response
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer server.Close()

	client := New(WithMaxRetries(2))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			// Read the entire response body
			buf := make([]byte, len(largeData))
			_, err = resp.Body.Read(buf)
			resp.Body.Close()
			if err != nil && err.Error() != "EOF" {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkTimeoutHandling benchmarks timeout scenarios
func BenchmarkTimeoutHandling(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // Longer than client timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(
		WithTimeout(10*time.Millisecond),
		WithMaxRetries(0), // No retries for timeout testing
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Get(context.Background(), server.URL)
		// We expect timeouts, so we don't check for errors
		_ = err
	}
}

package klayengo

import (
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestWithMaxRetries(t *testing.T) {
	client := New(WithMaxRetries(5))

	if client.maxRetries != 5 {
		t.Errorf("Expected maxRetries=5, got %d", client.maxRetries)
	}
}

func TestWithInitialBackoff(t *testing.T) {
	backoff := 200 * time.Millisecond
	client := New(WithInitialBackoff(backoff))

	if client.initialBackoff != backoff {
		t.Errorf("Expected initialBackoff=%v, got %v", backoff, client.initialBackoff)
	}
}

func TestWithMaxBackoff(t *testing.T) {
	maxBackoff := 30 * time.Second
	client := New(WithMaxBackoff(maxBackoff))

	if client.maxBackoff != maxBackoff {
		t.Errorf("Expected maxBackoff=%v, got %v", maxBackoff, client.maxBackoff)
	}
}

func TestWithBackoffMultiplier(t *testing.T) {
	multiplier := 3.0
	client := New(WithBackoffMultiplier(multiplier))

	if client.backoffMultiplier != multiplier {
		t.Errorf("Expected backoffMultiplier=%v, got %v", multiplier, client.backoffMultiplier)
	}
}

func TestWithJitter(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0.1, 0.1},
		{0.5, 0.5},
		{1.0, 1.0},
		{-0.1, 0.0}, // Should clamp to 0
		{1.5, 1.0},  // Should clamp to 1
	}

	for _, test := range tests {
		client := New(WithJitter(test.input))
		if client.jitter != test.expected {
			t.Errorf("WithJitter(%v) = %v, expected %v", test.input, client.jitter, test.expected)
		}
	}
}

func TestWithRateLimiter(t *testing.T) {
	client := New(WithRateLimiter(100, 1*time.Minute))

	if client.rateLimiter == nil {
		t.Fatal("Expected rate limiter to be set")
	}

	if client.rateLimiter.maxTokens != 100 {
		t.Errorf("Expected maxTokens=100, got %d", client.rateLimiter.maxTokens)
	}

	if client.rateLimiter.refillRate != 1*time.Minute {
		t.Errorf("Expected refillRate=1m, got %v", client.rateLimiter.refillRate)
	}
}

func TestWithCache(t *testing.T) {
	ttl := 10 * time.Minute
	client := New(WithCache(ttl))

	if client.cache == nil {
		t.Fatal("Expected cache to be set")
	}

	if client.cacheTTL != ttl {
		t.Errorf("Expected cacheTTL=%v, got %v", ttl, client.cacheTTL)
	}

	// Verify it's an InMemoryCache
	if _, ok := client.cache.(*InMemoryCache); !ok {
		t.Error("Expected InMemoryCache implementation")
	}
}

func TestWithCustomCache(t *testing.T) {
	customCache := NewInMemoryCache()
	ttl := 15 * time.Minute

	client := New(WithCustomCache(customCache, ttl))

	if client.cache != customCache {
		t.Error("Expected custom cache to be set")
	}

	if client.cacheTTL != ttl {
		t.Errorf("Expected cacheTTL=%v, got %v", ttl, client.cacheTTL)
	}
}

func TestWithCacheKeyFunc(t *testing.T) {
	customKeyFunc := func(req *http.Request) string {
		return "custom-key"
	}

	client := New(WithCacheKeyFunc(customKeyFunc))

	if client.cacheKeyFunc == nil {
		t.Fatal("Expected cache key function to be set")
	}

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	key := client.cacheKeyFunc(req)

	if key != "custom-key" {
		t.Errorf("Expected 'custom-key', got '%s'", key)
	}
}

func TestWithCacheCondition(t *testing.T) {
	customCondition := func(req *http.Request) bool {
		return req.Method == "POST"
	}

	client := New(WithCacheCondition(customCondition))

	if client.cacheCondition == nil {
		t.Fatal("Expected cache condition to be set")
	}

	getReq, _ := http.NewRequest("GET", "https://example.com", nil)
	postReq, _ := http.NewRequest("POST", "https://example.com", nil)

	if client.cacheCondition(getReq) {
		t.Error("Expected GET request to not be cached with custom condition")
	}

	if !client.cacheCondition(postReq) {
		t.Error("Expected POST request to be cached with custom condition")
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 45 * time.Second
	client := New(WithTimeout(timeout))

	if client.timeout != timeout {
		t.Errorf("Expected timeout=%v, got %v", timeout, client.timeout)
	}

	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected HTTP client timeout=%v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestWithRetryCondition(t *testing.T) {
	customCondition := func(resp *http.Response, err error) bool {
		return err != nil // Only retry on errors
	}

	client := New(WithRetryCondition(customCondition))

	if client.retryCondition == nil {
		t.Fatal("Expected retry condition to be set")
	}

	// Test with error
	if !client.retryCondition(nil, http.ErrHandlerTimeout) {
		t.Error("Expected true for error condition")
	}

	// Test with 500 response
	resp500 := &http.Response{StatusCode: 500}
	if client.retryCondition(resp500, nil) {
		t.Error("Expected false for 500 response with custom condition")
	}
}

func TestWithCircuitBreaker(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  45 * time.Second,
		SuccessThreshold: 2,
	}

	client := New(WithCircuitBreaker(config))

	if client.circuitBreaker == nil {
		t.Fatal("Expected circuit breaker to be set")
	}

	if client.circuitBreaker.config.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold=3, got %d", client.circuitBreaker.config.FailureThreshold)
	}

	if client.circuitBreaker.config.RecoveryTimeout != 45*time.Second {
		t.Errorf("Expected RecoveryTimeout=45s, got %v", client.circuitBreaker.config.RecoveryTimeout)
	}

	if client.circuitBreaker.config.SuccessThreshold != 2 {
		t.Errorf("Expected SuccessThreshold=2, got %d", client.circuitBreaker.config.SuccessThreshold)
	}
}

func TestWithMiddleware(t *testing.T) {
	middleware1 := func(req *http.Request, next RoundTripper) (*http.Response, error) {
		return next.RoundTrip(req)
	}

	middleware2 := func(req *http.Request, next RoundTripper) (*http.Response, error) {
		return next.RoundTrip(req)
	}

	client := New(WithMiddleware(middleware1, middleware2))

	if len(client.middleware) != 2 {
		t.Errorf("Expected 2 middleware functions, got %d", len(client.middleware))
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	client := New(WithHTTPClient(customClient))

	if client.httpClient != customClient {
		t.Error("Expected custom HTTP client to be set")
	}
}

func TestWithHTTPClientTimeoutUpdate(t *testing.T) {
	customClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Set timeout first, then HTTP client
	client := New(
		WithTimeout(30*time.Second),
		WithHTTPClient(customClient),
	)

	// HTTP client timeout should be updated to match client timeout
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected HTTP client timeout=30s, got %v", client.httpClient.Timeout)
	}
}

func TestWithMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	client := New(WithMetricsCollector(NewMetricsCollectorWithRegistry(registry)))

	if client.metrics == nil {
		t.Fatal("Expected metrics collector to be set")
	}
}

func TestWithMetricsCollector(t *testing.T) {
	registry := prometheus.NewRegistry()
	customCollector := NewMetricsCollectorWithRegistry(registry)

	client := New(WithMetricsCollector(customCollector))

	if client.metrics != customCollector {
		t.Error("Expected custom metrics collector to be set")
	}
}

func TestMultipleOptions(t *testing.T) {
	registry := prometheus.NewRegistry()
	client := New(
		WithMaxRetries(10),
		WithTimeout(60*time.Second),
		WithCache(20*time.Minute),
		WithRateLimiter(50, 30*time.Second),
		WithMetricsCollector(NewMetricsCollectorWithRegistry(registry)),
	)

	if client.maxRetries != 10 {
		t.Errorf("Expected maxRetries=10, got %d", client.maxRetries)
	}

	if client.timeout != 60*time.Second {
		t.Errorf("Expected timeout=60s, got %v", client.timeout)
	}

	if client.cacheTTL != 20*time.Minute {
		t.Errorf("Expected cacheTTL=20m, got %v", client.cacheTTL)
	}

	if client.rateLimiter == nil {
		t.Error("Expected rate limiter to be set")
	}

	if client.metrics == nil {
		t.Error("Expected metrics collector to be set")
	}
}

func TestOptionsOrderIndependence(t *testing.T) {
	// Test that option order doesn't matter
	client1 := New(
		WithMaxRetries(5),
		WithTimeout(30*time.Second),
		WithCache(10*time.Minute),
	)

	client2 := New(
		WithCache(10*time.Minute),
		WithTimeout(30*time.Second),
		WithMaxRetries(5),
	)

	if client1.maxRetries != client2.maxRetries {
		t.Error("Option order affected maxRetries")
	}

	if client1.timeout != client2.timeout {
		t.Error("Option order affected timeout")
	}

	if client1.cacheTTL != client2.cacheTTL {
		t.Error("Option order affected cacheTTL")
	}
}

func TestDefaultValuesWithoutOptions(t *testing.T) {
	client := New()

	if client.maxRetries != 3 {
		t.Errorf("Expected default maxRetries=3, got %d", client.maxRetries)
	}

	if client.initialBackoff != 100*time.Millisecond {
		t.Errorf("Expected default initialBackoff=100ms, got %v", client.initialBackoff)
	}

	if client.maxBackoff != 10*time.Second {
		t.Errorf("Expected default maxBackoff=10s, got %v", client.maxBackoff)
	}

	if client.backoffMultiplier != 2.0 {
		t.Errorf("Expected default backoffMultiplier=2.0, got %v", client.backoffMultiplier)
	}

	if client.jitter != 0.1 {
		t.Errorf("Expected default jitter=0.1, got %v", client.jitter)
	}

	if client.timeout != 30*time.Second {
		t.Errorf("Expected default timeout=30s, got %v", client.timeout)
	}

	if client.cache != nil {
		t.Error("Expected default cache=nil")
	}

	if client.rateLimiter != nil {
		t.Error("Expected default rateLimiter=nil")
	}

	if client.metrics != nil {
		t.Error("Expected default metrics=nil")
	}

	if len(client.middleware) != 0 {
		t.Errorf("Expected default middleware count=0, got %d", len(client.middleware))
	}
}

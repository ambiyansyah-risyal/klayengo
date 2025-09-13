package klayengo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	testEndpoint    = "test-endpoint"
	testAPIEndpoint = "example.com/api"
)

func TestNewMetricsCollector(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	if collector == nil {
		t.Fatal("NewMetricsCollectorWithRegistry() returned nil")
	}

	if collector.requestsTotal == nil {
		t.Error("requestsTotal metric not initialized")
	}

	if collector.requestDuration == nil {
		t.Error("requestDuration metric not initialized")
	}

	if collector.requestsInFlight == nil {
		t.Error("requestsInFlight metric not initialized")
	}

	if collector.retriesTotal == nil {
		t.Error("retriesTotal metric not initialized")
	}

	if collector.circuitBreakerState == nil {
		t.Error("circuitBreakerState metric not initialized")
	}

	if collector.rateLimiterTokens == nil {
		t.Error("rateLimiterTokens metric not initialized")
	}

	if collector.cacheHits == nil {
		t.Error("cacheHits metric not initialized")
	}

	if collector.cacheMisses == nil {
		t.Error("cacheMisses metric not initialized")
	}

	if collector.cacheSize == nil {
		t.Error("cacheSize metric not initialized")
	}

	if collector.errorsTotal == nil {
		t.Error("errorsTotal metric not initialized")
	}
}

func TestNewMetricsCollectorWithRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	if collector == nil {
		t.Fatal("NewMetricsCollectorWithRegistry() returned nil")
	}

	if collector.registry != registry {
		t.Error("Registry not set correctly")
	}
}

func TestRecordRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	method := "GET"
	endpoint := testAPIEndpoint
	statusCode := 200
	duration := 150 * time.Millisecond

	collector.RecordRequest(method, endpoint, statusCode, duration)

}

func TestRecordRequestStart(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	method := "POST"
	endpoint := testAPIEndpoint

	collector.RecordRequestStart(method, endpoint)

}

func TestRecordRequestEnd(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	method := "PUT"
	endpoint := testAPIEndpoint

	collector.RecordRequestEnd(method, endpoint)

}

func TestRecordRetry(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	method := "GET"
	endpoint := testAPIEndpoint
	attempt := 2

	collector.RecordRetry(method, endpoint, attempt)

}

func TestRecordCircuitBreakerState(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	name := "default"

	states := []CircuitState{StateClosed, StateOpen, StateHalfOpen}

	for _, state := range states {
		collector.RecordCircuitBreakerState(name, state)
	}
}

func TestRecordRateLimiterTokens(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	name := "default"
	tokens := 50

	collector.RecordRateLimiterTokens(name, tokens)

}

func TestRecordCacheHit(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	method := "GET"
	endpoint := testAPIEndpoint

	collector.RecordCacheHit(method, endpoint)

}

func TestRecordCacheMiss(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	method := "POST"
	endpoint := testAPIEndpoint

	collector.RecordCacheMiss(method, endpoint)

}

func TestRecordCacheSize(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	name := "default"
	size := 25

	collector.RecordCacheSize(name, size)

}

func TestRecordDeduplicationHit(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	method := "GET"
	endpoint := testAPIEndpoint

	collector.RecordDeduplicationHit(method, endpoint)

}

func TestGetRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	if collector.GetRegistry() != registry {
		t.Error("GetRegistry() returned wrong registry")
	}
}

func TestMetricsCollectorWithNil(t *testing.T) {
	var collector *MetricsCollector

	collector.RecordRequest("GET", "test", 200, time.Second)
	collector.RecordRequestStart("GET", "test")
	collector.RecordRequestEnd("GET", "test")
	collector.RecordRetry("GET", "test", 1)
	collector.RecordCircuitBreakerState("test", StateClosed)
	collector.RecordRateLimiterTokens("test", 10)
	collector.RecordCacheHit("GET", "test")
	collector.RecordCacheMiss("GET", "test")
	collector.RecordCacheSize("test", 5)
	collector.RecordError("test", "GET", "test")
}

func TestMetricsIntegration(t *testing.T) {
	registry := prometheus.NewRegistry()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := New(WithMetricsCollector(NewMetricsCollectorWithRegistry(registry)))

	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	_ = resp.Body.Close()

}

func TestMetricsWithCircuitBreaker(t *testing.T) {
	registry := prometheus.NewRegistry()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(
		WithMetricsCollector(NewMetricsCollectorWithRegistry(registry)),
		WithCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 2,
			RecoveryTimeout:  10 * time.Millisecond,
		}),
		WithMaxRetries(0),
	)

	for i := 0; i < 3; i++ {
		resp, err := client.Get(context.Background(), server.URL)
		if err == nil {
			_ = resp.Body.Close()
		}
	}

}

func TestMetricsWithRateLimiter(t *testing.T) {
	registry := prometheus.NewRegistry()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(
		WithMetricsCollector(NewMetricsCollectorWithRegistry(registry)),
		WithRateLimiter(2, 100*time.Millisecond),
	)

	for i := 0; i < 3; i++ {
		resp, err := client.Get(context.Background(), server.URL)
		if err == nil {
			_ = resp.Body.Close()
		}
	}

}

func TestMetricsWithCache(t *testing.T) {
	registry := prometheus.NewRegistry()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("cached response")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := New(
		WithMetricsCollector(NewMetricsCollectorWithRegistry(registry)),
		WithCache(1*time.Hour),
	)

	resp1, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	_ = resp1.Body.Close()

	resp2, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	_ = resp2.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected 1 server call (cached), got %d", callCount)
	}

}

func TestMetricsWithRetries(t *testing.T) {
	registry := prometheus.NewRegistry()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(
		WithMetricsCollector(NewMetricsCollectorWithRegistry(registry)),
		WithMaxRetries(3),
	)

	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	_ = resp.Body.Close()

	if callCount != 3 {
		t.Errorf("Expected 3 calls (with retries), got %d", callCount)
	}

}

func TestMetricsCollectorInitialization(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	if collector.requestsTotal == nil {
		t.Error("requestsTotal not initialized")
	}

	if collector.requestDuration == nil {
		t.Error("requestDuration not initialized")
	}

	if collector.requestsInFlight == nil {
		t.Error("requestsInFlight not initialized")
	}

	if collector.retriesTotal == nil {
		t.Error("retriesTotal not initialized")
	}

	if collector.circuitBreakerState == nil {
		t.Error("circuitBreakerState not initialized")
	}

	if collector.rateLimiterTokens == nil {
		t.Error("rateLimiterTokens not initialized")
	}

	if collector.cacheHits == nil {
		t.Error("cacheHits not initialized")
	}

	if collector.cacheMisses == nil {
		t.Error("cacheMisses not initialized")
	}

	if collector.cacheSize == nil {
		t.Error("cacheSize not initialized")
	}

	if collector.errorsTotal == nil {
		t.Error("errorsTotal not initialized")
	}
}

func TestMetricsWithCustomRegistry(t *testing.T) {
	customRegistry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(customRegistry)

	if collector.registry != customRegistry {
		t.Error("Custom registry not properly set")
	}

	collector.RecordRequest("GET", "test", 200, time.Second)

}

func TestMetricsStateTransitions(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	states := []CircuitState{StateClosed, StateOpen, StateHalfOpen, StateClosed}

	for _, state := range states {
		collector.RecordCircuitBreakerState("test", state)
	}

	for tokens := 0; tokens <= 10; tokens++ {
		collector.RecordRateLimiterTokens("test", tokens)
	}

	for size := 0; size <= 5; size++ {
		collector.RecordCacheSize("test", size)
	}

}

func TestMetricsErrorTypes(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	errorTypes := []string{
		"Network",
		"Server",
		"RateLimit",
		"CircuitBreaker",
		"Timeout",
		"Unknown",
	}

	for _, errorType := range errorTypes {
		collector.RecordError(errorType, "GET", testEndpoint)
	}

}

func TestMetricsHTTPMethods(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		collector.RecordRequest(method, testEndpoint, 200, time.Millisecond)
		collector.RecordRequestStart(method, testEndpoint)
		collector.RecordRequestEnd(method, testEndpoint)
		collector.RecordRetry(method, testEndpoint, 1)
		collector.RecordCacheHit(method, testEndpoint)
		collector.RecordCacheMiss(method, testEndpoint)
		collector.RecordError("Test", method, testEndpoint)
	}

}

func TestMetricsStatusCodes(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	statusCodes := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 422, 429, 500, 502, 503, 504}

	for _, statusCode := range statusCodes {
		collector.RecordRequest("GET", testEndpoint, statusCode, time.Millisecond)
	}

}

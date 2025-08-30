package klayengo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	testResponseBody         = "test response"
	cachedResponseBody       = "cached response"
	successResponseBody      = "success"
	fullFeaturedResponseBody = "full featured response"
	contentTypeJSON          = "application/json"
	expectedStatus200Msg     = "Expected status 200, got %d"
	expectedContentTypeMsg   = "Expected Content-Type application/json, got %s"
	failedWriteResponseMsg   = "Failed to write response: %v"
)

func TestNew(t *testing.T) {
	client := New()

	if client == nil {
		t.Fatal("New() returned nil")
	}

	// Test default values
	if client.maxRetries != 3 {
		t.Errorf("Expected maxRetries=3, got %d", client.maxRetries)
	}

	if client.initialBackoff != 100*time.Millisecond {
		t.Errorf("Expected initialBackoff=100ms, got %v", client.initialBackoff)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout=30s, got %v", client.httpClient.Timeout)
	}
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(testResponseBody)); err != nil {
			t.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New()
	resp, err := client.Get(context.Background(), server.URL)

	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf(expectedStatus200Msg, resp.StatusCode)
	}

	body := make([]byte, len(testResponseBody))
	if _, err := resp.Body.Read(body); err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != testResponseBody {
		t.Errorf("Expected '%s', got '%s'", testResponseBody, string(body))
	}
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != contentTypeJSON {
			t.Errorf(expectedContentTypeMsg, r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(testResponseBody)); err != nil {
			t.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New()
	resp, err := client.Post(context.Background(), server.URL, "application/json", strings.NewReader(`{"test": "data"}`))

	if err != nil {
		t.Fatalf("Post() returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf(expectedStatus200Msg, resp.StatusCode)
	}
}

func TestDoWithRetry(t *testing.T) {
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

	client := New(WithMaxRetries(3))
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf(expectedStatus200Msg, resp.StatusCode)
	}
}

func TestDoWithRetryFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(WithMaxRetries(2))
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}

	if callCount != 3 { // initial + 2 retries
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestCalculateBackoff(t *testing.T) {
	client := New(
		WithInitialBackoff(100*time.Millisecond),
		WithMaxBackoff(1*time.Second),
		WithBackoffMultiplier(2.0),
		WithJitter(0.0), // Disable jitter for predictable test
	)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1000 * time.Millisecond}, // capped at maxBackoff
	}

	for _, test := range tests {
		result := client.calculateBackoff(test.attempt)
		if result != test.expected {
			t.Errorf("Attempt %d: expected %v, got %v", test.attempt, test.expected, result)
		}
	}
}

func TestDefaultRetryCondition(t *testing.T) {
	tests := []struct {
		resp     *http.Response
		err      error
		expected bool
	}{
		{nil, http.ErrHandlerTimeout, true},
		{&http.Response{StatusCode: 200}, nil, false},
		{&http.Response{StatusCode: 404}, nil, false},
		{&http.Response{StatusCode: 500}, nil, true},
		{&http.Response{StatusCode: 502}, nil, true},
		{&http.Response{StatusCode: 503}, nil, true},
	}

	for _, test := range tests {
		result := DefaultRetryCondition(test.resp, test.err)
		if result != test.expected {
			t.Errorf("Retry condition failed: resp=%v, err=%v, expected %v, got %v",
				test.resp, test.err, test.expected, result)
		}
	}
}

func TestGetEndpointFromRequest(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://example.com", "example.com/"},
		{"http://example.com/path", "example.com/path"},
		{"http://example.com/path/to/resource", "example.com/path/to/resource"},
		{"https://api.example.com/v1/users", "api.example.com/v1/users"},
	}

	for _, test := range tests {
		req, _ := http.NewRequest("GET", test.url, nil)
		result := getEndpointFromRequest(req)
		if result != test.expected {
			t.Errorf("URL %s: expected %s, got %s", test.url, test.expected, result)
		}
	}
}

func TestExecuteMiddleware(t *testing.T) {
	callOrder := []string{}

	middleware1 := func(req *http.Request, next RoundTripper) (*http.Response, error) {
		callOrder = append(callOrder, "middleware1")
		return next.RoundTrip(req)
	}

	middleware2 := func(req *http.Request, next RoundTripper) (*http.Response, error) {
		callOrder = append(callOrder, "middleware2")
		return next.RoundTrip(req)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "handler")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(WithMiddleware(middleware1, middleware2))
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("Execute middleware failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf(expectedStatus200Msg, resp.StatusCode)
	}

	expectedOrder := []string{"middleware1", "middleware2", "handler"}
	if len(callOrder) != len(expectedOrder) {
		t.Errorf("Expected call order %v, got %v", expectedOrder, callOrder)
	}

	for i, expected := range expectedOrder {
		if i >= len(callOrder) || callOrder[i] != expected {
			t.Errorf("Expected call order %v, got %v", expectedOrder, callOrder)
		}
	}
}

func TestClientWithTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(WithTimeout(50 * time.Millisecond))
	req, _ := http.NewRequest("GET", server.URL, nil)

	_, err := client.Do(req)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestClientWithCustomHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	client := New(WithHTTPClient(customClient))

	if client.httpClient != customClient {
		t.Error("Custom HTTP client not set correctly")
	}
}

func TestClientWithMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewMetricsCollectorWithRegistry(registry)

	client := New(WithMetricsCollector(collector))

	if client.metrics != collector {
		t.Error("Metrics collector not set correctly")
	}
}

// Benchmark tests for performance measurement

func BenchmarkClientGet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(testResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New()

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

func BenchmarkClientPost(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(testResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Post(context.Background(), server.URL, "application/json", strings.NewReader(`{"test": "data"}`))
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkClientWithRetries(b *testing.B) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount%3 != 0 { // Fail 2 out of 3 requests
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(successResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New(
		WithMaxRetries(2),
		WithCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1000, // Very high threshold to avoid opening during benchmark
			RecoveryTimeout:  1 * time.Second,
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

func BenchmarkClientWithCache(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(cachedResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New(WithCache(1 * time.Hour))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(context.Background(), server.URL)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkClientWithCircuitBreaker(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(successResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New(WithCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  1 * time.Second,
	}))

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

func BenchmarkClientWithRateLimiter(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(successResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New(WithRateLimiter(100000, 1*time.Second)) // Very high limit to avoid blocking

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

func BenchmarkClientFullFeatures(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(fullFeaturedResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	// Create metrics collector once to avoid duplicate registration
	registry := prometheus.NewRegistry()
	metricsCollector := NewMetricsCollectorWithRegistry(registry)

	client := New(
		WithMaxRetries(2),
		WithCache(5*time.Minute),
		WithRateLimiter(10000, 1*time.Second),
		WithCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1000, // High threshold to avoid opening
			RecoveryTimeout:  5 * time.Second,
		}),
		WithMetricsCollector(metricsCollector),
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

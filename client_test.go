package klayengo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	resp, err := client.Post(context.Background(), server.URL, contentTypeJSON, strings.NewReader(`{"test": "data"}`))

	if err != nil {
		t.Fatalf("Post() returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf(expectedStatus200Msg, resp.StatusCode)
	}
}

func TestPostWithInvalidURL(t *testing.T) {
	client := New()
	_, err := client.Post(context.Background(), "http://invalid url with spaces", contentTypeJSON, strings.NewReader(`{}`))

	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestPostWithNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(testResponseBody)); err != nil {
			t.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New()
	resp, err := client.Post(context.Background(), server.URL, contentTypeJSON, nil)

	if err != nil {
		t.Fatalf("Post() with nil body returned error: %v", err)
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

	if callCount != 3 {
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
		WithJitter(0.0),
	)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1000 * time.Millisecond},
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

func TestGetEndpointFromRequestWithNilURL(t *testing.T) {
	req, _ := http.NewRequest("GET", "", nil)
	req.URL = nil

	result := getEndpointFromRequest(req)
	if result != "unknown" {
		t.Errorf("Expected 'unknown' for nil URL, got '%s'", result)
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
			_ = resp.Body.Close()
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
			resp, err := client.Post(context.Background(), server.URL, contentTypeJSON, strings.NewReader(`{"test": "data"}`))
			if err != nil {
				b.Fatal(err)
			}
			_ = resp.Body.Close()
		}
	})
}

func BenchmarkClientWithRetries(b *testing.B) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount%3 != 0 {
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
			FailureThreshold: 1000,
			RecoveryTimeout:  1 * time.Second,
		}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(context.Background(), server.URL)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()
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
		_ = resp.Body.Close()
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
			_ = resp.Body.Close()
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

	client := New(WithRateLimiter(100000, 1*time.Second))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			_ = resp.Body.Close()
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

	registry := prometheus.NewRegistry()
	metricsCollector := NewMetricsCollectorWithRegistry(registry)

	client := New(
		WithMaxRetries(2),
		WithCache(5*time.Minute),
		WithRateLimiter(10000, 1*time.Second),
		WithCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1000,
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
		_ = resp.Body.Close()
	}
}

func BenchmarkClientWithDeduplication(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(successResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New(WithDeduplication())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(context.Background(), server.URL)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()
	}
}

func BenchmarkClientConcurrentDeduplication(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(successResponseBody)); err != nil {
			b.Fatalf(failedWriteResponseMsg, err)
		}
	}))
	defer server.Close()

	client := New(WithDeduplication())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				b.Fatal(err)
			}
			_ = resp.Body.Close()
		}
	})
}

func TestClientWithPerHostRateLimiting(t *testing.T) {
	// Create a slow server to ensure rate limiting kicks in
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with per-host rate limiting
	hostLimiter := NewRateLimiter(2, 100*time.Millisecond) // Only 2 requests per 100ms

	client := New(
		WithLimiterKeyFunc(DefaultHostKeyFunc),
		WithLimiterFor("host:"+mustParseURL(server.URL).Host, hostLimiter),
		WithMaxRetries(0), // Disable retries for this test
	)

	// Make multiple requests to the same host
	var allowedCount, deniedCount int
	for i := 0; i < 5; i++ {
		_, err := client.Get(context.Background(), server.URL)
		if err != nil {
			if strings.Contains(err.Error(), "rate limit exceeded") {
				deniedCount++
			} else {
				t.Fatalf("Unexpected error: %v", err)
			}
		} else {
			allowedCount++
		}
	}

	// Should have some requests allowed and some denied due to rate limiting
	if allowedCount == 0 {
		t.Error("Expected at least some requests to be allowed")
	}
	if deniedCount == 0 {
		t.Error("Expected at least some requests to be denied due to rate limiting")
	}
}

func TestClientWithPerRouteRateLimiting(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with per-route rate limiting
	usersLimiter := NewRateLimiter(1, 500*time.Millisecond) // Only 1 request per 500ms to /users
	postsLimiter := NewRateLimiter(5, 500*time.Millisecond) // 5 requests per 500ms to /posts

	client := New(
		WithLimiterKeyFunc(DefaultRouteKeyFunc),
		WithLimiterFor("route:GET:/users", usersLimiter),
		WithLimiterFor("route:GET:/posts", postsLimiter),
		WithMaxRetries(0),
	)

	serverURL := mustParseURL(server.URL)

	// Test /users endpoint (strict limiting)
	var usersAllowed, usersDenied int
	usersURL := serverURL.String() + "/users"
	for i := 0; i < 4; i++ {
		_, err := client.Get(context.Background(), usersURL)
		if err != nil {
			if strings.Contains(err.Error(), "rate limit exceeded") {
				usersDenied++
			} else {
				t.Fatalf("Unexpected error: %v", err)
			}
		} else {
			usersAllowed++
		}
		time.Sleep(100 * time.Millisecond) // Wait a bit between requests
	}

	// Test /posts endpoint (more permissive limiting)
	var postsAllowed, postsDenied int
	postsURL := serverURL.String() + "/posts"
	for i := 0; i < 8; i++ {
		_, err := client.Get(context.Background(), postsURL)
		if err != nil {
			if strings.Contains(err.Error(), "rate limit exceeded") {
				postsDenied++
			} else {
				t.Fatalf("Unexpected error: %v", err)
			}
		} else {
			postsAllowed++
		}
		time.Sleep(50 * time.Millisecond) // Wait a bit between requests
	}

	// /users should have stricter limiting (more denials)
	if usersDenied <= postsDenied {
		t.Errorf("Expected /users to have more denials (%d) than /posts (%d)", usersDenied, postsDenied)
	}
}

func mustParseURL(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

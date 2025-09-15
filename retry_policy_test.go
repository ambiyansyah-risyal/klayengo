package klayengo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

const (
	retryTestURL             = "http://example.com"
	expectedPositiveDelayMsg = "Expected positive delay for retry"
	errFmtExpectedClientErr  = "Expected ClientError, got %T"
	errFmtBudgetExceededType = "Expected ErrorTypeRetryBudgetExceeded, got %s"
)

func TestNewDefaultRetryPolicy(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	if policy.maxRetries != 3 {
		t.Errorf("Expected maxRetries=3, got %d", policy.maxRetries)
	}
	if policy.initialBackoff != 100*time.Millisecond {
		t.Errorf("Expected initialBackoff=100ms, got %v", policy.initialBackoff)
	}
	if policy.maxBackoff != 5*time.Second {
		t.Errorf("Expected maxBackoff=5s, got %v", policy.maxBackoff)
	}
	if policy.backoffMultiplier != 2.0 {
		t.Errorf("Expected backoffMultiplier=2.0, got %f", policy.backoffMultiplier)
	}
	if policy.jitter != 0.1 {
		t.Errorf("Expected jitter=0.1, got %f", policy.jitter)
	}
}

func TestDefaultIsIdempotent(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"GET", true},
		{"HEAD", true},
		{"PUT", true},
		{"DELETE", true},
		{"OPTIONS", true},
		{"POST", false},
		{"PATCH", false},
		{"CONNECT", false},
		{"TRACE", false},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := DefaultIsIdempotent(tt.method)
			if result != tt.expected {
				t.Errorf("DefaultIsIdempotent(%s) = %v, want %v", tt.method, result, tt.expected)
			}
		})
	}
}

func TestRetryPolicyShouldRetryNetworkError(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	req, _ := http.NewRequest("GET", retryTestURL, nil)
	resp := &http.Response{Request: req}
	delay, shouldRetry := policy.ShouldRetry(resp, errors.New("network error"), 0)
	if !shouldRetry {
		t.Error("Expected to retry on network error")
	}
	if delay <= 0 {
		t.Error(expectedPositiveDelayMsg)
	}
}

func TestRetryPolicyShouldRetryServerError(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	req, _ := http.NewRequest("GET", retryTestURL, nil)
	resp := &http.Response{StatusCode: 500, Request: req, Header: make(http.Header)}
	delay, shouldRetry := policy.ShouldRetry(resp, nil, 0)
	if !shouldRetry {
		t.Error("Expected to retry on 500 error")
	}
	if delay <= 0 {
		t.Error(expectedPositiveDelayMsg)
	}
}

func TestRetryPolicyShouldRetryRateLimited(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	req, _ := http.NewRequest("GET", retryTestURL, nil)
	resp := &http.Response{StatusCode: 429, Request: req, Header: make(http.Header)}
	delay, shouldRetry := policy.ShouldRetry(resp, nil, 0)
	if !shouldRetry {
		t.Error("Expected to retry on 429 error")
	}
	if delay <= 0 {
		t.Error(expectedPositiveDelayMsg)
	}
}

func TestRetryPolicyShouldNotRetryNonIdempotent(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	req, _ := http.NewRequest("POST", retryTestURL, nil)
	resp := &http.Response{StatusCode: 500, Request: req, Header: make(http.Header)}
	_, shouldRetry := policy.ShouldRetry(resp, nil, 0)
	if shouldRetry {
		t.Error("Expected not to retry non-idempotent method")
	}
}

func TestRetryPolicyShouldNotRetryMaxAttempts(t *testing.T) {
	policy := NewDefaultRetryPolicy(2, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	req, _ := http.NewRequest("GET", retryTestURL, nil)
	resp := &http.Response{Request: req}
	_, shouldRetry := policy.ShouldRetry(resp, errors.New("error"), 2)
	if shouldRetry {
		t.Error("Expected not to retry when max attempts reached")
	}
}

func TestRetryPolicyShouldNotRetrySuccessfulResponse(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	req, _ := http.NewRequest("GET", retryTestURL, nil)
	resp := &http.Response{StatusCode: 200, Request: req, Header: make(http.Header)}
	_, shouldRetry := policy.ShouldRetry(resp, nil, 0)
	if shouldRetry {
		t.Error("Expected not to retry successful response")
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		value    string
		expected time.Duration
	}{
		{"5", 5 * time.Second},
		{"120", 2 * time.Minute},
		{"3600", time.Hour},
		{"7200", time.Hour}, // Capped at 1 hour
		{"0", 0},
		{"-5", 0},
		{"", 0},
		{"invalid", 0},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseRetryAfter(tt.value)
			if result != tt.expected {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	future := time.Now().Add(30 * time.Second)
	httpDate := future.UTC().Format(http.TimeFormat)
	result := parseRetryAfter(httpDate)
	if result <= 25*time.Second || result >= 35*time.Second {
		t.Errorf("parseRetryAfter with HTTP date should be around 30s, got %v", result)
	}
	past := time.Now().Add(-30 * time.Second)
	pastDate := past.UTC().Format(http.TimeFormat)
	result = parseRetryAfter(pastDate)
	if result != 0 {
		t.Errorf("parseRetryAfter with past date should return 0, got %v", result)
	}
}

func TestRetryPolicyWithRetryAfter(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	req, _ := http.NewRequest("GET", retryTestURL, nil)
	resp := &http.Response{StatusCode: 429, Request: req, Header: http.Header{"Retry-After": []string{"10"}}}
	delay, shouldRetry := policy.ShouldRetry(resp, nil, 0)
	if !shouldRetry {
		t.Error("Expected to retry on 429 with Retry-After")
	}
	if delay < 10*time.Second || delay > 11*time.Second {
		t.Errorf("Expected delay around 10s from Retry-After, got %v", delay)
	}
}

func TestNewRetryBudget(t *testing.T) {
	budget := NewRetryBudget(10, time.Minute)
	if budget.maxRetries != 10 {
		t.Errorf("Expected maxRetries=10, got %d", budget.maxRetries)
	}
	if budget.perWindow != time.Minute {
		t.Errorf("Expected perWindow=1m, got %v", budget.perWindow)
	}
}

func TestRetryBudgetAllow(t *testing.T) {
	budget := NewRetryBudget(3, time.Second)
	for i := 0; i < 3; i++ {
		if !budget.Allow() {
			t.Errorf("Expected call %d to be allowed", i+1)
		}
	}
	if budget.Allow() {
		t.Error("Expected 4th call to be rejected")
	}
}

func TestRetryBudgetWithMetricsCollector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	collector := NewMetricsCollector()
	client := New(
		WithRetryBudget(2, time.Second),
		WithMaxRetries(5),
		WithInitialBackoff(1*time.Millisecond),
		WithTimeout(time.Second),
		WithMetricsCollector(collector),
	)

	_, err1 := client.Get(context.TODO(), server.URL)
	if err1 == nil {
		t.Error("Expected error from first request consuming budget")
	}
	_, err2 := client.Get(context.TODO(), server.URL)
	if err2 == nil {
		t.Error("Expected immediate error from budget exhaustion")
	}
	ce, ok := err2.(*ClientError)
	if !ok {
		t.Fatalf(errFmtExpectedClientErr, err2)
	}
	if ce.Type != ErrorTypeRetryBudgetExceeded {
		t.Errorf(errFmtBudgetExceededType, ce.Type)
	}
	parsed, perr := url.Parse(server.URL)
	if perr != nil {
		t.Fatalf("Failed to parse server URL: %v", perr)
	}
	hostLabel := parsed.Host
	if exceeded := testutil.ToFloat64(collector.retryBudgetExceeded.WithLabelValues(hostLabel)); exceeded != 2 {
		t.Errorf("Expected retry_budget_exceeded=2, got %f", exceeded)
	}
	reqForEndpoint, _ := http.NewRequest("GET", server.URL, nil)
	endpoint := getEndpointFromRequest(reqForEndpoint)
	if r1 := testutil.ToFloat64(collector.retriesTotal.WithLabelValues("GET", endpoint, "1")); r1 != 1 {
		t.Errorf("Expected retriesTotal attempt=1 count=1, got %f", r1)
	}
	if r2 := testutil.ToFloat64(collector.retriesTotal.WithLabelValues("GET", endpoint, "2")); r2 != 1 {
		t.Errorf("Expected retriesTotal attempt=2 count=1, got %f", r2)
	}
	if rt := testutil.ToFloat64(collector.requestsTotal.WithLabelValues("GET", "0", endpoint)); rt != 2 {
		t.Errorf("Expected requestsTotal=2 for failed top-level requests (status_code=0), got %f", rt)
	}
	if et := testutil.ToFloat64(collector.errorsTotal.WithLabelValues("Server", "GET", endpoint)); et != 4 {
		t.Errorf("Expected errorsTotal Server count=4, got %f", et)
	}
	if inf := testutil.ToFloat64(collector.requestsInFlight.WithLabelValues("GET", endpoint)); inf != 0 {
		t.Errorf("Expected requestsInFlight=0 after completion, got %f", inf)
	}
}

func TestRetryBudgetGetStats(t *testing.T) {
	budget := NewRetryBudget(5, time.Minute)
	budget.Allow()
	budget.Allow()
	current, max, windowStart := budget.GetStats()
	if current != 2 {
		t.Errorf("Expected current=2, got %d", current)
	}
	if max != 5 {
		t.Errorf("Expected max=5, got %d", max)
	}
	if windowStart.IsZero() {
		t.Error("Expected non-zero window start time")
	}
}

func TestRetryPolicyIntegrationWithClient(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(500)
		} else {
			w.WriteHeader(200)
			if _, err := w.Write([]byte("success")); err != nil {
				// Best-effort error handling in test server
				http.Error(w, "write failed", http.StatusInternalServerError)
			}
		}
	}))
	defer server.Close()
	policy := NewDefaultRetryPolicy(3, 10*time.Millisecond, 100*time.Millisecond, 2.0, 0)
	client := New(
		WithRetryPolicy(policy),
		WithTimeout(time.Second),
	)
	resp, err := client.Get(context.TODO(), server.URL)
	if err != nil {
		t.Fatalf("Expected successful response after retries, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attempts)
	}
}

func TestRetryBudgetIntegrationWithClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()
	client := New(
		WithRetryBudget(2, time.Second),
		WithMaxRetries(5),
		WithInitialBackoff(1*time.Millisecond),
		WithTimeout(time.Second),
	)
	_, err1 := client.Get(context.TODO(), server.URL)
	if err1 == nil {
		t.Error("Expected error from first request")
	}
	_, err2 := client.Get(context.TODO(), server.URL)
	if err2 == nil {
		t.Error("Expected error from second request")
	}
	ce, ok := err2.(*ClientError)
	if !ok {
		t.Fatalf(errFmtExpectedClientErr, err2)
	}
	if ce.Type != ErrorTypeRetryBudgetExceeded {
		t.Errorf(errFmtBudgetExceededType, ce.Type)
	}
}

// duplicate simpler metrics test removed; enhanced metrics assertions are above

func TestRetryPolicyCalculateBackoff(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0)
	tests := []struct {
		attempt  int
		expected time.Duration
	}{{0, 100 * time.Millisecond}, {1, 200 * time.Millisecond}, {2, 400 * time.Millisecond}}
	for _, tt := range tests {
		t.Run("attempt_"+string(rune(tt.attempt+'0')), func(t *testing.T) {
			result := policy.calculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestRetryPolicyCalculateBackoffWithJitter(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.5)
	base := 100 * time.Millisecond
	minExpected := base
	maxExpected := base + time.Duration(float64(base)*0.5)
	result := policy.calculateBackoff(0)
	if result < minExpected || result > maxExpected {
		t.Errorf("calculateBackoff(0) with jitter = %v, expected between %v and %v", result, minExpected, maxExpected)
	}
}

func TestRetryPolicyCalculateBackoffMaxCap(t *testing.T) {
	policy := NewDefaultRetryPolicy(10, 100*time.Millisecond, 500*time.Millisecond, 2.0, 0)
	result := policy.calculateBackoff(10)
	if result != 500*time.Millisecond {
		t.Errorf("calculateBackoff(10) = %v, want %v (maxBackoff)", result, 500*time.Millisecond)
	}
}

func TestRetryAfterParsesHTTPDateFormat(t *testing.T) {
	future := time.Now().Add(45 * time.Second)
	rfc1123 := future.UTC().Format(http.TimeFormat)
	result := parseRetryAfter(rfc1123)
	if result < 40*time.Second || result > 50*time.Second {
		t.Errorf("parseRetryAfter RFC1123 should be around 45s, got %v", result)
	}
	result = parseRetryAfter("invalid-date")
	if result != 0 {
		t.Errorf("parseRetryAfter with invalid date should return 0, got %v", result)
	}
}

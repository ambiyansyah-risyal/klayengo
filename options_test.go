package klayengo

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const testURL = "https://example.com"

const (
	ErrMsgMaxRetriesNegative             = "maxRetries must be non-negative"
	ErrMsgInitialBackoffPositive         = "initialBackoff must be positive"
	ErrMsgMaxBackoffGTEInitial           = "maxBackoff must be greater than or equal to initialBackoff"
	ErrMsgBackoffMultiplierPositive      = "backoffMultiplier must be positive"
	ErrMsgJitterRange                    = "jitter must be between 0 and 1"
	ErrMsgTimeoutPositive                = "timeout must be positive"
	ErrMsgRateLimiterMaxTokens           = "rateLimiter maxTokens must be positive"
	ErrMsgRateLimiterRefillRate          = "rateLimiter refillRate must be positive"
	ErrMsgCacheTTLPositive               = "cacheTTL must be positive when cache is enabled"
	ErrMsgCircuitBreakerFailureThreshold = "circuitBreaker FailureThreshold must be positive"
	ErrMsgCircuitBreakerRecoveryTimeout  = "circuitBreaker RecoveryTimeout must be positive"
	ErrMsgCircuitBreakerSuccessThreshold = "circuitBreaker SuccessThreshold must be positive"
	ErrMsgDebugLoggerRequired            = "logger must be set when debug is enabled"
	ErrMsgDebugRequestIDGenRequired      = "debug RequestIDGen must be set when debug is enabled"
	ErrMsgMiddlewareNil                  = "middleware[0] cannot be nil"
)

const (
	ErrMsgExpectedContains       = "Expected error to contain '%s', got: %v"
	ErrMsgExpectedInvalid        = "Expected configuration to be invalid"
	ErrMsgExpectedInvalidExtreme = "Expected configuration to be invalid due to extreme values"
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
		{-0.1, 0.0},
		{1.5, 1.0},
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

	req, _ := http.NewRequest("GET", testURL, nil)
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

	getReq, _ := http.NewRequest("GET", testURL, nil)
	postReq, _ := http.NewRequest("POST", testURL, nil)

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
		return err != nil
	}

	client := New(WithRetryCondition(customCondition))

	if client.retryCondition == nil {
		t.Fatal("Expected retry condition to be set")
	}

	if !client.retryCondition(nil, http.ErrHandlerTimeout) {
		t.Error("Expected true for error condition")
	}

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

	client := New(
		WithTimeout(30*time.Second),
		WithHTTPClient(customClient),
	)

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

// Lightweight test ensuring WithMetrics() option constructor itself does not panic
// (full metrics behavior covered elsewhere with custom registry to avoid duplicate global registration).
func TestWithMetricsOptionLight(t *testing.T) {
	// Just assert the option factory returns a non-nil functional option.
	if WithMetrics() == nil {
		t.Error("WithMetrics should return a non-nil option")
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

func TestWithDeduplication(t *testing.T) {
	client := New(WithDeduplication())

	if client.deduplication == nil {
		t.Fatal("Expected deduplication tracker to be set")
	}

	if client.dedupKeyFunc == nil {
		t.Fatal("Expected default deduplication key function to be set")
	}

	if client.dedupCondition == nil {
		t.Fatal("Expected default deduplication condition to be set")
	}
}

func TestWithDeduplicationKeyFunc(t *testing.T) {
	customKeyFunc := func(req *http.Request) string {
		return "custom-dedup-key"
	}

	client := New(WithDeduplicationKeyFunc(customKeyFunc))

	if client.dedupKeyFunc == nil {
		t.Fatal("Expected deduplication key function to be set")
	}

	req, _ := http.NewRequest("GET", testURL, nil)
	key := client.dedupKeyFunc(req)

	if key != "custom-dedup-key" {
		t.Errorf("Expected 'custom-dedup-key', got '%s'", key)
	}
}

func TestWithDeduplicationCondition(t *testing.T) {
	customCondition := func(req *http.Request) bool {
		return req.Method == "POST"
	}

	client := New(WithDeduplicationCondition(customCondition))

	if client.dedupCondition == nil {
		t.Fatal("Expected deduplication condition to be set")
	}

	getReq, _ := http.NewRequest("GET", testURL, nil)
	postReq, _ := http.NewRequest("POST", testURL, nil)

	if client.dedupCondition(getReq) {
		t.Error("Expected GET request to not be deduplicated with custom condition")
	}

	if !client.dedupCondition(postReq) {
		t.Error("Expected POST request to be deduplicated with custom condition")
	}
}

func TestValidateConfigurationValid(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
	}{
		{
			name:    "default configuration",
			options: []Option{},
		},
		{
			name: "custom valid configuration",
			options: []Option{
				WithMaxRetries(5),
				WithInitialBackoff(200 * time.Millisecond),
				WithMaxBackoff(20 * time.Second),
				WithBackoffMultiplier(2.5),
				WithJitter(0.2),
				WithTimeout(60 * time.Second),
				WithRateLimiter(100, 1*time.Minute),
				WithCache(10 * time.Minute),
				WithCircuitBreaker(CircuitBreakerConfig{
					FailureThreshold: 3,
					RecoveryTimeout:  30 * time.Second,
					SuccessThreshold: 2,
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.options...)
			if !client.IsValid() {
				t.Errorf("Expected configuration to be valid, got error: %v", client.ValidationError())
			}
		})
	}
}

func TestValidateConfigurationRetryErrors(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		errorMsg string
	}{
		{
			name:     "maxRetries negative",
			options:  []Option{WithMaxRetries(-1)},
			errorMsg: ErrMsgMaxRetriesNegative,
		},
		{
			name:     "initialBackoff zero",
			options:  []Option{WithInitialBackoff(0)},
			errorMsg: ErrMsgInitialBackoffPositive,
		},
		{
			name:     "initialBackoff negative",
			options:  []Option{WithInitialBackoff(-100 * time.Millisecond)},
			errorMsg: ErrMsgInitialBackoffPositive,
		},
		{
			name: "maxBackoff less than initialBackoff",
			options: []Option{
				WithInitialBackoff(1 * time.Second),
				WithMaxBackoff(500 * time.Millisecond),
			},
			errorMsg: ErrMsgMaxBackoffGTEInitial,
		},
		{
			name:     "backoffMultiplier zero",
			options:  []Option{WithBackoffMultiplier(0)},
			errorMsg: ErrMsgBackoffMultiplierPositive,
		},
		{
			name:     "backoffMultiplier negative",
			options:  []Option{WithBackoffMultiplier(-1.0)},
			errorMsg: ErrMsgBackoffMultiplierPositive,
		},
		{
			name:     "jitter negative (clamped to 0)",
			options:  []Option{WithJitter(-0.1)},
			errorMsg: "",
		},
		{
			name:     "jitter greater than 1 (clamped to 1)",
			options:  []Option{WithJitter(1.5)},
			errorMsg: "",
		},
		{
			name:     "timeout zero",
			options:  []Option{WithTimeout(0)},
			errorMsg: ErrMsgTimeoutPositive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.options...)
			if tt.errorMsg == "" {
				if !client.IsValid() {
					t.Errorf("Expected configuration to be valid, got error: %v", client.ValidationError())
				}
			} else {
				if client.IsValid() {
					t.Error(ErrMsgExpectedInvalid)
				}
				if !strings.Contains(client.ValidationError().Error(), tt.errorMsg) {
					t.Errorf(ErrMsgExpectedContains, tt.errorMsg, client.ValidationError())
				}
			}
		})
	}
}

func TestValidateConfigurationRateLimiterErrors(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		errorMsg string
	}{
		{
			name:     "rateLimiter maxTokens zero",
			options:  []Option{WithRateLimiter(0, 1*time.Minute)},
			errorMsg: ErrMsgRateLimiterMaxTokens,
		},
		{
			name:     "rateLimiter refillRate zero",
			options:  []Option{WithRateLimiter(100, 0)},
			errorMsg: ErrMsgRateLimiterRefillRate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.options...)
			if client.IsValid() {
				t.Error(ErrMsgExpectedInvalid)
			}
			if !strings.Contains(client.ValidationError().Error(), tt.errorMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, client.ValidationError())
			}
		})
	}
}

func TestValidateConfigurationCacheErrors(t *testing.T) {
	client := New(WithCache(0))
	if client.IsValid() {
		t.Error(ErrMsgExpectedInvalid)
	}
	if !strings.Contains(client.ValidationError().Error(), ErrMsgCacheTTLPositive) {
		t.Errorf("Expected error to contain '%s', got: %v", ErrMsgCacheTTLPositive, client.ValidationError())
	}
}

func TestValidateConfigurationCircuitBreakerErrors(t *testing.T) {
	client := New(WithCircuitBreaker(CircuitBreakerConfig{}))
	if !client.IsValid() {
		t.Errorf("Expected configuration to be valid with default circuit breaker, got error: %v", client.ValidationError())
	}
	if client.circuitBreaker.config.FailureThreshold != 5 {
		t.Errorf("Expected default FailureThreshold=5, got %d", client.circuitBreaker.config.FailureThreshold)
	}
	if client.circuitBreaker.config.SuccessThreshold != 2 {
		t.Errorf("Expected default SuccessThreshold=2, got %d", client.circuitBreaker.config.SuccessThreshold)
	}
	if client.circuitBreaker.config.RecoveryTimeout != 60*time.Second {
		t.Errorf("Expected default RecoveryTimeout=60s, got %v", client.circuitBreaker.config.RecoveryTimeout)
	}
}

func TestValidateConfigurationDebugErrors(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		errorMsg string
	}{
		{
			name:     "debug without logger",
			options:  []Option{WithDebug()},
			errorMsg: ErrMsgDebugLoggerRequired,
		},
		{
			name: "debug without RequestIDGen",
			options: []Option{
				WithDebugConfig(&DebugConfig{
					Enabled:      true,
					RequestIDGen: nil,
				}),
				WithSimpleLogger(),
			},
			errorMsg: ErrMsgDebugRequestIDGenRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.options...)
			if client.IsValid() {
				t.Error(ErrMsgExpectedInvalid)
			}
			if !strings.Contains(client.ValidationError().Error(), tt.errorMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, client.ValidationError())
			}
		})
	}
}

func TestValidateConfigurationMiddlewareErrors(t *testing.T) {
	client := New(WithMiddleware(nil))
	if client.IsValid() {
		t.Error(ErrMsgExpectedInvalid)
	}
	if !strings.Contains(client.ValidationError().Error(), ErrMsgMiddlewareNil) {
		t.Errorf("Expected error to contain '%s', got: %v", ErrMsgMiddlewareNil, client.ValidationError())
	}
}

func TestValidateConfigurationMultipleErrors(t *testing.T) {
	client := New(
		WithMaxRetries(-1),
		WithInitialBackoff(0),
		WithTimeout(0),
	)

	if client.IsValid() {
		t.Error(ErrMsgExpectedInvalid)
	}

	err := client.ValidationError()
	if err == nil {
		t.Fatal("Expected validation error")
	}

	errStr := err.Error()
	expectedErrors := []string{
		"maxRetries must be non-negative",
		"initialBackoff must be positive",
		"timeout must be positive",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(errStr, expected) {
			t.Errorf("Expected error to contain '%s', got: %s", expected, errStr)
		}
	}
}

func TestValidateConfigurationExtremeValues(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		errorMsg string
	}{
		{
			name:     "extreme maxRetries",
			options:  []Option{WithMaxRetries(200)},
			errorMsg: "maxRetries > 100 may cause excessive resource usage",
		},
		{
			name:     "extreme initialBackoff",
			options:  []Option{WithInitialBackoff(15 * time.Minute)},
			errorMsg: "initialBackoff > 10m may cause very long delays",
		},
		{
			name:     "extreme maxBackoff",
			options:  []Option{WithMaxBackoff(2 * time.Hour)},
			errorMsg: "maxBackoff > 1h may cause extremely long delays",
		},
		{
			name:     "extreme timeout",
			options:  []Option{WithTimeout(15 * time.Minute)},
			errorMsg: "timeout > 10m may cause requests to hang for too long",
		},
		{
			name:     "extreme rateLimiter maxTokens",
			options:  []Option{WithRateLimiter(2000000, 1*time.Minute)},
			errorMsg: "rateLimiter maxTokens > 1M may cause memory issues",
		},
		{
			name:     "extreme rateLimiter refillRate",
			options:  []Option{WithRateLimiter(100, 500*time.Microsecond)},
			errorMsg: "rateLimiter refillRate < 1ms may cause excessive CPU usage",
		},
		{
			name:     "extreme cacheTTL",
			options:  []Option{WithCache(48 * time.Hour)},
			errorMsg: "cacheTTL > 24h may cause stale data issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.options...)
			if client.IsValid() {
				t.Error(ErrMsgExpectedInvalidExtreme)
			}
			if !strings.Contains(client.ValidationError().Error(), tt.errorMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, client.ValidationError())
			}
		})
	}
}

func TestValidateConfigurationStrict(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected ValidateConfigurationStrict to panic for invalid config")
		}
	}()

	client := New(WithMaxRetries(-1))
	client.ValidateConfigurationStrict()
}

func TestMustValidateConfiguration(t *testing.T) {
	client := New(WithMaxRetries(5))
	if err := client.MustValidateConfiguration(); err != nil {
		t.Errorf("Expected no validation error for valid config, got: %v", err)
	}

	client = New(WithMaxRetries(-1))
	if err := client.MustValidateConfiguration(); err == nil {
		t.Error("Expected validation error for invalid config")
	}
}

func TestIsValid(t *testing.T) {
	validClient := New(WithMaxRetries(5))
	if !validClient.IsValid() {
		t.Error("Expected valid client to return true for IsValid()")
	}

	invalidClient := New(WithMaxRetries(-1))
	if invalidClient.IsValid() {
		t.Error("Expected invalid client to return false for IsValid()")
	}
}

func TestValidationError(t *testing.T) {
	validClient := New(WithMaxRetries(5))
	if validClient.ValidationError() != nil {
		t.Errorf("Expected no validation error for valid config, got: %v", validClient.ValidationError())
	}

	invalidClient := New(WithMaxRetries(-1))
	if invalidClient.ValidationError() == nil {
		t.Error("Expected validation error for invalid config")
	}
}

package klayengo

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	typesTestURL         = "https://example.com"
	testContentType      = "Content-Type"
	testContentTypeValue = "application/json"
	testCustomHeader     = "X-Custom"
	testCustomValue      = "value"
	testHeaderFormat     = "Expected %s='%s', got '%s'"
	testStatusFormat     = "Expected status 200, got %d"
)

func TestCircuitStateConstants(t *testing.T) {
	if StateClosed != 0 {
		t.Errorf("Expected StateClosed=0, got %d", StateClosed)
	}

	if StateOpen != 1 {
		t.Errorf("Expected StateOpen=1, got %d", StateOpen)
	}

	if StateHalfOpen != 2 {
		t.Errorf("Expected StateHalfOpen=2, got %d", StateHalfOpen)
	}
}

func TestCircuitBreakerConfig(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		SuccessThreshold: 2,
	}

	if config.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold=5, got %d", config.FailureThreshold)
	}

	if config.RecoveryTimeout != 30*time.Second {
		t.Errorf("Expected RecoveryTimeout=30s, got %v", config.RecoveryTimeout)
	}

	if config.SuccessThreshold != 2 {
		t.Errorf("Expected SuccessThreshold=2, got %d", config.SuccessThreshold)
	}
}

func TestCacheEntry(t *testing.T) {
	body := []byte("test response")
	statusCode := 200
	header := http.Header{testContentType: []string{testContentTypeValue}}

	entry := &CacheEntry{
		Body:       body,
		StatusCode: statusCode,
		Header:     header,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	if string(entry.Body) != "test response" {
		t.Errorf("Expected body='test response', got '%s'", string(entry.Body))
	}

	if entry.StatusCode != 200 {
		t.Errorf("Expected StatusCode=200, got %d", entry.StatusCode)
	}

	if entry.Header.Get(testContentType) != testContentTypeValue {
		t.Errorf(testHeaderFormat, testContentType, testContentTypeValue, entry.Header.Get(testContentType))
	}

	if entry.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestCacheEntryExpiration(t *testing.T) {
	pastTime := time.Now().Add(-1 * time.Hour)

	entry := &CacheEntry{
		Body:      []byte("expired"),
		ExpiresAt: pastTime,
	}

	if !entry.ExpiresAt.Before(time.Now()) {
		t.Error("Entry should be expired")
	}
}

func TestCacheEntryHeader(t *testing.T) {
	header := make(http.Header)
	header.Set(testContentType, testContentTypeValue)
	header.Set(testCustomHeader, testCustomValue)

	entry := &CacheEntry{
		Header: header,
	}

	if entry.Header.Get(testContentType) != testContentTypeValue {
		t.Errorf(testHeaderFormat, testContentType, testContentTypeValue, entry.Header.Get(testContentType))
	}

	if entry.Header.Get(testCustomHeader) != testCustomValue {
		t.Errorf(testHeaderFormat, testCustomHeader, testCustomValue, entry.Header.Get(testCustomHeader))
	}
}

func TestRateLimiterFields(t *testing.T) {
	rl := &RateLimiter{
		tokens:     10,
		maxTokens:  10,
		refillRate: 1 * time.Second,
		lastRefill: time.Now().UnixNano(),
	}

	if rl.tokens != 10 {
		t.Errorf("Expected tokens=10, got %d", rl.tokens)
	}

	if rl.maxTokens != 10 {
		t.Errorf("Expected maxTokens=10, got %d", rl.maxTokens)
	}

	if rl.refillRate != 1*time.Second {
		t.Errorf("Expected refillRate=1s, got %v", rl.refillRate)
	}

	if time.Unix(0, rl.lastRefill).After(time.Now()) {
		t.Error("lastRefill should not be in the future")
	}
}

func TestContextKey(t *testing.T) {
	key := contextKey("test-key")

	if key != "test-key" {
		t.Errorf("Expected contextKey='test-key', got '%s'", key)
	}
}

func TestCacheControl(t *testing.T) {
	control := &CacheControl{
		Enabled: true,
		TTL:     30 * time.Minute,
	}

	if !control.Enabled {
		t.Error("Expected Enabled=true")
	}

	if control.TTL != 30*time.Minute {
		t.Errorf("Expected TTL=30m, got %v", control.TTL)
	}
}

func TestCacheControlDefaults(t *testing.T) {
	control := &CacheControl{}

	if control.Enabled {
		t.Error("Expected Enabled=false by default")
	}

	if control.TTL != 0 {
		t.Errorf("Expected TTL=0 by default, got %v", control.TTL)
	}
}

func TestRoundTripperFunc(t *testing.T) {
	callCount := 0

	roundTripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		return &http.Response{StatusCode: 200}, nil
	})

	req, _ := http.NewRequest("GET", typesTestURL, nil)
	resp, err := roundTripper.RoundTrip(req)

	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	if resp.StatusCode != 200 {
		t.Errorf(testStatusFormat, resp.StatusCode)
	}
}

func TestRoundTripperFuncNil(t *testing.T) {
	var roundTripper RoundTripperFunc

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling RoundTrip on nil RoundTripperFunc")
		}
	}()

	req, _ := http.NewRequest("GET", typesTestURL, nil)
	_, _ = roundTripper.RoundTrip(req)
}

func TestMiddlewareType(t *testing.T) {
	callOrder := []string{}

	middleware := Middleware(func(req *http.Request, next RoundTripper) (*http.Response, error) {
		callOrder = append(callOrder, "middleware")
		return next.RoundTrip(req)
	})

	next := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callOrder = append(callOrder, "next")
		return &http.Response{StatusCode: 200}, nil
	})

	req, _ := http.NewRequest("GET", typesTestURL, nil)
	resp, err := middleware(req, next)

	if err != nil {
		t.Fatalf("Middleware failed: %v", err)
	}

	if len(callOrder) != 2 {
		t.Errorf("Expected 2 calls, got %d", len(callOrder))
	}

	if callOrder[0] != "middleware" || callOrder[1] != "next" {
		t.Errorf("Expected call order ['middleware', 'next'], got %v", callOrder)
	}

	if resp.StatusCode != 200 {
		t.Errorf(testStatusFormat, resp.StatusCode)
	}
}

func TestRetryConditionType(t *testing.T) {
	condition := RetryCondition(func(resp *http.Response, err error) bool {
		return err != nil
	})

	retry := condition(nil, http.ErrHandlerTimeout)
	if !retry {
		t.Error("Expected true for error condition")
	}

	resp := &http.Response{StatusCode: 500}
	retry = condition(resp, nil)
	if retry {
		t.Error("Expected false for 500 response with custom condition")
	}
}

func TestCacheConditionType(t *testing.T) {
	condition := CacheCondition(func(req *http.Request) bool {
		return req.Method == "GET"
	})

	getReq, _ := http.NewRequest("GET", typesTestURL, nil)
	postReq, _ := http.NewRequest("POST", typesTestURL, nil)

	if !condition(getReq) {
		t.Error("Expected true for GET request")
	}

	if condition(postReq) {
		t.Error("Expected false for POST request")
	}
}

func TestOptionType(t *testing.T) {
	callCount := 0

	option := Option(func(c *Client) {
		callCount++
		c.maxRetries = 10
	})

	client := &Client{}
	option(client)

	if callCount != 1 {
		t.Errorf("Expected option to be called once, got %d", callCount)
	}

	if client.maxRetries != 10 {
		t.Errorf("Expected maxRetries=10, got %d", client.maxRetries)
	}
}

func TestCircuitBreakerStateValues(t *testing.T) {
	if StateClosed == StateOpen {
		t.Error("StateClosed should not equal StateOpen")
	}

	if StateOpen == StateHalfOpen {
		t.Error("StateOpen should not equal StateHalfOpen")
	}

	if StateClosed == StateHalfOpen {
		t.Error("StateClosed should not equal StateHalfOpen")
	}
}

func TestCircuitBreakerStateType(t *testing.T) {
	state := StateClosed

	if state != 0 {
		t.Errorf("Expected CircuitState(0), got %d", state)
	}

	state = StateOpen
	if state != 1 {
		t.Errorf("Expected CircuitState(1), got %d", state)
	}

	state = StateHalfOpen
	if state != 2 {
		t.Errorf("Expected CircuitState(2), got %d", state)
	}
}

func TestCacheEntryNilHeader(t *testing.T) {
	entry := &CacheEntry{
		Body:   []byte("test"),
		Header: nil,
	}

	if entry.Header != nil {
		t.Error("Expected nil header")
	}
}

func TestCacheEntryEmptyBody(t *testing.T) {
	entry := &CacheEntry{
		Body:       []byte{},
		StatusCode: 200,
	}

	if len(entry.Body) != 0 {
		t.Errorf("Expected empty body, got length %d", len(entry.Body))
	}

	if entry.StatusCode != 200 {
		t.Errorf(testStatusFormat, entry.StatusCode)
	}
}

func TestRateLimiterZeroValues(t *testing.T) {
	rl := &RateLimiter{}

	if rl.tokens != 0 {
		t.Errorf("Expected tokens=0, got %d", rl.tokens)
	}

	if rl.maxTokens != 0 {
		t.Errorf("Expected maxTokens=0, got %d", rl.maxTokens)
	}

	if rl.refillRate != 0 {
		t.Errorf("Expected refillRate=0, got %v", rl.refillRate)
	}
}

func TestContextKeyString(t *testing.T) {
	key := contextKey("test")
	if string(key) != "test" {
		t.Errorf("Expected string 'test', got '%s'", string(key))
	}
}

func TestCacheControlZeroTTL(t *testing.T) {
	control := &CacheControl{
		Enabled: true,
		TTL:     0,
	}

	if !control.Enabled {
		t.Error("Expected Enabled=true")
	}

	if control.TTL != 0 {
		t.Errorf("Expected TTL=0, got %v", control.TTL)
	}
}

// --- Logger & Request ID (migrated from coverage_test.go) ---
func TestGenerateRequestIDFormat(t *testing.T) {
	id := generateRequestID()
	if len(id) < 5 || !strings.HasPrefix(id, "req_") {
		t.Errorf("Expected request ID with 'req_' prefix, got %s", id)
	}
}

func TestSimpleLoggerBasicCalls(t *testing.T) {
	logger := NewSimpleLogger()
	logger.Debug("dbg")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("err")
}

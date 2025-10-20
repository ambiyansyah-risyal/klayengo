package klayengo

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	testErrorFormat         = "Expected '%s', got '%s'"
	testMessage             = "test message"
	testInternalServerError = "internal server error"
)

func TestClientError(t *testing.T) {
	err := &ClientError{
		Type:    "NetworkError",
		Message: "connection timeout",
	}

	expectedMsg := "NetworkError: connection timeout"
	if err.Error() != expectedMsg {
		t.Errorf(testErrorFormat, expectedMsg, err.Error())
	}

	cause := errors.New("underlying error")
	errWithCause := &ClientError{
		Type:    "ServerError",
		Message: testInternalServerError,
		Cause:   cause,
	}

	expectedMsgWithCause := "ServerError: internal server error (underlying error)"
	if errWithCause.Error() != expectedMsgWithCause {
		t.Errorf(testErrorFormat, expectedMsgWithCause, errWithCause.Error())
	}
}

func TestClientErrorWithRequestID(t *testing.T) {
	err := &ClientError{
		Type:       "NetworkError",
		Message:    "connection timeout",
		RequestID:  "req_123",
		Attempt:    1,
		MaxRetries: 3,
	}

	expectedMsg := "[req_123] NetworkError: connection timeout (attempt 1/3)"
	if err.Error() != expectedMsg {
		t.Errorf(testErrorFormat, expectedMsg, err.Error())
	}
}

func TestClientErrorDebugInfo(t *testing.T) {
	timestamp := time.Now()
	err := &ClientError{
		Type:       "TimeoutError",
		Message:    "request timed out",
		RequestID:  "req_456",
		Method:     "GET",
		URL:        "https://api.example.com/data",
		Attempt:    2,
		MaxRetries: 5,
		Timestamp:  timestamp,
		Duration:   10 * time.Second,
		StatusCode: 0,
		Endpoint:   "api.example.com/data",
		Cause:      errors.New("context deadline exceeded"),
	}

	debugInfo := err.DebugInfo()
	if !strings.Contains(debugInfo, "Error Type: TimeoutError") {
		t.Error("DebugInfo should contain error type")
	}
	if !strings.Contains(debugInfo, "Request ID: req_456") {
		t.Error("DebugInfo should contain request ID")
	}
	if !strings.Contains(debugInfo, "Method: GET") {
		t.Error("DebugInfo should contain method")
	}
	if !strings.Contains(debugInfo, "URL: https://api.example.com/data") {
		t.Error("DebugInfo should contain URL")
	}
	if !strings.Contains(debugInfo, "Attempt: 2/5") {
		t.Error("DebugInfo should contain attempt info")
	}
	if !strings.Contains(debugInfo, "Duration: 10s") {
		t.Error("DebugInfo should contain duration")
	}
	if !strings.Contains(debugInfo, "Cause: context deadline exceeded") {
		t.Error("DebugInfo should contain cause")
	}
}

func TestClientErrorDebugInfoWithStatusCode(t *testing.T) {
	err := &ClientError{
		Type:       "ServerError",
		Message:    testInternalServerError,
		RequestID:  "req_789",
		Method:     "POST",
		URL:        "https://api.example.com/submit",
		Attempt:    1,
		MaxRetries: 3,
		StatusCode: 500,
		Endpoint:   "api.example.com/submit",
		Cause:      errors.New("server returned 500"),
	}

	debugInfo := err.DebugInfo()
	if !strings.Contains(debugInfo, "Status Code: 500") {
		t.Error("DebugInfo should contain status code when > 0")
	}
}

func TestClientErrorUnwrap(t *testing.T) {
	cause := errors.New("original error")
	err := &ClientError{
		Type:    "TestError",
		Message: testMessage,
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected unwrapped error to be %v, got %v", cause, unwrapped)
	}
}

func TestClientErrorUnwrapNilCause(t *testing.T) {
	err := &ClientError{
		Type:    "TestError",
		Message: testMessage,
		Cause:   nil,
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Expected unwrapped error to be nil, got %v", unwrapped)
	}
}

func TestClientErrorTypes(t *testing.T) {
	testCases := []struct {
		errorType string
		message   string
		cause     error
	}{
		{"NetworkError", "connection failed", nil},
		{"TimeoutError", "request timed out", errors.New("deadline exceeded")},
		{"RateLimitError", "rate limit exceeded", nil},
		{"CircuitBreakerError", "circuit breaker open", nil},
		{"ServerError", testInternalServerError, errors.New("500 status")},
	}

	for _, tc := range testCases {
		err := &ClientError{
			Type:    tc.errorType,
			Message: tc.message,
			Cause:   tc.cause,
		}

		if err.Type != tc.errorType {
			t.Errorf("Expected Type='%s', got '%s'", tc.errorType, err.Type)
		}

		if err.Message != tc.message {
			t.Errorf("Expected Message='%s', got '%s'", tc.message, err.Message)
		}

		if err.Cause != tc.cause {
			t.Errorf("Expected Cause=%v, got %v", tc.cause, err.Cause)
		}
	}
}

func TestClientErrorFormatting(t *testing.T) {
	testCases := []struct {
		err      *ClientError
		expected string
	}{
		{
			&ClientError{Type: "Simple", Message: "simple message"},
			"Simple: simple message",
		},
		{
			&ClientError{Type: "WithCause", Message: "with cause", Cause: errors.New("cause")},
			"WithCause: with cause (cause)",
		},
		{
			&ClientError{Type: "", Message: "no type"},
			": no type",
		},
		{
			&ClientError{Type: "EmptyMessage", Message: ""},
			"EmptyMessage: ",
		},
	}

	for _, tc := range testCases {
		result := tc.err.Error()
		if result != tc.expected {
			t.Errorf("Error() = '%s', expected '%s'", result, tc.expected)
		}
	}
}

func TestClientErrorChain(t *testing.T) {
	rootCause := errors.New("root cause")
	middleErr := &ClientError{
		Type:    "MiddleError",
		Message: "middle layer failed",
		Cause:   rootCause,
	}
	topErr := &ClientError{
		Type:    "TopError",
		Message: "top layer failed",
		Cause:   middleErr,
	}

	if topErr.Unwrap() != middleErr {
		t.Error("Top error should unwrap to middle error")
	}

	if middleErr.Unwrap() != rootCause {
		t.Error("Middle error should unwrap to root cause")
	}

	if rootCause != errors.New("root cause") {
		_ = rootCause
	}
}

func TestClientErrorAs(t *testing.T) {
	err := &ClientError{
		Type:    "TestError",
		Message: testMessage,
	}

	var clientErr *ClientError
	if !errors.As(err, &clientErr) {
		t.Error("Should be able to cast to ClientError")
	}

	if clientErr.Type != "TestError" {
		t.Errorf("Casted error Type should be 'TestError', got '%s'", clientErr.Type)
	}
}

func TestClientErrorIs(t *testing.T) {
	err1 := &ClientError{Type: "NetworkError", Message: "connection failed"}
	err2 := &ClientError{Type: "NetworkError", Message: "timeout"}

	_ = err2

	if !errors.Is(err1, &ClientError{Type: "NetworkError"}) {
		t.Error("Should match errors with same type")
	}

	if errors.Is(err1, &ClientError{Type: "DifferentError"}) {
		t.Error("Should not match errors with different types")
	}

	if errors.Is(err1, errors.New("some error")) {
		t.Error("Should not match non-ClientError types")
	}

}

func TestClientErrorNilHandling(t *testing.T) {
	var err *ClientError

	result := err.Error()
	if result != "<nil>" {
		t.Errorf("Nil error Error() should return '<nil>', got '%s'", result)
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Nil error Unwrap() should return nil, got %v", unwrapped)
	}

	targetErr := &ClientError{Type: "test"}
	if err.Is(targetErr) {
		t.Error("Nil error Is() should return false")
	}

	if targetErr.Type != "test" {
		t.Errorf("Target error type should be 'test', got '%s'", targetErr.Type)
	}

	debugInfo := err.DebugInfo()
	if debugInfo != "Error: <nil>" {
		t.Errorf("Nil error DebugInfo() should return 'Error: <nil>', got '%s'", debugInfo)
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		expected string
	}{
		{"ErrCircuitOpen", ErrCircuitOpen, "klayengo: circuit open"},
		{"ErrRateLimited", ErrRateLimited, "klayengo: rate limited"},
		{"ErrCacheMiss", ErrCacheMiss, "klayengo: cache miss"},
		{"ErrRetryBudgetExceeded", ErrRetryBudgetExceeded, "klayengo: retry budget exceeded"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.sentinel.Error() != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, test.sentinel.Error())
			}
		})
	}
}

func TestErrorsIs(t *testing.T) {
	// Test that ClientError with wrapped sentinels work with errors.Is
	networkErr := &ClientError{
		Type:    ErrorTypeNetwork,
		Message: "network request failed",
		Cause:   fmt.Errorf("network request failed: %w", fmt.Errorf("connection timeout")),
	}

	rateLimitErr := &ClientError{
		Type:    ErrorTypeRateLimit,
		Message: "rate limit exceeded",
		Cause:   fmt.Errorf("rate limit exceeded: %w", ErrRateLimited),
	}

	circuitErr := &ClientError{
		Type:    ErrorTypeCircuitOpen,
		Message: "circuit breaker is open",
		Cause:   fmt.Errorf("circuit breaker is open: %w", ErrCircuitOpen),
	}

	retryBudgetErr := &ClientError{
		Type:    ErrorTypeRetryBudgetExceeded,
		Message: "retry budget exceeded",
		Cause:   fmt.Errorf("retry budget exceeded: %w", ErrRetryBudgetExceeded),
	}

	// Test that errors.Is works with our sentinel errors
	if !errors.Is(rateLimitErr, ErrRateLimited) {
		t.Error("Rate limit error should match ErrRateLimited sentinel")
	}

	if !errors.Is(circuitErr, ErrCircuitOpen) {
		t.Error("Circuit breaker error should match ErrCircuitOpen sentinel")
	}

	if !errors.Is(retryBudgetErr, ErrRetryBudgetExceeded) {
		t.Error("Retry budget error should match ErrRetryBudgetExceeded sentinel")
	}

	// Test that errors.Is doesn't match wrong sentinels
	if errors.Is(networkErr, ErrRateLimited) {
		t.Error("Network error should not match ErrRateLimited sentinel")
	}

	if errors.Is(rateLimitErr, ErrCircuitOpen) {
		t.Error("Rate limit error should not match ErrCircuitOpen sentinel")
	}
}

func TestErrorsAs(t *testing.T) {
	originalErr := &ClientError{
		Type:    ErrorTypeServer,
		Message: "server error",
		Cause:   fmt.Errorf("server error: %w", fmt.Errorf("500 status")),
	}

	// Test that errors.As works with ClientError
	var clientErr *ClientError
	if !errors.As(originalErr, &clientErr) {
		t.Error("Should be able to extract ClientError with errors.As")
	}

	if clientErr.Type != ErrorTypeServer {
		t.Errorf("Expected Type='%s', got '%s'", ErrorTypeServer, clientErr.Type)
	}

	// Test that errors.As works through wrapping
	wrappedErr := fmt.Errorf("wrapped: %w", originalErr)
	if !errors.As(wrappedErr, &clientErr) {
		t.Error("Should be able to extract ClientError through wrapping")
	}

	if clientErr.Type != ErrorTypeServer {
		t.Errorf("Expected Type='%s' through wrapping, got '%s'", ErrorTypeServer, clientErr.Type)
	}
}

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		transient bool
	}{
		{
			name:      "nil error",
			err:       nil,
			transient: false,
		},
		{
			name:      "circuit open sentinel",
			err:       ErrCircuitOpen,
			transient: true,
		},
		{
			name:      "rate limited sentinel",
			err:       ErrRateLimited,
			transient: true,
		},
		{
			name:      "retry budget exceeded sentinel",
			err:       ErrRetryBudgetExceeded,
			transient: true,
		},
		{
			name:      "cache miss sentinel",
			err:       ErrCacheMiss,
			transient: false,
		},
		{
			name: "network error",
			err: &ClientError{
				Type:    ErrorTypeNetwork,
				Message: "connection failed",
			},
			transient: true,
		},
		{
			name: "timeout error",
			err: &ClientError{
				Type:    ErrorTypeTimeout,
				Message: "request timed out",
			},
			transient: true,
		},
		{
			name: "server error (5xx)",
			err: &ClientError{
				Type:       ErrorTypeServer,
				Message:    "internal server error",
				StatusCode: 500,
			},
			transient: true,
		},
		{
			name: "client error (4xx - not 429)",
			err: &ClientError{
				Type:       ErrorTypeClient,
				Message:    "bad request",
				StatusCode: 400,
			},
			transient: false,
		},
		{
			name: "client error (429 Too Many Requests)",
			err: &ClientError{
				Type:       ErrorTypeClient,
				Message:    "too many requests",
				StatusCode: 429,
			},
			transient: true,
		},
		{
			name: "configuration error",
			err: &ClientError{
				Type:    ErrorTypeConfig,
				Message: "invalid configuration",
			},
			transient: false,
		},
		{
			name:      "non-ClientError",
			err:       fmt.Errorf("some random error"),
			transient: false,
		},
		{
			name:      "wrapped sentinel error",
			err:       fmt.Errorf("wrapped: %w", ErrRateLimited),
			transient: true,
		},
		{
			name: "wrapped ClientError",
			err: fmt.Errorf("wrapper: %w", &ClientError{
				Type:    ErrorTypeNetwork,
				Message: "connection failed",
			}),
			transient: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsTransient(test.err)
			if result != test.transient {
				t.Errorf("IsTransient(%v) = %v, expected %v", test.err, result, test.transient)
			}
		})
	}
}

func TestErrorWrappingChain(t *testing.T) {
	// Test that error wrapping preserves the root cause through multiple layers
	rootCause := fmt.Errorf("connection refused")

	wrappedErr := &ClientError{
		Type:    ErrorTypeNetwork,
		Message: "network request failed",
		Cause:   fmt.Errorf("network request failed: %w", rootCause),
	}

	outerErr := fmt.Errorf("request failed: %w", wrappedErr)

	// Should be able to unwrap to ClientError
	var clientErr *ClientError
	if !errors.As(outerErr, &clientErr) {
		t.Error("Should be able to extract ClientError from wrapped chain")
	}

	// Should be able to find root cause
	if !errors.Is(outerErr, rootCause) {
		t.Error("Should be able to find root cause through wrapping chain")
	}

	// Should preserve ClientError type matching
	if !errors.Is(outerErr, &ClientError{Type: ErrorTypeNetwork}) {
		t.Error("Should match ClientError type through wrapping chain")
	}
}

func FuzzErrorWrapping(f *testing.F) {
	// Add seed inputs
	f.Add("network error", "connection failed")
	f.Add("timeout", "request timed out")
	f.Add("server error", "internal server error")

	f.Fuzz(func(t *testing.T, errorType, message string) {
		// Create a root cause error
		rootCause := fmt.Errorf("root cause: %s", message)

		// Create a ClientError with wrapped root cause
		clientErr := &ClientError{
			Type:    errorType,
			Message: message,
			Cause:   fmt.Errorf("%s: %w", message, rootCause),
		}

		// Wrap it multiple times
		wrapped1 := fmt.Errorf("layer1: %w", clientErr)
		wrapped2 := fmt.Errorf("layer2: %w", wrapped1)
		wrapped3 := fmt.Errorf("layer3: %w", wrapped2)

		// Should always be able to extract the ClientError
		var extractedErr *ClientError
		if !errors.As(wrapped3, &extractedErr) {
			t.Error("Should always be able to extract ClientError from wrapped chain")
		}

		// Should preserve the original type and message
		if extractedErr.Type != errorType {
			t.Errorf("Error type not preserved: expected %s, got %s", errorType, extractedErr.Type)
		}

		if extractedErr.Message != message {
			t.Errorf("Error message not preserved: expected %s, got %s", message, extractedErr.Message)
		}

		// Should be able to find root cause
		if !errors.Is(wrapped3, rootCause) {
			t.Error("Root cause should be findable through wrapping chain")
		}
	})
}

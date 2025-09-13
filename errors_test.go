package klayengo

import (
	"errors"
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

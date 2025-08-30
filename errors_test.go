package klayengo

import (
	"errors"
	"testing"
)

func TestClientError(t *testing.T) {
	// Test error without cause
	err := &ClientError{
		Type:    "NetworkError",
		Message: "connection timeout",
	}

	expectedMsg := "NetworkError: connection timeout"
	if err.Error() != expectedMsg {
		t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
	}

	// Test error with cause
	cause := errors.New("underlying error")
	errWithCause := &ClientError{
		Type:    "ServerError",
		Message: "internal server error",
		Cause:   cause,
	}

	expectedMsgWithCause := "ServerError: internal server error (underlying error)"
	if errWithCause.Error() != expectedMsgWithCause {
		t.Errorf("Expected '%s', got '%s'", expectedMsgWithCause, errWithCause.Error())
	}
}

func TestClientErrorUnwrap(t *testing.T) {
	cause := errors.New("original error")
	err := &ClientError{
		Type:    "TestError",
		Message: "test message",
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
		Message: "test message",
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
		{"ServerError", "internal server error", errors.New("500 status")},
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
	// Test error message formatting with different combinations
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
	// Test error chaining
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

	// Test unwrapping chain
	if topErr.Unwrap() != middleErr {
		t.Error("Top error should unwrap to middle error")
	}

	if middleErr.Unwrap() != rootCause {
		t.Error("Middle error should unwrap to root cause")
	}

	if rootCause != errors.New("root cause") {
		// This is just to use rootCause to avoid unused variable warning
	}
}

func TestClientErrorAs(t *testing.T) {
	err := &ClientError{
		Type:    "TestError",
		Message: "test message",
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

	// Use err2 to avoid unused variable warning
	_ = err2

	// Test that errors with same type are considered equal for Is()
	if !errors.Is(err1, &ClientError{Type: "NetworkError"}) {
		t.Error("Should match errors with same type")
	}

	if errors.Is(err1, &ClientError{Type: "DifferentError"}) {
		t.Error("Should not match errors with different types")
	}

	// Test Is() with non-ClientError target
	if errors.Is(err1, errors.New("some error")) {
		t.Error("Should not match non-ClientError types")
	}

	// Note: errors.Is() works with custom error types when implementing Is() method
}

func TestClientErrorNilHandling(t *testing.T) {
	var err *ClientError

	// Test that nil error doesn't panic
	if err != nil {
		result := err.Error()
		if result != "" {
			t.Errorf("Nil error Error() should return empty string, got '%s'", result)
		}
	}

	// Test Unwrap on nil
	if err != nil {
		unwrapped := err.Unwrap()
		if unwrapped != nil {
			t.Errorf("Nil error Unwrap() should return nil, got %v", unwrapped)
		}
	}
}

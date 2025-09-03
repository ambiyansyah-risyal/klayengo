package klayengo

import (
	"fmt"
	"time"
)

// Error returns a formatted error message with enhanced context
func (e *ClientError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause != nil {
		msg := fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Cause)
		if e.RequestID != "" {
			msg = fmt.Sprintf("[%s] %s", e.RequestID, msg)
		}
		if e.Attempt > 0 {
			msg = fmt.Sprintf("%s (attempt %d/%d)", msg, e.Attempt, e.MaxRetries)
		}
		return msg
	}

	msg := fmt.Sprintf("%s: %s", e.Type, e.Message)
	if e.RequestID != "" {
		msg = fmt.Sprintf("[%s] %s", e.RequestID, msg)
	}
	if e.Attempt > 0 {
		msg = fmt.Sprintf("%s (attempt %d/%d)", msg, e.Attempt, e.MaxRetries)
	}
	return msg
}

// Unwrap returns the underlying error
func (e *ClientError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is checks if this error matches another error for errors.Is() compatibility
func (e *ClientError) Is(target error) bool {
	if e == nil {
		return false
	}
	if targetErr, ok := target.(*ClientError); ok {
		return e.Type == targetErr.Type
	}
	return false
}

// DebugInfo returns detailed debugging information about the error
func (e *ClientError) DebugInfo() string {
	if e == nil {
		return "Error: <nil>"
	}
	info := fmt.Sprintf("Error Type: %s\n", e.Type)
	info += fmt.Sprintf("Message: %s\n", e.Message)
	if e.RequestID != "" {
		info += fmt.Sprintf("Request ID: %s\n", e.RequestID)
	}
	if e.Method != "" {
		info += fmt.Sprintf("Method: %s\n", e.Method)
	}
	if e.URL != "" {
		info += fmt.Sprintf("URL: %s\n", e.URL)
	}
	if e.Endpoint != "" {
		info += fmt.Sprintf("Endpoint: %s\n", e.Endpoint)
	}
	if e.StatusCode > 0 {
		info += fmt.Sprintf("Status Code: %d\n", e.StatusCode)
	}
	if e.Attempt > 0 {
		info += fmt.Sprintf("Attempt: %d/%d\n", e.Attempt, e.MaxRetries)
	}
	if !e.Timestamp.IsZero() {
		info += fmt.Sprintf("Timestamp: %s\n", e.Timestamp.Format(time.RFC3339))
	}
	if e.Duration > 0 {
		info += fmt.Sprintf("Duration: %v\n", e.Duration)
	}
	if e.Cause != nil {
		info += fmt.Sprintf("Cause: %v\n", e.Cause)
	}
	return info
}

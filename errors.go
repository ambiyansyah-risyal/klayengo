package klayengo

import "fmt"

// Error returns a formatted error message
func (e *ClientError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *ClientError) Unwrap() error {
	return e.Cause
}

// Is checks if this error matches another error for errors.Is() compatibility
func (e *ClientError) Is(target error) bool {
	if targetErr, ok := target.(*ClientError); ok {
		return e.Type == targetErr.Type
	}
	return false
}

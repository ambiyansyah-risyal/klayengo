package klayengo

import "testing"

// Logger focused tests migrated from coverage_test.go to keep tests organized by concern.
// These are light smoke tests ensuring exported logger APIs do not panic and remain callable.
// If richer logging behavior (format, sinks, filtering) is added later, expand assertions here.
func TestSimpleLoggerLevels(t *testing.T) {
	logger := NewSimpleLogger()

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestSimpleLoggerReusability(t *testing.T) {
	logger := NewSimpleLogger()
	for i := 0; i < 5; i++ {
		logger.Info("loop message")
	}
}

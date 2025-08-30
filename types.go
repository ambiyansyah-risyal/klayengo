package klayengo

import (
	"fmt"
	"net/http"
	"time"
)

// RetryCondition determines whether a request should be retried
type RetryCondition func(resp *http.Response, err error) bool

// Middleware represents a middleware function
type Middleware func(req *http.Request, next RoundTripper) (*http.Response, error)

// RoundTripper represents the HTTP transport interface
type RoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	SuccessThreshold int
}

// CircuitBreaker represents a circuit breaker with atomic operations
type CircuitBreaker struct {
	config      CircuitBreakerConfig
	state       int64 // CircuitState as int64 for atomic operations
	failures    int64
	lastFailure int64 // Unix nano timestamp
	successes   int64
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// CacheEntry represents a cached response
type CacheEntry struct {
	Response   *http.Response
	Body       []byte
	StatusCode int
	Header     http.Header
	ExpiresAt  time.Time
}

// Cache interface for response caching
type Cache interface {
	Get(key string) (*CacheEntry, bool)
	Set(key string, entry *CacheEntry, ttl time.Duration)
	Delete(key string)
	Clear()
}

// CacheCondition determines whether a request should be cached
type CacheCondition func(req *http.Request) bool

// Context keys for cache control
type contextKey string

const (
	CacheControlKey contextKey = "klayengo_cache_control"
)

// CacheControl holds cache control options for a request
type CacheControl struct {
	Enabled bool
	TTL     time.Duration
}

// ClientError represents an error from the client
type ClientError struct {
	Type       string        // Error type (NetworkError, TimeoutError, RateLimitError, etc.)
	Message    string        // Human-readable error message
	Cause      error         // Underlying error that caused this error
	RequestID  string        // Unique identifier for request tracing
	Method     string        // HTTP method
	URL        string        // Request URL
	Attempt    int           // Current retry attempt (0-based)
	MaxRetries int           // Maximum retry attempts configured
	Timestamp  time.Time     // When the error occurred
	Duration   time.Duration // How long the request took before failing
	StatusCode int           // HTTP status code if applicable (0 if not HTTP-related)
	Endpoint   string        // Simplified endpoint for metrics/logging
}

// Error types for better categorization
const (
	ErrorTypeNetwork     = "NetworkError"
	ErrorTypeTimeout     = "TimeoutError"
	ErrorTypeRateLimit   = "RateLimitError"
	ErrorTypeCircuitOpen = "CircuitBreakerError"
	ErrorTypeServer      = "ServerError"
	ErrorTypeClient      = "ClientError"
	ErrorTypeCache       = "CacheError"
	ErrorTypeConfig      = "ConfigurationError"
	ErrorTypeValidation  = "ValidationError"
)

// RateLimiter represents a simple rate limiter with atomic operations
type RateLimiter struct {
	tokens     int64
	maxTokens  int64
	refillRate time.Duration
	lastRefill int64 // Unix nano timestamp
}

// Option represents a configuration option
type Option func(*Client)

// Logger interface for debug logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// DebugConfig holds debug configuration
type DebugConfig struct {
	Enabled      bool          // Enable debug logging
	LogRequests  bool          // Log all requests
	LogRetries   bool          // Log retry attempts
	LogCache     bool          // Log cache operations
	LogRateLimit bool          // Log rate limiting
	LogCircuit   bool          // Log circuit breaker state changes
	RequestIDGen func() string // Function to generate request IDs
}

// DefaultDebugConfig returns default debug configuration
func DefaultDebugConfig() *DebugConfig {
	return &DebugConfig{
		Enabled:      false,
		LogRequests:  true,
		LogRetries:   true,
		LogCache:     false,
		LogRateLimit: false,
		LogCircuit:   true,
		RequestIDGen: generateRequestID,
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// SimpleLogger is a basic logger implementation
type SimpleLogger struct{}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}

// Debug logs debug messages
func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[DEBUG] "+msg+"\n", args...)
	} else {
		fmt.Printf("[DEBUG] %s\n", msg)
	}
}

// Info logs info messages
func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[INFO] "+msg+"\n", args...)
	} else {
		fmt.Printf("[INFO] %s\n", msg)
	}
}

// Warn logs warning messages
func (l *SimpleLogger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[WARN] "+msg+"\n", args...)
	} else {
		fmt.Printf("[WARN] %s\n", msg)
	}
}

// Error logs error messages
func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[ERROR] "+msg+"\n", args...)
	} else {
		fmt.Printf("[ERROR] %s\n", msg)
	}
}

// RoundTripperFunc is a helper type for middleware
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

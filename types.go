package klayengo

import (
	"fmt"
	"net/http"
	"time"
)

type RetryCondition func(resp *http.Response, err error) bool

type Middleware func(req *http.Request, next RoundTripper) (*http.Response, error)

type RoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	SuccessThreshold int
}

type CircuitBreaker struct {
	config      CircuitBreakerConfig
	state       int64
	failures    int64
	lastFailure int64
	successes   int64
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CacheEntry struct {
	Response   *http.Response
	Body       []byte
	StatusCode int
	Header     http.Header
	ExpiresAt  time.Time
}

type Cache interface {
	Get(key string) (*CacheEntry, bool)
	Set(key string, entry *CacheEntry, ttl time.Duration)
	Delete(key string)
	Clear()
}

type CacheCondition func(req *http.Request) bool

type contextKey string

const (
	CacheControlKey contextKey = "klayengo_cache_control"
)

type CacheControl struct {
	Enabled bool
	TTL     time.Duration
}

type ClientError struct {
	Type       string
	Message    string
	Cause      error
	RequestID  string
	Method     string
	URL        string
	Attempt    int
	MaxRetries int
	Timestamp  time.Time
	Duration   time.Duration
	StatusCode int
	Endpoint   string
}

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

type RateLimiter struct {
	tokens     int64
	maxTokens  int64
	refillRate time.Duration
	lastRefill int64
}

type Option func(*Client)

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type DebugConfig struct {
	Enabled      bool
	LogRequests  bool
	LogRetries   bool
	LogCache     bool
	LogRateLimit bool
	LogCircuit   bool
	RequestIDGen func() string
}

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

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

type SimpleLogger struct{}

func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[DEBUG] "+msg+"\n", args...)
	} else {
		fmt.Printf("[DEBUG] %s\n", msg)
	}
}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[INFO] "+msg+"\n", args...)
	} else {
		fmt.Printf("[INFO] %s\n", msg)
	}
}

func (l *SimpleLogger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[WARN] "+msg+"\n", args...)
	} else {
		fmt.Printf("[WARN] %s\n", msg)
	}
}

func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("[ERROR] "+msg+"\n", args...)
	} else {
		fmt.Printf("[ERROR] %s\n", msg)
	}
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

package klayengo

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	internalbackoff "github.com/ambiyansyah-risyal/klayengo/internal/backoff"
)

// RetryCondition returns true if the operation should be retried.
type RetryCondition func(resp *http.Response, err error) bool

// Middleware composes request handling around the underlying transport.
type Middleware func(req *http.Request, next RoundTripper) (*http.Response, error)

// RoundTripper minimal interface subset for middleware chaining.
type RoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

// CircuitBreakerConfig defines thresholds for circuit breaker transitions.
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	SuccessThreshold int
}

// CircuitBreaker is a lock-free state machine implementing open/half-open/closed.
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

// CacheEntry stores a cached HTTP response body + metadata.
type CacheEntry struct {
	Response   *http.Response
	Body       []byte
	StatusCode int
	Header     http.Header
	ExpiresAt  time.Time
	// HTTP cache semantics fields
	ETag         string
	LastModified *time.Time
	MaxAge       *time.Duration
	StaleAt      *time.Time // When entry becomes stale for SWR
	IsStale      bool       // Whether entry is stale but still servable in SWR mode
}

// Cache abstracts a simple TTL key/value store used for responses.
type Cache interface {
	Get(key string) (*CacheEntry, bool)
	Set(key string, entry *CacheEntry, ttl time.Duration)
	Delete(key string)
	Clear()
}

// CacheProvider is the new interface supporting HTTP cache semantics and context.
type CacheProvider interface {
	Get(ctx context.Context, key string) (resp *http.Response, ok bool)
	Set(ctx context.Context, key string, resp *http.Response, ttl time.Duration)
	Invalidate(ctx context.Context, key string)
}

// CacheMode determines cache behavior strategy.
type CacheMode int

const (
	// TTLOnly uses fixed TTL-based caching (default behavior).
	TTLOnly CacheMode = iota
	// HTTPSemantics honors Cache-Control, ETag, Last-Modified headers.
	HTTPSemantics
	// SWR enables Stale-While-Revalidate mode.
	SWR
)

// singleFlightEntry represents a single in-flight request for stampede protection.
type singleFlightEntry struct {
	wg   sync.WaitGroup
	resp *http.Response
	err  error
	done bool
}

// CacheCondition returns true if a request should be cached.
type CacheCondition func(req *http.Request) bool

type contextKey string

const (
	CacheControlKey contextKey = "klayengo_cache_control"
)

// CacheControl provides per-request overrides for cache behavior.
type CacheControl struct {
	Enabled bool
	TTL     time.Duration
}

// ClientError wraps contextual information about a request failure.
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
	// ErrorTypeNetwork classifies network-related failures (I/O, DNS, connection).
	ErrorTypeNetwork = "NetworkError"
	// ErrorTypeTimeout classifies request timeout failures.
	ErrorTypeTimeout = "TimeoutError"
	// ErrorTypeRateLimit classifies rate limiter denials.
	ErrorTypeRateLimit = "RateLimitError"
	// ErrorTypeCircuitOpen classifies open circuit breaker denials.
	ErrorTypeCircuitOpen = "CircuitBreakerError"
	// ErrorTypeServer classifies 5xx server responses.
	ErrorTypeServer = "ServerError"
	// ErrorTypeClient classifies 4xx client responses or validation at request level.
	ErrorTypeClient = "ClientError"
	// ErrorTypeCache classifies cache-related failures.
	ErrorTypeCache = "CacheError"
	// ErrorTypeConfig classifies configuration errors during setup.
	ErrorTypeConfig = "ConfigurationError"
	// ErrorTypeValidation classifies aggregated validation errors.
	ErrorTypeValidation = "ValidationError"
	// ErrorTypeRetryBudgetExceeded indicates retry was denied due to budget exhaustion.
	ErrorTypeRetryBudgetExceeded = "RetryBudgetExceededError"
)

// Limiter interface abstracts rate limiting behavior.
type Limiter interface {
	Allow() bool
}

// KeyFunc generates a rate limiter key from a request.
type KeyFunc func(*http.Request) string

// RateLimiter is a token bucket implementation.
type RateLimiter struct {
	tokens     int64
	maxTokens  int64
	refillRate time.Duration
	lastRefill int64
}

// RateLimiterRegistry manages per-key rate limiters with concurrent access safety.
type RateLimiterRegistry struct {
	limiters map[string]Limiter
	keyFunc  KeyFunc
	fallback Limiter
	mutex    sync.RWMutex
}

// Option configures a Client instance.
type Option func(*Client)

// Logger is a minimal structured logging interface.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// DebugConfig toggles verbose instrumentation for a client instance.
type DebugConfig struct {
	Enabled      bool
	LogRequests  bool
	LogRetries   bool
	LogCache     bool
	LogRateLimit bool
	LogCircuit   bool
	RequestIDGen func() string
}

// DefaultDebugConfig returns a disabled debug configuration.
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

// NewSimpleLogger returns a basic stdout logger.
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

// RoundTrip implements RoundTripper.
func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// RetryPolicy determines retry behavior including delays and retry conditions.
type RetryPolicy interface {
	ShouldRetry(resp *http.Response, err error, attempt int) (delay time.Duration, ok bool)
}

// RetryBudget tracks retry attempts within time windows to prevent thundering herd.
type RetryBudget struct {
	maxRetries  int64
	perWindow   time.Duration
	window      int64
	current     int64
	windowStart int64
}

// DefaultRetryPolicy implements the standard retry policy with configurable backoff strategy.
type DefaultRetryPolicy struct {
	maxRetries        int
	initialBackoff    time.Duration
	maxBackoff        time.Duration
	backoffMultiplier float64
	jitter            float64
	backoffStrategy   BackoffStrategy
	backoffCalculator *internalbackoff.Calculator
	isIdempotent      func(method string) bool
}

// BackoffStrategy defines the algorithm used for calculating retry delays.
type BackoffStrategy int

const (
	// ExponentialJitter applies exponential backoff with uniform jitter (current default).
	ExponentialJitter BackoffStrategy = iota
	// DecorrelatedJitter applies decorrelated jitter as per AWS paper for smoother tail latencies.
	DecorrelatedJitter
)

// String returns the string representation of the BackoffStrategy.
func (bs BackoffStrategy) String() string {
	switch bs {
	case ExponentialJitter:
		return "ExponentialJitter"
	case DecorrelatedJitter:
		return "DecorrelatedJitter"
	default:
		return "Unknown"
	}
}



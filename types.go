package klayengo

import (
	"net/http"
	"sync"
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

// CircuitBreaker represents a circuit breaker
type CircuitBreaker struct {
	config      CircuitBreakerConfig
	state       CircuitState
	failures    int
	lastFailure time.Time
	successes   int
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
	Type    string
	Message string
	Cause   error
}

// RateLimiter represents a simple rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// Option represents a configuration option
type Option func(*Client)

// RoundTripperFunc is a helper type for middleware
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

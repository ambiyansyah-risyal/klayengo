package klayengo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Client represents a resilient HTTP client with retry logic
type Client struct {
	httpClient        *http.Client
	maxRetries        int
	initialBackoff    time.Duration
	maxBackoff        time.Duration
	backoffMultiplier float64
	jitter            float64
	timeout           time.Duration
	retryCondition    RetryCondition
	circuitBreaker    *CircuitBreaker
	middleware        []Middleware
	rateLimiter       *RateLimiter
	cache             Cache
	cacheTTL          time.Duration
	cacheKeyFunc      func(*http.Request) string
	cacheCondition    CacheCondition
	metrics           *MetricsCollector
}

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

// InMemoryCache is a simple in-memory cache implementation
type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string]*CacheEntry
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		store: make(map[string]*CacheEntry),
	}
}

// Get retrieves a cached entry
func (c *InMemoryCache) Get(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.store[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		delete(c.store, key)
		return nil, false
	}

	return entry, true
}

// Set stores a cache entry
func (c *InMemoryCache) Set(key string, entry *CacheEntry, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry.ExpiresAt = time.Now().Add(ttl)
	c.store[key] = entry
}

// Delete removes a cache entry
func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
}

// Clear removes all cache entries
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*CacheEntry)
}

// createResponseFromCache creates an HTTP response from a cached entry
func (c *Client) createResponseFromCache(entry *CacheEntry) *http.Response {
	resp := &http.Response{
		StatusCode: entry.StatusCode,
		Header:     entry.Header,
		Body:       io.NopCloser(bytes.NewReader(entry.Body)),
	}
	return resp
}

// createCacheEntry creates a cache entry from an HTTP response
func (c *Client) createCacheEntry(resp *http.Response) *CacheEntry {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Restore the body for the caller
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return &CacheEntry{
		Response:   resp,
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
	}
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// Error types
type ClientError struct {
	Type    string
	Message string
	Cause   error
}

func (e *ClientError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ClientError) Unwrap() error {
	return e.Cause
}

// RateLimiter represents a simple rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		maxTokens:  maxTokens,
		tokens:     maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed by the rate limiter
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// Option represents a configuration option
type Option func(*Client)

// DefaultCacheKeyFunc generates a cache key from the request
func DefaultCacheKeyFunc(req *http.Request) string {
	return fmt.Sprintf("%s:%s", req.Method, req.URL.String())
}

// DefaultCacheCondition determines if a request should be cached
func DefaultCacheCondition(req *http.Request) bool {
	// Only cache GET requests by default
	return req.Method == "GET"
}

// shouldCacheRequest determines if a request should be cached
func (c *Client) shouldCacheRequest(req *http.Request) bool {
	// Cache must be enabled
	if c.cache == nil {
		return false
	}

	// Check context-based cache control
	if cacheControl, ok := req.Context().Value(CacheControlKey).(*CacheControl); ok {
		return cacheControl.Enabled
	}

	// Check cache condition
	return c.cacheCondition(req)
}

// getCacheTTLForRequest gets the TTL for a request
func (c *Client) getCacheTTLForRequest(req *http.Request) time.Duration {
	// Check context-based cache control
	if cacheControl, ok := req.Context().Value(CacheControlKey).(*CacheControl); ok && cacheControl.TTL > 0 {
		return cacheControl.TTL
	}

	// Use default TTL
	return c.cacheTTL
}

// WithContextCacheEnabled creates a context that enables caching for the request
func WithContextCacheEnabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, CacheControlKey, &CacheControl{Enabled: true})
}

// WithContextCacheDisabled creates a context that disables caching for the request
func WithContextCacheDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, CacheControlKey, &CacheControl{Enabled: false})
}

// WithContextCacheTTL creates a context with custom TTL for the request
func WithContextCacheTTL(ctx context.Context, ttl time.Duration) context.Context {
	cacheControl := &CacheControl{Enabled: true, TTL: ttl}
	return context.WithValue(ctx, CacheControlKey, cacheControl)
}

// New creates a new retry client with default options
func New(options ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries:        3,
		initialBackoff:    100 * time.Millisecond,
		maxBackoff:        10 * time.Second,
		backoffMultiplier: 2.0,
		jitter:            0.1, // 10% jitter by default
		timeout:           30 * time.Second,
		retryCondition:    DefaultRetryCondition,
		circuitBreaker:    NewCircuitBreaker(CircuitBreakerConfig{}),
		middleware:        []Middleware{},
		rateLimiter:       nil, // No rate limiting by default
		cache:             nil, // No caching by default
		cacheTTL:          5 * time.Minute,
		cacheKeyFunc:      DefaultCacheKeyFunc,
		cacheCondition:    DefaultCacheCondition,
		metrics:           nil, // No metrics by default
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// DefaultRetryCondition is the default retry condition
func DefaultRetryCondition(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp.StatusCode >= 500
}

// Get performs a GET request with retry logic
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post performs a POST request with retry logic
func (c *Client) Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// Do executes the HTTP request with retry logic
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	start := time.Now()
	endpoint := getEndpointFromRequest(req)

	// Record request start
	c.metrics.RecordRequestStart(req.Method, endpoint)

	// Check if caching is enabled for this request
	cacheEnabled := c.shouldCacheRequest(req)

	// Check cache first if enabled
	if cacheEnabled {
		cacheKey := c.cacheKeyFunc(req)
		if entry, found := c.cache.Get(cacheKey); found {
			// Record cache hit
			c.metrics.RecordCacheHit(req.Method, endpoint)

			// Record request end and metrics
			duration := time.Since(start)
			c.metrics.RecordRequestEnd(req.Method, endpoint)
			c.metrics.RecordRequest(req.Method, endpoint, entry.StatusCode, duration)

			// Return cached response
			return c.createResponseFromCache(entry), nil
		}
		// Record cache miss
		c.metrics.RecordCacheMiss(req.Method, endpoint)
	}

	resp, err := c.doWithRetry(req, 0)

	// Record request end
	c.metrics.RecordRequestEnd(req.Method, endpoint)

	// Record final metrics
	duration := time.Since(start)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	c.metrics.RecordRequest(req.Method, endpoint, statusCode, duration)

	// Cache successful responses if caching is enabled
	if cacheEnabled && err == nil && resp.StatusCode < 400 {
		cacheKey := c.cacheKeyFunc(req)
		entry := c.createCacheEntry(resp)
		ttl := c.getCacheTTLForRequest(req)
		c.cache.Set(cacheKey, entry, ttl)

		// Update cache size metric
		if inMemoryCache, ok := c.cache.(*InMemoryCache); ok {
			c.metrics.RecordCacheSize("default", len(inMemoryCache.store))
		}
	}

	return resp, err
}

// doWithRetry executes the request with retry logic
func (c *Client) doWithRetry(req *http.Request, attempt int) (*http.Response, error) {
	endpoint := getEndpointFromRequest(req)

	// Check rate limiter
	if c.rateLimiter != nil && !c.rateLimiter.Allow() {
		c.metrics.RecordError("RateLimit", req.Method, endpoint)
		return nil, &ClientError{Type: "RateLimit", Message: "rate limit exceeded"}
	}

	// Record rate limiter tokens if rate limiter is enabled
	if c.rateLimiter != nil {
		c.metrics.RecordRateLimiterTokens("default", c.rateLimiter.tokens)
	}

	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		c.metrics.RecordError("CircuitBreaker", req.Method, endpoint)
		return nil, &ClientError{Type: "CircuitBreaker", Message: "circuit breaker is open"}
	}

	// Record retry attempt if not the first attempt
	if attempt > 0 {
		c.metrics.RecordRetry(req.Method, endpoint, attempt)
	}

	// Execute middleware chain
	resp, err := c.executeMiddleware(req)

	// Handle circuit breaker
	if err != nil || (resp != nil && resp.StatusCode >= 500) {
		c.circuitBreaker.RecordFailure()
		c.metrics.RecordCircuitBreakerState("default", c.circuitBreaker.state)
		if err != nil {
			c.metrics.RecordError("Network", req.Method, endpoint)
		} else {
			c.metrics.RecordError("Server", req.Method, endpoint)
		}
	} else {
		c.circuitBreaker.RecordSuccess()
		c.metrics.RecordCircuitBreakerState("default", c.circuitBreaker.state)
	}

	// Check if retry is needed
	if attempt < c.maxRetries && c.retryCondition(resp, err) {
		backoff := c.calculateBackoff(attempt)
		time.Sleep(backoff)
		return c.doWithRetry(req, attempt+1)
	}

	return resp, err
}

// executeMiddleware executes the middleware chain
func (c *Client) executeMiddleware(req *http.Request) (*http.Response, error) {
	if len(c.middleware) == 0 {
		return c.httpClient.Do(req)
	}

	// Start with the base transport
	current := RoundTripperFunc(c.httpClient.Do)

	// Apply middleware in reverse order (last middleware wraps first)
	for i := len(c.middleware) - 1; i >= 0; i-- {
		middleware := c.middleware[i]
		next := current
		current = RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return middleware(r, next)
		})
	}

	return current.RoundTrip(req)
}

// calculateBackoff calculates the backoff duration for the given attempt
func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := time.Duration(float64(c.initialBackoff) * pow(c.backoffMultiplier, attempt))
	if backoff > c.maxBackoff {
		backoff = c.maxBackoff
	}
	// Add jitter
	if c.jitter > 0 {
		jitterAmount := time.Duration(float64(backoff) * c.jitter * rand.Float64())
		backoff += jitterAmount
	}
	return backoff
}

// pow calculates base^exponent for float64
func pow(base float64, exponent int) float64 {
	result := 1.0
	for i := 0; i < exponent; i++ {
		result *= base
	}
	return result
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.RecoveryTimeout == 0 {
		config.RecoveryTimeout = 60 * time.Second
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 2
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Allow checks if the request should be allowed through the circuit breaker
func (cb *CircuitBreaker) Allow() bool {
	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.lastFailure) >= cb.config.RecoveryTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordFailure records a failure in the circuit breaker
func (cb *CircuitBreaker) RecordFailure() {
	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateClosed && cb.failures >= cb.config.FailureThreshold {
		cb.state = StateOpen
	}
}

// RecordSuccess records a success in the circuit breaker
func (cb *CircuitBreaker) RecordSuccess() {
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
		}
	}
}

// RoundTripperFunc is a helper type for middleware
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// getEndpointFromRequest extracts a simplified endpoint from the request for metrics
func getEndpointFromRequest(req *http.Request) string {
	if req.URL == nil {
		return "unknown"
	}

	// Use host + path for endpoint identification
	endpoint := req.URL.Host
	if req.URL.Path != "" && req.URL.Path != "/" {
		endpoint += req.URL.Path
	} else {
		endpoint += "/"
	}

	return endpoint
}

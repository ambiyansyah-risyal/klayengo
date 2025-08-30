package klayengo

import (
	"context"
	"io"
	"math/rand"
	"net/http"
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

// DefaultRetryCondition is the default retry condition
func DefaultRetryCondition(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp.StatusCode >= 500
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

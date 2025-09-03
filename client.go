package klayengo

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"net/url"
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
	debug             *DebugConfig
	logger            Logger
	deduplication     *DeduplicationTracker
	dedupKeyFunc      DeduplicationKeyFunc
	dedupCondition    DeduplicationCondition
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
		debug:             DefaultDebugConfig(),
		logger:            nil, // No logging by default
		deduplication:     nil, // No deduplication by default
		dedupKeyFunc:      DefaultDeduplicationKeyFunc,
		dedupCondition:    DefaultDeduplicationCondition,
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

	// Initialize request context
	requestID := c.initializeRequest(req, endpoint)

	// Handle deduplication if enabled
	dedupResp, dedupHandled, dedupInfo, dedupErr := c.handleDeduplication(req, requestID, endpoint, start)
	if dedupHandled {
		return dedupResp, dedupErr
	}

	// Handle cache lookup if enabled
	if cachedResp, handled := c.handleCacheLookup(req, requestID, endpoint, start); handled {
		return cachedResp, nil
	}

	// Execute the request with retry logic
	resp, err := c.doWithRetry(req, 0, requestID, start)

	// Finalize request with metrics and cleanup
	c.finalizeRequest(req, resp, err, requestID, endpoint, start, dedupInfo)

	return resp, err
}

// initializeRequest sets up request tracing and initial logging
func (c *Client) initializeRequest(req *http.Request, endpoint string) string {
	// Generate request ID for tracing
	requestID := ""
	if c.debug != nil && c.debug.RequestIDGen != nil {
		requestID = c.debug.RequestIDGen()
	}

	// Debug logging
	if c.debug != nil && c.debug.Enabled && c.debug.LogRequests && c.logger != nil {
		c.logger.Debug("Starting request", "requestID", requestID, "method", req.Method, "url", req.URL.String(), "endpoint", endpoint)
	}

	// Record request start
	if c.metrics != nil {
		c.metrics.RecordRequestStart(req.Method, endpoint)
	}

	return requestID
}

// handleDeduplication manages request deduplication logic
func (c *Client) handleDeduplication(req *http.Request, requestID, endpoint string, start time.Time) (*http.Response, bool, deduplicationInfo, error) {
	dedupEnabled := c.deduplication != nil && c.dedupCondition(req)
	if !dedupEnabled {
		return nil, false, deduplicationInfo{}, nil
	}

	dedupKey := c.dedupKeyFunc(req)
	dedupEntry, isDedupOwner := c.deduplication.GetOrCreateEntry(dedupKey)

	// If this is not the owner, wait for the result
	if !isDedupOwner {
		resp, err := dedupEntry.Wait(req.Context())
		duration := time.Since(start)

		if c.metrics != nil {
			statusCode := 0
			if resp != nil {
				statusCode = resp.StatusCode
			}
			c.metrics.RecordRequest(req.Method, endpoint, statusCode, duration)
			c.metrics.RecordDeduplicationHit(req.Method, endpoint)
		}

		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.logger != nil {
			c.logger.Debug("Deduplication hit", "requestID", requestID, "dedupKey", dedupKey)
		}

		return resp, true, deduplicationInfo{key: dedupKey, isOwner: false}, err
	}

	// Debug logging for owner
	if c.debug != nil && c.debug.Enabled && c.logger != nil {
		c.logger.Debug("Deduplication miss - proceeding with request", "requestID", requestID, "dedupKey", dedupKey)
	}

	return nil, false, deduplicationInfo{key: dedupKey, isOwner: true}, nil
}

// handleCacheLookup checks cache for existing response
func (c *Client) handleCacheLookup(req *http.Request, requestID, endpoint string, start time.Time) (*http.Response, bool) {
	cacheEnabled := c.cache != nil && c.cacheCondition(req)
	if !cacheEnabled {
		return nil, false
	}

	cacheKey := c.cacheKeyFunc(req)
	if entry, found := c.cache.Get(cacheKey); found {
		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
			c.logger.Debug("Cache hit", "requestID", requestID, "cacheKey", cacheKey)
		}

		// Record cache hit
		if c.metrics != nil {
			c.metrics.RecordCacheHit(req.Method, endpoint)
		}

		// Record request end and metrics
		duration := time.Since(start)
		if c.metrics != nil {
			c.metrics.RecordRequestEnd(req.Method, endpoint)
			c.metrics.RecordRequest(req.Method, endpoint, entry.StatusCode, duration)
		}

		return c.createResponseFromCache(entry), true
	}

	// Record cache miss
	if c.metrics != nil {
		c.metrics.RecordCacheMiss(req.Method, endpoint)
	}

	// Debug logging
	if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
		c.logger.Debug("Cache miss", "requestID", requestID, "cacheKey", cacheKey)
	}

	return nil, false
}

// finalizeRequest handles final metrics recording and cleanup
func (c *Client) finalizeRequest(req *http.Request, resp *http.Response, err error, requestID, endpoint string, start time.Time, dedupInfo deduplicationInfo) {
	// Record request end
	if c.metrics != nil {
		c.metrics.RecordRequestEnd(req.Method, endpoint)
	}

	// Record final metrics
	duration := time.Since(start)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if c.metrics != nil {
		c.metrics.RecordRequest(req.Method, endpoint, statusCode, duration)
	}

	// Cache successful responses if caching is enabled
	c.handleCacheStorage(req, resp, err, requestID, endpoint)

	// Complete deduplication if enabled and this was the owner
	if dedupInfo.key != "" && dedupInfo.isOwner {
		c.deduplication.Complete(dedupInfo.key, resp, err)
	}
}

// deduplicationInfo holds deduplication state information
type deduplicationInfo struct {
	key     string
	isOwner bool
}

// handleCacheStorage stores successful responses in cache
func (c *Client) handleCacheStorage(req *http.Request, resp *http.Response, err error, requestID, endpoint string) {
	cacheEnabled := c.cache != nil && c.cacheCondition(req)
	if !cacheEnabled || err != nil || resp == nil || resp.StatusCode >= 400 {
		return
	}

	cacheKey := c.cacheKeyFunc(req)
	entry := c.createCacheEntry(resp)
	ttl := c.getCacheTTLForRequest(req)
	c.cache.Set(cacheKey, entry, ttl)

	// Update cache size metric
	if inMemoryCache, ok := c.cache.(*InMemoryCache); ok {
		totalSize := 0
		for _, shard := range inMemoryCache.shards {
			shard.mu.RLock()
			totalSize += len(shard.store)
			shard.mu.RUnlock()
		}
		if c.metrics != nil {
			c.metrics.RecordCacheSize("default", totalSize)
		}
	}

	// Debug logging
	if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
		c.logger.Debug("Response cached", "requestID", requestID, "endpoint", endpoint, "cacheKey", cacheKey, "ttl", ttl)
	}
}

// doWithRetry executes the request with retry logic
func (c *Client) doWithRetry(req *http.Request, attempt int, requestID string, startTime time.Time) (*http.Response, error) {
	endpoint := getEndpointFromRequest(req)

	// Check rate limiter
	if err := c.checkRateLimit(requestID, endpoint, req, attempt, startTime); err != nil {
		return nil, err
	}

	// Check circuit breaker
	if err := c.checkCircuitBreaker(requestID, endpoint, req, attempt, startTime); err != nil {
		return nil, err
	}

	// Log retry attempt if not the first attempt
	c.logRetryAttempt(attempt, requestID, endpoint)
	if attempt > 0 && c.metrics != nil {
		c.metrics.RecordRetry(req.Method, endpoint, attempt)
	}

	// Execute the request
	resp, err := c.executeRequest(req)

	// Handle circuit breaker state based on response
	c.handleCircuitBreakerState(resp, err, requestID, endpoint, req)

	// Check if retry is needed
	if c.shouldRetry(attempt, resp, err) {
		return c.performRetry(req, attempt, requestID, startTime, endpoint)
	}

	// Return final result or wrap error
	return c.finalizeAttempt(resp, err, req, attempt, startTime, requestID)
}

// checkRateLimit checks if the request should be rate limited
func (c *Client) checkRateLimit(requestID, endpoint string, req *http.Request, attempt int, startTime time.Time) error {
	if c.rateLimiter == nil || c.rateLimiter.Allow() {
		// Record rate limiter tokens if rate limiter is enabled
		if c.rateLimiter != nil && c.metrics != nil {
			c.metrics.RecordRateLimiterTokens("default", int(c.rateLimiter.tokens))
		}
		return nil
	}

	// Debug logging
	if c.debug != nil && c.debug.Enabled && c.debug.LogRateLimit && c.logger != nil {
		c.logger.Warn("Rate limit exceeded", "requestID", requestID, "endpoint", endpoint)
	}

	if c.metrics != nil {
		c.metrics.RecordError("RateLimit", req.Method, endpoint)
	}
	return c.createClientError(ErrorTypeRateLimit, "rate limit exceeded", nil, requestID, req, attempt, time.Since(startTime))
}

// checkCircuitBreaker checks if the circuit breaker allows the request
func (c *Client) checkCircuitBreaker(requestID, endpoint string, req *http.Request, attempt int, startTime time.Time) error {
	if c.circuitBreaker.Allow() {
		return nil
	}

	// Debug logging
	if c.debug != nil && c.debug.Enabled && c.debug.LogCircuit && c.logger != nil {
		c.logger.Warn("Circuit breaker open", "requestID", requestID, "endpoint", endpoint, "state", c.circuitBreaker.state)
	}

	if c.metrics != nil {
		c.metrics.RecordError("CircuitBreaker", req.Method, endpoint)
	}
	return c.createClientError(ErrorTypeCircuitOpen, "circuit breaker is open", nil, requestID, req, attempt, time.Since(startTime))
}

// logRetryAttempt logs retry attempt information
func (c *Client) logRetryAttempt(attempt int, requestID, endpoint string) {
	if attempt > 0 {
		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.debug.LogRetries && c.logger != nil {
			c.logger.Info("Retry attempt", "requestID", requestID, "attempt", attempt, "maxRetries", c.maxRetries, "endpoint", endpoint)
		}
	}
}

// executeRequest executes the HTTP request through middleware
func (c *Client) executeRequest(req *http.Request) (*http.Response, error) {
	return c.executeMiddleware(req)
}

// handleCircuitBreakerState updates circuit breaker state based on response
func (c *Client) handleCircuitBreakerState(resp *http.Response, err error, requestID, endpoint string, req *http.Request) {
	if c.isFailureResponse(resp, err) {
		c.recordCircuitBreakerFailure(resp, err, requestID, endpoint, req)
	} else {
		c.recordCircuitBreakerSuccess()
	}
}

// isFailureResponse determines if the response indicates a failure
func (c *Client) isFailureResponse(resp *http.Response, err error) bool {
	return err != nil || (resp != nil && resp.StatusCode >= 500)
}

// recordCircuitBreakerFailure handles circuit breaker failure recording
func (c *Client) recordCircuitBreakerFailure(resp *http.Response, err error, requestID, endpoint string, req *http.Request) {
	c.circuitBreaker.RecordFailure()
	if c.metrics != nil {
		c.metrics.RecordCircuitBreakerState("default", CircuitState(c.circuitBreaker.state))
	}

	// Debug logging
	if c.debug != nil && c.debug.Enabled && c.debug.LogCircuit && c.logger != nil {
		if err != nil {
			c.logger.Warn("Circuit breaker failure recorded", "requestID", requestID, "error", err.Error())
		} else {
			c.logger.Warn("Circuit breaker failure recorded", "requestID", requestID, "statusCode", resp.StatusCode)
		}
	}

	// Record error metrics
	c.recordErrorMetrics(err, resp, req.Method, endpoint)
}

// recordCircuitBreakerSuccess handles circuit breaker success recording
func (c *Client) recordCircuitBreakerSuccess() {
	c.circuitBreaker.RecordSuccess()
	if c.metrics != nil {
		c.metrics.RecordCircuitBreakerState("default", CircuitState(c.circuitBreaker.state))
	}
}

// recordErrorMetrics records appropriate error metrics based on error type
func (c *Client) recordErrorMetrics(err error, resp *http.Response, method, endpoint string) {
	if c.metrics == nil {
		return
	}

	if err != nil {
		c.metrics.RecordError("Network", method, endpoint)
	} else if resp != nil {
		c.metrics.RecordError("Server", method, endpoint)
	}
}

// shouldRetry determines if the request should be retried
func (c *Client) shouldRetry(attempt int, resp *http.Response, err error) bool {
	return attempt < c.maxRetries && c.retryCondition(resp, err)
}

// performRetry handles the retry logic with backoff
func (c *Client) performRetry(req *http.Request, attempt int, requestID string, startTime time.Time, endpoint string) (*http.Response, error) {
	backoff := c.calculateBackoff(attempt)

	// Debug logging
	if c.debug != nil && c.debug.Enabled && c.debug.LogRetries && c.logger != nil {
		c.logger.Info("Scheduling retry", "requestID", requestID, "attempt", attempt+1, "backoff", backoff, "endpoint", endpoint)
	}

	time.Sleep(backoff)
	return c.doWithRetry(req, attempt+1, requestID, startTime)
}

// finalizeAttempt returns the final result or wraps errors
func (c *Client) finalizeAttempt(resp *http.Response, err error, req *http.Request, attempt int, startTime time.Time, requestID string) (*http.Response, error) {
	if err != nil {
		return nil, c.createClientError(ErrorTypeNetwork, "network request failed", err, requestID, req, attempt, time.Since(startTime))
	}
	return resp, nil
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

// createClientError creates a ClientError with enhanced context
func (c *Client) createClientError(errorType, message string, cause error, requestID string, req *http.Request, attempt int, duration time.Duration) *ClientError {
	endpoint := getEndpointFromRequest(req)
	statusCode := 0

	// Try to get status code from cause if it's an HTTP error
	if cause != nil {
		if httpErr, ok := cause.(*url.Error); ok && httpErr.Err != nil {
			// Could be a more specific error type - handled in error creation below
			_ = httpErr // Avoid unused variable warning
		}
	}

	return &ClientError{
		Type:       errorType,
		Message:    message,
		Cause:      cause,
		RequestID:  requestID,
		Method:     req.Method,
		URL:        req.URL.String(),
		Attempt:    attempt,
		MaxRetries: c.maxRetries,
		Timestamp:  time.Now(),
		Duration:   duration,
		StatusCode: statusCode,
		Endpoint:   endpoint,
	}
}

// getEndpointFromRequest extracts a simplified endpoint from the request for metrics
func getEndpointFromRequest(req *http.Request) string {
	if req.URL == nil {
		return "unknown"
	}

	host := req.URL.Host
	path := req.URL.Path

	// Pre-allocate buffer for efficiency
	var buf []byte
	buf = append(buf, host...)

	if path != "" && path != "/" {
		buf = append(buf, path...)
	} else {
		buf = append(buf, '/')
	}

	return string(buf)
}

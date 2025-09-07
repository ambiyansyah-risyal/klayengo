package klayengo

import (
	"context"
	"fmt"
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
	validationError   error // Stores validation error if configuration is invalid
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

	// Validate configuration after applying all options
	if err := client.ValidateConfiguration(); err != nil {
		// Return a client with validation error for backward compatibility
		// The error will be accessible via a method or field if needed
		client.validationError = err
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

	// Check if deduplication is enabled for this request
	dedupEnabled := c.deduplication != nil && c.dedupCondition(req)

	var dedupEntry *DeduplicationEntry
	var isDedupOwner bool
	if dedupEnabled {
		dedupKey := c.dedupKeyFunc(req)
		dedupEntry, isDedupOwner = c.deduplication.GetOrCreateEntry(dedupKey)

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

			return resp, err
		}

		// Debug logging for owner
		if c.debug != nil && c.debug.Enabled && c.logger != nil {
			c.logger.Debug("Deduplication miss - proceeding with request", "requestID", requestID, "dedupKey", dedupKey)
		}
	}

	// Check if caching is enabled for this request
	cacheEnabled := c.cache != nil && c.cacheCondition(req)

	// Check cache first if enabled
	if cacheEnabled {
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

			// Return cached response
			return c.createResponseFromCache(entry), nil
		}
		// Record cache miss
		if c.metrics != nil {
			c.metrics.RecordCacheMiss(req.Method, endpoint)
		}

		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
			cacheKey := c.cacheKeyFunc(req)
			c.logger.Debug("Cache miss", "requestID", requestID, "cacheKey", cacheKey)
		}
	}

	resp, err := c.doWithRetry(req, 0, requestID, start)

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
	if cacheEnabled && err == nil && resp.StatusCode < 400 {
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
			c.logger.Debug("Response cached", "requestID", requestID, "cacheKey", cacheKey, "ttl", ttl)
		}
	}

	// Complete deduplication if enabled and this was the owner
	if dedupEnabled && isDedupOwner && dedupEntry != nil {
		dedupKey := c.dedupKeyFunc(req)
		c.deduplication.Complete(dedupKey, resp, err)
	}

	return resp, err
}

// doWithRetry executes the request with retry logic
func (c *Client) doWithRetry(req *http.Request, attempt int, requestID string, startTime time.Time) (*http.Response, error) {
	endpoint := getEndpointFromRequest(req)

	// Check rate limiter
	if c.rateLimiter != nil && !c.rateLimiter.Allow() {
		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.debug.LogRateLimit && c.logger != nil {
			c.logger.Warn("Rate limit exceeded", "requestID", requestID, "endpoint", endpoint)
		}

		if c.metrics != nil {
			c.metrics.RecordError("RateLimit", req.Method, endpoint)
		}
		return nil, c.createClientError(ErrorTypeRateLimit, "rate limit exceeded", nil, requestID, req, attempt, time.Since(startTime))
	}

	// Record rate limiter tokens if rate limiter is enabled
	if c.rateLimiter != nil && c.metrics != nil {
		c.metrics.RecordRateLimiterTokens("default", int(c.rateLimiter.tokens))
	}

	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.debug.LogCircuit && c.logger != nil {
			c.logger.Warn("Circuit breaker open", "requestID", requestID, "endpoint", endpoint, "state", c.circuitBreaker.state)
		}

		if c.metrics != nil {
			c.metrics.RecordError("CircuitBreaker", req.Method, endpoint)
		}
		return nil, c.createClientError(ErrorTypeCircuitOpen, "circuit breaker is open", nil, requestID, req, attempt, time.Since(startTime))
	}

	// Record retry attempt if not the first attempt
	if attempt > 0 {
		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.debug.LogRetries && c.logger != nil {
			c.logger.Info("Retry attempt", "requestID", requestID, "attempt", attempt, "maxRetries", c.maxRetries, "endpoint", endpoint)
		}

		if c.metrics != nil {
			c.metrics.RecordRetry(req.Method, endpoint, attempt)
		}
	}

	// Execute middleware chain
	resp, err := c.executeMiddleware(req)

	// Handle circuit breaker
	if err != nil || (resp != nil && resp.StatusCode >= 500) {
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

		if err != nil {
			if c.metrics != nil {
				c.metrics.RecordError("Network", req.Method, endpoint)
			}
		} else {
			if c.metrics != nil {
				c.metrics.RecordError("Server", req.Method, endpoint)
			}
		}
	} else {
		c.circuitBreaker.RecordSuccess()
		if c.metrics != nil {
			c.metrics.RecordCircuitBreakerState("default", CircuitState(c.circuitBreaker.state))
		}
	}

	// Check if retry is needed
	if attempt < c.maxRetries && c.retryCondition(resp, err) {
		backoff := c.calculateBackoff(attempt)

		// Debug logging
		if c.debug != nil && c.debug.Enabled && c.debug.LogRetries && c.logger != nil {
			c.logger.Info("Scheduling retry", "requestID", requestID, "attempt", attempt+1, "backoff", backoff, "endpoint", endpoint)
		}

		time.Sleep(backoff)
		return c.doWithRetry(req, attempt+1, requestID, startTime)
	}

	// If there's an error, wrap it with enhanced context
	if err != nil {
		return nil, c.createClientError(ErrorTypeNetwork, "network request failed", err, requestID, req, attempt, time.Since(startTime))
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
	// Add jitter (clamp to valid range)
	jitter := c.jitter
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	if jitter > 0 {
		jitterAmount := time.Duration(float64(backoff) * jitter * rand.Float64())
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

// IsValid returns true if the client configuration is valid
func (c *Client) IsValid() bool {
	return c.validationError == nil
}

// ValidationError returns the validation error if the configuration is invalid, nil otherwise
func (c *Client) ValidationError() error {
	return c.validationError
}

// ValidateConfigurationStrict validates the client configuration and panics if invalid
// This is useful for development and testing where invalid configurations should not be allowed
func (c *Client) ValidateConfigurationStrict() {
	if err := c.ValidateConfiguration(); err != nil {
		panic(fmt.Sprintf("invalid client configuration: %v", err))
	}
}

// MustValidateConfiguration validates the client configuration and returns an error if invalid
// This is an alias for ValidateConfiguration for clarity
func (c *Client) MustValidateConfiguration() error {
	return c.ValidateConfiguration()
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

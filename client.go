package klayengo

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client is a resilient HTTP client that layers retries, circuit breaking,
// rate limiting, caching, de‑duplication, middleware and metrics around
// the standard net/http Client. It is safe for concurrent use.
type Client struct {
	httpClient        *http.Client
	maxRetries        int
	initialBackoff    time.Duration
	maxBackoff        time.Duration
	backoffMultiplier float64
	jitter            float64
	backoffStrategy   BackoffStrategy
	timeout           time.Duration
	retryCondition    RetryCondition
	retryPolicy       RetryPolicy
	retryBudget       *RetryBudget
	circuitBreaker    *CircuitBreaker
	middleware        []Middleware
	rateLimiter       *RateLimiter
	limiterRegistry   *RateLimiterRegistry
	limiterKeyFunc    KeyFunc
	cache             Cache
	cacheTTL          time.Duration
	cacheKeyFunc      func(*http.Request) string
	cacheCondition    CacheCondition
	cacheProvider     CacheProvider
	cacheMode         CacheMode
	singleFlight      map[string]*singleFlightEntry
	singleFlightMu    sync.RWMutex
	metrics           *MetricsCollector
	debug             *DebugConfig
	logger            Logger
	deduplication     *DeduplicationTracker
	dedupKeyFunc      DeduplicationKeyFunc
	dedupCondition    DeduplicationCondition
	validationError   error
}

// New constructs a Client using the provided functional options. A best effort
// validation is performed; call IsValid / ValidationError for errors.
func New(options ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries:        3,
		initialBackoff:    100 * time.Millisecond,
		maxBackoff:        10 * time.Second,
		backoffMultiplier: 2.0,
		jitter:            0.1,
		backoffStrategy:   ExponentialJitter, // Default to current behavior
		timeout:           30 * time.Second,
		retryCondition:    DefaultRetryCondition,
		retryPolicy:       nil, // Will use legacy retry logic if nil
		retryBudget:       nil,
		circuitBreaker:    NewCircuitBreaker(CircuitBreakerConfig{}),
		middleware:        []Middleware{},
		rateLimiter:       nil,
		limiterRegistry:   nil,
		limiterKeyFunc:    nil,
		cache:             nil,
		cacheTTL:          5 * time.Minute,
		cacheKeyFunc:      DefaultCacheKeyFunc,
		cacheCondition:    DefaultCacheCondition,
		cacheProvider:     nil,
		cacheMode:         TTLOnly,
		singleFlight:      make(map[string]*singleFlightEntry),
		metrics:           nil,
		debug:             DefaultDebugConfig(),
		logger:            nil,
		deduplication:     nil,
		dedupKeyFunc:      DefaultDeduplicationKeyFunc,
		dedupCondition:    DefaultDeduplicationCondition,
	}

	for _, option := range options {
		option(client)
	}

	if err := client.ValidateConfiguration(); err != nil {
		client.validationError = err
	}

	return client
}

// Get performs an HTTP GET with context.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post performs an HTTP POST with the given content type.
func (c *Client) Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// Do executes a prepared *http.Request applying all reliability features.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	start := time.Now()
	endpoint := getEndpointFromRequest(req)

	var requestID string
	if c.debug != nil && c.debug.Enabled && c.debug.RequestIDGen != nil {
		requestID = c.debug.RequestIDGen()
	}

	if c.debug != nil && c.debug.Enabled && c.debug.LogRequests && c.logger != nil {
		c.logger.Debug("Starting request", "requestID", requestID, "method", req.Method, "url", req.URL.String(), "endpoint", endpoint)
	}

	if c.metrics != nil {
		c.metrics.RecordRequestStart(req.Method, endpoint)
	}

	dedupEnabled := c.deduplication != nil && c.dedupCondition(req)

	var dedupEntry *DeduplicationEntry
	var isDedupOwner bool
	if dedupEnabled {
		dedupKey := c.dedupKeyFunc(req)
		dedupEntry, isDedupOwner = c.deduplication.GetOrCreateEntry(dedupKey)

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

			if c.debug != nil && c.debug.Enabled && c.logger != nil {
				c.logger.Debug("Deduplication hit", "requestID", requestID, "dedupKey", dedupKey)
			}

			return resp, err
		}

		if c.debug != nil && c.debug.Enabled && c.logger != nil {
			c.logger.Debug("Deduplication miss - proceeding with request", "requestID", requestID, "dedupKey", dedupKey)
		}
	}

	// Handle cache with new modes
	cacheEnabled := (c.cache != nil || c.cacheProvider != nil) && c.cacheCondition(req)

	if cacheEnabled {
		cacheKey := c.cacheKeyFunc(req)

		// Try new cache provider first if available
		if c.cacheProvider != nil && (c.cacheMode == HTTPSemantics || c.cacheMode == SWR) {
			if resp, found := c.cacheProvider.Get(req.Context(), cacheKey); found {
				if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
					c.logger.Debug("Cache provider hit", "requestID", requestID, "cacheKey", cacheKey, "mode", c.cacheMode)
				}

				if c.metrics != nil {
					c.metrics.RecordCacheHit(req.Method, endpoint)
				}

				duration := time.Since(start)
				if c.metrics != nil {
					c.metrics.RecordRequestEnd(req.Method, endpoint)
					c.metrics.RecordRequest(req.Method, endpoint, resp.StatusCode, duration)
				}

				return resp, nil
			}
		} else if c.cache != nil {
			// Fall back to legacy cache for TTLOnly mode
			if entry, found := c.cache.Get(cacheKey); found {
				// For HTTP semantics mode, handle conditional requests even with legacy cache
				if c.cacheMode == HTTPSemantics {
					return c.handleHTTPSemanticsCacheHit(req, entry, cacheKey)
				}

				if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
					c.logger.Debug("Cache hit", "requestID", requestID, "cacheKey", cacheKey)
				}

				if c.metrics != nil {
					c.metrics.RecordCacheHit(req.Method, endpoint)
				}

				duration := time.Since(start)
				if c.metrics != nil {
					c.metrics.RecordRequestEnd(req.Method, endpoint)
					c.metrics.RecordRequest(req.Method, endpoint, entry.StatusCode, duration)
				}

				return c.createResponseFromCache(entry), nil
			}
		}

		if c.metrics != nil {
			c.metrics.RecordCacheMiss(req.Method, endpoint)
		}

		if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
			c.logger.Debug("Cache miss", "requestID", requestID, "cacheKey", cacheKey)
		}
	}

	// Use single-flight protection for cache misses to prevent stampedes
	var resp *http.Response
	var err error
	var skipCaching bool

	if cacheEnabled && (c.cacheMode == HTTPSemantics || c.cacheMode == SWR) {
		cacheKey := c.cacheKeyFunc(req)
		resp, err = c.singleFlightDo(cacheKey, func() (*http.Response, error) {
			httpResp, httpErr := c.doWithRetry(req, 0, requestID, start)

			// Cache the response here in the single-flight function to avoid races
			if httpErr == nil && httpResp.StatusCode < 400 && c.cacheProvider != nil {
				ttl := c.getCacheTTLForRequest(req)
				c.cacheProvider.Set(req.Context(), cacheKey, httpResp, ttl)

				if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
					c.logger.Debug("Response cached via provider", "requestID", requestID, "cacheKey", cacheKey, "mode", c.cacheMode)
				}
			}

			return httpResp, httpErr
		})
		skipCaching = true // Skip caching below since we already did it in single-flight
	} else {
		resp, err = c.doWithRetry(req, 0, requestID, start)
	}

	if c.metrics != nil {
		c.metrics.RecordRequestEnd(req.Method, endpoint)
	}

	duration := time.Since(start)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if c.metrics != nil {
		c.metrics.RecordRequest(req.Method, endpoint, statusCode, duration)
	}

	if cacheEnabled && err == nil && resp.StatusCode < 400 && !skipCaching {
		cacheKey := c.cacheKeyFunc(req)

		// Use new cache provider if available
		if c.cacheProvider != nil {
			ttl := c.getCacheTTLForRequest(req)
			c.cacheProvider.Set(req.Context(), cacheKey, resp, ttl)

			if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
				c.logger.Debug("Response cached via provider", "requestID", requestID, "cacheKey", cacheKey, "mode", c.cacheMode)
			}
		} else if c.cache != nil {
			// Fall back to legacy cache
			entry := c.createCacheEntry(resp)
			ttl := c.getCacheTTLForRequest(req)
			c.cache.Set(cacheKey, entry, ttl)

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

			if c.debug != nil && c.debug.Enabled && c.debug.LogCache && c.logger != nil {
				c.logger.Debug("Response cached", "requestID", requestID, "cacheKey", cacheKey, "ttl", ttl)
			}
		}
	}

	if dedupEnabled && isDedupOwner && dedupEntry != nil {
		dedupKey := c.dedupKeyFunc(req)
		c.deduplication.Complete(dedupKey, resp, err)
	}

	return resp, err
}

func (c *Client) doWithRetry(req *http.Request, attempt int, requestID string, startTime time.Time) (*http.Response, error) {
	endpoint := getEndpointFromRequest(req)

	var allowed bool
	var limiterKey string
	if c.limiterRegistry != nil {
		allowed, limiterKey = c.limiterRegistry.Allow(req)
	} else if c.rateLimiter != nil {
		allowed = c.rateLimiter.Allow()
		limiterKey = "default"
	} else {
		allowed = true
		limiterKey = "none"
	}

	if !allowed {
		if c.debug != nil && c.debug.Enabled && c.debug.LogRateLimit && c.logger != nil {
			c.logger.Warn("Rate limit exceeded", "requestID", requestID, "endpoint", endpoint, "limiterKey", limiterKey)
		}

		if c.metrics != nil {
			c.metrics.RecordError("RateLimit", req.Method, endpoint)
			c.metrics.RecordRateLimiterExceeded(limiterKey)
		}
		return nil, c.createClientError(ErrorTypeRateLimit, "rate limit exceeded", nil, requestID, req, attempt, time.Since(startTime))
	}

	if c.limiterRegistry != nil && c.metrics != nil {
		// Record metrics for all active limiters in the registry
		c.limiterRegistry.mutex.RLock()
		for key, limiter := range c.limiterRegistry.limiters {
			if rl, ok := limiter.(*RateLimiter); ok {
				c.metrics.RecordRateLimiterTokens(key, int(rl.tokens))
			}
		}
		c.limiterRegistry.mutex.RUnlock()
		// Also record fallback limiter if it exists
		if c.rateLimiter != nil {
			c.metrics.RecordRateLimiterTokens("default", int(c.rateLimiter.tokens))
		}
	} else if c.rateLimiter != nil && c.metrics != nil {
		c.metrics.RecordRateLimiterTokens("default", int(c.rateLimiter.tokens))
	}

	if !c.circuitBreaker.Allow() {
		if c.debug != nil && c.debug.Enabled && c.debug.LogCircuit && c.logger != nil {
			c.logger.Warn("Circuit breaker open", "requestID", requestID, "endpoint", endpoint, "state", c.circuitBreaker.state)
		}

		if c.metrics != nil {
			c.metrics.RecordError("CircuitBreaker", req.Method, endpoint)
		}
		return nil, c.createClientError(ErrorTypeCircuitOpen, "circuit breaker is open", nil, requestID, req, attempt, time.Since(startTime))
	}

	if attempt > 0 {
		if c.debug != nil && c.debug.Enabled && c.debug.LogRetries && c.logger != nil {
			c.logger.Info("Retry attempt", "requestID", requestID, "attempt", attempt, "maxRetries", c.maxRetries, "endpoint", endpoint)
		}

		if c.metrics != nil {
			c.metrics.RecordRetry(req.Method, endpoint, attempt)
		}
	}

	resp, err := c.executeMiddleware(req)

	if err != nil || (resp != nil && resp.StatusCode >= 500) {
		c.circuitBreaker.RecordFailure()
		if c.metrics != nil {
			c.metrics.RecordCircuitBreakerState("default", CircuitState(c.circuitBreaker.state))
		}

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

	// Check retry eligibility using either new RetryPolicy or legacy condition
	var shouldRetry bool
	var delay time.Duration

	if c.retryPolicy != nil {
		delay, shouldRetry = c.retryPolicy.ShouldRetry(resp, err, attempt)
	} else {
		shouldRetry = attempt < c.maxRetries && c.retryCondition(resp, err)
		if shouldRetry {
			delay = c.calculateBackoff(attempt)
		}
	}

	if shouldRetry {
		// Check retry budget if configured
		if c.retryBudget != nil && !c.retryBudget.Allow() {
			if c.metrics != nil {
				c.metrics.RecordRetryBudgetExceeded(endpoint)
			}
			if c.debug != nil && c.debug.Enabled && c.debug.LogRetries && c.logger != nil {
				c.logger.Warn("Retry budget exceeded", "requestID", requestID, "endpoint", endpoint)
			}
			return nil, c.createClientError(ErrorTypeRetryBudgetExceeded, "retry budget exceeded", nil, requestID, req, attempt, time.Since(startTime))
		}

		if c.debug != nil && c.debug.Enabled && c.debug.LogRetries && c.logger != nil {
			c.logger.Info("Scheduling retry", "requestID", requestID, "attempt", attempt+1, "backoff", delay, "endpoint", endpoint)
		}

		time.Sleep(delay)
		return c.doWithRetry(req, attempt+1, requestID, startTime)
	}

	if err != nil {
		return nil, c.createClientError(ErrorTypeNetwork, "network request failed", err, requestID, req, attempt, time.Since(startTime))
	}

	return resp, err
}

func (c *Client) executeMiddleware(req *http.Request) (*http.Response, error) {
	if len(c.middleware) == 0 {
		return c.httpClient.Do(req)
	}

	current := RoundTripperFunc(c.httpClient.Do)

	for i := len(c.middleware) - 1; i >= 0; i-- {
		middleware := c.middleware[i]
		next := current
		current = RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return middleware(r, next)
		})
	}

	return current.RoundTrip(req)
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	switch c.backoffStrategy {
	case ExponentialJitter:
		return c.calculateExponentialBackoff(attempt)
	case DecorrelatedJitter:
		return c.calculateDecorrelatedBackoff(attempt)
	default:
		// Fallback to exponential jitter for unknown strategies
		return c.calculateExponentialBackoff(attempt)
	}
}

func (c *Client) calculateExponentialBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Prevent overflow by limiting attempt
	if attempt > 30 {
		attempt = 30
	}

	backoff := time.Duration(float64(c.initialBackoff) * pow(c.backoffMultiplier, attempt))
	if backoff < 0 || backoff > c.maxBackoff {
		backoff = c.maxBackoff
	}

	jitter := c.jitter
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	if jitter > 0 {
		jitterAmount := time.Duration(float64(backoff) * jitter * rand.Float64())
		if backoff+jitterAmount > c.maxBackoff {
			backoff = c.maxBackoff
		} else {
			backoff += jitterAmount
		}
	}
	return backoff
}

func (c *Client) calculateDecorrelatedBackoff(attempt int) time.Duration {
	// Decorrelated jitter as per AWS: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
	// Formula: random_between(base, min(cap, base * 3))
	// For subsequent attempts: random_between(base, min(cap, previous_delay * 3))

	if attempt <= 0 {
		return c.initialBackoff
	}

	// Prevent overflow by limiting attempt
	if attempt > 10 {
		attempt = 10
	}

	// For decorrelated jitter, we need to track the previous delay
	// Since we don't have state, we'll use a simplified version:
	// random_between(base, min(cap, base * 3^attempt))

	base := float64(c.initialBackoff)
	factor := pow(3.0, attempt) // Use 3x multiplier for decorrelated jitter
	upper := base * factor

	// Prevent overflow and respect maxBackoff
	maxBackoffFloat := float64(c.maxBackoff)
	if upper > maxBackoffFloat || upper < 0 {
		upper = maxBackoffFloat
	}

	// Ensure upper is at least base
	if upper < base {
		upper = base
	}

	// Generate random delay between base and upper
	delay := base + rand.Float64()*(upper-base)

	result := time.Duration(delay)
	if result < 0 || result > c.maxBackoff {
		result = c.maxBackoff
	}

	return result
}

func pow(base float64, exponent int) float64 {
	result := 1.0
	for i := 0; i < exponent; i++ {
		result *= base
	}
	return result
}

func DefaultRetryCondition(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp.StatusCode >= 500
}

func (c *Client) createClientError(errorType, message string, cause error, requestID string, req *http.Request, attempt int, duration time.Duration) *ClientError {
	endpoint := getEndpointFromRequest(req)

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
		StatusCode: 0,
		Endpoint:   endpoint,
	}
}

// IsValid reports whether configuration validation passed at construction.
func (c *Client) IsValid() bool {
	return c.validationError == nil
}

// ValidationError returns the configuration validation error, if any.
func (c *Client) ValidationError() error {
	return c.validationError
}

// ValidateConfigurationStrict panics if configuration is invalid.
func (c *Client) ValidateConfigurationStrict() {
	if err := c.ValidateConfiguration(); err != nil {
		panic(fmt.Sprintf("invalid client configuration: %v", err))
	}
}

// MustValidateConfiguration re-runs validation returning an error (no panic).
func (c *Client) MustValidateConfiguration() error {
	return c.ValidateConfiguration()
}

func getEndpointFromRequest(req *http.Request) string {
	if req.URL == nil {
		return "unknown"
	}

	host := req.URL.Host
	path := req.URL.Path

	var builder strings.Builder
	builder.WriteString(host)

	if path != "" && path != "/" {
		builder.WriteString(path)
	} else {
		builder.WriteByte('/')
	}

	return builder.String()
}

// singleFlightDo executes a request with single-flight protection to prevent stampede.
func (c *Client) singleFlightDo(key string, fn func() (*http.Response, error)) (*http.Response, error) {
	c.singleFlightMu.Lock()
	if entry, exists := c.singleFlight[key]; exists {
		c.singleFlightMu.Unlock()
		entry.wg.Wait()
		return entry.resp, entry.err
	}

	// Create new entry
	entry := &singleFlightEntry{}
	entry.wg.Add(1)
	c.singleFlight[key] = entry
	c.singleFlightMu.Unlock()

	// Execute the function
	resp, err := fn()

	// Complete the entry
	entry.resp = resp
	entry.err = err
	entry.done = true
	entry.wg.Done()

	// Clean up after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		c.singleFlightMu.Lock()
		delete(c.singleFlight, key)
		c.singleFlightMu.Unlock()
	}()

	return resp, err
}

// performConditionalRequest performs a conditional request using If-None-Match/If-Modified-Since.
func (c *Client) performConditionalRequest(req *http.Request, entry *CacheEntry) (*http.Response, error) {
	// Clone request to avoid modifying original
	conditionalReq := req.Clone(req.Context())
	addConditionalHeaders(conditionalReq, entry)

	// Perform the request
	return c.doWithRetry(conditionalReq, 0, "", time.Now())
}

// handleHTTPSemanticsCacheHit handles cache hits with HTTP semantics.
func (c *Client) handleHTTPSemanticsCacheHit(req *http.Request, entry *CacheEntry, cacheKey string) (*http.Response, error) {
	cacheControl := parseCacheControl(entry.Header.Get("Cache-Control"))

	// Check if revalidation is needed
	if shouldRevalidate(entry, cacheControl) {
		// In SWR mode, serve stale immediately and revalidate in background
		if c.cacheMode == SWR && entry.IsStale {
			// Start background revalidation
			go func() {
				_, _ = c.singleFlightDo("revalidate:"+cacheKey, func() (*http.Response, error) {
					resp, err := c.performConditionalRequest(req, entry)
					if err == nil && c.cacheProvider != nil {
						// Update cache with fresh response
						c.cacheProvider.Set(req.Context(), cacheKey, resp, 0)
					}
					return resp, err
				})
			}()

			// Serve stale response immediately
			return c.createResponseFromCache(entry), nil
		}

		// Perform conditional request
		resp, err := c.performConditionalRequest(req, entry)
		if err != nil {
			return nil, err
		}

		// If not modified, serve cached version
		if isNotModified(resp) {
			return c.createResponseFromCache(entry), nil
		}

		// Cache the new response
		if c.cacheProvider != nil {
			c.cacheProvider.Set(req.Context(), cacheKey, resp, 0)
		}

		return resp, nil
	}

	// Entry is fresh, serve from cache
	return c.createResponseFromCache(entry), nil
}

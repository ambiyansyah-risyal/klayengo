package klayengo

import (
	"fmt"
	"net/http"
	"time"
)

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		c.maxRetries = n
	}
}

// WithInitialBackoff sets the initial backoff duration
func WithInitialBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.initialBackoff = d
	}
}

// WithMaxBackoff sets the maximum backoff duration
func WithMaxBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.maxBackoff = d
	}
}

// WithBackoffMultiplier sets the backoff multiplier
func WithBackoffMultiplier(f float64) Option {
	return func(c *Client) {
		c.backoffMultiplier = f
	}
}

// WithJitter sets the jitter factor for backoff (0.0 to 1.0)
func WithJitter(f float64) Option {
	return func(c *Client) {
		if f < 0 {
			f = 0
		}
		if f > 1 {
			f = 1
		}
		c.jitter = f
	}
}

// WithRateLimiter sets the rate limiter
func WithRateLimiter(maxTokens int, refillRate time.Duration) Option {
	return func(c *Client) {
		c.rateLimiter = NewRateLimiter(maxTokens, refillRate)
	}
}

// WithCache enables caching with the default in-memory cache
func WithCache(ttl time.Duration) Option {
	return func(c *Client) {
		c.cache = NewInMemoryCache()
		c.cacheTTL = ttl
	}
}

// WithCustomCache sets a custom cache implementation
func WithCustomCache(cache Cache, ttl time.Duration) Option {
	return func(c *Client) {
		c.cache = cache
		c.cacheTTL = ttl
	}
}

// WithCacheKeyFunc sets a custom cache key function
func WithCacheKeyFunc(fn func(*http.Request) string) Option {
	return func(c *Client) {
		c.cacheKeyFunc = fn
	}
}

// WithCacheCondition sets a custom cache condition function
func WithCacheCondition(fn CacheCondition) Option {
	return func(c *Client) {
		c.cacheCondition = fn
	}
}

// WithTimeout sets the request timeout
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
		if c.httpClient != nil {
			c.httpClient.Timeout = d
		}
	}
}

// WithRetryCondition sets a custom retry condition
func WithRetryCondition(fn RetryCondition) Option {
	return func(c *Client) {
		c.retryCondition = fn
	}
}

// WithCircuitBreaker sets the circuit breaker configuration
func WithCircuitBreaker(config CircuitBreakerConfig) Option {
	return func(c *Client) {
		c.circuitBreaker = NewCircuitBreaker(config)
	}
}

// WithMiddleware adds middleware to the client
func WithMiddleware(middleware ...Middleware) Option {
	return func(c *Client) {
		c.middleware = append(c.middleware, middleware...)
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
		// Update timeout if it was set
		if c.timeout != 0 {
			c.httpClient.Timeout = c.timeout
		}
	}
}

// WithMetrics enables Prometheus metrics collection
func WithMetrics() Option {
	return func(c *Client) {
		c.metrics = NewMetricsCollector()
	}
}

// WithMetricsCollector sets a custom metrics collector
func WithMetricsCollector(collector *MetricsCollector) Option {
	return func(c *Client) {
		c.metrics = collector
	}
}

// WithDebug enables debug logging with default configuration
func WithDebug() Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.Enabled = true
	}
}

// WithDebugConfig sets custom debug configuration
func WithDebugConfig(config *DebugConfig) Option {
	return func(c *Client) {
		c.debug = config
	}
}

// WithLogger sets a custom logger for debug output
func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithSimpleLogger enables debug logging with a simple console logger
func WithSimpleLogger() Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.Enabled = true
		c.logger = NewSimpleLogger()
	}
}

// WithRequestIDGenerator sets a custom function for generating request IDs
func WithRequestIDGenerator(gen func() string) Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.RequestIDGen = gen
	}
}

// WithDeduplication enables request deduplication
func WithDeduplication() Option {
	return func(c *Client) {
		c.deduplication = NewDeduplicationTracker()
	}
}

// WithDeduplicationKeyFunc sets a custom deduplication key function
func WithDeduplicationKeyFunc(fn DeduplicationKeyFunc) Option {
	return func(c *Client) {
		c.dedupKeyFunc = fn
	}
}

// WithDeduplicationCondition sets a custom deduplication condition function
func WithDeduplicationCondition(fn DeduplicationCondition) Option {
	return func(c *Client) {
		c.dedupCondition = fn
	}
}

// ValidateConfiguration validates the client configuration and returns an error if invalid
func (c *Client) ValidateConfiguration() error {
	var errors []string

	// Validate each configuration section
	errors = append(errors, c.validateRetryConfig()...)
	errors = append(errors, c.validateRateLimiterConfig()...)
	errors = append(errors, c.validateCacheConfig()...)
	errors = append(errors, c.validateCircuitBreakerConfig()...)
	errors = append(errors, c.validateDebugConfig()...)
	errors = append(errors, c.validateDeduplicationConfig()...)
	errors = append(errors, c.validateMiddlewareConfig()...)
	errors = append(errors, c.validateHTTPClientConfig()...)
	errors = append(errors, c.validateOptionCombinations()...)
	errors = append(errors, c.validateExtremeValues()...)

	if len(errors) > 0 {
		return &ClientError{
			Type:    ErrorTypeValidation,
			Message: "configuration validation failed",
			Cause:   fmt.Errorf("validation errors: %v", errors),
		}
	}

	return nil
}

// validateRetryConfig validates retry-related configuration
func (c *Client) validateRetryConfig() []string {
	var errors []string

	if c.maxRetries < 0 {
		errors = append(errors, "maxRetries must be non-negative")
	}

	if c.initialBackoff <= 0 {
		errors = append(errors, "initialBackoff must be positive")
	}

	if c.maxBackoff < c.initialBackoff {
		errors = append(errors, "maxBackoff must be greater than or equal to initialBackoff")
	}

	if c.backoffMultiplier <= 0 {
		errors = append(errors, "backoffMultiplier must be positive")
	}

	// Note: jitter is clamped to [0,1] in WithJitter, so this validation
	// is mainly for detecting if someone manually set an invalid value
	if c.jitter < 0 || c.jitter > 1 {
		errors = append(errors, "jitter must be between 0 and 1 (will be clamped automatically)")
	}

	if c.timeout <= 0 {
		errors = append(errors, "timeout must be positive")
	}

	return errors
}

// validateRateLimiterConfig validates rate limiter configuration
func (c *Client) validateRateLimiterConfig() []string {
	var errors []string

	if c.rateLimiter != nil {
		if c.rateLimiter.maxTokens <= 0 {
			errors = append(errors, "rateLimiter maxTokens must be positive")
		}
		if c.rateLimiter.refillRate <= 0 {
			errors = append(errors, "rateLimiter refillRate must be positive")
		}
	}

	return errors
}

// validateCacheConfig validates cache configuration
func (c *Client) validateCacheConfig() []string {
	var errors []string

	if c.cache != nil && c.cacheTTL <= 0 {
		errors = append(errors, "cacheTTL must be positive when cache is enabled")
	}

	return errors
}

// validateCircuitBreakerConfig validates circuit breaker configuration
func (c *Client) validateCircuitBreakerConfig() []string {
	var errors []string

	if c.circuitBreaker != nil {
		if c.circuitBreaker.config.FailureThreshold <= 0 {
			errors = append(errors, "circuitBreaker FailureThreshold must be positive")
		}
		if c.circuitBreaker.config.RecoveryTimeout <= 0 {
			errors = append(errors, "circuitBreaker RecoveryTimeout must be positive")
		}
		if c.circuitBreaker.config.SuccessThreshold <= 0 {
			errors = append(errors, "circuitBreaker SuccessThreshold must be positive")
		}
	}

	return errors
}

// validateDebugConfig validates debug configuration
func (c *Client) validateDebugConfig() []string {
	var errors []string

	if c.debug != nil && c.debug.Enabled {
		if c.debug.RequestIDGen == nil {
			errors = append(errors, "debug RequestIDGen must be set when debug is enabled")
		}
		if c.logger == nil {
			errors = append(errors, "logger must be set when debug is enabled")
		}
	}

	return errors
}

// validateDeduplicationConfig validates deduplication configuration
func (c *Client) validateDeduplicationConfig() []string {
	var errors []string

	if c.deduplication != nil {
		if c.dedupKeyFunc == nil {
			errors = append(errors, "deduplication key function must be set when deduplication is enabled")
		}
		if c.dedupCondition == nil {
			errors = append(errors, "deduplication condition must be set when deduplication is enabled")
		}
	}

	return errors
}

// validateMiddlewareConfig validates middleware configuration
func (c *Client) validateMiddlewareConfig() []string {
	var errors []string

	for i, middleware := range c.middleware {
		if middleware == nil {
			errors = append(errors, fmt.Sprintf("middleware[%d] cannot be nil", i))
		}
	}

	return errors
}

// validateHTTPClientConfig validates HTTP client configuration
func (c *Client) validateHTTPClientConfig() []string {
	var errors []string

	if c.httpClient == nil {
		errors = append(errors, "HTTP client cannot be nil")
	}

	return errors
}

// validateOptionCombinations validates that option combinations make sense together
func (c *Client) validateOptionCombinations() []string {
	var errors []string

	// Cache and deduplication can work together but warn about potential conflicts
	// This is allowed but could be confusing - both cache and deduplication
	// might try to handle the same request
	// Note: Currently allowing this combination, but monitoring for issues

	// Circuit breaker and retries work well together
	// Rate limiter and retries work well together
	// These combinations are fine

	// Debug logging without logger is invalid (already checked in validateDebugConfig)
	// Cache without TTL is invalid (already checked in validateCacheConfig)

	return errors
}

// validateExtremeValues validates that configuration values are within reasonable bounds
func (c *Client) validateExtremeValues() []string {
	var errors []string

	// Check for extreme retry values that could cause issues
	if c.maxRetries > 100 {
		errors = append(errors, "maxRetries > 100 may cause excessive resource usage")
	}

	// Check for extreme backoff values
	if c.initialBackoff > 10*time.Minute {
		errors = append(errors, "initialBackoff > 10m may cause very long delays")
	}
	if c.maxBackoff > 1*time.Hour {
		errors = append(errors, "maxBackoff > 1h may cause extremely long delays")
	}

	// Check for extreme timeout values
	if c.timeout > 10*time.Minute {
		errors = append(errors, "timeout > 10m may cause requests to hang for too long")
	}

	// Check for extreme rate limiter values
	if c.rateLimiter != nil {
		if c.rateLimiter.maxTokens > 1000000 {
			errors = append(errors, "rateLimiter maxTokens > 1M may cause memory issues")
		}
		if c.rateLimiter.refillRate < time.Millisecond {
			errors = append(errors, "rateLimiter refillRate < 1ms may cause excessive CPU usage")
		}
	}

	// Check for extreme cache TTL
	if c.cache != nil && c.cacheTTL > 24*time.Hour {
		errors = append(errors, "cacheTTL > 24h may cause stale data issues")
	}

	return errors
}

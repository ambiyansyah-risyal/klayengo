package klayengo

import (
	"fmt"
	"net/http"
	"time"
)

func WithMaxRetries(n int) Option {
	return func(c *Client) {
		c.maxRetries = n
	}
}

func WithInitialBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.initialBackoff = d
	}
}

func WithMaxBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.maxBackoff = d
	}
}

func WithBackoffMultiplier(f float64) Option {
	return func(c *Client) {
		c.backoffMultiplier = f
	}
}

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

func WithRateLimiter(maxTokens int, refillRate time.Duration) Option {
	return func(c *Client) {
		c.rateLimiter = NewRateLimiter(maxTokens, refillRate)
	}
}

func WithCache(ttl time.Duration) Option {
	return func(c *Client) {
		c.cache = NewInMemoryCache()
		c.cacheTTL = ttl
	}
}

func WithCustomCache(cache Cache, ttl time.Duration) Option {
	return func(c *Client) {
		c.cache = cache
		c.cacheTTL = ttl
	}
}

func WithCacheKeyFunc(fn func(*http.Request) string) Option {
	return func(c *Client) {
		c.cacheKeyFunc = fn
	}
}

func WithCacheCondition(fn CacheCondition) Option {
	return func(c *Client) {
		c.cacheCondition = fn
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
		if c.httpClient != nil {
			c.httpClient.Timeout = d
		}
	}
}

func WithRetryCondition(fn RetryCondition) Option {
	return func(c *Client) {
		c.retryCondition = fn
	}
}

func WithCircuitBreaker(config CircuitBreakerConfig) Option {
	return func(c *Client) {
		c.circuitBreaker = NewCircuitBreaker(config)
	}
}

func WithMiddleware(middleware ...Middleware) Option {
	return func(c *Client) {
		c.middleware = append(c.middleware, middleware...)
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
		if c.timeout != 0 {
			c.httpClient.Timeout = c.timeout
		}
	}
}

func WithMetrics() Option {
	return func(c *Client) {
		c.metrics = NewMetricsCollector()
	}
}

func WithMetricsCollector(collector *MetricsCollector) Option {
	return func(c *Client) {
		c.metrics = collector
	}
}

func WithDebug() Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.Enabled = true
	}
}

func WithDebugConfig(config *DebugConfig) Option {
	return func(c *Client) {
		c.debug = config
	}
}

func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

func WithSimpleLogger() Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.Enabled = true
		c.logger = NewSimpleLogger()
	}
}

func WithRequestIDGenerator(gen func() string) Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.RequestIDGen = gen
	}
}

func WithDeduplication() Option {
	return func(c *Client) {
		c.deduplication = NewDeduplicationTracker()
	}
}

func WithDeduplicationKeyFunc(fn DeduplicationKeyFunc) Option {
	return func(c *Client) {
		c.dedupKeyFunc = fn
	}
}

func WithDeduplicationCondition(fn DeduplicationCondition) Option {
	return func(c *Client) {
		c.dedupCondition = fn
	}
}

func (c *Client) ValidateConfiguration() error {
	var errors []string

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

	if c.jitter < 0 || c.jitter > 1 {
		errors = append(errors, "jitter must be between 0 and 1 (will be clamped automatically)")
	}

	if c.timeout <= 0 {
		errors = append(errors, "timeout must be positive")
	}

	return errors
}

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

func (c *Client) validateCacheConfig() []string {
	var errors []string

	if c.cache != nil && c.cacheTTL <= 0 {
		errors = append(errors, "cacheTTL must be positive when cache is enabled")
	}

	return errors
}

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

func (c *Client) validateMiddlewareConfig() []string {
	var errors []string

	for i, middleware := range c.middleware {
		if middleware == nil {
			errors = append(errors, fmt.Sprintf("middleware[%d] cannot be nil", i))
		}
	}

	return errors
}

func (c *Client) validateHTTPClientConfig() []string {
	var errors []string

	if c.httpClient == nil {
		errors = append(errors, "HTTP client cannot be nil")
	}

	return errors
}

func (c *Client) validateOptionCombinations() []string {
	var errors []string

	return errors
}

func (c *Client) validateExtremeValues() []string {
	var errors []string

	if c.maxRetries > 100 {
		errors = append(errors, "maxRetries > 100 may cause excessive resource usage")
	}

	if c.initialBackoff > 10*time.Minute {
		errors = append(errors, "initialBackoff > 10m may cause very long delays")
	}
	if c.maxBackoff > 1*time.Hour {
		errors = append(errors, "maxBackoff > 1h may cause extremely long delays")
	}

	if c.timeout > 10*time.Minute {
		errors = append(errors, "timeout > 10m may cause requests to hang for too long")
	}

	if c.rateLimiter != nil {
		if c.rateLimiter.maxTokens > 1000000 {
			errors = append(errors, "rateLimiter maxTokens > 1M may cause memory issues")
		}
		if c.rateLimiter.refillRate < time.Millisecond {
			errors = append(errors, "rateLimiter refillRate < 1ms may cause excessive CPU usage")
		}
	}

	if c.cache != nil && c.cacheTTL > 24*time.Hour {
		errors = append(errors, "cacheTTL > 24h may cause stale data issues")
	}

	return errors
}

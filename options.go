package klayengo

import (
	"fmt"
	"net/http"
	"time"
)

// WithMaxRetries sets the maximum retry attempts.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		c.maxRetries = n
	}
}

// WithInitialBackoff sets the initial backoff duration.
func WithInitialBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.initialBackoff = d
	}
}

// WithMaxBackoff sets the maximum backoff duration.
func WithMaxBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.maxBackoff = d
	}
}

// WithBackoffMultiplier sets the exponential backoff multiplier.
func WithBackoffMultiplier(f float64) Option {
	return func(c *Client) {
		c.backoffMultiplier = f
	}
}

// WithJitter sets jitter factor (0..1) applied to backoff.
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

// WithBackoffStrategy sets the backoff algorithm for retry delays.
func WithBackoffStrategy(strategy BackoffStrategy) Option {
	return func(c *Client) {
		c.backoffStrategy = strategy
		// Update the calculator will be done in New() after all options are applied
	}
}

// WithRateLimiter enables a token bucket rate limiter.
func WithRateLimiter(maxTokens int, refillRate time.Duration) Option {
	return func(c *Client) {
		c.rateLimiter = NewRateLimiter(maxTokens, refillRate)
	}
}

// WithLimiterFor registers a rate limiter for a specific key.
func WithLimiterFor(key string, limiter Limiter) Option {
	return func(c *Client) {
		if c.limiterRegistry == nil {
			c.limiterRegistry = NewRateLimiterRegistry(c.limiterKeyFunc, c.rateLimiter)
		}
		c.limiterRegistry.RegisterLimiter(key, limiter)
	}
}

// WithLimiterKeyFunc sets the function used to generate rate limiter keys from requests.
func WithLimiterKeyFunc(keyFunc KeyFunc) Option {
	return func(c *Client) {
		c.limiterKeyFunc = keyFunc
		if c.limiterRegistry != nil {
			// Update existing registry with new key function
			c.limiterRegistry = NewRateLimiterRegistry(keyFunc, c.rateLimiter)
		}
	}
}

// WithCache enables the default in-memory cache with a global TTL.
func WithCache(ttl time.Duration) Option {
	return func(c *Client) {
		c.cache = NewInMemoryCache()
		c.cacheTTL = ttl
	}
}

// WithCustomCache supplies a custom cache implementation plus TTL.
func WithCustomCache(cache Cache, ttl time.Duration) Option {
	return func(c *Client) {
		c.cache = cache
		c.cacheTTL = ttl
	}
}

// WithCacheKeyFunc overrides the cache key generator.
func WithCacheKeyFunc(fn func(*http.Request) string) Option {
	return func(c *Client) {
		c.cacheKeyFunc = fn
	}
}

// WithCacheCondition overrides the cacheable request predicate.
func WithCacheCondition(fn CacheCondition) Option {
	return func(c *Client) {
		c.cacheCondition = fn
	}
}

// WithCacheProvider sets a new cache provider supporting HTTP semantics.
func WithCacheProvider(provider CacheProvider) Option {
	return func(c *Client) {
		c.cacheProvider = provider
	}
}

// WithCacheMode sets the cache behavior mode.
func WithCacheMode(mode CacheMode) Option {
	return func(c *Client) {
		c.cacheMode = mode
	}
}

// WithTimeout sets per-request timeout (sets underlying http.Client timeout).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
		if c.httpClient != nil {
			c.httpClient.Timeout = d
		}
	}
}

// WithRetryCondition overrides when a response/error should trigger a retry.
func WithRetryCondition(fn RetryCondition) Option {
	return func(c *Client) {
		c.retryCondition = fn
	}
}

// WithRetryPolicy sets a custom retry policy that determines retry behavior.
func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *Client) {
		c.retryPolicy = policy
	}
}

// WithRetryBudget enables retry budget tracking to prevent thundering herd.
func WithRetryBudget(maxRetries int, perWindow time.Duration) Option {
	return func(c *Client) {
		c.retryBudget = NewRetryBudget(maxRetries, perWindow)
	}
}

// WithCircuitBreaker configures a circuit breaker.
func WithCircuitBreaker(config CircuitBreakerConfig) Option {
	return func(c *Client) {
		c.circuitBreaker = NewCircuitBreaker(config)
	}
}

// WithMiddleware appends middleware to execution chain (outermost first).
func WithMiddleware(middleware ...Middleware) Option {
	return func(c *Client) {
		c.middleware = append(c.middleware, middleware...)
	}
}

// WithHTTPClient supplies a custom *http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
		if c.timeout != 0 {
			c.httpClient.Timeout = c.timeout
		}
	}
}

// WithMetrics enables metrics collection using default registry.
func WithMetrics() Option {
	return func(c *Client) {
		c.metrics = NewMetricsCollector()
	}
}

// WithMetricsCollector injects an existing metrics collector.
func WithMetricsCollector(collector *MetricsCollector) Option {
	return func(c *Client) {
		c.metrics = collector
	}
}

// WithDebug enables debug flags using default config.
func WithDebug() Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.Enabled = true
	}
}

// WithDebugConfig sets a custom debug config.
func WithDebugConfig(config *DebugConfig) Option {
	return func(c *Client) {
		c.debug = config
	}
}

// WithLogger sets the logger used when debug is enabled.
func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithSimpleLogger attaches a minimal stdout logger and enables debug.
func WithSimpleLogger() Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.Enabled = true
		c.logger = NewSimpleLogger()
	}
}

// WithRequestIDGenerator overrides the request id generator used in debug logs.
func WithRequestIDGenerator(gen func() string) Option {
	return func(c *Client) {
		if c.debug == nil {
			c.debug = DefaultDebugConfig()
		}
		c.debug.RequestIDGen = gen
	}
}

// WithDeduplication enables in-flight request de-duplication.
func WithDeduplication() Option {
	return func(c *Client) {
		c.deduplication = NewDeduplicationTracker()
	}
}

// WithDeduplicationKeyFunc sets custom deduplication key generator.
func WithDeduplicationKeyFunc(fn DeduplicationKeyFunc) Option {
	return func(c *Client) {
		c.dedupKeyFunc = fn
	}
}

// WithDeduplicationCondition sets custom predicate for deduplication eligibility.
func WithDeduplicationCondition(fn DeduplicationCondition) Option {
	return func(c *Client) {
		c.dedupCondition = fn
	}
}

// WithUnmarshaler sets a custom response unmarshaler for typed response methods.
func WithUnmarshaler(unmarshaler ResponseUnmarshaler) Option {
	return func(c *Client) {
		c.unmarshaler = unmarshaler
	}
}

// ValidateConfiguration performs shallow validation returning aggregated error.
func (c *Client) ValidateConfiguration() error {
	var errors []string

	errors = append(errors, c.validateRetryConfig()...)
	errors = append(errors, c.validateRetryPolicyConfig()...)
	errors = append(errors, c.validateRetryBudgetConfig()...)
	errors = append(errors, c.validateBackoffStrategyConfig()...)
	errors = append(errors, c.validateRateLimiterConfig()...)
	errors = append(errors, c.validateCacheConfig()...)
	errors = append(errors, c.validateCircuitBreakerConfig()...)
	errors = append(errors, c.validateDebugConfig()...)
	errors = append(errors, c.validateDeduplicationConfig()...)
	errors = append(errors, c.validateMiddlewareConfig()...)
	errors = append(errors, c.validateHTTPClientConfig()...)
	errors = append(errors, c.validateUnmarshalerConfig()...)
	errors = append(errors, c.validateOptionCombinations()...)
	errors = append(errors, c.validateExtremeValues()...)

	if len(errors) > 0 {
		return &ClientError{
			Type:    ErrorTypeValidation,
			Message: "configuration validation failed",
			Cause:   fmt.Errorf("configuration validation failed: %v", errors),
		}
	}

	return nil
}

// WithSingleFlightEnabled enables the internal singleflight group for deduplicating concurrent requests.
// This is experimental and not yet wired to cache functionality.
func WithSingleFlightEnabled(enabled bool) Option {
	return func(c *Client) {
		c.singleFlightEnabled = enabled
	}
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

	if c.limiterRegistry != nil {
		if c.limiterKeyFunc == nil {
			errors = append(errors, "limiterKeyFunc must be set when limiterRegistry is enabled")
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

func (c *Client) validateUnmarshalerConfig() []string {
	var errors []string

	if c.unmarshaler == nil {
		errors = append(errors, "unmarshaler cannot be nil")
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

func (c *Client) validateRetryPolicyConfig() []string {
	var errors []string

	// RetryPolicy is optional, no validation needed if nil
	// If both retryPolicy and legacy retry fields are set, that's allowed (retryPolicy takes precedence)

	return errors
}

func (c *Client) validateRetryBudgetConfig() []string {
	var errors []string

	if c.retryBudget != nil {
		if c.retryBudget.maxRetries <= 0 {
			errors = append(errors, "retryBudget maxRetries must be positive")
		}
		if c.retryBudget.perWindow <= 0 {
			errors = append(errors, "retryBudget perWindow must be positive")
		}
		if c.retryBudget.maxRetries > 1000 {
			errors = append(errors, "retryBudget maxRetries > 1000 may cause excessive resource usage")
		}
		if c.retryBudget.perWindow < time.Second {
			errors = append(errors, "retryBudget perWindow < 1s may cause excessive retry limiting")
		}
	}

	return errors
}

func (c *Client) validateBackoffStrategyConfig() []string {
	var errors []string

	switch c.backoffStrategy {
	case ExponentialJitter, DecorrelatedJitter:
		// Valid strategies - no error
	default:
		errors = append(errors, fmt.Sprintf("invalid backoffStrategy: %d", int(c.backoffStrategy)))
	}

	return errors
}

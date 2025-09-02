package klayengo

import (
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

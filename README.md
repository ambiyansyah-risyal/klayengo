# klayengo

[![Go Reference](https://pkg.go.dev/badge/github.com/ambiyansyah-risyal/klayengo)](https://pkg.go.dev/github.com/ambiyansyah-risyal/klayengo)
[![Go Report Card](https://goreportcard.com/badge/github.com/ambiyansyah-risyal/klayengo)](https://goreportcard.com/report/github.com/ambiyansyah-risyal/klayengo)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A resilient HTTP client wrapper for Go with retry logic, exponential backoff, and circuit breaker pattern.

## Features

- **Retry Logic**: Automatic retry on failed HTTP requests with configurable retry attempts
- **Exponential Backoff**: Intelligent backoff strategy with jitter to avoid overwhelming servers
- **Rate Limiting**: Configurable token bucket rate limiting to control request frequency
- **Response Caching**: In-memory caching for GET requests with customizable TTL and conditions
- **Circuit Breaker**: Prevents cascading failures by temporarily stopping requests to failing services
- **Custom Error Types**: Structured error handling with specific error types for different failure modes
- **Context Cancellation**: Full support for Go context for request cancellation and timeouts
- **Middleware Hooks**: Extensible middleware system for logging, metrics, and custom logic
- **Configurable Options**: Highly customizable retry policies, timeouts, and behavior using functional options
- **Thread-Safe**: Safe for concurrent use across multiple goroutines
- **Prometheus Metrics**: Built-in metrics collection for monitoring and observability
- **Per-Request Overrides**: Override global settings on a per-request basis using context

## Installation

```bash
go get github.com/ambiyansyah-risyal/klayengo
```

**Requirements:**
- Go 1.23.0 or later (tested with Go 1.24.6)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/ambiyansyah-risyal/klayengo"
)

func main() {
    // Create a new resilient HTTP client with default options
    client := klayengo.New()

    // Make a request with automatic retry and circuit breaker protection
    resp, err := client.Get(context.Background(), "https://api.example.com/data")
    if err != nil {
        fmt.Printf("Request failed: %v\n", err)
        return
    }
    defer resp.Body.Close()

    fmt.Printf("Response status: %s\n", resp.Status)
}
```

## Configuration

### Basic Configuration

### Basic Configuration

```go
client := klayengo.New(
    klayengo.WithMaxRetries(5),
    klayengo.WithInitialBackoff(100*time.Millisecond),
    klayengo.WithMaxBackoff(10*time.Second),
    klayengo.WithBackoffMultiplier(2.0),
    klayengo.WithJitter(0.1), // 10% jitter to avoid thundering herd
    klayengo.WithTimeout(30*time.Second),
)
```

### Rate Limiting Configuration

```go
client := klayengo.New(
    klayengo.WithRateLimiter(10, 1*time.Second), // 10 requests per second
)
```

### Caching Configuration

```go
// Enable caching with default in-memory cache
client := klayengo.New(
    klayengo.WithCache(5*time.Minute), // Cache responses for 5 minutes
)

// Use custom cache implementation
customCache := klayengo.NewInMemoryCache()
client := klayengo.New(
    klayengo.WithCustomCache(customCache, 10*time.Minute),
)

// Custom cache key function
client := klayengo.New(
    klayengo.WithCacheKeyFunc(func(req *http.Request) string {
        // Include headers in cache key for more specific caching
        return fmt.Sprintf("%s:%s:%s", req.Method, req.URL.String(), req.Header.Get("Authorization"))
    }),
)

// Custom cache condition - only cache specific requests
client := klayengo.New(
    klayengo.WithCacheCondition(func(req *http.Request) bool {
        // Only cache GET requests to /api/data endpoint
        return req.Method == "GET" && req.URL.Path == "/api/data"
    }),
)
```

### Per-Request Cache Control

You can control caching on a per-request basis using context:

```go
// Enable caching for specific request
ctx := klayengo.WithContextCacheEnabled(context.Background())
resp, err := client.Get(ctx, "https://api.example.com/data")

// Disable caching for specific request
ctx := klayengo.WithContextCacheDisabled(context.Background())
resp, err := client.Get(ctx, "https://api.example.com/dynamic-data")

// Custom TTL for specific request
ctx := klayengo.WithContextCacheTTL(context.Background(), 30*time.Minute)
resp, err := client.Get(ctx, "https://api.example.com/important-data")
```

### Circuit Breaker Configuration

```go
client := klayengo.New(
    klayengo.WithCircuitBreaker(
        klayengo.CircuitBreakerConfig{
            FailureThreshold: 5,
            RecoveryTimeout:  60 * time.Second,
            SuccessThreshold: 2,
        },
    ),
)
```

### Custom Retry Conditions

```go
client := klayengo.New(
    klayengo.WithRetryCondition(func(resp *http.Response, err error) bool {
        // Retry on 5xx errors or network errors
        if err != nil {
            return true
        }
        return resp.StatusCode >= 500
    }),
)
```

### Metrics Configuration

Klayengo provides built-in Prometheus metrics for monitoring HTTP client performance and behavior:

```go
// Enable default metrics collection
client := klayengo.New(
    klayengo.WithMetrics(),
)

// Use custom metrics collector
customCollector := klayengo.NewMetricsCollector()
client := klayengo.New(
    klayengo.WithMetricsCollector(customCollector),
)
```

#### Available Metrics

- **`klayengo_requests_total`**: Total number of HTTP requests made (counter)
  - Labels: `method`, `status_code`, `endpoint`
- **`klayengo_request_duration_seconds`**: Duration of HTTP requests (histogram)
  - Labels: `method`, `status_code`, `endpoint`
- **`klayengo_requests_in_flight`**: Number of requests currently in flight (gauge)
  - Labels: `method`, `endpoint`
- **`klayengo_retries_total`**: Total number of retry attempts (counter)
  - Labels: `method`, `endpoint`, `attempt`
- **`klayengo_circuit_breaker_state`**: Current circuit breaker state (gauge)
  - Labels: `name`
  - Values: 0=closed, 1=open, 2=half-open
- **`klayengo_rate_limiter_tokens`**: Current number of available rate limiter tokens (gauge)
  - Labels: `name`
- **`klayengo_cache_hits_total`**: Total number of cache hits (counter)
  - Labels: `method`, `endpoint`
- **`klayengo_cache_misses_total`**: Total number of cache misses (counter)
  - Labels: `method`, `endpoint`
- **`klayengo_cache_size`**: Current number of entries in cache (gauge)
  - Labels: `name`
- **`klayengo_errors_total`**: Total number of errors encountered (counter)
  - Labels: `type`, `method`, `endpoint`

#### Exposing Metrics

To expose metrics via HTTP endpoint:

```go
import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

client := klayengo.New(klayengo.WithMetrics())

// Expose metrics at /metrics endpoint
http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":8080", nil)
```

#### Custom Metrics Registry

You can use a custom Prometheus registry:

```go
registry := prometheus.NewRegistry()
collector := klayengo.NewMetricsCollectorWithRegistry(registry)
client := klayengo.New(klayengo.WithMetricsCollector(collector))

// Use registry with your metrics endpoint
http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
```

## Advanced Usage

### Using with Custom HTTP Client

```go
customClient := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}

retryClient := klayengo.New(
    klayengo.WithHTTPClient(customClient),
)
```

### Middleware for Logging

```go
client := klayengo.New(
    klayengo.WithMiddleware(func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
        start := time.Now()
        fmt.Printf("Starting request to %s\n", req.URL)

        resp, err := next.RoundTrip(req)

        duration := time.Since(start)
        if err != nil {
            fmt.Printf("Request failed after %v: %v\n", duration, err)
        } else {
            fmt.Printf("Request completed in %v with status %s\n", duration, resp.Status)
        }

        return resp, err
    }),
)
```

### Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Get(ctx, "https://api.example.com/slow-endpoint")
```

## Error Handling

Klayengo provides structured error handling with specific error types:

```go
import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "time"

    "github.com/ambiyansyah-risyal/klayengo"
)

resp, err := client.Get(context.Background(), "https://api.example.com/endpoint")
if err != nil {
    var clientErr *klayengo.ClientError
    if errors.As(err, &clientErr) {
        switch clientErr.Type {
        case "RateLimit":
            // Handle rate limit exceeded
        case "CircuitBreaker":
            // Handle circuit breaker open
        default:
            // Handle other errors
        }
    }
}
```

## Caching

Klayengo supports response caching to improve performance and reduce server load:

### Cache Interface

```go
type Cache interface {
    Get(key string) (*CacheEntry, bool)
    Set(key string, entry *CacheEntry, ttl time.Duration)
    Delete(key string)
    Clear()
}
```

### Cache Key Generation

By default, cache keys are generated from the request method and URL. You can customize this:

```go
client := klayengo.New(
    klayengo.WithCacheKeyFunc(func(req *http.Request) string {
        // Include query parameters in cache key
        return fmt.Sprintf("%s:%s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
    }),
)
```

### Cache Behavior

- **Conditional Caching**: Only requests that meet the cache condition are cached
- **Context Override**: Context values can override global cache settings per request
- **TTL Control**: Different TTL values can be set globally or per request
- **Thread-Safe**: Safe for concurrent access across multiple goroutines
- **Automatic Expiration**: Cache entries are automatically removed when they expire

## API Reference

### Client Methods

- `New(options ...Option) *Client` - Create a new retry client
- `Get(ctx context.Context, url string) (*http.Response, error)` - GET request
- `Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error)` - POST request
- `Do(req *http.Request) (*http.Response, error)` - Execute custom request

### Configuration Options

- `WithMaxRetries(n int)` - Maximum number of retry attempts
- `WithInitialBackoff(d time.Duration)` - Initial backoff duration
- `WithMaxBackoff(d time.Duration)` - Maximum backoff duration
- `WithBackoffMultiplier(f float64)` - Backoff multiplier (default: 2.0)
- `WithJitter(f float64)` - Jitter factor for backoff randomization (0.0 to 1.0)
- `WithRateLimiter(maxTokens int, refillRate time.Duration)` - Rate limiter configuration
- `WithCache(ttl time.Duration)` - Enable in-memory caching with TTL
- `WithCustomCache(cache Cache, ttl time.Duration)` - Use custom cache implementation
- `WithCacheKeyFunc(fn func(*http.Request) string)` - Custom cache key generation function
- `WithCacheCondition(fn CacheCondition)` - Custom cache condition function
- `WithTimeout(d time.Duration)` - Request timeout
- `WithRetryCondition(fn RetryCondition)` - Custom retry condition
- `WithCircuitBreaker(config CircuitBreakerConfig)` - Circuit breaker configuration
- `WithMiddleware(middleware ...Middleware)` - Request middleware
- `WithHTTPClient(client *http.Client)` - Custom HTTP client
- `WithMetrics()` - Enable Prometheus metrics collection
- `WithMetricsCollector(collector *MetricsCollector)` - Use custom metrics collector

### Context Helper Functions

- `WithContextCacheEnabled(ctx context.Context) context.Context` - Enable caching for specific request
- `WithContextCacheDisabled(ctx context.Context) context.Context` - Disable caching for specific request
- `WithContextCacheTTL(ctx context.Context, ttl time.Duration) context.Context` - Set custom TTL for specific request

### Circuit Breaker States

- `Closed`: Normal operation, requests pass through
- `Open`: Circuit is open, requests fail immediately
- `HalfOpen`: Testing if service has recovered

## Examples

See the [`examples/`](examples/) directory for complete examples:

- [`basic/`](examples/basic/) - Basic retry usage with middleware
- [`advanced/`](examples/advanced/) - Advanced usage with rate limiting, jitter, and multiple middleware
- [`metrics/`](examples/metrics/) - Metrics collection with Prometheus integration
- Circuit breaker implementation
- Custom middleware
- Advanced configuration

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Testing

```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [go-retry](https://github.com/go-retry/retry) - Simple retry library
- [circuit](https://github.com/rubyist/circuitbreaker) - Circuit breaker implementation
- [backoff](https://github.com/cenkalti/backoff) - Exponential backoff library

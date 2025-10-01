# klayengo

🚀 **Resilient HTTP client for Go** with advanced reliability patterns including retry, rate limiting, caching, circuit breaker, request deduplication, and observability.

[![Go Reference](https://pkg.go.dev/badge/github.com/ambiyansyah-risyal/klayengo.svg)](https://pkg.go.dev/github.com/ambiyansyah-risyal/klayengo)
[![Go Report Card](https://goreportcard.com/badge/github.com/ambiyansyah-risyal/klayengo)](https://goreportcard.com/report/github.com/ambiyansyah-risyal/klayengo)

## ✨ Features

- 🔄 **Retry Logic** - Configurable backoff strategies (exponential/decorrelated jitter) and custom retry conditions
- 🚦 **Rate Limiting** - Token bucket algorithm with configurable limits
- 💾 **Response Caching** - In-memory HTTP response caching with TTL
- ⚡ **Circuit Breaker** - Fail-fast pattern with automatic recovery
- 🔗 **Request Deduplication** - Merge concurrent identical requests
- 📊 **Metrics** - Prometheus integration for monitoring
- 🔧 **Middleware** - Extensible request/response interceptors
- 🐛 **Debug & Logging** - Comprehensive observability
- ⚙️ **Configuration Validation** - Runtime validation of client settings

## 📦 Installation

```bash
go get github.com/ambiyansyah-risyal/klayengo
```

## 🚀 Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/ambiyansyah-risyal/klayengo"
)

func main() {
    client := klayengo.New(
        klayengo.WithMaxRetries(3),
        klayengo.WithRateLimiter(10, time.Second),
        klayengo.WithCache(5*time.Minute),
        klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
            FailureThreshold: 5,
            RecoveryTimeout:  30 * time.Second,
        }),
        klayengo.WithDeduplication(),
    )

    resp, err := client.Get(context.Background(), "https://api.example.com/data")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
}
```

## 📖 Examples

### Basic Usage with All Features

```go
client := klayengo.New(
    // Retry configuration
    klayengo.WithMaxRetries(3),
    klayengo.WithInitialBackoff(100*time.Millisecond),
    klayengo.WithMaxBackoff(10*time.Second),
    klayengo.WithBackoffMultiplier(2.0),
    klayengo.WithJitter(0.1),

    // Rate limiting (10 requests per second)
    klayengo.WithRateLimiter(10, time.Second),

    // Response caching
    klayengo.WithCache(5*time.Minute),

    // Circuit breaker
    klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
        FailureThreshold: 5,
        RecoveryTimeout:  30 * time.Second,
        SuccessThreshold: 3,
    }),

    // Request deduplication
    klayengo.WithDeduplication(),

    // Timeout
    klayengo.WithTimeout(30*time.Second),
)
```

### Advanced Rate Limiting

klayengo supports per-host and per-route rate limiting to prevent one hot endpoint from starving others:

```go
// Per-host rate limiting
apiLimiter := klayengo.NewRateLimiter(50, time.Minute)  // 50 req/min for API
webLimiter := klayengo.NewRateLimiter(200, time.Minute) // 200 req/min for web

client := klayengo.New(
    klayengo.WithLimiterKeyFunc(klayengo.DefaultHostKeyFunc),
    klayengo.WithLimiterFor("host:api.example.com", apiLimiter),
    klayengo.WithLimiterFor("host:web.example.com", webLimiter),
    klayengo.WithRateLimiter(10, time.Second), // Fallback limiter
)

// Per-route rate limiting
usersLimiter := klayengo.NewRateLimiter(100, time.Minute) // 100 req/min for /users
postsLimiter := klayengo.NewRateLimiter(500, time.Minute) // 500 req/min for /posts

client := klayengo.New(
    klayengo.WithLimiterKeyFunc(klayengo.DefaultRouteKeyFunc),
    klayengo.WithLimiterFor("route:GET:/users", usersLimiter),
    klayengo.WithLimiterFor("route:GET:/posts", postsLimiter),
)

// Custom key function
client := klayengo.New(
    klayengo.WithLimiterKeyFunc(func(req *http.Request) string {
        return fmt.Sprintf("service:%s", req.Header.Get("X-Service"))
    }),
    klayengo.WithLimiterFor("service:auth", klayengo.NewRateLimiter(1000, time.Minute)),
    klayengo.WithLimiterFor("service:api", klayengo.NewRateLimiter(5000, time.Minute)),
)
```

### Backoff Strategies

klayengo supports different backoff strategies for retry delays to optimize for different scenarios:

```go
// Exponential Jitter (default) - stable, predictable backoff
client := klayengo.New(
    klayengo.WithBackoffStrategy(klayengo.ExponentialJitter),
    klayengo.WithMaxRetries(5),
    klayengo.WithInitialBackoff(100*time.Millisecond),
    klayengo.WithMaxBackoff(10*time.Second),
    klayengo.WithJitter(0.1),
)

// Decorrelated Jitter - smoother tail latencies, avoids synchronized retry storms
client := klayengo.New(
    klayengo.WithBackoffStrategy(klayengo.DecorrelatedJitter),
    klayengo.WithMaxRetries(5),
    klayengo.WithInitialBackoff(100*time.Millisecond),
    klayengo.WithMaxBackoff(10*time.Second),
)

// With custom retry policy - backoff strategy applies within the policy
customPolicy := klayengo.NewDefaultRetryPolicyWithStrategy(
    5,                                    // maxRetries
    100*time.Millisecond,                // initialBackoff
    30*time.Second,                      // maxBackoff
    2.0,                                 // multiplier
    0.1,                                 // jitter
    klayengo.DecorrelatedJitter,         // strategy
)

client := klayengo.New(
    klayengo.WithRetryPolicy(customPolicy),
)
```

**When to use which strategy:**
- **ExponentialJitter**: Default choice for most applications. Provides predictable, stable backoff with uniform jitter.
- **DecorrelatedJitter**: Use when you need smoother tail latencies and want to minimize synchronized retry storms across multiple clients. Particularly useful in high-throughput scenarios with many concurrent clients.

### Custom Middleware

```go
// Authentication middleware
authMiddleware := func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
    req.Header.Set("Authorization", "Bearer "+getToken())
    return next.RoundTrip(req)
}

// Logging middleware
loggingMiddleware := func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
    start := time.Now()
    resp, err := next.RoundTrip(req)
    log.Printf("Request to %s took %v", req.URL, time.Since(start))
    return resp, err
}

client := klayengo.New(
    klayengo.WithMiddleware(authMiddleware, loggingMiddleware),
    // ... other options
)
```

### Production Configuration with Metrics

```go
// Create custom metrics collector for access to registry
metricsCollector := klayengo.NewMetricsCollector()

client := klayengo.New(
    // Conservative retry settings
    klayengo.WithMaxRetries(3),
    klayengo.WithInitialBackoff(100*time.Millisecond),
    klayengo.WithMaxBackoff(30*time.Second),

    // Rate limiting based on API limits
    klayengo.WithRateLimiter(100, time.Minute),

    // Longer cache for production
    klayengo.WithCache(15*time.Minute),

    // Circuit breaker tuned for stability
    klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
        FailureThreshold: 10,
        RecoveryTimeout:  60 * time.Second,
        SuccessThreshold: 5,
    }),

    // Enable all features
    klayengo.WithDeduplication(),
    klayengo.WithMetricsCollector(metricsCollector),
    klayengo.WithTimeout(60*time.Second),
)

// Access Prometheus registry for /metrics endpoint
registry := metricsCollector.GetRegistry()
// Use with prometheus/client_golang/prometheus/promhttp:
// http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
```

### Custom Retry Logic

```go
client := klayengo.New(
    klayengo.WithRetryCondition(func(resp *http.Response, err error) bool {
        if err != nil {
            return true // Always retry network errors
        }
        // Retry on 5xx, 429 (rate limited), and specific gateway errors
        return resp.StatusCode >= 500 ||
               resp.StatusCode == 429 ||
               resp.StatusCode == 502 ||
               resp.StatusCode == 503 ||
               resp.StatusCode == 504
    }),
    klayengo.WithMaxRetries(5),
)
```

### Custom Caching Strategy

```go
client := klayengo.New(
    klayengo.WithCache(10*time.Minute),
    klayengo.WithCacheCondition(func(req *http.Request) bool {
        // Only cache GET requests without auth headers
        return req.Method == "GET" && req.Header.Get("Authorization") == ""
    }),
    klayengo.WithCacheKeyFunc(func(req *http.Request) string {
        // Custom cache key including query parameters
        return fmt.Sprintf("%s:%s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
    }),
)
```

### Debug and Logging

```go
client := klayengo.New(
    klayengo.WithSimpleLogger(),
    klayengo.WithDebugConfig(&klayengo.DebugConfig{
        Enabled:      true,
        LogRequests:  true,
        LogRetries:   true,
        LogCache:     true,
        LogRateLimit: true,
        LogCircuit:   true,
        RequestIDGen: func() string {
            return fmt.Sprintf("req_%d", time.Now().UnixNano())
        },
    }),
)
```

## 🏗️ Architecture

klayengo uses a middleware-based architecture where each feature is implemented as a composable layer:

```
Request → Middleware → Rate Limiter → Circuit Breaker → Cache → Retry → HTTP Client
Response ← Middleware ← Rate Limiter ← Circuit Breaker ← Cache ← Retry ← HTTP Client
```

## 📊 Observability

### Prometheus Metrics

When metrics are enabled, klayengo exposes the following Prometheus metrics:

- `klayengo_requests_total` - Total HTTP requests
- `klayengo_request_duration_seconds` - Request duration histogram
- `klayengo_requests_in_flight` - Current in-flight requests
- `klayengo_retries_total` - Total retry attempts
- `klayengo_circuit_breaker_state` - Circuit breaker state (0=closed, 1=open, 2=half-open)
- `klayengo_rate_limiter_tokens` - Available rate limiter tokens
- `klayengo_cache_hits_total` - Cache hit count
- `klayengo_cache_misses_total` - Cache miss count
- `klayengo_cache_size` - Current cache size
- `klayengo_deduplication_hits_total` - Request deduplication hits
- `klayengo_retry_budget_exceeded_total` - Retry attempts denied due to retry budget exhaustion (label: host)
- `klayengo_errors_total` - Errors by type (Network, Server, RateLimit, etc.)

### Debug Logging

Enable comprehensive logging to understand client behavior:

```go
client := klayengo.New(
    klayengo.WithDebug(),
    klayengo.WithSimpleLogger(),
)
```

### Advanced Retry: Policy and Budget

```go
// Define a custom retry policy (overrides legacy backoff knobs when provided)
type MyPolicy struct{}

func (p MyPolicy) ShouldRetry(resp *http.Response, err error, attempt int) (time.Duration, bool) {
    if err != nil || (resp != nil && resp.StatusCode >= 500) {
        // constant 200ms delay up to 5 attempts
        if attempt < 5 {
            return 200 * time.Millisecond, true
        }
    }
    return 0, false
}

client := klayengo.New(
    klayengo.WithRetryPolicy(MyPolicy{}),
    // Budget: allow up to 10 retries per minute across calls
    klayengo.WithRetryBudget(10, time.Minute),
)
```
## 🧪 Testing

Run the comprehensive example to see all features in action:

```bash
cd example
go run main.go
```

## ⚙️ Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithMaxRetries(n)` | Maximum retry attempts | 3 |
| `WithInitialBackoff(d)` | Initial retry backoff | 100ms |
| `WithMaxBackoff(d)` | Maximum retry backoff | 10s |
| `WithBackoffMultiplier(f)` | Backoff multiplier | 2.0 |
| `WithJitter(f)` | Jitter factor (0-1) | 0.1 |
| `WithBackoffStrategy(s)` | Backoff algorithm (ExponentialJitter/DecorrelatedJitter) | ExponentialJitter |
| `WithTimeout(d)` | Request timeout | 30s |
| `WithRateLimiter(tokens, interval)` | Global rate limiting | None |
| `WithLimiterKeyFunc(fn)` | Key function for per-key limiting | None |
| `WithLimiterFor(key, limiter)` | Rate limiter for specific key | None |
| `WithCache(ttl)` | Response caching | None |
| `WithCircuitBreaker(config)` | Circuit breaker | Disabled |
| `WithRetryPolicy(policy)` | Custom retry policy (overrides legacy knobs) | None |
| `WithRetryBudget(max, window)` | Cap retries per time window | None |
| `WithDeduplication()` | Request deduplication | Disabled |
| `WithMetrics()` | Prometheus metrics | Disabled |
| `WithMiddleware(...)` | Custom middleware | None |
| `WithDebug()` | Debug logging | Disabled |

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

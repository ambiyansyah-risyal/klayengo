# klayengo

🚀 **Resilient HTTP client for Go** with advanced reliability patterns including retry, rate limiting, caching, circuit breaker, request deduplication, and observability.

[![Go Reference](https://pkg.go.dev/badge/github.com/ambiyansyah-risyal/klayengo.svg)](https://pkg.go.dev/github.com/ambiyansyah-risyal/klayengo)
[![Go Report Card](https://goreportcard.com/badge/github.com/ambiyansyah-risyal/klayengo)](https://goreportcard.com/report/github.com/ambiyansyah-risyal/klayengo)

## ✨ Features

- 🔄 **Retry Logic** - Configurable backoff strategies (exponential/decorrelated jitter) and custom retry conditions
- 🚦 **Rate Limiting** - Token bucket algorithm with configurable limits
- 💾 **HTTP Cache Semantics** - Smart caching with ETag, Cache-Control, SWR, and stampede protection
- ⚡ **Circuit Breaker** - Fail-fast pattern with automatic recovery
- 🔗 **Request Deduplication** - Merge concurrent identical requests
- 📊 **Metrics** - Prometheus integration for monitoring
- 🔧 **Middleware** - Extensible request/response interceptors
- 🐛 **Debug & Logging** - Comprehensive observability
- ⚙️ **Configuration Validation** - Runtime validation of client settings
- 🎯 **Typed Responses** - Automatic JSON unmarshaling into struct types with compile-time type safety

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

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Email string `json:"email"`
}

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

    // Traditional approach - manual JSON handling
    resp, err := client.Get(context.Background(), "https://api.example.com/users/1")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    
    // New typed approach - automatic JSON unmarshaling
    var user User
    err = client.GetJSON(context.Background(), "https://api.example.com/users/1", &user)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("User: %+v", user)
}
```

## 📖 Examples

### Typed Responses (New!)

klayengo now supports automatic JSON unmarshaling into struct types, eliminating the need for manual JSON parsing:

```go
type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Username string `json:"username"`
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

client := klayengo.New(
    klayengo.WithMaxRetries(3),
    klayengo.WithRateLimiter(10, time.Second),
    klayengo.WithCache(5*time.Minute),
)

// GET with automatic JSON unmarshaling
var user User
err := client.GetJSON(ctx, "https://api.example.com/users/1", &user)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User: %+v", user)

// POST with JSON request/response
request := CreateUserRequest{Name: "John", Email: "john@example.com"}
var newUser User
err = client.PostJSON(ctx, "https://api.example.com/users", request, &newUser)
if err != nil {
    log.Fatal(err)
}

// Get both HTTP response metadata AND typed data
var users []User
typedResp, err := client.GetTyped(ctx, "https://api.example.com/users", &users)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Status: %d, Users: %+v", typedResp.StatusCode, users)
```

**Benefits:**
- ✅ **Type Safety** - Compile-time type checking prevents runtime errors
- ✅ **No Manual Parsing** - Automatic JSON unmarshaling
- ✅ **Same Resilience** - All retry, caching, circuit breaker features work seamlessly
- ✅ **Backward Compatible** - Traditional `Get()`, `Post()`, `Do()` methods still work
- ✅ **Custom Unmarshalers** - Support for XML, Protocol Buffers, or custom formats

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

### HTTP Cache Semantics

klayengo supports intelligent HTTP caching with ETag, Cache-Control, and Stale-While-Revalidate (SWR):

```go
// HTTP semantics mode - honors Cache-Control, ETag, Last-Modified
cache := klayengo.NewInMemoryCache()
provider := klayengo.NewHTTPSemanticsCacheProvider(cache, 15*time.Minute, klayengo.HTTPSemantics)

client := klayengo.New(
    klayengo.WithCacheProvider(provider),
    klayengo.WithCacheMode(klayengo.HTTPSemantics),
)

// SWR mode - serves stale responses immediately, revalidates in background
swrProvider := klayengo.NewHTTPSemanticsCacheProvider(cache, 15*time.Minute, klayengo.SWR)
swrClient := klayengo.New(
    klayengo.WithCacheProvider(swrProvider),
    klayengo.WithCacheMode(klayengo.SWR),
)
```

### Cache Modes

| Mode | Description | Benefits |
|------|-------------|----------|
| **TTLOnly** | Traditional fixed TTL caching | Simple, backwards compatible |
| **HTTPSemantics** | Respects Cache-Control, ETag, Last-Modified | Efficient conditional requests, correct cache behavior |
| **SWR** | Stale-While-Revalidate mode | Instant responses, background refresh |

**HTTPSemantics Features:**
- Parses `Cache-Control` directives (max-age, no-cache, no-store, must-revalidate)
- Handles `ETag` and `If-None-Match` for efficient updates
- Supports `Last-Modified` and `If-Modified-Since` headers
- Respects `304 Not Modified` responses
- Single-flight protection prevents cache stampedes

**SWR Benefits:**
- Serves stale responses immediately (no latency penalty)
- Automatically refreshes cache in background
- Configurable stale window via `stale-while-revalidate` directive
- Ideal for high-traffic, latency-sensitive applications

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

## 🚨 Error Handling

klayengo provides typed errors and consistent error wrapping for better observability and easier error handling.

### Sentinel Errors

Use sentinel errors to check for specific failure conditions:

```go
resp, err := client.Get(ctx, "https://api.example.com/data")
if err != nil {
    // Check for specific error types
    if errors.Is(err, klayengo.ErrCircuitOpen) {
        log.Println("Circuit breaker is open - service unavailable")
        return
    }
    
    if errors.Is(err, klayengo.ErrRateLimited) {
        log.Println("Rate limited - backing off")
        time.Sleep(time.Second)
        return
    }
    
    if errors.Is(err, klayengo.ErrRetryBudgetExceeded) {
        log.Println("Retry budget exhausted - failing fast")
        return
    }
    
    // Extract detailed error information
    var clientErr *klayengo.ClientError
    if errors.As(err, &clientErr) {
        log.Printf("Request failed: %s (attempt %d/%d)", 
            clientErr.Message, clientErr.Attempt, clientErr.MaxRetries)
        
        // Print detailed debug information
        fmt.Println(clientErr.DebugInfo())
    }
}
```

### Transient Error Detection

Use `IsTransient()` to determine if an error is worth retrying:

```go
resp, err := client.Get(ctx, "https://api.example.com/data")
if err != nil {
    if klayengo.IsTransient(err) {
        log.Println("Transient error - will be retried automatically")
        // Network errors, timeouts, 5xx responses, 429 rate limiting
    } else {
        log.Println("Permanent error - won't retry")
        // 4xx client errors (except 429), configuration errors
        return err
    }
}
```

### Error Wrapping Chain

All errors maintain proper wrapping chains for root cause analysis:

```go
resp, err := client.Get(ctx, "https://api.example.com/data")
if err != nil {
    // Walk through the error chain
    fmt.Printf("Error chain: %+v\n", err)
    
    // Find root cause
    cause := err
    for errors.Unwrap(cause) != nil {
        cause = errors.Unwrap(cause)
    }
    fmt.Printf("Root cause: %v\n", cause)
}
```

### Available Sentinel Errors

- `ErrCircuitOpen` - Circuit breaker is in open state
- `ErrRateLimited` - Request denied due to rate limiting  
- `ErrCacheMiss` - Cache lookup failed (for custom cache implementations)
- `ErrRetryBudgetExceeded` - Retry budget exhausted

## ⚡ Performance

### HTTP Cache Semantics Benchmarks

The new HTTP cache semantics provide significant performance improvements:

```
BenchmarkCacheModes/TTLOnly-4         1,219,240 ops   985.5 ns/op  (99.99%+ cache hit rate)
BenchmarkCacheModes/HTTPSemantics-4     671,010 ops  1636 ns/op   (99.99%+ cache hit rate) 
BenchmarkCacheModes/SWR-4               695,712 ops  1636 ns/op   (99.99%+ cache hit rate)

BenchmarkSingleFlight/WithoutSingleFlight-4    447 ops  2.69ms/op   (447 server calls)
BenchmarkSingleFlight/WithSingleFlight-4   1,380,601 ops   864.6 ns/op  (99.99%+ reduction)

BenchmarkHeaderParsing/ParseCacheControl-4  6,541,195 ops  194.9 ns/op
BenchmarkHeaderParsing/ParseExpires-4       3,034,178 ops  374.4 ns/op
```

**Key Benefits:**
- **99.99%+ cache hit rates** with proper HTTP cache headers
- **99.99% reduction in server calls** with single-flight protection
- **Sub-microsecond header parsing** for Cache-Control and Expires
- **Stale-While-Revalidate** serves responses instantly while updating cache

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
| `WithCache(ttl)` | Response caching (TTL-based) | None |
| `WithCacheProvider(provider)` | HTTP cache provider with semantics | None |
| `WithCacheMode(mode)` | Cache mode (TTLOnly/HTTPSemantics/SWR) | TTLOnly |
| `WithCircuitBreaker(config)` | Circuit breaker | Disabled |
| `WithRetryPolicy(policy)` | Custom retry policy (overrides legacy knobs) | None |
| `WithRetryBudget(max, window)` | Cap retries per time window | None |
| `WithDeduplication()` | Request deduplication | Disabled |
| `WithMetrics()` | Prometheus metrics | Disabled |
| `WithMiddleware(...)` | Custom middleware | None |
| `WithDebug()` | Debug logging | Disabled |
| `WithUnmarshaler(unmarshaler)` | Custom response unmarshaler for typed responses | JSON |

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

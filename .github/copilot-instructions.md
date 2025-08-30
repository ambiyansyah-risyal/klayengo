# Klayengo AI Coding Instructions

## Project Overview
Klayengo is a resilient HTTP client wrapper for Go that provides retry logic, exponential backoff, circuit breaker pattern, caching, rate limiting, and Prometheus metrics. It's designed as a library package for building fault-tolerant HTTP clients.

## Core Architecture Patterns

### 1. Functional Options Pattern
All client configuration uses functional options. **Always use `WithXXX()` functions** instead of direct field access:

```go
// ✅ Correct - Use functional options
client := klayengo.New(
    klayengo.WithMaxRetries(5),
    klayengo.WithInitialBackoff(100*time.Millisecond),
    klayengo.WithCache(5*time.Minute),
)

// ❌ Incorrect - Never modify fields directly
client.maxRetries = 5
```

### 2. Middleware Chain Pattern
Use the `RoundTripper` interface for composable middleware. Middleware wraps requests in reverse order:

```go
client := klayengo.New(
    klayengo.WithMiddleware(
        loggingMiddleware,
        metricsMiddleware,
        authMiddleware, // Applied first
    ),
)
```

### 3. Context-Based Per-Request Overrides
Override global settings per request using context:

```go
// Disable caching for specific request
ctx := klayengo.WithContextCacheDisabled(context.Background())
resp, err := client.Get(ctx, url)

// Custom TTL for specific request
ctx := klayengo.WithContextCacheTTL(context.Background(), 30*time.Minute)
resp, err := client.Get(ctx, url)
```

### 4. Interface-Based Design
Use interfaces for testability and extensibility:

```go
// Cache interface allows custom implementations
type Cache interface {
    Get(key string) (*CacheEntry, bool)
    Set(key string, entry *CacheEntry, ttl time.Duration)
    Delete(key string)
    Clear()
}
```

## Key Files and Their Purposes

- **`klayengo.go`**: Core client implementation with retry logic, circuit breaker, caching
- **`options.go`**: Functional options for client configuration
- **`metrics.go`**: Prometheus metrics collection and reporting
- **`klayengo_test.go`**: Comprehensive unit tests with httptest servers and benchmarks
- **`examples/`**: Usage examples (basic, advanced, metrics integration)

## Critical Developer Workflows

### Testing Patterns
Always use `httptest.Server` for HTTP client testing:

```go
func TestClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    }))
    defer server.Close()

    client := klayengo.New()
    resp, err := client.Get(context.Background(), server.URL)
    // ... assertions
}
```

### Error Handling
Use structured error types and proper error checking:

```go
resp, err := client.Get(ctx, url)
if err != nil {
    var clientErr *klayengo.ClientError
    if errors.As(err, &clientErr) {
        switch clientErr.Type {
        case "RateLimit":
            // Handle rate limit
        case "CircuitBreaker":
            // Handle circuit breaker open
        }
    }
    return err
}
```

### Metrics Integration
Enable Prometheus metrics for observability:

```go
client := klayengo.New(klayengo.WithMetrics())

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":8080", nil)
```

## Project-Specific Conventions

### 1. Configuration Naming
- `WithXXX()` for all configuration options
- `WithContextXXX()` for context-based overrides
- Consistent parameter naming (e.g., `maxTokens int, refillRate time.Duration`)

### 2. Cache Key Generation
Default cache keys combine method and URL. Customize for specific needs:

```go
client := klayengo.New(
    klayengo.WithCacheKeyFunc(func(req *http.Request) string {
        return fmt.Sprintf("%s:%s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
    }),
)
```

### 3. Retry Conditions
Default retries on network errors and 5xx status codes. Customize for specific APIs:

```go
client := klayengo.New(
    klayengo.WithRetryCondition(func(resp *http.Response, err error) bool {
        if err != nil {
            return true // Retry on network errors
        }
        return resp.StatusCode >= 500 // Retry on server errors
    }),
)
```

### 4. Circuit Breaker Configuration
Use sensible defaults and adjust based on service characteristics:

```go
client := klayengo.New(
    klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
        FailureThreshold: 5,        // Open after 5 failures
        RecoveryTimeout:  60 * time.Second, // Wait 60s before testing
        SuccessThreshold: 2,        // Require 2 successes to close
    }),
)
```

## Common Patterns and Anti-Patterns

### ✅ Recommended Patterns

1. **Always handle context cancellation**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
resp, err := client.Get(ctx, url)
```

2. **Use cache conditions for selective caching**:
```go
client := klayengo.New(
    klayengo.WithCacheCondition(func(req *http.Request) bool {
        return req.Method == "GET" && req.URL.Path == "/api/data"
    }),
)
```

3. **Configure rate limiting based on API limits**:
```go
client := klayengo.New(
    klayengo.WithRateLimiter(100, 1*time.Minute), // 100 requests per minute
)
```

### ❌ Anti-Patterns to Avoid

1. **Don't ignore circuit breaker errors** - they're intentional failures
2. **Don't use very short timeouts** without retries - defeats resilience purpose
3. **Don't cache POST/PUT/DELETE requests** without careful consideration
4. **Don't modify cached response bodies** - they're read-only

## Performance Considerations

### 1. Connection Reuse
The underlying `http.Client` handles connection pooling automatically. Configure when needed:

```go
customClient := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}
client := klayengo.New(klayengo.WithHTTPClient(customClient))
```

### 2. Cache TTL Strategy
Balance cache freshness with performance. Use shorter TTLs for volatile data:

```go
// Short TTL for frequently changing data
client := klayengo.New(klayengo.WithCache(30*time.Second))

// Long TTL for static configuration
client := klayengo.New(klayengo.WithCache(24*time.Hour))
```

### 3. Rate Limiting
Configure based on API rate limits, not arbitrary values:

```go
// GitHub API: 5000 requests per hour
client := klayengo.New(klayengo.WithRateLimiter(5000, 1*time.Hour))
```

## Debugging and Monitoring

### 1. Enable Detailed Logging
Use middleware for request/response logging:

```go
loggingMiddleware := func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
    start := time.Now()
    log.Printf("Starting %s %s", req.Method, req.URL)

    resp, err := next.RoundTrip(req)

    duration := time.Since(start)
    if err != nil {
        log.Printf("Request failed after %v: %v", duration, err)
    } else {
        log.Printf("Request completed in %v with status %s", duration, resp.Status)
    }

    return resp, err
}
```

### 2. Monitor Circuit Breaker State
Circuit breaker state is exposed via metrics. Monitor for service health.

### 3. Cache Hit/Miss Analysis
Use cache metrics to optimize cache strategies and TTL values.

## Testing Best Practices

### 1. Use httptest for Isolation
Always test against `httptest.Server` to avoid external dependencies:

```go
func TestCircuitBreaker(t *testing.T) {
    callCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        if callCount <= 3 {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    client := klayengo.New(
        klayengo.WithMaxRetries(2),
        klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
            FailureThreshold: 3,
            RecoveryTimeout:  100 * time.Millisecond,
            SuccessThreshold: 1,
        }),
    )

    // First 3 requests should fail and open circuit
    for i := 0; i < 3; i++ {
        _, err := client.Get(context.Background(), server.URL)
        assert.Error(t, err)
    }

    // Circuit should be open, requests should fail fast
    _, err := client.Get(context.Background(), server.URL)
    assert.Contains(t, err.Error(), "circuit breaker is open")
}
```

### 2. Test Concurrent Access
Klayengo is thread-safe. Test concurrent usage:

```go
func TestConcurrentRequests(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(10 * time.Millisecond) // Simulate network delay
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    client := klayengo.New()
    var wg sync.WaitGroup
    errors := make(chan error, 100)

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := client.Get(context.Background(), server.URL)
            if err != nil {
                errors <- err
            }
        }()
    }

    wg.Wait()
    close(errors)

    // Check for any errors
    for err := range errors {
        t.Errorf("Concurrent request failed: %v", err)
    }
}
```

### 3. Benchmark Performance
Use Go's benchmarking tools to measure performance impact:

```go
func BenchmarkClientWithRetries(b *testing.B) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    client := klayengo.New(klayengo.WithMaxRetries(3))

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            resp, err := client.Get(context.Background(), server.URL)
            if err != nil {
                b.Fatal(err)
            }
            resp.Body.Close()
        }
    })
}
```

## Integration Examples

### With Popular Go Libraries

**Using with go-retryablehttp** (if needed for comparison):
```go
// Klayengo provides similar functionality with more features
client := klayengo.New(
    klayengo.WithMaxRetries(3),
    klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{...}),
    klayengo.WithMetrics(),
)
```

**Using with Prometheus**:
```go
registry := prometheus.NewRegistry()
collector := klayengo.NewMetricsCollectorWithRegistry(registry)
client := klayengo.New(klayengo.WithMetricsCollector(collector))

// Expose custom registry
http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
```

**Using with custom HTTP client**:
```go
transport := &http.Transport{
    TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // For testing
}
httpClient := &http.Client{Transport: transport}

client := klayengo.New(klayengo.WithHTTPClient(httpClient))
```

## Troubleshooting Common Issues

### 1. Circuit Breaker Not Opening
- Check failure threshold configuration
- Verify retry condition is triggering failures
- Monitor circuit breaker metrics

### 2. Cache Not Working
- Ensure cache is enabled with `WithCache()`
- Check cache condition function
- Verify cache key generation

### 3. Rate Limiting Too Aggressive
- Adjust refill rate or max tokens
- Consider API-specific rate limits
- Monitor rate limiter metrics

### 4. High Memory Usage
- Check cache size metrics
- Implement cache size limits
- Use shorter TTLs for cached responses

## Code Review Checklist

- [ ] Uses functional options pattern for configuration
- [ ] Handles context cancellation properly
- [ ] Includes appropriate error handling
- [ ] Uses httptest.Server for testing
- [ ] Includes benchmarks for performance-critical code
- [ ] Follows middleware pattern for cross-cutting concerns
- [ ] Enables metrics for observability
- [ ] Configures circuit breaker appropriately
- [ ] Uses cache conditions for selective caching
- [ ] Includes comprehensive test coverage</content>
<parameter name="filePath">/home/risyal/project/klayengo/.github/copilot-instructions.md

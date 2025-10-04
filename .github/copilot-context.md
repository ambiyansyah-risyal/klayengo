# Technical Context for klayengo

## Architecture Deep Dive

### Client Structure
The `Client` struct is the central component that composes all reliability patterns:

```go
type Client struct {
    // Core HTTP
    httpClient *http.Client
    transport  http.RoundTripper
    
    // Retry configuration  
    maxRetries         int
    initialBackoff     time.Duration
    maxBackoff         time.Duration
    backoffMultiplier  float64
    jitter             float64
    backoffStrategy    BackoffStrategy
    retryCondition     RetryCondition
    retryPolicy        RetryPolicy
    
    // Rate limiting
    rateLimiter        *RateLimiter
    limiterRegistry    *RateLimiterRegistry
    
    // Caching
    cache              Cache
    cacheTTL           time.Duration
    cacheCondition     CacheCondition
    
    // Circuit breaker
    circuitBreaker     *CircuitBreaker
    
    // Deduplication
    deduplication      *Deduplication
    
    // Middleware & Observability
    middleware         []Middleware
    metrics            *MetricsCollector
    logger             Logger
    debug              *DebugConfig
}
```

### Request Flow
1. **Middleware Chain**: Custom middleware applied in order
2. **Deduplication**: Check for identical in-flight requests
3. **Cache Check**: Attempt cache retrieval (if enabled)
4. **Rate Limiting**: Token bucket or per-key limiting  
5. **Circuit Breaker**: Check state (closed/open/half-open)
6. **HTTP Request**: Actual network call with timeout
7. **Retry Logic**: Exponential backoff with jitter on failures
8. **Response Processing**: Cache storage, metrics recording

### Key Patterns

#### Functional Options
All configuration uses the functional options pattern:
- Options are functions that modify a `*Client`
- Applied during `New()` construction
- Enables clean, extensible API
- Supports validation during construction

#### Lock-Free Concurrency
Critical paths use atomic operations:
- Circuit breaker state (atomic int64)
- Metrics counters (sync/atomic)
- Rate limiter tokens (atomic operations)
- Minimizes contention in high-concurrency scenarios

#### Middleware Composition
Middleware follows HTTP handler pattern:
```go
type Middleware func(req *http.Request, next RoundTripper) (*http.Response, error)
type RoundTripper interface {
    RoundTrip(*http.Request) (*http.Response, error)
}
```

## Component Details

### Retry Policy
Two backoff strategies:
- **ExponentialJitter**: Standard exponential backoff with uniform jitter
- **DecorrelatedJitter**: Smoother distribution, better for many concurrent clients

Formula for ExponentialJitter:
```
backoff = min(maxBackoff, initialBackoff * multiplier^attempt)
jittered = backoff * (1 + jitter * (2*random() - 1))
```

### Circuit Breaker
Three states implemented as atomic state machine:
- **Closed**: Normal operation, count failures
- **Open**: Fail fast, occasional health check
- **Half-Open**: Limited requests to test recovery

State transitions based on:
- `FailureThreshold`: Failures before opening
- `RecoveryTimeout`: Time before trying half-open
- `SuccessThreshold`: Successes needed to close

### Rate Limiter
Token bucket algorithm with:
- Fixed capacity (`maxTokens`)
- Periodic refill (`refillRate`)
- Per-key limiting via registry
- Atomic operations for thread safety

### Caching
In-memory cache with TTL:
- LRU eviction policy
- Per-request cache control overrides
- Cache key includes method + URL + headers
- Response body stored separately for streaming

### Deduplication
Merges concurrent identical requests:
- Request signature: method + URL + headers + body hash
- In-flight request tracking with sync.Map
- Response shared across waiting goroutines
- Reduces backend load for duplicate requests

## Performance Characteristics

### Memory Allocation
Hot paths minimize allocations:
- Reuse HTTP requests where possible
- Object pooling for frequently created objects
- Atomic counters instead of mutex-protected fields
- Efficient cache key generation

### Concurrency Model
- Single `*Client` instance safe for concurrent use
- Lock-free operations on critical paths
- Separate locks for different subsystems
- Rate limiter uses atomic operations
- Circuit breaker uses atomic state machine

## Error Handling Strategy

### ClientError Type
Wraps all errors with rich context:
```go
type ClientError struct {
    Type       string    // Error category
    Message    string    // Human-readable message
    Cause      error     // Original error
    RequestID  string    // Request identifier
    Method     string    // HTTP method
    URL        string    // Request URL
    Attempt    int       // Retry attempt number
    MaxRetries int       // Maximum retries allowed
    Timestamp  time.Time // Error occurrence time
    Duration   time.Duration // Request duration
    StatusCode int       // HTTP status code (if available)
    Endpoint   string    // Endpoint identifier
}
```

### Error Categories
- `NetworkError`: Connection failures, timeouts
- `HTTPError`: HTTP-level errors (4xx, 5xx)
- `RateLimitError`: Rate limiting triggered
- `CircuitBreakerError`: Circuit breaker open
- `ValidationError`: Configuration issues
- `CacheError`: Cache operation failures

## Metrics and Observability

### Prometheus Metrics
All metrics use `klayengo_` prefix:
- `klayengo_requests_total`: Request counter with labels
- `klayengo_request_duration_seconds`: Request histogram
- `klayengo_errors_total`: Error counter by type
- `klayengo_cache_operations_total`: Cache hit/miss counters
- `klayengo_rate_limit_events_total`: Rate limiting events
- `klayengo_circuit_breaker_state`: Circuit breaker state gauge

### Debug Configuration
Structured logging with configurable verbosity:
```go
type DebugConfig struct {
    Enabled      bool
    LogRequests  bool   // Log request/response details
    LogRetries   bool   // Log retry attempts
    LogCache     bool   // Log cache operations
    LogRateLimit bool   // Log rate limiting
    LogCircuit   bool   // Log circuit breaker events
    RequestIDGen func() string  // Custom request ID generator
}
```

## Configuration Validation

### Safety Checks
Validation prevents resource exhaustion:
- Maximum retries capped at 100
- Timeouts limited to reasonable ranges
- Rate limiter tokens capped at 1M
- Backoff durations validated

### Extreme Value Warnings
Configuration that might cause issues:
- Very high retry counts
- Excessive timeouts  
- Very short rate limiter intervals
- Large cache TTLs

## Testing Strategy

### Test Categories
- **Unit Tests**: Individual component behavior
- **Integration Tests**: Component interaction
- **Race Tests**: Concurrent access patterns
- **Benchmark Tests**: Performance validation
- **Property Tests**: Edge case generation

### Mock Strategy
- Mock external HTTP services
- Controllable time for timeout testing
- Deterministic randomness for jitter testing
- Configurable failure injection

## Extension Points

### Custom Middleware
Implement cross-cutting concerns:
```go
func AuthMiddleware(token string) Middleware {
    return func(req *http.Request, next RoundTripper) (*http.Response, error) {
        req.Header.Set("Authorization", "Bearer "+token)
        return next.RoundTrip(req)
    }
}
```

### Custom Cache
Implement `Cache` interface for external stores:
```go
type Cache interface {
    Get(key string) (*CacheEntry, bool)
    Set(key string, entry *CacheEntry, ttl time.Duration)
    Delete(key string)
    Clear()
}
```

### Custom Retry Policy
Implement `RetryPolicy` interface for complex retry logic:
```go
type RetryPolicy interface {
    ShouldRetry(attempt int, resp *http.Response, err error) bool
    GetDelay(attempt int) time.Duration
}
```

This architecture enables high performance, reliability, and extensibility while maintaining clean separation of concerns.
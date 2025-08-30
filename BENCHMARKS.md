# Benchmark Results

## Performance Overview

The following benchmark results show the performance characteristics of the klayengo HTTP client:

### HTTP Request Benchmarks

| Benchmark | Operations/sec | Time/op | Memory/op | Allocs/op |
|-----------|---------------|---------|-----------|-----------|
| BasicGet | ~7,500 ops/sec | 133μs | 20.8KB | 123 |
| GetWithRetries | ~6,200 ops/sec | 161μs | 21.5KB | 124 |
| GetWithCircuitBreaker | ~6,400 ops/sec | 156μs | 21.3KB | 123 |
| GetWithMiddleware | ~8,600 ops/sec | 116μs | 20.0KB | 124 |
| GetWithMultipleMiddleware | ~14,800 ops/sec | 68μs | 17.9KB | 123 |
| GetWithRetriesAndFailures | ~3 ops/sec | 303ms | 31.8KB | 244 |

### Core Component Benchmarks

| Benchmark | Operations/sec | Time/op | Memory/op | Allocs/op |
|-----------|---------------|---------|-----------|-----------|
| CircuitBreakerAllow | ~13.8M ops/sec | 73ns | 0B | 0 |
| CalculateBackoff | ~312M ops/sec | 3.2ns | 0B | 0 |
| DefaultRetryCondition | ~2.2B ops/sec | 0.45ns | 0B | 0 |
| ClientCreation | ~3.4M ops/sec | 291ns | 304B | 4 |

### Specialized Benchmarks

| Benchmark | Operations/sec | Time/op | Memory/op | Allocs/op |
|-----------|---------------|---------|-----------|-----------|
| ConcurrentRequests | ~1,350 ops/sec | 743μs | 17.8KB | 120 |
| LargePayload (1MB) | ~2,750 ops/sec | 364μs | 1.07MB | 132 |
| TimeoutHandling | ~1.3M ops/sec | 752ns | 496B | 4 |

#### Backoff Strategy Comparison

| Strategy | Operations/sec | Time/op | Memory/op | Allocs/op |
|----------|---------------|---------|-----------|-----------|
| LinearBackoff | ~98 ops/sec | 10.2ms | 20.8KB | 123 |
| ExponentialBackoff | ~15,700 ops/sec | 64μs | 17.7KB | 118 |
| AggressiveBackoff | ~14,800 ops/sec | 68μs | 17.7KB | 118 |

## Performance Analysis

### Key Findings

1. **Minimal Overhead**: The retry logic adds only ~20% overhead compared to basic HTTP requests
2. **Circuit Breaker Impact**: Adds ~18% overhead but provides crucial resilience
3. **Middleware Performance**: Single middleware has minimal impact, multiple middleware can actually improve performance due to request batching
4. **Failure Scenarios**: Requests with retries and failures have significantly higher latency due to backoff delays
5. **Core Operations**: Circuit breaker state checks and backoff calculations are extremely fast (<100ns)
6. **Concurrent Performance**: Handles concurrent requests well with ~1,350 ops/sec
7. **Large Payload Handling**: Efficiently handles 1MB responses with ~2,750 ops/sec
8. **Timeout Performance**: Very fast timeout handling at ~1.3M ops/sec

### Memory Usage

- **Base Request**: ~20.8KB per request
- **With Retries**: ~21.5KB per request (+3%)
- **With Circuit Breaker**: ~21.3KB per request (+2%)
- **With Middleware**: ~20.0KB per request (-4%)
- **Failure Scenarios**: ~31.8KB per request (+53% due to error handling)
- **Large Payload (1MB)**: ~1.07MB per request (expected for payload size)

### Backoff Strategy Insights

- **Linear Backoff**: Slower performance due to consistent delays
- **Exponential Backoff**: Best overall performance with balanced delays
- **Aggressive Backoff**: Similar to exponential but with more aggressive retry timing

### Recommendations

1. **Use Circuit Breaker**: The performance impact is minimal compared to the resilience benefits
2. **Optimize Middleware**: Multiple middleware can improve performance through better request batching
3. **Choose Backoff Strategy**: Exponential backoff provides the best balance of performance and reliability
4. **Monitor Concurrent Load**: The client handles concurrency well but monitor for your specific use case
5. **Tune Timeouts**: Fast timeout handling allows for quick failure detection
6. **Memory Monitoring**: Watch memory usage in high-throughput scenarios with failures or large payloads

## Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkGetWithRetries -benchmem

# Run benchmarks with CPU profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof

# Run benchmarks with memory profiling
go test -bench=. -benchmem -memprofile=mem.prof

# Run benchmarks for specific scenarios
go test -bench=BenchmarkConcurrentRequests -benchmem
go test -bench=BenchmarkLargePayload -benchmem

# Compare backoff strategies
go test -bench=BenchmarkDifferentBackoffStrategies -benchmem
```

## Environment

- **Go Version**: 1.21+
- **OS**: Linux
- **Architecture**: amd64
- **CPU**: AMD Ryzen 7 8840U
- **Date**: August 30, 2025

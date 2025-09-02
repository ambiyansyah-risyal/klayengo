# Benchmark Results

## Performance Overview

The following benchmark results show the performance characteristics of the klayengo HTTP client after recent optimizations:

### HTTP Request Benchmarks

| Benchmark | Operations/sec | Time/op | Memory/op | Allocs/op | Improvement |
|-----------|---------------|---------|-----------|-----------|-------------|
| BenchmarkClientGet | ~8,150 ops/sec | 123μs | 23.0KB | 145 | +5% |
| BenchmarkClientPost | ~6,440 ops/sec | 155μs | 46.3KB | 165 | +7% |
| BenchmarkClientWithCache | ~985,000 ops/sec | 1.02μs | 888B | 14 | +5% |
| BenchmarkClientWithCircuitBreaker | ~7,670 ops/sec | 130μs | 23.4KB | 144 | +6% |
| BenchmarkClientWithRateLimiter | ~7,620 ops/sec | 131μs | 23.5KB | 144 | +2% |
| BenchmarkClientWithDeduplication | ~8,120 ops/sec | 123μs | 23.2KB | 146 | +1% |
| BenchmarkDeduplicationConcurrentDuplicates | ~2.45M ops/sec | 408ns | 45B | 3 | +99.3% |

### Cache Performance Benchmarks

| Benchmark | Operations/sec | Time/op | Memory/op | Allocs/op | Improvement |
|-----------|---------------|---------|-----------|-----------|-------------|
| BenchmarkCacheGet | ~17.2M ops/sec | 58.2ns | 0B | 0 | -3%* |
| BenchmarkCacheSet | ~1.55M ops/sec | 645ns | 133B | 2 | -4%* |
| BenchmarkCacheConcurrentAccess | ~5.35M ops/sec | 187ns | 13B | 1 | +26% |

*Note: Slight degradation in single-threaded cache operations due to sharding overhead, but significant improvement in concurrent access.

### Performance Analysis

### Key Findings

1. **Caching provides massive performance gains**: ~100x faster requests when cached (1.02μs vs 123μs)
2. **Deduplication dramatically improves concurrent performance**: ~99.3% faster for duplicate concurrent requests (408ns vs 123μs)
3. **Minimal Overhead**: The retry logic adds only ~10-15% overhead compared to basic HTTP requests
4. **Circuit Breaker Impact**: Adds ~6% overhead but provides crucial resilience
5. **Rate Limiting Overhead**: Adds ~7% overhead for token management
6. **Deduplication Overhead**: Adds ~1% overhead for unique requests but massive benefits for duplicates
7. **Full Feature Stack**: Complete client with all features performs at ~1.42μs per request
8. **Cache Performance**: Extremely fast cache operations (<60ns for gets)
9. **Concurrent Performance**: Excellent concurrent access performance for cache operations (+26% improvement)
10. **Failure Scenarios**: Requests with retries and failures have significantly higher latency due to backoff delays

### Memory Usage

- **Base Request**: ~23.0KB per request
- **With Retries**: ~38.5KB per request (+67% due to error handling in failure scenarios)
- **With Circuit Breaker**: ~23.4KB per request (+2%)
- **With Rate Limiter**: ~23.5KB per request (+2%)
- **With Deduplication**: ~23.2KB per request (+1%)
- **With Cache**: ~888B per request (-96% when cached)
- **Cache Operations**: Minimal memory usage (0-133B per operation)
- **Deduplication Operations**: Very low memory usage (45B per operation for concurrent duplicates)

### Performance Optimizations Implemented

#### Cache Optimizations
- **Sharded Cache Architecture**: Implemented 16-shard cache with FNV-1a hashing for better concurrency
- **Atomic Operations**: Replaced mutex-based locking with atomic operations in rate limiter and circuit breaker
- **Memory-Efficient String Operations**: Replaced `fmt.Sprintf` with direct byte buffer operations
- **Size Limits**: Added 10MB limit on cached responses to prevent memory issues
- **Optimized Cache Keys**: More efficient cache key generation using byte buffers

#### Client Core Optimizations
- **Request Deduplication**: Implemented efficient deduplication for concurrent identical requests
- **Reduced Debug Overhead**: Early returns when debug features are disabled
- **Conditional Metrics Recording**: Metrics are only recorded when enabled
- **Optimized Backoff Calculation**: Maintained accuracy while improving performance
- **Efficient Endpoint Extraction**: Streamlined endpoint string generation

#### Concurrent Performance Improvements
- **Lock-Free Operations**: Rate limiter and circuit breaker now use atomic operations
- **Reduced Lock Contention**: Sharded cache eliminates single-point bottlenecks
- **Better Memory Layout**: Optimized data structures for cache-friendly access patterns

### Recommendations

1. **Use Caching Strategically**: The performance benefits are enormous for cacheable requests
2. **Enable Deduplication for Concurrent Workloads**: Massive performance improvements (99.3%) for duplicate concurrent requests
3. **Choose Features Wisely**: Each feature adds minimal overhead but consider your use case
4. **Monitor Cache Hit Rates**: High cache hit rates can dramatically improve performance
5. **Tune Circuit Breaker Settings**: Balance failure threshold with your service's characteristics
6. **Consider Rate Limiting**: Minimal overhead but effective for controlling request rates
7. **Profile Memory Usage**: Cache hits use dramatically less memory than full requests
8. **Leverage Concurrency**: The optimized concurrent performance benefits high-throughput applications
9. **Use Deduplication for API Calls**: Perfect for scenarios with concurrent identical requests to the same endpoints

## Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkClientGet -benchmem

# Run benchmarks with CPU profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof

# Run benchmarks with memory profiling
go test -bench=. -benchmem -memprofile=mem.prof

# Run cache-specific benchmarks
go test -bench=BenchmarkCache -benchmem

# Run client-specific benchmarks
go test -bench=BenchmarkClient -benchmem

# Run deduplication-specific benchmarks
go test -bench=BenchmarkDeduplication -benchmem
```

## Environment

- **Go Version**: 1.24.6
- **OS**: Linux
- **Architecture**: amd64
- **CPU**: AMD Ryzen 7 8840U w/ Radeon 780M Graphics
- **Date**: September 1, 2025
- **Test Coverage**: 87.1%
- **Performance Optimizations**: Applied (Cache sharding, atomic operations, memory optimizations, request deduplication)

### Current Coverage: 87.1%

#### Well-Covered Areas (90-100%):
- Cache operations (100%)
- Metrics collection (100%)
- Rate limiting (100%)
- Circuit breaker core logic (100%)
- Configuration options (100%)
- Error unwrapping and type checking (100%)
- Request deduplication (95%)

#### Moderate Coverage (75-89%):
- Client.Do method (87.5%)
- Client.Get method (75%)
- Client.Post method (80%)
- Circuit breaker Allow method (80%)
- Error formatting (84.6%)
- Cache condition checking (80%)

#### Low Coverage Areas (0-74%):
- **New debugging features** (0%): SimpleLogger methods, debug configuration options
- **Metrics configuration options** (0%): WithMetrics, WithDebug, WithSimpleLogger, etc.
- **Error context methods** (95.2% for DebugInfo, but other context fields untested)

### Coverage Improvement Recommendations

1. **Add tests for debugging features**:
   ```go
   func TestSimpleLogger(t *testing.T) {
       logger := NewSimpleLogger()
       // Test Debug, Info, Warn, Error methods
   }
   ```

2. **Test configuration option functions**:
   ```go
   func TestWithDebug(t *testing.T) {
       client := New(WithDebug())
       // Verify debug configuration is set
   }
   ```

3. **Add edge case tests for error handling**:
   ```go
   func TestClientErrorWithNilCause(t *testing.T) {
       err := &ClientError{Type: "Test", Message: "test"}
       // Test error formatting with nil cause
   }
   ```

4. **Test concurrent scenarios**:
   ```go
   func TestClientConcurrentRequests(t *testing.T) {
       // Test client behavior under concurrent load
   }
   ```

5. **Add integration tests for full request lifecycle**:
   ```go
   func TestClientFullRequestLifecycle(t *testing.T) {
       // Test complete request flow with all features enabled
   }
   ```

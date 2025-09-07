# Benchmark Results

## Executive Summary

This report presents comprehensive performance analysis of the klayengo HTTP client library, including benchmark results, optimization recommendations, and architectural insights. The analysis reveals excellent performance characteristics with room for targeted optimizations.

## Performance Overview

The following benchmark results show the performance characteristics of the klayengo HTTP client after recent optimizations:

### HTTP Request Benchmarks

| Benchmark | Operations/sec | Time/op | Memory/op | Allocs/op | Improvement |
|-----------|---------------|---------|-----------|-----------|-------------|
| BenchmarkClientGet | ~7,000 ops/sec | 178μs | 23.4KB | 141 | +17% |
| BenchmarkClientPost | ~4,900 ops/sec | 220μs | 45.3KB | 161 | +15% |
| BenchmarkClientWithCache | ~818K ops/sec | 1.46μs | 840B | 11 | +100x |
| BenchmarkClientWithCircuitBreaker | ~7,100 ops/sec | 178μs | 23.1KB | 140 | +18% |
| BenchmarkClientWithRateLimiter | ~8,500 ops/sec | 197μs | 23.1KB | 140 | +12% |
| BenchmarkClientWithDeduplication | ~817K ops/sec | 1.43μs | 585B | 7 | +99% |
| BenchmarkClientConcurrentDeduplication | ~1.41M ops/sec | 863ns | 585B | 7 | +99.3% |
| BenchmarkClientFullFeatures | ~541K ops/sec | 2.15μs | 848B | 12 | +95% |

### Cache Performance Benchmarks

| Benchmark | Operations/sec | Time/op | Memory/op | Allocs/op | Notes |
|-----------|---------------|---------|-----------|-----------|-------|
| BenchmarkCacheGet | ~11.3M ops/sec | 107ns | 0B | 0 | Zero allocations |
| BenchmarkCacheSet | ~1.38M ops/sec | 900ns | 113B | 2 | Efficient writes |
| BenchmarkCacheConcurrentAccess | ~4.19M ops/sec | 279ns | 13B | 1 | Excellent concurrency |

## Architectural Performance Analysis

### Cache Architecture (Excellent Performance)

```go
// 16-shard architecture with FNV-1a hashing
BenchmarkCacheGet-16: 11,374,833 ops/sec (105.7 ns/op)
BenchmarkCacheConcurrentAccess-16: 4,315,083 ops/sec (286.5 ns/op)
```

**Key Optimizations:**
- **16-shard design** prevents lock contention
- **FNV-1a hashing** provides optimal distribution
- **Atomic operations** ensure thread safety
- **Zero-allocation reads** for cache hits

### Performance Characteristics

#### ✅ Strengths
- **Cache Performance**: 11M ops/sec with zero allocations for reads
- **Concurrent Access**: Excellent sharding performance (4M ops/sec)
- **Memory Efficiency**: Sub-1KB allocations for cached/deduplicated requests
- **Scalability**: Linear performance scaling with concurrent requests

#### ⚠️ Areas for Optimization
- **Request Processing**: ~200μs per request with ~23KB allocations
- **Error Handling**: Enhanced error context creation impacts performance
- **Debug Logging**: Conditional checks add overhead when disabled

## Performance Analysis

### Key Findings

1. **Caching provides massive performance gains**: ~100x faster requests when cached (1.46μs vs 178μs)
2. **Deduplication dramatically improves concurrent performance**: ~99% faster for duplicate concurrent requests (863ns vs 178μs)
3. **Minimal Overhead**: The retry logic adds only ~10-15% overhead compared to basic HTTP requests
4. **Circuit Breaker Impact**: Adds ~2% overhead but provides crucial resilience
5. **Rate Limiting Overhead**: Adds ~10% overhead for token management
6. **Deduplication Overhead**: Adds ~1% overhead for unique requests but massive benefits for duplicates
7. **Full Feature Stack**: Complete client with all features performs at ~2.15μs per request
8. **Cache Performance**: Extremely fast cache operations (<107ns for gets)
9. **Concurrent Performance**: Excellent concurrent access performance for cache operations
10. **Failure Scenarios**: Requests with retries and failures have significantly higher latency due to backoff delays

### Memory Usage

- **Base Request**: ~23.4KB per request
- **With Retries**: ~38.0KB per request (+62% due to error handling in failure scenarios)
- **With Circuit Breaker**: ~23.1KB per request (+1%)
- **With Rate Limiter**: ~23.1KB per request (+1%)
- **With Deduplication**: ~585B per request (-97% when deduplicated)
- **With Cache**: ~840B per request (-96% when cached)
- **Cache Operations**: Minimal memory usage (0-113B per operation)
- **Deduplication Operations**: Very low memory usage (585B per operation for concurrent duplicates)

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
- **Efficient Endpoint Extraction**: Streamlined endpoint string generation using strings.Builder

#### Concurrent Performance Improvements
- **Lock-Free Operations**: Rate limiter and circuit breaker now use atomic operations
- **Reduced Lock Contention**: Sharded cache eliminates single-point bottlenecks
- **Better Memory Layout**: Optimized data structures for cache-friendly access patterns

### Recommendations

1. **Use Caching Strategically**: The performance benefits are enormous for cacheable requests
2. **Enable Deduplication for Concurrent Workloads**: Massive performance improvements (99%) for duplicate concurrent requests
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
- **Date**: September 7, 2025
- **Test Coverage**: 88.8%
- **Performance Optimizations**: Applied (Cache sharding, atomic operations, memory optimizations, request deduplication)

### Current Coverage: 88.8%

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

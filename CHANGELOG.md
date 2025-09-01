# Changelog

All notable changes to **Klayengo** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Request Deduplication**: Prevents duplicate concurrent requests by deduplicating identical in-flight requests
  - Automatic deduplication of concurrent identical requests
  - Configurable deduplication key generation
  - Custom deduplication conditions
  - Thread-safe implementation with minimal overhead
  - Performance improvement of 99.3% for duplicate concurrent requests
  - New metrics: `klayengo_deduplication_hits_total`
  - Comprehensive test coverage with integration tests and benchmarks
- Initial public release
- Comprehensive versioning system with build-time injection
- Makefile for automated builds and releases

### Changed
- Updated documentation with versioning guidelines

### Fixed
- Minor documentation improvements

## [1.0.0] - 2025-01-01

### Added
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
- **Enhanced Error Handling**: Structured error types with detailed debugging information
- **Comprehensive Debugging**: Request tracing, debug logging, and detailed error context
- **Performance Optimizations**: Sharded cache, atomic operations, and memory-efficient implementations
- **Security Features**: Rate limiting, input validation, and secure defaults
- **CI/CD Integration**: Automated testing, linting, and quality assurance

### Performance
- **16-shard cache architecture** with FNV-1a hashing (26% faster concurrent access)
- **Atomic operations** for rate limiter and circuit breaker (28-29% faster)
- **Optimized backoff calculations** with reduced computational overhead
- **Memory-efficient operations** with 14% reduction in allocations

### Security
- **Rate Limiting**: Prevents abuse and protects against DoS attacks
- **Circuit Breaker**: Provides resilience against cascading failures
- **Input Validation**: Validates configuration parameters
- **Secure Defaults**: Conservative default settings for security
- **Audit Logging**: Comprehensive logging for security monitoring

### Documentation
- **Comprehensive README** with usage examples and API reference
- **Security Policy** for responsible disclosure
- **Contributing Guidelines** for community contributions
- **Performance Benchmarks** documentation
- **Example Code** for basic and advanced usage patterns

### Testing
- **86.7% code coverage** with comprehensive unit tests
- **Integration tests** for end-to-end scenarios
- **Benchmark tests** for performance validation
- **Concurrent safety tests**
- **Error handling tests**

### CI/CD
- **GitHub Actions** for automated testing and quality assurance
- **Multi-version Go testing** (1.23, 1.24)
- **Automated linting** with golangci-lint
- **Security scanning** integration
- **Coverage reporting** with codecov

---

## Types of changes
- `Added` for new features
- `Changed` for changes in existing functionality
- `Deprecated` for soon-to-be removed features
- `Removed` for now removed features
- `Fixed` for any bug fixes
- `Security` in case of vulnerabilities

## Version Format
This project uses [Semantic Versioning](https://semver.org/):
- **MAJOR.MINOR.PATCH** (e.g., `1.2.3`)
- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

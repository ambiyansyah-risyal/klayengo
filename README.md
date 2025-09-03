# klayengo

[![Go Reference](https://pkg.go.dev/badge/github.com/ambiyansyah-risyal/klayengo)](https://pkg.go.dev/github.com/ambiyansyah-risyal/klayengo)
[![Go Report Card](https://goreportcard.com/badge/github.com/ambiyansyah-risyal/klayengo)](https://goreportcard.com/report/github.com/ambiyansyah-risyal/klayengo)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/ambiyansyah-risyal/klayengo/workflows/CI/badge.svg)](https://github.com/ambiyansyah-risyal/klayengo/actions)
[![codecov](https://codecov.io/gh/ambiyansyah-risyal/klayengo/branch/main/graph/badge.svg)](https://codecov.io/gh/ambiyansyah-risyal/klayengo)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-00ADD8.svg)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo/releases)
[![Contributors](https://img.shields.io/github/contributors/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo/graphs/contributors)
[![Stars](https://img.shields.io/github/stars/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo/stargazers)
[![Forks](https://img.shields.io/github/forks/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo/network/members)
[![Issues](https://img.shields.io/github/issues/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo/pulls)
[![Last Commit](https://img.shields.io/github/last-commit/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo/commits/main)
[![Repository Size](https://img.shields.io/github/repo-size/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo)
[![Lines of Code](https://img.shields.io/tokei/lines/github/ambiyansyah-risyal/klayengo)](https://github.com/ambiyansyah-risyal/klayengo)
[![GoDoc](https://godoc.org/github.com/ambiyansyah-risyal/klayengo?status.svg)](https://godoc.org/github.com/ambiyansyah-risyal/klayengo)
[![Security](https://img.shields.io/badge/Security-Policy-green.svg)](SECURITY.md)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/ambiyansyah-risyal/klayengo/graphs/commit-activity)

A resilient HTTP client wrapper for Go with retry logic, exponential backoff, and circuit breaker pattern.

## Status

**Version**: 1.1.0  
**Go Version**: 1.23+ (tested with Go 1.24.6)  
**Test Coverage**: 87.1%  
**License**: MIT

## Versioning

Klayengo follows [Semantic Versioning](https://semver.org/) for releases:

- **MAJOR.MINOR.PATCH** (e.g., `v1.2.3`)
- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Current Version

```go
import "github.com/ambiyansyah-risyal/klayengo"

// Get version information
fmt.Println(klayengo.GetVersion())

// Get detailed version info
versionInfo := klayengo.GetVersionInfo()
fmt.Printf("Version: %s\n", versionInfo["version"])
fmt.Printf("Commit: %s\n", versionInfo["commit"])
fmt.Printf("Build Date: %s\n", versionInfo["build_date"])
```

### Automated Release Process

Klayengo uses GitHub Actions for fully automated releases:

#### 🚀 Automatic Releases
- **Trigger**: Push to `main` branch (after PR merge)
- **Process**: 
  1. Analyzes commit messages using conventional commits
  2. Determines version bump type (major/minor/patch)
  3. Updates version files and CHANGELOG.md
  4. Creates git tag and pushes to remote
  5. Builds binaries with version injection
  6. Creates GitHub release with changelog

#### 📋 Conventional Commits
Use conventional commit format for automatic version bumping:

```bash
# Feature (minor bump)
git commit -m "feat: add configuration validation"

# Bug fix (patch bump)
git commit -m "fix: handle timeout errors properly"

# Breaking change (major bump)
git commit -m "feat!: change API interface"
# or
git commit -m "feat: change API interface

BREAKING CHANGE: This changes the interface"
```

#### 🔍 Pull Request Analysis
When you create a PR, GitHub Actions will:
- Analyze your commits
- Suggest the version bump type
- Show what the new version will be
- Provide tips for better commit messages

#### 🛠️ Manual Releases
For special cases, trigger manual releases via GitHub Actions:

1. Go to **Actions** tab
2. Select **Manual Release** workflow
3. Choose version bump type or specify custom version
4. Click **Run workflow**

#### 📊 Release Workflow
```
PR Merged to main → Auto-tag Workflow → Version Bump → Git Tag → Release Workflow → GitHub Release
```

### Using the Version Management Script

For local development and manual version management:

```bash
# Show current version information
./scripts/version.sh show

# Create a specific version
./scripts/version.sh create v1.1.0

# Bump version automatically
./scripts/version.sh bump minor    # v1.0.0 -> v1.1.0
./scripts/version.sh bump major    # v1.0.0 -> v2.0.0
./scripts/version.sh bump patch    # v1.0.0 -> v1.0.1

# Dry run (see what would happen without making changes)
./scripts/version.sh create v1.1.0 --dry-run
./scripts/version.sh bump minor --dry-run
```

The script automatically:
- Updates `version.go` with the new version
- Updates the version in `README.md`
- Adds an entry to `CHANGELOG.md`
- Creates a git commit and tag
- Pushes changes to the remote repository
- Triggers automated GitHub releases

### Build with Version Info

The library includes build-time version injection:

```bash
# Build with current git info
make build

# Build with specific version
make build VERSION=v1.1.0

# Check version info
make version
```

### Go Module Versioning

When you release a new version, users can update using:

```bash
# Update to latest
go get -u github.com/ambiyansyah-risyal/klayengo

# Update to specific version
go get github.com/ambiyansyah-risyal/klayengo@v1.1.0

# Update to latest patch/minor
go get github.com/ambiyansyah-risyal/klayengo@latest
```

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Donate](#donate)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Advanced Usage](#advanced-usage)
- [Error Handling](#error-handling)
- [Debugging](#debugging)
- [Caching](#caching)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Contributing](#contributing)
- [Security](#security)
- [Testing](#testing)
- [License](#license)

## Donate

Klayengo is a free and open-source project that I maintain in my spare time. If you've found it helpful and would like to show your appreciation, here are some ways you can support the project:

### 💝 **Ways to Support**

#### PayPal
[![PayPal](https://img.shields.io/badge/PayPal-00457C?style=for-the-badge&logo=paypal&logoColor=white)](https://paypal.me/ambiyansyah)

#### GitHub Sponsors
[![GitHub Sponsors](https://img.shields.io/badge/sponsor-30363D?style=for-the-badge&logo=GitHub-Sponsors&logoColor=#EA4AAA)](https://github.com/sponsors/ambiyansyah-risyal)

### 🌟 **Your Support Helps With**

Your generous contributions help sustain the project by supporting:
- **Time Investment**: Dedicated time for maintenance, improvements, and community support
- **Development Resources**: Tools, software licenses, and development environment costs
- **Community Engagement**: Responding to issues, reviewing contributions, and helping users
- **Future Development**: Planning and implementing new features and enhancements

### 🙏 **Thank You**

I'm truly grateful for the support from the community. Every star, issue report, pull request, and contribution - big or small - helps make Klayengo better. Thank you for being part of this journey!

## Features

- **Retry Logic**: Automatic retry on failed HTTP requests with configurable retry attempts
- **Exponential Backoff**: Intelligent backoff strategy with jitter to avoid overwhelming servers
- **Rate Limiting**: Configurable token bucket rate limiting to control request frequency
- **Response Caching**: In-memory caching for GET requests with customizable TTL and conditions
- **Request Deduplication**: Prevents duplicate concurrent requests by deduplicating identical in-flight requests
- **Circuit Breaker**: Prevents cascading failures by temporarily stopping requests to failing services
- **Configuration Validation**: Validates all configuration parameters at client creation time
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

## What's New in v1.0.0

### 🚀 Performance Improvements
- **16-shard cache architecture** with FNV-1a hashing (26% faster concurrent access)
- **Atomic operations** for rate limiter and circuit breaker (28-29% faster)
- **Optimized backoff calculations** with reduced computational overhead
- **Memory-efficient operations** with 14% reduction in allocations

### 🔧 Enhanced Features
- **Request ID generation** for distributed tracing
- **Per-request cache control** via context
- **Custom cache key functions** for advanced caching strategies
- **Conditional caching** based on request characteristics
- **Enhanced middleware system** with better error propagation

### 🛡️ Security & Reliability
- **Comprehensive error types** with detailed context
- **Debug logging system** with configurable verbosity
- **Input validation** for all configuration parameters
- **Secure defaults** for production safety
- **Audit logging** for security monitoring

### 📊 Observability
- **Prometheus metrics** with 10+ metric types
- **Circuit breaker state monitoring**
- **Rate limiter token tracking**
- **Cache hit/miss statistics**
- **Request duration histograms**

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

### Configuration Validation

Klayengo validates all configuration parameters at client creation time to prevent runtime issues from invalid configurations:

```go
// Valid configuration - client created successfully
client, err := klayengo.New(
    klayengo.WithMaxRetries(3),
    klayengo.WithInitialBackoff(100*time.Millisecond),
    klayengo.WithMaxBackoff(5*time.Second),
    klayengo.WithJitter(0.1),
    klayengo.WithRateLimiter(10, 1*time.Second),
)
if err != nil {
    // Handle validation error
    fmt.Printf("Configuration error: %v\n", err)
}
```

#### Validation Rules

Configuration validation ensures:

- **Retry Configuration**: MaxRetries ≥ 0, InitialBackoff > 0, MaxBackoff > InitialBackoff, BackoffMultiplier > 0, Jitter ∈ [0,1]
- **Rate Limiting**: MaxTokens > 0, RefillRate > 0
- **Cache Settings**: TTL > 0, CacheSize > 0
- **Circuit Breaker**: FailureThreshold > 0, RecoveryTimeout > 0, SuccessThreshold > 0
- **Debug Settings**: Valid logger configuration
- **Deduplication**: Valid key function and condition
- **Middleware**: Non-nil middleware functions
- **HTTP Client**: Valid timeout and transport settings

#### Error Handling

Invalid configurations return detailed error messages:

```go
client, err := klayengo.New(
    klayengo.WithMaxRetries(-1), // Invalid: negative retries
    klayengo.WithJitter(1.5),    // Invalid: jitter > 1.0
)
if err != nil {
    // Error: "configuration validation failed: maxRetries must be >= 0, jitter must be between 0.0 and 1.0"
    fmt.Printf("Validation failed: %v\n", err)
}
```

#### Runtime Validation

For configurations that can't be validated at creation time, Klayengo provides runtime validation:

```go
// Check if client configuration is still valid
if !client.IsValid() {
    // Get detailed validation errors
    validationErr := client.ValidationError()
    fmt.Printf("Configuration issues: %v\n", validationErr)
}
```

### Rate Limiting Configuration

```go
client := klayengo.New(
    klayengo.WithRateLimiter(10, 1*time.Second), // 10 requests per second
)
```

### Request Deduplication Configuration

Request deduplication prevents multiple identical concurrent requests from being sent to the server. Only one request is actually executed, while others wait for the result:

```go
// Enable request deduplication
client := klayengo.New(
    klayengo.WithDeduplication(),
)

// Custom deduplication key function
client := klayengo.New(
    klayengo.WithDeduplication(),
    klayengo.WithDeduplicationKeyFunc(func(req *http.Request) string {
        // Include query parameters in deduplication key
        return fmt.Sprintf("%s:%s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
    }),
)

// Custom deduplication condition - only deduplicate specific requests
client := klayengo.New(
    klayengo.WithDeduplication(),
    klayengo.WithDeduplicationCondition(func(req *http.Request) bool {
        // Only deduplicate GET and HEAD requests
        return req.Method == "GET" || req.Method == "HEAD"
    }),
)
```

#### Deduplication Behavior

- **Concurrent Requests**: Multiple identical requests are automatically deduplicated
- **First Request Wins**: The first request executes normally, others wait
- **Shared Results**: All waiting requests receive the same response
- **Timeout Handling**: Waiting requests respect context timeouts
- **Memory Efficient**: Completed requests are cleaned up automatically
- **Thread-Safe**: Safe for concurrent use across multiple goroutines
- **Performance**: 99.3% improvement for duplicate concurrent requests

#### Deduplication Metrics

When deduplication is enabled, additional metrics are available:

- **`klayengo_deduplication_hits_total`**: Total number of deduplication hits (counter)
  - Labels: `method`, `endpoint`

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
- **`klayengo_deduplication_hits_total`**: Total number of deduplication hits (counter)
  - Labels: `method`, `endpoint`
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

## Roadmap

### 🔄 In Progress
- **Redis Cache Backend**: External cache support for distributed deployments
- **Advanced Circuit Breaker**: Adaptive thresholds and machine learning-based failure detection
- **Distributed Tracing**: OpenTelemetry integration for end-to-end observability

### 🚀 Planned Features
- **HTTP/2 Support**: Enhanced protocol support with multiplexing
- **WebSocket Support**: Real-time communication capabilities
- **Plugin System**: Extensible architecture for custom middleware
- **Configuration Management**: YAML/TOML configuration files
- **Kubernetes Integration**: Native support for Kubernetes environments

### 📈 Future Enhancements
- **GraphQL Client**: Specialized support for GraphQL APIs
- **gRPC Support**: Protocol buffer-based communication
- **Multi-Region Support**: Global deployment optimizations
- **AI/ML Integration**: Intelligent retry strategies and anomaly detection

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

Klayengo provides structured error handling with specific error types and enhanced debugging information:

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
        case klayengo.ErrorTypeRateLimit:
            // Handle rate limit exceeded
            fmt.Printf("Rate limited: %s\n", clientErr.Message)
        case klayengo.ErrorTypeCircuitOpen:
            // Handle circuit breaker open
            fmt.Printf("Circuit breaker open: %s\n", clientErr.Message)
        case klayengo.ErrorTypeNetwork:
            // Handle network errors
            fmt.Printf("Network error: %s\n", clientErr.Message)
        default:
            // Handle other errors
            fmt.Printf("Error: %s\n", clientErr.Message)
        }

        // Get detailed debugging information
        if clientErr.RequestID != "" {
            fmt.Printf("Request ID: %s\n", clientErr.RequestID)
        }
        fmt.Printf("Debug Info:\n%s\n", clientErr.DebugInfo())
    }
}
```

### Error Types

Klayengo defines specific error types for better error categorization:

- `ErrorTypeNetwork` - Network-related errors (connection failures, DNS issues, etc.)
- `ErrorTypeTimeout` - Request timeout errors
- `ErrorTypeRateLimit` - Rate limiting errors
- `ErrorTypeCircuitOpen` - Circuit breaker open errors
- `ErrorTypeServer` - Server-side errors (5xx status codes)
- `ErrorTypeClient` - Client-side errors (4xx status codes)
- `ErrorTypeCache` - Cache-related errors
- `ErrorTypeConfig` - Configuration errors
- `ErrorTypeValidation` - Input validation errors

### Enhanced Error Context

ClientError now includes rich context information:

- **RequestID**: Unique identifier for request tracing
- **Method**: HTTP method used
- **URL**: Request URL
- **Attempt**: Current retry attempt (0-based)
- **MaxRetries**: Maximum configured retries
- **Timestamp**: When the error occurred
- **Duration**: How long the request took
- **StatusCode**: HTTP status code (if applicable)
- **Endpoint**: Simplified endpoint for logging
- **Cause**: Underlying error that caused this error

## Debugging

Klayengo provides comprehensive debugging capabilities to help troubleshoot issues:

### Debug Logging

Enable debug logging to see detailed information about request processing:

```go
// Enable debug logging with simple console logger
client := klayengo.New(
    klayengo.WithSimpleLogger(),
)

// Or use a custom logger
client := klayengo.New(
    klayengo.WithLogger(customLogger),
)

// Configure debug options
client := klayengo.New(
    klayengo.WithDebugConfig(&klayengo.DebugConfig{
        Enabled:      true,
        LogRequests:  true,
        LogRetries:   true,
        LogCache:     false,
        LogRateLimit: true,
        LogCircuit:   true,
    }),
)
```

### Debug Configuration Options

- `LogRequests`: Log all HTTP requests
- `LogRetries`: Log retry attempts
- `LogCache`: Log cache operations
- `LogRateLimit`: Log rate limiting events
- `LogCircuit`: Log circuit breaker state changes

### Request Tracing

Each request gets a unique ID for tracing through the system:

```go
// Custom request ID generator
client := klayengo.New(
    klayengo.WithRequestIDGenerator(func() string {
        return fmt.Sprintf("myapp_%d", time.Now().UnixNano())
    }),
)
```

### Debug Information

Use the `DebugInfo()` method to get comprehensive debugging information:

```go
if err != nil {
    var clientErr *klayengo.ClientError
    if errors.As(err, &clientErr) {
        fmt.Println(clientErr.DebugInfo())
    }
}
```

This provides detailed information including:
- Error type and message
- Request details (ID, method, URL, endpoint)
- Timing information (timestamp, duration)
- Retry information (attempt count, max retries)
- HTTP status code
- Underlying cause

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
- `WithDebug()` - Enable debug logging with default configuration
- `WithDebugConfig(config *DebugConfig)` - Set custom debug configuration
- `WithLogger(logger Logger)` - Set a custom logger for debug output
- `WithSimpleLogger()` - Enable debug logging with a simple console logger
- `WithDeduplication()` - Enable request deduplication
- `WithDeduplicationKeyFunc(fn DeduplicationKeyFunc)` - Custom deduplication key generation function
- `WithDeduplicationCondition(fn DeduplicationCondition)` - Custom deduplication condition function

### Context Helper Functions

- `WithContextCacheEnabled(ctx context.Context) context.Context` - Enable caching for specific request
- `WithContextCacheDisabled(ctx context.Context) context.Context` - Disable caching for specific request
- `WithContextCacheTTL(ctx context.Context, ttl time.Duration) context.Context` - Set custom TTL for specific request

### Circuit Breaker States

- `Closed`: Normal operation, requests pass through
- `Open`: Circuit is open, requests fail immediately
- `HalfOpen`: Testing if service has recovered

## Project Structure

```
klayengo/
├── client.go              # Main HTTP client implementation
├── cache.go               # In-memory caching with sharding
├── circuit_breaker.go     # Circuit breaker pattern implementation
├── rate_limiter.go        # Token bucket rate limiting
├── deduplication.go       # Request deduplication for concurrent requests
├── deduplication_test.go  # Tests for deduplication functionality
├── metrics.go             # Prometheus metrics collection
├── errors.go              # Structured error handling
├── options.go             # Functional options pattern
├── types.go               # Common types and interfaces
├── examples/              # Usage examples
│   ├── basic/            # Basic retry usage
│   ├── advanced/         # Advanced features
│   ├── metrics/          # Metrics integration
│   └── deduplication/    # Request deduplication example
├── .github/              # GitHub configuration
│   ├── workflows/        # CI/CD pipelines
│   ├── ISSUE_TEMPLATE/   # Issue templates
│   └── pull_request_template.md
├── CONTRIBUTING.md        # Contribution guidelines
├── SECURITY.md           # Security policy
├── BENCHMARKS.md         # Performance benchmarks
└── README.md            # This file
```

## Documentation

- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project
- **[Security Policy](SECURITY.md)** - Security vulnerability reporting
- **[Benchmarks](BENCHMARKS.md)** - Performance analysis and results
- **[API Documentation](https://pkg.go.dev/github.com/ambiyansyah-risyal/klayengo)** - Go package documentation

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for detailed information on how to contribute to Klayengo.

### Quick Start for Contributors

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following our [contributing guidelines](CONTRIBUTING.md)
4. Add tests for new functionality
5. Ensure all tests pass: `go test ./...`
6. Update documentation if needed
7. Commit your changes (`git commit -m 'Add some amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request using our [PR template](.github/pull_request_template.md)

### Issue Reporting

- **Bug Reports**: Use our [bug report template](.github/ISSUE_TEMPLATE/bug_report.md)
- **Feature Requests**: Use our [feature request template](.github/ISSUE_TEMPLATE/feature_request.md)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/ambiyansyah-risyal/klayengo.git
cd klayengo

# Install dependencies
go mod download

# Run tests
go test ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run linting
golangci-lint run
```

### Development Guidelines

- **Code Coverage**: Maintain >90% test coverage
- **Documentation**: Update docs for any API changes
- **Benchmarks**: Add benchmarks for performance-critical code
- **Security**: Follow secure coding practices
- **Compatibility**: Ensure Go 1.21+ compatibility

### Areas for Contribution

We're looking for contributions in these areas:

- **Performance Optimizations**: Improve existing algorithms and data structures
- **New Features**: Implement planned features from our roadmap
- **Documentation**: Improve examples, guides, and API documentation
- **Testing**: Add more comprehensive test cases and integration tests
- **Bug Fixes**: Help resolve existing issues and improve stability
- **Security**: Enhance security features and review code for vulnerabilities

### Getting Help

- **Discussions**: Use [GitHub Discussions](https://github.com/ambiyansyah-risyal/klayengo/discussions) for questions and general discussion
- **Issues**: Report bugs and request features using our issue templates
- **Documentation**: Check our [Contributing Guide](CONTRIBUTING.md) for detailed guidelines

For more detailed development setup and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

Security is a top priority for Klayengo. If you discover a security vulnerability, please report it responsibly.

### Reporting Security Issues

Please **DO NOT** report security vulnerabilities through public GitHub issues. Instead, follow our [Security Policy](SECURITY.md) for responsible disclosure.

### Security Features

Klayengo includes several security-focused features:

- **Rate Limiting**: Prevents abuse and protects against DoS attacks
- **Circuit Breaker**: Provides resilience against cascading failures
- **Input Validation**: Validates configuration parameters
- **Secure Defaults**: Conservative default settings for security
- **Audit Logging**: Comprehensive logging for security monitoring

For more information about security considerations and best practices, see our [Security Policy](SECURITY.md).

## Testing

Klayengo maintains high code quality with comprehensive test coverage and automated CI/CD pipelines.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run specific test
go test -run TestClientGet ./...
```

### Test Coverage

Current test coverage: **87.1%**

- Unit tests for all core functionality
- Integration tests for end-to-end scenarios
- Benchmark tests for performance validation
- Concurrent safety tests
- Error handling tests

### CI/CD Pipeline

Klayengo uses GitHub Actions for automated testing and quality assurance:

- **Linting**: golangci-lint for code quality
- **Testing**: Multi-version Go testing (1.23, 1.24)
- **Examples**: Automated testing of example code
- **Security**: Automated security scanning

[![CI](https://github.com/ambiyansyah-risyal/klayengo/workflows/CI/badge.svg)](https://github.com/ambiyansyah-risyal/klayengo/actions)

### Performance Testing

For detailed performance analysis and benchmarking results, see [`BENCHMARKS.md`](BENCHMARKS.md).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Klayengo is built with the following technologies and inspirations:

- **Go**: The programming language that makes this possible
- **Prometheus**: For metrics collection and monitoring
- **Circuit Breaker Pattern**: Inspired by Netflix Hystrix and similar implementations
- **Exponential Backoff**: Based on proven retry strategies
- **Token Bucket Algorithm**: For efficient rate limiting

## Related Projects

- [go-retry](https://github.com/go-retry/retry) - Simple retry library
- [circuit](https://github.com/rubyist/circuitbreaker) - Circuit breaker implementation
- [backoff](https://github.com/cenkalti/backoff) - Exponential backoff library
- [prometheus/client_golang](https://github.com/prometheus/client_golang) - Prometheus Go client

---

<p align="center">
  <strong>Klayengo</strong> - Making HTTP requests resilient and reliable
  <br>
  Built with ❤️ in Go
</p>

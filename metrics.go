package klayengo

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector holds all Prometheus metrics for the HTTP client
type MetricsCollector struct {
	// Request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec

	// Retry metrics
	retriesTotal *prometheus.CounterVec

	// Circuit breaker metrics
	circuitBreakerState *prometheus.GaugeVec

	// Rate limiter metrics
	rateLimiterTokens *prometheus.GaugeVec

	// Cache metrics
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec
	cacheSize   *prometheus.GaugeVec

	// Deduplication metrics
	deduplicationHits *prometheus.CounterVec

	// Error metrics
	errorsTotal *prometheus.CounterVec

	// Registry for custom metrics
	registry *prometheus.Registry
}

// NewMetricsCollector creates a new metrics collector with default Prometheus registry
func NewMetricsCollector() *MetricsCollector {
	return NewMetricsCollectorWithRegistry(prometheus.DefaultRegisterer)
}

// NewMetricsCollectorWithRegistry creates a new metrics collector with custom registry
func NewMetricsCollectorWithRegistry(registry prometheus.Registerer) *MetricsCollector {
	mc := &MetricsCollector{
		requestsTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "klayengo_requests_total",
				Help: "Total number of HTTP requests made",
			},
			[]string{"method", "status_code", "endpoint"},
		),
		requestDuration: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "klayengo_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "status_code", "endpoint"},
		),
		requestsInFlight: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "klayengo_requests_in_flight",
				Help: "Number of HTTP requests currently in flight",
			},
			[]string{"method", "endpoint"},
		),
		retriesTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "klayengo_retries_total",
				Help: "Total number of retry attempts",
			},
			[]string{"method", "endpoint", "attempt"},
		),
		circuitBreakerState: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "klayengo_circuit_breaker_state",
				Help: "Current state of circuit breaker (0=closed, 1=open, 2=half-open)",
			},
			[]string{"name"},
		),
		rateLimiterTokens: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "klayengo_rate_limiter_tokens",
				Help: "Current number of available rate limiter tokens",
			},
			[]string{"name"},
		),
		cacheHits: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "klayengo_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"method", "endpoint"},
		),
		cacheMisses: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "klayengo_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"method", "endpoint"},
		),
		cacheSize: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "klayengo_cache_size",
				Help: "Current number of entries in cache",
			},
			[]string{"name"},
		),
		deduplicationHits: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "klayengo_deduplication_hits_total",
				Help: "Total number of deduplication hits",
			},
			[]string{"method", "endpoint"},
		),
		errorsTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "klayengo_errors_total",
				Help: "Total number of errors encountered",
			},
			[]string{"type", "method", "endpoint"},
		),
		registry: registry.(*prometheus.Registry),
	}

	return mc
}

// RecordRequest records a completed HTTP request
func (mc *MetricsCollector) RecordRequest(method, endpoint string, statusCode int, duration time.Duration) {
	if mc == nil {
		return
	}

	statusCodeStr := strconv.Itoa(statusCode)
	mc.requestsTotal.WithLabelValues(method, statusCodeStr, endpoint).Inc()
	mc.requestDuration.WithLabelValues(method, statusCodeStr, endpoint).Observe(duration.Seconds())
}

// RecordRequestStart records the start of an HTTP request
func (mc *MetricsCollector) RecordRequestStart(method, endpoint string) {
	if mc == nil {
		return
	}

	mc.requestsInFlight.WithLabelValues(method, endpoint).Inc()
}

// RecordRequestEnd records the end of an HTTP request
func (mc *MetricsCollector) RecordRequestEnd(method, endpoint string) {
	if mc == nil {
		return
	}

	mc.requestsInFlight.WithLabelValues(method, endpoint).Dec()
}

// RecordRetry records a retry attempt
func (mc *MetricsCollector) RecordRetry(method, endpoint string, attempt int) {
	if mc == nil {
		return
	}

	attemptStr := strconv.Itoa(attempt)
	mc.retriesTotal.WithLabelValues(method, endpoint, attemptStr).Inc()
}

// RecordCircuitBreakerState records the current circuit breaker state
func (mc *MetricsCollector) RecordCircuitBreakerState(name string, state CircuitState) {
	if mc == nil {
		return
	}

	var stateValue float64
	switch state {
	case StateClosed:
		stateValue = 0
	case StateOpen:
		stateValue = 1
	case StateHalfOpen:
		stateValue = 2
	}

	mc.circuitBreakerState.WithLabelValues(name).Set(stateValue)
}

// RecordRateLimiterTokens records the current number of available tokens
func (mc *MetricsCollector) RecordRateLimiterTokens(name string, tokens int) {
	if mc == nil {
		return
	}

	mc.rateLimiterTokens.WithLabelValues(name).Set(float64(tokens))
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit(method, endpoint string) {
	if mc == nil {
		return
	}

	mc.cacheHits.WithLabelValues(method, endpoint).Inc()
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss(method, endpoint string) {
	if mc == nil {
		return
	}

	mc.cacheMisses.WithLabelValues(method, endpoint).Inc()
}

// RecordCacheSize records the current cache size
func (mc *MetricsCollector) RecordCacheSize(name string, size int) {
	if mc == nil {
		return
	}

	mc.cacheSize.WithLabelValues(name).Set(float64(size))
}

// RecordError records an error
func (mc *MetricsCollector) RecordError(errorType, method, endpoint string) {
	if mc == nil {
		return
	}

	mc.errorsTotal.WithLabelValues(errorType, method, endpoint).Inc()
}

// RecordDeduplicationHit records a deduplication hit
func (mc *MetricsCollector) RecordDeduplicationHit(method, endpoint string) {
	if mc == nil {
		return
	}

	mc.deduplicationHits.WithLabelValues(method, endpoint).Inc()
}

// GetRegistry returns the Prometheus registry
func (mc *MetricsCollector) GetRegistry() *prometheus.Registry {
	return mc.registry
}

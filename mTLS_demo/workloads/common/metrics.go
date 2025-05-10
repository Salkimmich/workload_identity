package common

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for monitoring various aspects of the application
var (
	// HTTP request metrics
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status"},
	)

	// Total HTTP request counter
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	// HTTP error counter
	HTTPRequestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total number of HTTP request errors",
		},
		[]string{"path", "method", "error_type"},
	)

	// TLS handshake metrics
	TLSHandshakeDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tls_handshake_duration_seconds",
			Help:    "Duration of TLS handshakes in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"version", "cipher_suite"},
	)

	// TLS error counter
	TLSHandshakeErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tls_handshake_errors_total",
			Help: "Number of TLS handshake errors",
		},
		[]string{"error_type"},
	)

	// TLS version usage counter
	TLSVersion = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tls_version_total",
			Help: "Number of TLS connections by version",
		},
		[]string{"version"},
	)

	// Connection pool metrics
	ConnectionPoolSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "connection_pool_size",
			Help: "Current size of the connection pool",
		},
		[]string{"service"},
	)

	// Connection error counter
	ConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "connection_errors_total",
			Help: "Number of connection errors",
		},
		[]string{"service", "error_type"},
	)

	// Connection latency metrics
	ConnectionLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "connection_latency_seconds",
			Help:    "Connection establishment latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service"},
	)

	// Memory usage metrics
	MemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
		[]string{"type"},
	)

	// Goroutine count gauge
	GoRoutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines_total",
			Help: "Number of goroutines",
		},
	)

	// Garbage collection metrics
	GCStats = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gc_stats",
			Help: "Garbage collection statistics",
		},
		[]string{"type"},
	)
)

// MetricsCollector provides methods for collecting various metrics
type MetricsCollector struct {
	startTime time.Time // Application start time
}

// NewMetricsCollector creates a new metrics collector instance
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// RecordHTTPRequest records metrics for an HTTP request
func (m *MetricsCollector) RecordHTTPRequest(path, method string, status int, duration time.Duration) {
	HTTPRequestDuration.WithLabelValues(path, method, string(status)).Observe(duration.Seconds())
	HTTPRequestsTotal.WithLabelValues(path, method, string(status)).Inc()
}

// RecordHTTPError records metrics for an HTTP error
func (m *MetricsCollector) RecordHTTPError(path, method, errorType string) {
	HTTPRequestErrors.WithLabelValues(path, method, errorType).Inc()
}

// RecordTLSHandshake records metrics for a TLS handshake
func (m *MetricsCollector) RecordTLSHandshake(version, cipherSuite string, duration time.Duration) {
	TLSHandshakeDuration.WithLabelValues(version, cipherSuite).Observe(duration.Seconds())
	TLSVersion.WithLabelValues(version).Inc()
}

// RecordTLSHandshakeError records metrics for a TLS handshake error
func (m *MetricsCollector) RecordTLSHandshakeError(errorType string) {
	TLSHandshakeErrors.WithLabelValues(errorType).Inc()
}

// RecordConnectionPoolSize records the current connection pool size
func (m *MetricsCollector) RecordConnectionPoolSize(service string, size int) {
	ConnectionPoolSize.WithLabelValues(service).Set(float64(size))
}

// RecordConnectionError records a connection error
func (m *MetricsCollector) RecordConnectionError(service, errorType string) {
	ConnectionErrors.WithLabelValues(service, errorType).Inc()
}

// RecordConnectionLatency records connection establishment latency
func (m *MetricsCollector) RecordConnectionLatency(service string, duration time.Duration) {
	ConnectionLatency.WithLabelValues(service).Observe(duration.Seconds())
}

// RecordMemoryUsage records memory usage metrics
func (m *MetricsCollector) RecordMemoryUsage(memoryType string, bytes int64) {
	MemoryUsage.WithLabelValues(memoryType).Set(float64(bytes))
}

// RecordGoRoutines records the number of goroutines
func (m *MetricsCollector) RecordGoRoutines(count int) {
	GoRoutines.Set(float64(count))
}

// RecordGCStats records garbage collection statistics
func (m *MetricsCollector) RecordGCStats(statsType string, value float64) {
	GCStats.WithLabelValues(statsType).Set(value)
} 
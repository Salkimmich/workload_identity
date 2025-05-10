package common

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Transport metrics
	TransportRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transport_request_duration_seconds",
			Help:    "Duration of transport requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "endpoint", "status"},
	)

	TransportRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transport_requests_total",
			Help: "Total number of transport requests",
		},
		[]string{"service", "endpoint", "status"},
	)

	TransportErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transport_errors_total",
			Help: "Total number of transport errors",
		},
		[]string{"service", "endpoint", "error_type"},
	)

	TransportConnectionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "transport_connections_active",
			Help: "Number of active transport connections",
		},
		[]string{"service"},
	)

	TransportConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transport_connection_errors_total",
			Help: "Total number of connection errors",
		},
		[]string{"service", "error_type"},
	)

	TransportConnectionLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transport_connection_latency_seconds",
			Help:    "Connection establishment latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service"},
	)
)

// MetricsCollector handles transport layer metrics
type MetricsCollector struct {
	startTime time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// RecordRequest records a transport request
func (m *MetricsCollector) RecordRequest(service, endpoint, status string, duration time.Duration) {
	TransportRequestDuration.WithLabelValues(service, endpoint, status).Observe(duration.Seconds())
	TransportRequestsTotal.WithLabelValues(service, endpoint, status).Inc()
}

// RecordError records a transport error
func (m *MetricsCollector) RecordError(service, endpoint, errorType string) {
	TransportErrorsTotal.WithLabelValues(service, endpoint, errorType).Inc()
}

// RecordConnection records a new connection
func (m *MetricsCollector) RecordConnection(service string) {
	TransportConnectionsActive.WithLabelValues(service).Inc()
}

// RecordDisconnection records a connection termination
func (m *MetricsCollector) RecordDisconnection(service string) {
	TransportConnectionsActive.WithLabelValues(service).Dec()
}

// RecordConnectionError records a connection error
func (m *MetricsCollector) RecordConnectionError(service, errorType string) {
	TransportConnectionErrors.WithLabelValues(service, errorType).Inc()
}

// RecordConnectionLatency records connection establishment latency
func (m *MetricsCollector) RecordConnectionLatency(service string, duration time.Duration) {
	TransportConnectionLatency.WithLabelValues(service).Observe(duration.Seconds())
} 
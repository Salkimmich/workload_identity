package common

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Authentication metrics
	AuthRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_requests_total",
			Help: "Total number of authentication requests",
		},
		[]string{"service", "method", "status"},
	)

	AuthRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_request_duration_seconds",
			Help:    "Duration of authentication requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)

	AuthErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_errors_total",
			Help: "Total number of authentication errors",
		},
		[]string{"service", "method", "error_type"},
	)

	TokenValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "token_validations_total",
			Help: "Total number of token validations",
		},
		[]string{"service", "status"},
	)

	TokenValidationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "token_validation_errors_total",
			Help: "Total number of token validation errors",
		},
		[]string{"service", "error_type"},
	)

	TokenExpirationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "token_expirations_total",
			Help: "Total number of token expirations",
		},
		[]string{"service"},
	)

	ActiveTokens = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_tokens",
			Help: "Number of active tokens",
		},
		[]string{"service"},
	)
)

// AuthMetricsCollector implements the AuthMetricsCollector interface
type AuthMetricsCollector struct {
	authRequests *prometheus.CounterVec
	authErrors   *prometheus.CounterVec
	authDuration *prometheus.HistogramVec
	mu           sync.RWMutex
}

// NewAuthMetricsCollector creates a new metrics collector
func NewAuthMetricsCollector() *AuthMetricsCollector {
	collector := &AuthMetricsCollector{
		authRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_requests_total",
				Help: "Total number of authentication requests",
			},
			[]string{"service", "method", "result"},
		),
		authErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_errors_total",
				Help: "Total number of authentication errors",
			},
			[]string{"service", "method", "error_type"},
		),
		authDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auth_duration_seconds",
				Help:    "Authentication request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method"},
		),
	}

	// Register metrics
	prometheus.MustRegister(collector.authRequests)
	prometheus.MustRegister(collector.authErrors)
	prometheus.MustRegister(collector.authDuration)

	return collector
}

// RecordAuthRequest records a successful authentication request
func (c *AuthMetricsCollector) RecordAuthRequest(serviceName, authMethod, result string, duration float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.authRequests.WithLabelValues(serviceName, authMethod, result).Inc()
	c.authDuration.WithLabelValues(serviceName, authMethod).Observe(duration)
}

// RecordAuthError records an authentication error
func (c *AuthMetricsCollector) RecordAuthError(serviceName, authMethod, errorType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.authErrors.WithLabelValues(serviceName, authMethod, errorType).Inc()
}

// Describe implements prometheus.Collector
func (c *AuthMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.authRequests.Describe(ch)
	c.authErrors.Describe(ch)
	c.authDuration.Describe(ch)
}

// Collect implements prometheus.Collector
func (c *AuthMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.authRequests.Collect(ch)
	c.authErrors.Collect(ch)
	c.authDuration.Collect(ch)
}

// RecordTokenValidation records a token validation
func (m *AuthMetricsCollector) RecordTokenValidation(service, status string) {
	TokenValidationsTotal.WithLabelValues(service, status).Inc()
}

// RecordTokenValidationError records a token validation error
func (m *AuthMetricsCollector) RecordTokenValidationError(service, errorType string) {
	TokenValidationErrors.WithLabelValues(service, errorType).Inc()
}

// RecordTokenExpiration records a token expiration
func (m *AuthMetricsCollector) RecordTokenExpiration(service string) {
	TokenExpirationsTotal.WithLabelValues(service).Inc()
	ActiveTokens.WithLabelValues(service).Dec()
}

// RecordNewToken records a new token
func (m *AuthMetricsCollector) RecordNewToken(service string) {
	ActiveTokens.WithLabelValues(service).Inc()
} 
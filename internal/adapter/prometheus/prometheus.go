package prometheus

import (
	nethttp "net/http"
	"strconv"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	labelMethod     = "method"
	labelRoute      = "route"
	labelStatusCode = "status_code"
)

type HTTPMetrics struct {
	registry *prometheusclient.Registry
	requests *prometheusclient.CounterVec
	errors   *prometheusclient.CounterVec
	duration *prometheusclient.HistogramVec
}

func NewHTTPMetrics() *HTTPMetrics {
	registry := prometheusclient.NewRegistry()

	labels := []string{
		labelMethod,
		labelRoute,
		labelStatusCode,
	}

	metrics := &HTTPMetrics{
		registry: registry,
		requests: prometheusclient.NewCounterVec(
			prometheusclient.CounterOpts{
				Namespace: "task_manager",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests.",
			},
			labels,
		),
		errors: prometheusclient.NewCounterVec(
			prometheusclient.CounterOpts{
				Namespace: "task_manager",
				Subsystem: "http",
				Name:      "errors_total",
				Help:      "Total number of HTTP responses with status code 400 or greater.",
			},
			labels,
		),
		duration: prometheusclient.NewHistogramVec(
			prometheusclient.HistogramOpts{
				Namespace: "task_manager",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request processing duration in seconds.",
				Buckets:   prometheusclient.DefBuckets,
			},
			labels,
		),
	}

	registry.MustRegister(
		metrics.requests,
		metrics.errors,
		metrics.duration,
	)

	return metrics
}

func (m *HTTPMetrics) Observe(
	method string,
	route string,
	statusCode int,
	duration time.Duration,
) {
	status := strconv.Itoa(statusCode)
	labels := []string{method, route, status}

	m.requests.WithLabelValues(labels...).Inc()
	m.duration.WithLabelValues(labels...).Observe(
		duration.Seconds(),
	)

	if statusCode >= nethttp.StatusBadRequest {
		m.errors.WithLabelValues(labels...).Inc()
	}
}

func (m *HTTPMetrics) Handler() nethttp.Handler {
	return promhttp.HandlerFor(
		m.registry,
		promhttp.HandlerOpts{},
	)
}

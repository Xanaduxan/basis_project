package middleware

import (
	nethttp "net/http"
	"strings"
	"time"
)

const unmatchedRoute = "unmatched"

type HTTPMetricsObserver interface {
	Observe(
		method string,
		route string,
		statusCode int,
		duration time.Duration,
	)
}

type metricsResponseWriter struct {
	nethttp.ResponseWriter
	statusCode int
}

func (w *metricsResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode != 0 {
		return
	}

	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *metricsResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(nethttp.StatusOK)
	}

	return w.ResponseWriter.Write(data)
}

func (w *metricsResponseWriter) Unwrap() nethttp.ResponseWriter {
	return w.ResponseWriter
}

func Metrics(
	metrics HTTPMetricsObserver,
	next nethttp.Handler,
) nethttp.Handler {
	return nethttp.HandlerFunc(
		func(w nethttp.ResponseWriter, r *nethttp.Request) {
			startedAt := time.Now()
			responseWriter := &metricsResponseWriter{
				ResponseWriter: w,
			}

			next.ServeHTTP(responseWriter, r)

			statusCode := responseWriter.statusCode
			if statusCode == 0 {
				statusCode = nethttp.StatusOK
			}

			metrics.Observe(
				r.Method,
				routeFromPattern(r.Pattern),
				statusCode,
				time.Since(startedAt),
			)
		},
	)
}

func routeFromPattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return unmatchedRoute
	}

	_, route, found := strings.Cut(pattern, " ")
	if found && route != "" {
		return route
	}

	return pattern
}

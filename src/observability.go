package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)



// The RED signals for every endpoint: Rate, Errors, Duration.

var (
	// Rate: all requests, by method, route and status code.
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Requests handled, by method, route and status code.",
	}, []string{"method", "route", "code"})

	// Errors: the requests that did not return a 2xx status.
	httpRequestErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_errors_total",
		Help: "Requests that returned a non-2xx status, by method, route and status code.",
	}, []string{"method", "route", "code"})

	// Duration: latency histogram. The buckets run from 0.1 microsec to 2.5 s so that
	// p50, p95 and p99 can be computed in Prometheus with histogram_quantile.
	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Request latency in seconds, by method and route.",
		Buckets: []float64{0.000001, 0.000002, 0.000005, 0.00001, 0.00002, 0.00005, 0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"method", "route"})
)

// logger writes one JSON object per line to stdout
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// observabilityMiddleware records the RED metrics and writes one structured log
// line per request, with the endpoint, the status code and the duration.
//
// It must be registered before recoverMiddleware. A panic is turned into a 500
// by the recovery middleware, and by the time c.Next() returns here the status
// is already 500, so failed requests are counted too.
func observabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prometheus scrapes /metrics every few seconds. Leave it out so that it
		// does not distort the numbers or fill the logs.
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		elapsed := time.Since(start)

		route := c.FullPath()
		if route == "" {
			route = "unmatched" // no route matched, for example an unknown path
		}
		status := c.Writer.Status()
		code := strconv.Itoa(status)
		method := c.Request.Method

		httpRequestsTotal.WithLabelValues(method, route, code).Inc()
		httpRequestDuration.WithLabelValues(method, route).Observe(elapsed.Seconds())
		if status < 200 || status > 299 {
			httpRequestErrorsTotal.WithLabelValues(method, route, code).Inc()
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		logger.LogAttrs(c.Request.Context(), level, "request",
			slog.String("method", method),
			slog.String("endpoint", route),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(elapsed.Microseconds())/1000),
		)
	}
}

// metricsHandler serves the Prometheus metrics at GET /metrics.
func metricsHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

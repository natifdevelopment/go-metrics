package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"service", "method", "route", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "route"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"service", "method", "route"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"service", "method", "route"},
	)

	httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed.",
		},
		[]string{"service"},
	)
)

// GinMiddleware returns a Gin middleware that records Prometheus metrics
// for each HTTP request: request count, latency histogram, response size,
// and in-flight gauge. The service name is used as a label so metrics from
// multiple services can be distinguished when scraped to the same Prometheus
// instance or displayed on a single Grafana dashboard.
func GinMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		httpRequestsInFlight.WithLabelValues(serviceName).Inc()

		c.Next()

		httpRequestsInFlight.WithLabelValues(serviceName).Dec()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())
		elapsed := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(serviceName, method, route, status).Inc()
		httpRequestDuration.WithLabelValues(serviceName, method, route).Observe(elapsed)

		if c.Request.ContentLength > 0 {
			httpRequestSize.WithLabelValues(serviceName, method, route).Observe(float64(c.Request.ContentLength))
		}
		if size := c.Writer.Size(); size > 0 {
			httpResponseSize.WithLabelValues(serviceName, method, route).Observe(float64(size))
		}
	}
}

// Handler returns a Gin handler that serves the Prometheus metrics endpoint.
// Register it on a route that is excluded from authentication middleware,
// e.g. r.GET("/metrics", metrics.Handler())
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}

// BusinessCounter returns a counter for business-level metrics (e.g. active
// users, orders processed). The counter is registered once on first call and
// reused on subsequent calls with the same name.
func BusinessCounter(name, help string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
}

// BusinessGauge returns a gauge for business-level metrics.
func BusinessGauge(name, help string, labels []string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
}

// BusinessHistogram returns a histogram for business-level metrics.
func BusinessHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}
	if buckets == nil {
		opts.Buckets = prometheus.DefBuckets
	}
	return promauto.NewHistogramVec(opts, labels)
}

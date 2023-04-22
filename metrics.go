package chiutil

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics endpoint middleware useful to setting up a path like
// `/metrics` ...

func Metrics(service, prefix string) func(h http.Handler) http.Handler {
	labels := []string{"service", "handler", "method", "code"}

	var (
		requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prefix,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "The latency of the HTTP requests.",
			Buckets:   prometheus.DefBuckets,
		}, labels)

		responseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prefix,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "The size of the HTTP responses.",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		}, labels)

		requestsInflight = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prefix,
			Subsystem: "http",
			Name:      "requests_inflight",
			Help:      "The number of inflight requests being handled at the same time.",
		}, labels[:2])
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			start := time.Now()
			requestsInflight.WithLabelValues(service, r.URL.Path).Add(1)
			defer func() {
				var (
					values  = []string{service, r.URL.Path, r.Method, strconv.Itoa(ww.Status())}
					elapsed = time.Since(start)
				)
				requestDuration.WithLabelValues(values...).Observe(elapsed.Seconds())
				responseSize.WithLabelValues(values...).Observe(float64(ww.BytesWritten()))
				requestsInflight.WithLabelValues(values[:2]...).Add(-1)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

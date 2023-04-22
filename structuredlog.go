package chiutil

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Logger returns a request logging middleware
func Logger(logger *zap.Logger) func(h http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			start := time.Now()
			defer func() {
				elapsed := time.Since(start)

				remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					remoteIP = r.RemoteAddr
				}

				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}

				reqID := middleware.GetReqID(r.Context())

				logger.Info("incoming request",
					zap.String("method", r.Method),
					zap.String("proto", r.Proto),
					zap.String("remote_ip", remoteIP),
					zap.String("duration_display", elapsed.String()),
					zap.String("scheme", scheme),
					zap.String("host", r.Host),
					zap.String("path", r.RequestURI),
					zap.String("request_id", reqID),
					zap.Int("status_code", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.Duration("duration", elapsed),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

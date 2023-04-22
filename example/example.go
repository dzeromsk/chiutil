package main

import (
	"io"
	"net/http"
	"time"

	"chiutil"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	r := chi.NewRouter()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	r.Use(middleware.RequestID)
	r.Use(chiutil.Logger(logger))
	r.Use(chiutil.Metrics("example", ""))
	r.Use(chiutil.LoadShedding(10, 1*time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "welcome")
	})
	r.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":3000", r); err != nil {
		panic(err)
	}
}

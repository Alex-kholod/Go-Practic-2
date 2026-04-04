package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Метрики объявляются один раз глобально в пакете.
// promauto регистрирует их в prometheus.DefaultRegisterer автоматически.
var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Общее количество HTTP-запросов.",
		},
		[]string{"service", "method", "route", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Длительность обработки HTTP-запросов.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.3, 1, 3},
		},
		[]string{"service", "method", "route"},
	)

	inFlightRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_in_flight_requests",
			Help: "Текущее количество обрабатываемых запросов.",
		},
		[]string{"service"},
	)
)

func normalizeRoute(path string) string {
	// Простая эвристика: если последний сегмент не пустой и
	// предпоследний — "tasks", считаем это item-роутом.
	if len(path) > len("/v1/tasks/") && path[:len("/v1/tasks/")] == "/v1/tasks/" {
		return "/v1/tasks/:id"
	}
	return path
}

func Metrics(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := normalizeRoute(r.URL.Path)

			// Увеличиваем gauge при входе, уменьшаем при выходе.
			inFlightRequests.WithLabelValues(serviceName).Inc()
			defer inFlightRequests.WithLabelValues(serviceName).Dec()

			start := time.Now()

			// Оборачиваем writer, чтобы поймать статус-код.
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rec.status)

			requestsTotal.WithLabelValues(serviceName, r.Method, route, status).Inc()
			requestDuration.WithLabelValues(serviceName, r.Method, route).Observe(duration)
		})
	}
}

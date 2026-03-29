package middleware

import (
	"log"
	"net/http"
	"time"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		rid := GetRequestID(r)
		log.Printf("[%s] %s %s %s", rid, r.Method, r.URL.Path, time.Since(start))
	})
}

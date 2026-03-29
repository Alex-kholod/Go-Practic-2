package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"go.uber.org/zap"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
)

func GenerateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func CSRF(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := GetRequestID(r)

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(csrfCookieName)
			if err != nil || cookie.Value == "" {
				log.Warn("csrf: missing csrf cookie",
					zap.String("request_id", rid),
					zap.String("component", "csrf_middleware"),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			headerToken := r.Header.Get(csrfHeaderName)
			if headerToken == "" || headerToken != cookie.Value {
				log.Warn("csrf: token mismatch",
					zap.String("request_id", rid),
					zap.String("component", "csrf_middleware"),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

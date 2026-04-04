package http

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"pz1/services/auth/internal/service"
	"pz1/shared/middleware"
)

func NewMux(log *zap.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	// выдаёт session cookie и csrf_token cookie.
	mux.HandleFunc("/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetRequestID(r)

		// Проверяем логин/пароль
		username, ok := service.LoginCheck(r)
		if !ok {
			log.Warn("auth: login failed",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
			)
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
			return
		}

		// Session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    username + "-" + middleware.GenerateCSRFToken()[:16],
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		// CSRF cookie
		csrfToken := middleware.GenerateCSRFToken()
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: false,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		log.Info("auth: login success",
			zap.String("request_id", rid),
			zap.String("component", "handler"),
			zap.String("username", username),
		)

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]string{
			"username":   username,
			"csrf_token": csrfToken,
		})
	})

	mux.HandleFunc("/v1/auth/verify", service.VerifyHandler)

	h := middleware.RequestID(
		middleware.Metrics("auth")(
			middleware.SecurityHeaders(
				middleware.AccessLog(log)(mux),
			),
		),
	)
	return h
}

func Run(log *zap.Logger) error {
	port := os.Getenv("AUTH_PORT")
	if port == "" {
		port = "8081"
	}
	log.Info("auth HTTP server listening", zap.String("port", port))
	return http.ListenAndServe(":"+port, NewMux(log))
}

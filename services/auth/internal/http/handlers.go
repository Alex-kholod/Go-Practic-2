package http

import (
	"net/http"
	"os"

	"go.uber.org/zap"

	"pz1/services/auth/internal/service"
	"pz1/shared/middleware"
)

func NewMux(log *zap.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", service.LoginHandler)
	mux.HandleFunc("/v1/auth/verify", service.VerifyHandler)

	h := middleware.RequestID(middleware.AccessLog(log)(mux))
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

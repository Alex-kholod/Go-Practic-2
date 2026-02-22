package http

import (
	"net/http"
	"os"
	"pz1/services/auth/internal/service"
	"pz1/shared/middleware"
)

func NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", service.LoginHandler)
	mux.HandleFunc("/v1/auth/verify", service.VerifyHandler)
	// middleware
	h := middleware.RequestID(middleware.Logging(mux))
	return h
}

func Run() error {
	port := os.Getenv("AUTH_PORT")
	if port == "" {
		port = "8081"
	}
	addr := ":" + port
	return http.ListenAndServe(addr, NewMux())
}

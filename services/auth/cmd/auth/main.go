package main

import (
	"os"
	"sync"

	"go.uber.org/zap"

	"pz1/services/auth/internal/grpc"
	"pz1/services/auth/internal/http"
	"pz1/shared/logger"
)

func main() {
	httpPort := os.Getenv("AUTH_PORT")
	if httpPort == "" {
		httpPort = "8081"
	}
	grpcPort := os.Getenv("AUTH_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	// Создаём логгер один раз и передаём в сервисы.
	zapLog := logger.Must("auth")
	defer zapLog.Sync()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		zapLog.Info("starting HTTP auth server", zap.String("port", httpPort))
		if err := http.Run(zapLog); err != nil {
			zapLog.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	go func() {
		defer wg.Done()
		zapLog.Info("starting gRPC auth server", zap.String("port", grpcPort))
		if err := grpc.RunGRPCServer(grpcPort); err != nil {
			zapLog.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	wg.Wait()
}

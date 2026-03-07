package main

import (
	"os"

	"go.uber.org/zap"

	"pz1/services/tasks/internal/client/authclient"
	"pz1/services/tasks/internal/http"
	"pz1/shared/logger"
)

func main() {
	addr := os.Getenv("AUTH_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	// Создаём логгер один раз и передаём в сервисы.
	zapLog := logger.Must("tasks")
	defer zapLog.Sync()

	_, err := authclient.NewGRPCClient(addr)
	if err != nil {
		zapLog.Fatal("failed to connect to gRPC auth", zap.String("addr", addr), zap.Error(err))
	}

	zapLog.Info("connected to auth gRPC", zap.String("addr", addr))

	if err := http.Run(zapLog); err != nil {
		zapLog.Fatal("tasks server error", zap.Error(err))
	}
}

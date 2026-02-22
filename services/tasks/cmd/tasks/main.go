package main

import (
	"log"
	"os"

	"pz1/services/tasks/internal/client/authclient"
	"pz1/services/tasks/internal/http"
)

func main() {
	addr := os.Getenv("AUTH_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	_, err := authclient.NewGRPCClient(addr)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC Auth: %v", err)
	}

	log.Printf("Connected to Auth gRPC at %s", addr)
	if err := http.Run(); err != nil {
		log.Fatalf("Tasks server error: %v", err)
	}
}

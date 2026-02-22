package main

import (
	"log"
	"os"
	"sync"

	"pz1/services/auth/internal/grpc"
	"pz1/services/auth/internal/http"
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

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("Starting HTTP Auth server on :%s", httpPort)
		if err := http.Run(); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := grpc.RunGRPCServer(grpcPort); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	wg.Wait()
}

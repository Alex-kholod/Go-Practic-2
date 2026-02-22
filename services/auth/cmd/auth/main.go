package main

import (
	"log"
	"pz1/services/auth/internal/http"
)

func main() {
	log.Println("Starting auth service...")
	if err := http.Run(); err != nil {
		log.Fatalf("auth server error: %v", err)
	}
}

package main

import (
	"log"

	"pz1/services/tasks/internal/http"
)

func main() {
	log.Println("Starting Tasks service...")

	if err := http.Run(); err != nil {
		log.Fatalf("Tasks server error: %v", err)
	}
}

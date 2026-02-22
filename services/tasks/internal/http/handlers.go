package http

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"pz1/services/tasks/internal/client/authclient"
	"pz1/services/tasks/internal/service"
	"pz1/shared/middleware"
)

var store = service.NewStore()

func parseID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

func NewMux(client *authclient.GRPCClient) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		// Проверка авторизации
		if _, ok := service.VerifyAuth(w, r, client); !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			list := store.List()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)

		case http.MethodPost:
			var t service.Task
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
				return
			}
			if t.Title == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "title required"})
				return
			}
			res := store.Create(t)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(res)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		id := parseID(r.URL.Path)
		if id == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if _, ok := service.VerifyAuth(w, r, client); !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			if t, ok := store.Get(id); ok {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(t)
				return
			}
			w.WriteHeader(http.StatusNotFound)

		case http.MethodPatch:
			var patch service.Task
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
				return
			}
			if t, ok := store.Update(id, patch); ok {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(t)
				return
			}
			w.WriteHeader(http.StatusNotFound)

		case http.MethodDelete:
			if store.Delete(id) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	h := middleware.RequestID(middleware.Logging(mux))
	return h
}

func Run() error {
	port := os.Getenv("TASKS_PORT")
	if port == "" {
		port = "8082"
	}

	grpcAddr := os.Getenv("AUTH_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = "localhost:50051"
	}

	client, err := authclient.NewGRPCClient(grpcAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Auth gRPC: %v", err)
	}

	log.Printf("Tasks service listening on :%s, using auth gRPC %s", port, grpcAddr)
	addr := ":" + port
	return http.ListenAndServe(addr, NewMux(client))
}

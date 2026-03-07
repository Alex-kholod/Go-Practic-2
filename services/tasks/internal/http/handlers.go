package http

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"

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

func NewMux(log *zap.Logger, client *authclient.GRPCClient) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetRequestID(r)

		username, ok := service.VerifyAuth(w, r, client)
		if !ok {
			log.Warn("tasks: unauthorized",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)
			return
		}

		switch r.Method {
		case http.MethodGet:
			list := store.List()
			log.Info("tasks: list fetched",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("username", username),
				zap.Int("count", len(list)),
			)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)

		case http.MethodPost:
			var t service.Task
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				log.Warn("tasks: invalid json",
					zap.String("request_id", rid),
					zap.String("component", "handler"),
					zap.String("error", err.Error()),
				)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
				return
			}
			if t.Title == "" {
				log.Warn("tasks: missing title",
					zap.String("request_id", rid),
					zap.String("component", "handler"),
				)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "title required"})
				return
			}
			res := store.Create(t)
			log.Info("tasks: task created",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("username", username),
				zap.String("task_id", res.ID),
				zap.String("title", res.Title),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(res)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetRequestID(r)

		id := parseID(r.URL.Path)
		if id == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		username, ok := service.VerifyAuth(w, r, client)
		if !ok {
			log.Warn("tasks: unauthorized",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("task_id", id),
			)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if t, ok := store.Get(id); ok {
				log.Info("tasks: task fetched",
					zap.String("request_id", rid),
					zap.String("component", "handler"),
					zap.String("username", username),
					zap.String("task_id", id),
				)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(t)
				return
			}
			log.Warn("tasks: task not found",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("task_id", id),
			)
			w.WriteHeader(http.StatusNotFound)

		case http.MethodPatch:
			var patch service.Task
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				log.Warn("tasks: invalid json on patch",
					zap.String("request_id", rid),
					zap.String("component", "handler"),
					zap.String("error", err.Error()),
				)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
				return
			}
			if t, ok := store.Update(id, patch); ok {
				log.Info("tasks: task updated",
					zap.String("request_id", rid),
					zap.String("component", "handler"),
					zap.String("username", username),
					zap.String("task_id", id),
				)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(t)
				return
			}
			log.Warn("tasks: task not found on patch",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("task_id", id),
			)
			w.WriteHeader(http.StatusNotFound)

		case http.MethodDelete:
			if store.Delete(id) {
				log.Info("tasks: task deleted",
					zap.String("request_id", rid),
					zap.String("component", "handler"),
					zap.String("username", username),
					zap.String("task_id", id),
				)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			log.Warn("tasks: task not found on delete",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("task_id", id),
			)
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	h := middleware.RequestID(middleware.AccessLog(log)(mux))
	return h
}

func Run(log *zap.Logger) error {
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
		log.Fatal("failed to connect to auth gRPC",
			zap.String("addr", grpcAddr),
			zap.Error(err),
		)
	}

	log.Info("tasks HTTP server listening",
		zap.String("port", port),
		zap.String("auth_grpc", grpcAddr),
	)
	return http.ListenAndServe(":"+port, NewMux(log, client))
}

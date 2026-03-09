package http

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"pz1/services/tasks/internal/client/authclient"
	"pz1/services/tasks/internal/repository"
	"pz1/services/tasks/internal/service"
	"pz1/shared/middleware"
)

func parseID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

func writeError(w http.ResponseWriter, log *zap.Logger, rid, component, internal string, status int) {
	log.Error("internal error",
		zap.String("request_id", rid),
		zap.String("component", component),
		zap.String("error", internal),
	)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
}

func NewMux(log *zap.Logger, client *authclient.GRPCClient, repo repository.TaskRepository) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	// ── /v1/tasks/search ────────────────────────────────────────────────────
	mux.HandleFunc("/v1/tasks/search", func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetRequestID(r)

		if _, ok := service.VerifyAuth(w, r, client); !ok {
			log.Warn("tasks: unauthorized search",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
			)
			return
		}

		title := r.URL.Query().Get("title")
		tasks, err := repo.Search(r.Context(), title)
		if err != nil {
			writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("tasks: search completed",
			zap.String("request_id", rid),
			zap.String("component", "handler"),
			zap.String("query", title),
			zap.Int("count", len(tasks)),
		)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})

	// ── /v1/tasks ───────────────────────────────────────────────────────────
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetRequestID(r)

		username, ok := service.VerifyAuth(w, r, client)
		if !ok {
			log.Warn("tasks: unauthorized",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("method", r.Method),
			)
			return
		}

		switch r.Method {
		case http.MethodGet:
			list, err := repo.List(r.Context())
			if err != nil {
				writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
				return
			}
			log.Info("tasks: list fetched",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("username", username),
				zap.Int("count", len(list)),
			)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)

		case http.MethodPost:
			var t repository.Task
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
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "title required"})
				return
			}

			// ── XSS санитизация ────────────────────────────────────────────
			// Заменяем < > " ' на HTML-сущности перед сохранением в БД.
			// Так даже если данные попадут в HTML-страницу — они не выполнятся.
			t.Title = middleware.SanitizeText(t.Title)
			t.Description = middleware.SanitizeText(t.Description)

			res, err := repo.Create(r.Context(), t)
			if err != nil {
				writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
				return
			}
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

	// ── /v1/tasks/{id} ──────────────────────────────────────────────────────
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
			t, err := repo.Get(r.Context(), id)
			if err != nil {
				if err.Error() == "task not found" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
				return
			}
			log.Info("tasks: task fetched",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("username", username),
				zap.String("task_id", id),
			)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(t)

		case http.MethodPatch:
			var patch repository.Task
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
				return
			}
			// XSS санитизация при обновлении.
			patch.Title = middleware.SanitizeText(patch.Title)
			patch.Description = middleware.SanitizeText(patch.Description)

			t, err := repo.Update(r.Context(), id, patch)
			if err != nil {
				if err.Error() == "task not found" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
				return
			}
			log.Info("tasks: task updated",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("username", username),
				zap.String("task_id", id),
			)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(t)

		case http.MethodDelete:
			if err := repo.Delete(r.Context(), id); err != nil {
				if err.Error() == "task not found" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
				return
			}
			log.Info("tasks: task deleted",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
				zap.String("username", username),
				zap.String("task_id", id),
			)
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Цепочка: RequestID → Metrics → SecurityHeaders → CSRF → AccessLog → handlers
	h := middleware.RequestID(
		middleware.Metrics("tasks")(
			middleware.SecurityHeaders(
				middleware.CSRF(log)(
					middleware.AccessLog(log)(mux),
				),
			),
		),
	)
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
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/tasks?sslmode=disable"
	}

	repo, err := repository.New(dsn, log)
	if err != nil {
		log.Fatal("failed to connect to database",
			zap.String("component", "repository"),
			zap.Error(err),
		)
	}
	log.Info("connected to database")

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
	return http.ListenAndServe(":"+port, NewMux(log, client, repo))
}

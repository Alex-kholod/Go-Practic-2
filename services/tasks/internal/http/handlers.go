package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"pz1/services/tasks/internal/cache"
	"pz1/services/tasks/internal/client/authclient"
	"pz1/services/tasks/internal/service"
	"pz1/shared/middleware"
	"pz1/shared/rabbit"
	"pz1/shared/repository"
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

func NewMux(
	log *zap.Logger,
	client *authclient.GRPCClient,
	repo repository.TaskRepository,
	c *cache.Cache,
	producer *rabbit.Producer,
	jobProducer *rabbit.JobProducer,
) http.Handler {
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "tasks-unknown"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Instance-ID", instanceID)
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "ok",
			"instance_id": instanceID,
		})
	})

	mux.Handle("/metrics", promhttp.Handler())

	// ── POST /v1/jobs/process-task ───────────────────────────────────────────
	// Ставит задачу в очередь task_jobs для "тяжёлой" обработки worker'ом.
	mux.HandleFunc("/v1/jobs/process-task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		rid := middleware.GetRequestID(r)
		w.Header().Set("X-Instance-ID", instanceID)

		if _, ok := service.VerifyAuth(w, r, client); !ok {
			log.Warn("jobs: unauthorized",
				zap.String("request_id", rid),
				zap.String("component", "handler"),
			)
			return
		}

		var req struct {
			TaskID string `json:"task_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TaskID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "task_id required"})
			return
		}

		if jobProducer == nil {
			writeError(w, log, rid, "job_producer", "job producer not available", http.StatusServiceUnavailable)
			return
		}

		job := rabbit.JobMessage{
			Job:       "process_task",
			TaskID:    req.TaskID,
			Attempt:   1,
			MessageID: uuid.NewString(), // уникальный ID для идемпотентности
			RequestID: rid,
		}

		if err := jobProducer.PublishJob(r.Context(), job); err != nil {
			writeError(w, log, rid, "job_producer", err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("jobs: job enqueued",
			zap.String("request_id", rid),
			zap.String("component", "handler"),
			zap.String("task_id", req.TaskID),
			zap.String("message_id", job.MessageID),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202 — принято в обработку
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "accepted",
			"message_id": job.MessageID,
			"task_id":    req.TaskID,
		})
	})

	// ── /v1/tasks/search ────────────────────────────────────────────────────
	mux.HandleFunc("/v1/tasks/search", func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetRequestID(r)
		w.Header().Set("X-Instance-ID", instanceID)
		if _, ok := service.VerifyAuth(w, r, client); !ok {
			return
		}
		title := r.URL.Query().Get("title")
		tasks, err := repo.Search(r.Context(), title)
		if err != nil {
			writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})

	// ── /v1/tasks ───────────────────────────────────────────────────────────
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetRequestID(r)
		w.Header().Set("X-Instance-ID", instanceID)

		username, ok := service.VerifyAuth(w, r, client)
		if !ok {
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
				zap.String("username", username),
				zap.Int("count", len(list)),
				zap.String("instance", instanceID),
			)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)

		case http.MethodPost:
			var t repository.Task
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
			t.Title = middleware.SanitizeText(t.Title)
			t.Description = middleware.SanitizeText(t.Description)

			res, err := repo.Create(r.Context(), t)
			if err != nil {
				writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
				return
			}
			if producer != nil {
				producer.Publish(r.Context(), rabbit.TaskEvent{
					Event: "task.created", TaskID: res.ID,
					RequestID: rid, Producer: instanceID,
				})
			}
			if c != nil {
				c.InvalidateList(r.Context())
			}
			log.Info("tasks: task created",
				zap.String("request_id", rid),
				zap.String("username", username),
				zap.String("task_id", res.ID),
				zap.String("instance", instanceID),
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
		w.Header().Set("X-Instance-ID", instanceID)
		id := parseID(r.URL.Path)
		if id == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		username, ok := service.VerifyAuth(w, r, client)
		if !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			if c != nil {
				if task, hit := c.GetTask(r.Context(), id); hit {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(task)
					return
				}
			}
			t, err := repo.Get(r.Context(), id)
			if err != nil {
				if err.Error() == "task not found" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeError(w, log, rid, "repository", err.Error(), http.StatusInternalServerError)
				return
			}
			if c != nil {
				c.SetTask(r.Context(), t)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(t)

		case http.MethodPatch:
			var patch repository.Task
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
				return
			}
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
			if c != nil {
				c.DeleteTask(r.Context(), id)
				c.InvalidateList(r.Context())
			}
			if producer != nil {
				producer.Publish(r.Context(), rabbit.TaskEvent{
					Event: "task.updated", TaskID: id,
					RequestID: rid, Producer: instanceID,
				})
			}
			log.Info("tasks: task updated",
				zap.String("request_id", rid),
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
			if c != nil {
				c.DeleteTask(r.Context(), id)
				c.InvalidateList(r.Context())
			}
			if producer != nil {
				producer.Publish(r.Context(), rabbit.TaskEvent{
					Event: "task.deleted", TaskID: id,
					RequestID: rid, Producer: instanceID,
				})
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

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

	redisAddrs := strings.Split(os.Getenv("REDIS_ADDRS"), ",")
	if len(redisAddrs) == 0 || redisAddrs[0] == "" {
		redisAddrs = []string{"localhost:7001", "localhost:7002", "localhost:7003"}
	}
	baseTTL := 120 * time.Second
	if v := os.Getenv("CACHE_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			baseTTL = time.Duration(n) * time.Second
		}
	}
	jitterMax := 30 * time.Second
	if v := os.Getenv("CACHE_TTL_JITTER_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			jitterMax = time.Duration(n) * time.Second
		}
	}
	var c *cache.Cache
	c, err := cache.New(redisAddrs, os.Getenv("REDIS_PASSWORD"), baseTTL, jitterMax, log)
	if err != nil {
		log.Warn("redis unavailable", zap.String("error", err.Error()))
	} else {
		log.Info("connected to redis cluster")
	}

	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	var producer *rabbit.Producer
	var jobProducer *rabbit.JobProducer

	for {
		producer, err = rabbit.NewProducer(rabbitURL, log)
		if err == nil {
			break
		}

		log.Warn("rabbitmq not ready, retrying...",
			zap.String("error", err.Error()),
		)

		time.Sleep(3 * time.Second)
	}

	if producer != nil {
		jobProducer, err = rabbit.NewJobProducer(rabbitURL, log)
		if err != nil {
			log.Warn("job producer unavailable", zap.String("error", err.Error()))
		} else {
			defer jobProducer.Close()
		}
	}

	repo, repoErr := repository.New(dsn, log)
	if repoErr != nil {
		log.Fatal("failed to connect to database", zap.Error(repoErr))
	}
	log.Info("connected to database")

	authClient, authErr := authclient.NewGRPCClient(grpcAddr)
	if authErr != nil {
		log.Fatal("failed to connect to auth gRPC", zap.Error(authErr))
	}

	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "tasks-unknown"
	}
	log.Info("tasks HTTP server listening",
		zap.String("port", port),
		zap.String("instance", instanceID),
	)

	// Подавляем неиспользуемый импорт fmt.
	_ = fmt.Sprintf

	return http.ListenAndServe(":"+port, NewMux(log, authClient, repo, c, producer, jobProducer))
}

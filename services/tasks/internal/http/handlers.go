package http

import (
	"encoding/json"
	"io"
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

func NewMux(authBase string) http.Handler {
	mux := http.NewServeMux()
	client := authclient.New(authBase, 3*1e9) // 3s timeout

	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if _, ok := service.VerifyAuth(w, r, client); !ok {
				return
			}
			list := store.List()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)

		case http.MethodPost:
			if _, ok := service.VerifyAuth(w, r, client); !ok {
				return
			}
			var t service.Task
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &t); err != nil {
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
	authBase := os.Getenv("AUTH_BASE_URL")
	if authBase == "" {
		authBase = "http://localhost:8081"
	}
	log.Printf("Tasks service listening on :%s, using auth %s", port, authBase)
	addr := ":" + port
	return http.ListenAndServe(addr, NewMux(authBase))
}

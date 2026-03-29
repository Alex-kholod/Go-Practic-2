package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"pz1/services/tasks/internal/client/authclient"
)

type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Done        bool   `json:"done"`
}

type Store struct {
	mu    sync.RWMutex
	items map[string]Task
	next  int
}

func NewStore() *Store {
	return &Store{items: make(map[string]Task), next: 1}
}

func (s *Store) Create(t Task) Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("t_%03d", s.next)
	t.ID = id
	s.next++
	s.items[id] = t
	return t
}

func (s *Store) List() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Task, 0, len(s.items))
	for _, v := range s.items {
		out = append(out, v)
	}
	return out
}

func (s *Store) Get(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	return v, ok
}

func (s *Store) Update(id string, patch Task) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.items[id]
	if !ok {
		return Task{}, false
	}
	if patch.Title != "" {
		cur.Title = patch.Title
	}

	if patch.Description != "" {
		cur.Description = patch.Description
	}
	if patch.DueDate != "" {
		cur.DueDate = patch.DueDate
	}
	cur.Done = patch.Done
	s.items[id] = cur
	return cur, true
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	return true
}

func VerifyAuth(w http.ResponseWriter, r *http.Request, client *authclient.GRPCClient) (string, bool) {
	auth := r.Header.Get("Authorization")
	ctx := r.Context()

	subject, status, err := client.VerifyToken(ctx, auth)

	if err != nil {
		if status == 401 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return "", false
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "auth grpc unavailable"})
		return "", false
	}
	return subject, true
}

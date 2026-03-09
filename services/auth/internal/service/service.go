package service

import (
	"encoding/json"
	"net/http"
)

const (
	DemoToken       = "demo-token"
	DemoUsername    = "student"
	DemoPassword    = "student"
	AuthHeaderToken = "Bearer " + DemoToken
)

// для выдачи cookies.
func LoginCheck(r *http.Request) (string, bool) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", false
	}
	if body.Username != DemoUsername || body.Password != DemoPassword {
		return "", false
	}
	return body.Username, true
}

// Возвращает access_token в теле ответа (Bearer-схема).
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	if body.Username != DemoUsername || body.Password != DemoPassword {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": DemoToken,
		"token_type":   "Bearer",
	})
}

// VerifyHandler проверяет Bearer-токен из заголовка Authorization.
func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" || auth != AuthHeaderToken {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"valid": false, "error": "unauthorized"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"valid": true, "subject": DemoUsername})
}

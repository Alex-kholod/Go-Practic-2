package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxKeyRequestID struct{}

func RequestIDKey() ctxKeyRequestID {
	return ctxKeyRequestID{}
}

func GetRequestID(r *http.Request) string {
	v := r.Context().Value(RequestIDKey())
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func GetRequestIDFromCtx(ctx context.Context) string {
	if s, ok := ctx.Value(RequestIDKey()).(string); ok {
		return s
	}
	return ""
}

func genID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "rid-unknown"
	}
	return hex.EncodeToString(b)
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = genID()
		}

		// Кладём ID в контекст, чтобы он был доступен всем обработчикам.
		ctx := context.WithValue(r.Context(), RequestIDKey(), rid)

		w.Header().Set("X-Request-ID", rid)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

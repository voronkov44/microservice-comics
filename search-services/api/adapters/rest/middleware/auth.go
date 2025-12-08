package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func writeUnauthed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("unauthorized"))
}

func NewLoginHandler(log *slog.Logger, adminUser, adminPassword string, tokenTTL time.Duration) http.Handler {
	j := NewJWT(tokenTTL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Debug("failed to close response body in Get", "error", err)
			}
		}()

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Name != adminUser || req.Password != adminPassword {
			writeUnauthed(w)
			return
		}

		token, err := j.GenerateSuperuserToken()
		if err != nil {
			log.Error("failed to generate token", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(token))
	})
}

func RequireSuperuser(next http.Handler, tokenTTL time.Duration) http.Handler {
	j := NewJWT(tokenTTL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeUnauthed(w)
			return
		}

		if !strings.HasPrefix(authHeader, "Token ") {
			writeUnauthed(w)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Token ")
		tokenStr = strings.TrimSpace(tokenStr)
		if tokenStr == "" {
			writeUnauthed(w)
			return
		}

		if !j.IsSuperuserToken(tokenStr) {
			writeUnauthed(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

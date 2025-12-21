package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

// UserJWTClaims - минимальный набор для JWT
// (только user_id, jwt.RegisteredClaims из коробки выдает поля sub, exp, iat)
type UserJWTClaims struct {
	UserID uint32 `json:"user_id"`
	jwt.RegisteredClaims
}

func RequireUser(next http.Handler, jwtSecret string) http.Handler {
	secret := []byte(jwtSecret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr, ok := readToken(r)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}

		claims := &UserJWTClaims{}
		t, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			func(token *jwt.Token) (interface{}, error) {
				// алгоритм HS256 (sha-256)
				if token.Method != jwt.SigningMethodHS256 {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return secret, nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		)

		if err != nil || !t.Valid || claims.UserID == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (uint32, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(uint32)
	return id, ok
}

func readToken(r *http.Request) (string, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Token ") {
		return "", false
	}

	tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Token "))
	if tokenStr == "" {
		return "", false
	}

	return tokenStr, true
}

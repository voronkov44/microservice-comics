package middleware

import (
	"golang.org/x/time/rate"
	"net/http"
)

// WithConcurrencyLimit ограничиваем количество одновременных запросов
func WithConcurrencyLimit(next http.Handler, max int) http.Handler {
	if max <= 0 {
		return next
	}

	sem := make(chan struct{}, max)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		}
	})
}

// WithRateLimit ограничиваем скорость запросов к хендлеру
func WithRateLimit(next http.Handler, rps int) http.Handler {
	if rps <= 0 {
		return next
	}
	lim := rate.NewLimiter(rate.Limit(rps), 1)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := lim.Wait(r.Context()); err != nil {
			return
		}
		next.ServeHTTP(w, r)
	})
}

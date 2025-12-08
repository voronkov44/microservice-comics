package rest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"yadro.com/course/api/pkg/res"

	"yadro.com/course/api/core"
)

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		replies := make(map[string]string, len(pingers))
		for name, p := range pingers {
			if err := p.Ping(ctx); err != nil {
				replies[name] = "unavailable"
				log.Warn("ping failed", "service", name, "error", err)
			} else {
				replies[name] = "ok"
			}
		}

		res.Json(w, pingResponse{Replies: replies}, http.StatusOK)
		log.Info("ping handled", "replies", replies, "duration", time.Since(start))
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := context.Background()

		if err := updater.Update(ctx); err != nil {
			switch {
			case errors.Is(err, core.ErrAlreadyExists):
				// идемпотентный повтор - задача уже запущена
				res.Json(w, updateStatusResponse{Status: "already running"}, http.StatusAccepted)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("update failed", slog.Any("err", err))
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, updateStatusResponse{Status: "started"}, http.StatusOK)
		log.Info("update started", "duration", time.Since(start))
	}
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		st, err := updater.Stats(ctx)
		if err != nil {
			if errors.Is(err, core.ErrUnavailable) {
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			} else {
				log.Error("stats failed", slog.Any("err", err))
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, updateStatsResponse{
			WordsTotal:    st.WordsTotal,
			WordsUnique:   st.WordsUnique,
			ComicsFetched: st.ComicsFetched,
			ComicsTotal:   st.ComicsTotal,
		}, http.StatusOK)

		log.Info(
			"stats ok",
			"words_total", st.WordsTotal,
			"words_unique", st.WordsUnique,
			"comics_fetched", st.ComicsFetched,
			"comics_total", st.ComicsTotal,
			"duration", time.Since(start),
		)
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		st, err := updater.Status(ctx)
		if err != nil {
			if errors.Is(err, core.ErrUnavailable) {
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			} else {
				log.Error("status failed", slog.Any("err", err))
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, updateStatusResponse{Status: string(st)}, http.StatusOK)
		log.Info("status ok", "status", st, "duration", time.Since(start))
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		if err := updater.Drop(ctx); err != nil {
			if errors.Is(err, core.ErrUnavailable) {
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			} else {
				log.Error("drop failed", slog.Any("err", err))
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		log.Info("drop ok", "duration", time.Since(start))
	}
}

func NewSearchHandler(log *slog.Logger, search core.Searcher, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		q := r.URL.Query()

		phrase := q.Get("phrase")

		var limit uint32

		if limitStr := q.Get("limit"); limitStr != "" {
			n, err := strconv.ParseUint(limitStr, 10, 32)
			if err != nil {
				res.Json(w, errorResponse{Error: "bad limit"}, http.StatusBadRequest)
				return
			}
			limit = uint32(n)
		}

		result, err := search.Find(ctx, phrase, limit)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("search failed", "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		comics := make([]comicResponse, 0, len(result.Comics))
		for _, cmt := range result.Comics {
			comics = append(comics, comicResponse{
				ID:  cmt.ID,
				URL: cmt.URL,
			})
		}

		res.Json(w, searchResponse{
			Comics: comics,
			Total:  result.Total,
		}, http.StatusOK)

		log.Info(
			"search ok",
			"phrase", phrase,
			"limit", limit,
			"total", result.Total,
			"duration", time.Since(start),
		)
	}
}

func NewIndexedSearchHandler(log *slog.Logger, search core.Searcher, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		q := r.URL.Query()
		phrase := q.Get("phrase")

		var limit uint32
		if limitStr := q.Get("limit"); limitStr != "" {
			n, err := strconv.ParseUint(limitStr, 10, 32)
			if err != nil {
				res.Json(w, errorResponse{Error: "bad limit"}, http.StatusBadRequest)
				return
			}
			limit = uint32(n)
		}

		result, err := search.IndexedSearch(ctx, phrase, limit)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("indexed search failed", "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		comics := make([]comicResponse, 0, len(result.Comics))
		for _, cmt := range result.Comics {
			comics = append(comics, comicResponse{
				ID:  cmt.ID,
				URL: cmt.URL,
			})
		}

		res.Json(w, searchResponse{
			Comics: comics,
			Total:  result.Total,
		}, http.StatusOK)

		log.Info(
			"indexed search ok",
			"phrase", phrase,
			"limit", limit,
			"total", result.Total,
			"duration", time.Since(start),
		)
	}
}

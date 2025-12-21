package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"yadro.com/course/api/adapters/rest/middleware"
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

// UPDATE HANDLERS

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

// SEARCH HANDLERS

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

// SEARCH COMICS HANDLERS
// get comics by id
// get all(list) comics
// get random comic

func NewComicByIDHandler(log *slog.Logger, search core.Searcher, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		idStr := r.PathValue("id")
		if idStr == "" {
			res.Json(w, errorResponse{Error: "id is required"}, http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			res.Json(w, errorResponse{Error: "invalid id"}, http.StatusBadRequest)
			return
		}

		comic, err := search.GetComic(ctx, id)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "comic not found"}, http.StatusNotFound)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("get comic by id failed", "id", id, "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, comicResponse{ID: comic.ID, URL: comic.URL}, http.StatusOK)

		log.Info("comic fetched by id",
			"id", id,
			"duration", time.Since(start),
		)
	}
}

func NewComicsListHandler(log *slog.Logger, search core.Searcher, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		q := r.URL.Query()

		page := uint32(1)
		limit := uint32(10)

		if pageStr := q.Get("page"); pageStr != "" {
			n, err := strconv.ParseUint(pageStr, 10, 32)
			if err != nil || n == 0 {
				res.Json(w, errorResponse{Error: "bad page"}, http.StatusBadRequest)
				return
			}
			page = uint32(n)
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			n, err := strconv.ParseUint(limitStr, 10, 32)
			if err != nil || n == 0 {
				res.Json(w, errorResponse{Error: "bad limit"}, http.StatusBadRequest)
				return
			}
			limit = uint32(n)
		}

		result, err := search.ListComics(ctx, page, limit)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("list comics failed", "error", err)
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

		log.Info("comics page ok",
			"page", page,
			"limit", limit,
			"total", result.Total,
			"duration", time.Since(start),
		)
	}
}

func NewRandomComicHandler(log *slog.Logger, search core.Searcher, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		comic, err := search.RandomComic(ctx)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("get random comic failed", "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, comicResponse{ID: comic.ID, URL: comic.URL}, http.StatusOK)

		log.Info("random comic fetched",
			"id", comic.ID,
			"duration", time.Since(start),
		)
	}
}

// AUTH HANDLERS
// Registers
// Login
// Telegram login

func NewRegisterHandler(log *slog.Logger, auth core.Auth, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			res.Json(w, errorResponse{Error: "email and password must not be empty"}, http.StatusBadRequest)
			return
		}

		token, err := auth.Register(ctx, req.Email, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrInvalidEmail):
				res.Json(w, errorResponse{Error: "invalid email format"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrAlreadyExists):
				res.Json(w, errorResponse{Error: "user already exists"}, http.StatusConflict)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("register failed", "email", req.Email, "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, tokenResponse{Token: token}, http.StatusOK)

		log.Info(
			"user registered",
			"email", req.Email,
			"duration", time.Since(start),
		)
	}
}

func NewUserLoginHandler(log *slog.Logger, auth core.Auth, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			res.Json(w, errorResponse{Error: "email and password must not be empty"}, http.StatusBadRequest)
			return
		}

		token, err := auth.Login(ctx, req.Email, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrInvalidEmail):
				res.Json(w, errorResponse{Error: "invalid email format"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrInvalidCredentials):
				res.Json(w, errorResponse{Error: "invalid credentials"}, http.StatusUnauthorized)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("user login failed", "email", req.Email, "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, tokenResponse{Token: token}, http.StatusOK)

		log.Info(
			"user login ok",
			"email", req.Email,
			"duration", time.Since(start),
		)
	}
}

func NewBotTelegramLoginHandler(log *slog.Logger, auth core.Auth, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		var req botTelegramLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			return
		}
		if req.TgID == 0 {
			res.Json(w, errorResponse{Error: "tg_id is required"}, http.StatusBadRequest)
			return
		}

		token, err := auth.BotLoginTelegram(ctx, core.TelegramProfile{
			TgID:      req.TgID,
			Username:  req.Username,
			FirstName: req.FirstName,
			LastName:  req.LastName,
		})
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("bot telegram login failed", "tg_id", req.TgID, "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		res.Json(w, tokenResponse{Token: token}, http.StatusOK)
		log.Info("bot telegram login ok", "tg_id", req.TgID, "duration", time.Since(start))
	}
}

// FAVORITES HANDLERS
// list comics
// add comic
// delete comic

func NewFavoritesListHandler(log *slog.Logger, fav core.Favorites, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok || userID == 0 {
			res.Json(w, errorResponse{Error: "unauthorized"}, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		items, err := fav.List(ctx, userID)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("favorites list failed", "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		resp := favoritesListResponse{Items: make([]favoriteItemResponse, 0, len(items))}
		for _, it := range items {
			resp.Items = append(resp.Items, favoriteItemResponse{
				ComicID:       it.ComicID,
				CreatedAtUnix: it.CreatedAtUnix,
			})
		}

		res.Json(w, resp, http.StatusOK)
		log.Info("favorites list ok", "user_id", userID, "count", len(items), "duration", time.Since(start))
	}
}

func NewFavoritesAddHandler(log *slog.Logger, fav core.Favorites, search core.Searcher, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok || userID == 0 {
			res.Json(w, errorResponse{Error: "unauthorized"}, http.StatusUnauthorized)
			return
		}

		idStr := r.PathValue("id")
		comicID, err := strconv.Atoi(idStr)
		if err != nil || comicID <= 0 {
			res.Json(w, errorResponse{Error: "invalid id"}, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		// проверяем, что комикс существует
		_, err = search.GetComic(ctx, comicID)
		if err != nil {
			switch {
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "comic not found"}, http.StatusNotFound)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("get comic before favorite add failed", "comic_id", comicID, "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		// сохраняем
		if err := fav.Add(ctx, userID, int32(comicID)); err != nil {
			switch {
			case errors.Is(err, core.ErrAlreadyExists):
				res.Json(w, errorResponse{Error: "already exists"}, http.StatusConflict)
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("favorites add failed", "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusNoContent)
		log.Info("favorites add ok", "user_id", userID, "comic_id", comicID, "duration", time.Since(start))
	}
}

func NewFavoritesDeleteHandler(log *slog.Logger, fav core.Favorites, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok || userID == 0 {
			res.Json(w, errorResponse{Error: "unauthorized"}, http.StatusUnauthorized)
			return
		}

		idStr := r.PathValue("id")
		comicID, err := strconv.Atoi(idStr)
		if err != nil || comicID <= 0 {
			res.Json(w, errorResponse{Error: "invalid id"}, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		if err := fav.Delete(ctx, userID, int32(comicID)); err != nil {
			switch {
			case errors.Is(err, core.ErrNotFound):
				res.Json(w, errorResponse{Error: "not found"}, http.StatusNotFound)
			case errors.Is(err, core.ErrBadArguments):
				res.Json(w, errorResponse{Error: "bad request"}, http.StatusBadRequest)
			case errors.Is(err, core.ErrUnavailable):
				res.Json(w, errorResponse{Error: "dependency unavailable"}, http.StatusServiceUnavailable)
			default:
				log.Error("favorites delete failed", "error", err)
				res.Json(w, errorResponse{Error: "internal error"}, http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusNoContent)
		log.Info("favorites delete ok", "user_id", userID, "comic_id", comicID, "duration", time.Since(start))
	}
}

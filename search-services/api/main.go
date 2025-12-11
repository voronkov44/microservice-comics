package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"yadro.com/course/api/adapters/auth"
	"yadro.com/course/api/adapters/rest/middleware"
	"yadro.com/course/api/adapters/search"
	"yadro.com/course/api/adapters/words"
	"yadro.com/course/api/core"

	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/adapters/update"
	"yadro.com/course/api/config"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting server")
	log.Debug("debug messages are enabled")

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		log.Error("cannot init update adapter", "error", err)
		os.Exit(1)
	}
	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		log.Error("cannot init words adapter", "error", err)
		os.Exit(1)
	}
	searchClient, err := search.NewClient(cfg.SearchAddress, log)
	if err != nil {
		log.Error("cannot init search adapter", "error", err)
		os.Exit(1)
	}
	authClient, err := auth.NewClient(cfg.AuthAddress, log)
	if err != nil {
		log.Error("cannot init auth adapter", "error", err)
		os.Exit(1)
	}

	// приведение типов для компилятора
	pingmap := map[string]core.Pinger{
		"words":  wordsClient,
		"update": updateClient,
		"search": searchClient,
		"auth":   authClient,
	}

	mux := http.NewServeMux()

	mux.Handle("GET /api/ping", rest.NewPingHandler(log, pingmap, cfg.HTTPConfig.Timeout))

	// login
	mux.Handle("POST /api/login", middleware.NewLoginHandler(log, cfg.AdminUser, cfg.AdminPassword, cfg.TokenTTL))

	// search api
	searchHandler := rest.NewSearchHandler(log, searchClient, cfg.HTTPConfig.Timeout)
	mux.Handle("GET /api/search",
		middleware.WithConcurrencyLimit(searchHandler, cfg.SearchConcurrency),
	)

	isearchHandler := rest.NewIndexedSearchHandler(log, searchClient, cfg.HTTPConfig.Timeout)
	mux.Handle("GET /api/isearch",
		middleware.WithRateLimit(isearchHandler, cfg.SearchRate),
	)

	// update api
	mux.Handle("POST /api/db/update",
		middleware.RequireSuperuser(rest.NewUpdateHandler(log, updateClient), cfg.TokenTTL),
	)
	mux.Handle("GET /api/db/stats",
		rest.NewUpdateStatsHandler(log, updateClient, cfg.HTTPConfig.Timeout),
	)
	mux.Handle("GET /api/db/status",
		rest.NewUpdateStatusHandler(log, updateClient, cfg.HTTPConfig.Timeout),
	)
	mux.Handle("DELETE /api/db",
		middleware.RequireSuperuser(rest.NewDropHandler(log, updateClient, cfg.HTTPConfig.Timeout), cfg.TokenTTL),
	)

	// auth api
	mux.Handle("POST /api/auth/register",
		rest.NewRegisterHandler(log, authClient, cfg.HTTPConfig.Timeout),
	)
	mux.Handle("POST /api/auth/login",
		rest.NewUserLoginHandler(log, authClient, cfg.HTTPConfig.Timeout),
	)

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}

		_ = updateClient.Close()
		_ = wordsClient.Close()
		_ = searchClient.Close()
		_ = authClient.Close()
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server closed unexpectedly", "error", err)
			return
		}
	}
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

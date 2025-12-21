package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"yadro.com/course/auth/adapters/db"
	authgrpc "yadro.com/course/auth/adapters/grpc"
	"yadro.com/course/auth/config"
	"yadro.com/course/auth/core"
	authpb "yadro.com/course/proto/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	// config
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "auth server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	// logger
	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting auth server")
	log.Debug("debug messages are enabled")

	// database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}

	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %v", err)
	}

	// service
	authorization, err := core.NewService(log, storage, cfg.JWTSecret, cfg.TokenTTL)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %v", err)
	}

	// grpc
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	authpb.RegisterAuthServer(s, authgrpc.NewServer(log, authorization))
	reflection.Register(s)

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down auth server")
		s.GracefulStop()
	}()

	log.Info("auth gRPC server is running", "address", cfg.Address)

	// blocking
	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

func mustMakeLogger(levelStr string) *slog.Logger {
	var level slog.Level
	switch levelStr {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

package core

import (
	"context"
	"log/slog"
)

type Service struct {
	log *slog.Logger
	db  DB
}

func NewService(log *slog.Logger, db DB) *Service {
	return &Service{log: log, db: db}
}

func (s *Service) Add(ctx context.Context, userID uint32, comicID int32) error {
	if userID == 0 || comicID <= 0 {
		return ErrInvalidArgs
	}
	return s.db.Add(ctx, userID, comicID)
}

func (s *Service) Delete(ctx context.Context, userID uint32, comicID int32) error {
	if userID == 0 || comicID <= 0 {
		return ErrInvalidArgs
	}
	return s.db.Delete(ctx, userID, comicID)
}

func (s *Service) List(ctx context.Context, userID uint32) ([]Favorite, error) {
	if userID == 0 {
		return nil, ErrInvalidArgs
	}
	return s.db.List(ctx, userID)
}

func (s *Service) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

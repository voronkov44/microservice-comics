package core

import (
	"context"
)

type Search interface {
	Find(ctx context.Context, phrase string, limit uint32) ([]Comics, error)
	IndexedSearch(ctx context.Context, phrase string, limit uint32) ([]Comics, uint32, error)
	Ping(ctx context.Context) error

	GetComicByID(ctx context.Context, id int) (Comics, error)
	RandomComic(ctx context.Context) (Comics, error)
	GetAllComics(ctx context.Context, page, limit uint32) ([]Comics, uint32, error)
}

type DB interface {
	Find(ctx context.Context, tokens []string) ([]Comics, error)
	All(ctx context.Context) ([]Comics, error)
	Ping(ctx context.Context) error

	GetByID(ctx context.Context, id int) (Comics, error)
	GetAll(ctx context.Context, offset, limit int) ([]Comics, error)
	Count(ctx context.Context) (int, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

package core

import (
	"context"
)

type Search interface {
	Find(ctx context.Context, phrase string, limit uint32) ([]Comics, error)
	IndexedSearch(ctx context.Context, phrase string, limit uint32) ([]Comics, uint32, error)
	Ping(ctx context.Context) error
}

type DB interface {
	Find(ctx context.Context, tokens []string) ([]Comics, error)
	All(ctx context.Context) ([]Comics, error)
	Ping(ctx context.Context) error
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

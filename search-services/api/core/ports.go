package core

import "context"

type Normalizer interface {
	Norm(context.Context, string) ([]string, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
	Ping(ctx context.Context) error
}

type Searcher interface {
	Find(ctx context.Context, phrase string, limit uint32) (SearchResult, error)
	IndexedSearch(ctx context.Context, phrase string, limit uint32) (SearchResult, error)
	Ping(ctx context.Context) error

	GetComic(ctx context.Context, id int) (SearchComic, error)
	RandomComic(ctx context.Context) (SearchComic, error)
	ListComics(ctx context.Context, page, limit uint32) (SearchResult, error)
}

type Auth interface {
	Register(ctx context.Context, email, password string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	Ping(ctx context.Context) error
}

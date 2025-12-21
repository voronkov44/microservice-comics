package core

import "context"

type DB interface {
	Add(ctx context.Context, userID uint32, comicID int32) error
	Delete(ctx context.Context, userID uint32, comicID int32) error
	List(ctx context.Context, userID uint32) ([]Favorite, error)
	Ping(ctx context.Context) error
}

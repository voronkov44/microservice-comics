package core

import "context"

type DB interface {
	CreateUser(ctx context.Context, users Users) (Users, error)
	GetUserByEmail(ctx context.Context, email string) (Users, error)
	Ping(ctx context.Context) error
}

package core

import "context"

type DB interface {
	CreateComicshubUser(ctx context.Context, profile ComicsHubProfile) (User, error)
	GetComicshubByEmail(ctx context.Context, email string) (User, string, error)
	UpsertTelegramUser(ctx context.Context, tg TelegramProfile) (User, error)

	Ping(ctx context.Context) error
}

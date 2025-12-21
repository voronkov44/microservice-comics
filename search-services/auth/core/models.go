package core

import "time"

type User struct {
	ID        int64
	CreatedAt time.Time
}

type TelegramProfile struct {
	TgID      int64
	Username  string
	FirstName string
	LastName  string
}

type ComicsHubProfile struct {
	Email        string
	PasswordHash string
}

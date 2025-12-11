package core

import "time"

type Users struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

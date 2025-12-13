package core

import "errors"

var (
	ErrAlreadyExists = errors.New("favorite already exists")
	ErrNotFound      = errors.New("favorite not found")
	ErrInvalidArgs   = errors.New("invalid args")
)

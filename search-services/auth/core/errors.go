package core

import "errors"

var ErrUserAlreadyExists = errors.New("user already exists")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidEmail = errors.New("invalid email format")
var ErrNotFound = errors.New("not found")

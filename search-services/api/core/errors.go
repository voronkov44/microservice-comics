package core

import "errors"

var ErrBadArguments = errors.New("arguments are not acceptable")
var ErrAlreadyExists = errors.New("resource or task already exists")
var ErrUnavailable = errors.New("dependency unavailable")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidEmail = errors.New("invalid email format")
var ErrNotFound = errors.New("not found")

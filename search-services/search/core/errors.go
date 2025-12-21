package core

import "errors"

var (
	ErrEmptyPhrase   = errors.New("empty phrase")
	ErrToLargeLimit  = errors.New("too large limit")
	ErrUnavailable   = errors.New("dependency unavailable")
	ErrBadArguments  = errors.New("arguments are not acceptable")
	ErrNonePhrase    = errors.New("this is too philosophical, try something less abstract))")
	ErrComicNotFound = errors.New("comic not found")
)

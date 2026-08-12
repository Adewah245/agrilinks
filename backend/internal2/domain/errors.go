package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalidPrice   = errors.New("price must be greater than zero")
	ErrInvalidInput   = errors.New("missing required field")
)

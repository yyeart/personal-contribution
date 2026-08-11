package domainErrors

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrEmptyID         = errors.New("empty id")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrValidation      = errors.New("validation failed")
)

package domainErrors

import "errors"

var (
	ErrInvalidUserID = errors.New("user id must not be nil")
)

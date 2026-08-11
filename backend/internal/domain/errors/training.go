package domainErrors

import "errors"

var (
	ErrForbidden       = errors.New("forbidden")
	ErrOutOfOrder      = errors.New("scene is out of order")
	ErrUnknownOption   = errors.New("unknown option")
	ErrAttemptFinished = errors.New("attempt already finished")
	ErrInvalidScenario = errors.New("invalid scenario")
)

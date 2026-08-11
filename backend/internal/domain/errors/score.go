package domainErrors

import "errors"

var (
	ErrInvalidMaxPoints = errors.New("max points must be greater than zero")
	ErrNegativePoints   = errors.New("points must not be negative")
	ErrNegativeDelta    = errors.New("delta must not be negative")
	ErrScoreOverflow    = errors.New("score points exceed max points")
	ErrInvalidPassRate  = errors.New("pass percent must be between 1 and 100")
)

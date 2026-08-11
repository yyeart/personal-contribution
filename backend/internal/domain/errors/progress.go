package domainErrors

import "errors"

var (
	ErrInvalidProgressData   = errors.New("progress data violates domain invariants")
	ErrDependencyUnavailable = errors.New("progress dependency unavailable")
	ErrDataInconsistent      = errors.New("progress data is inconsistent")
)

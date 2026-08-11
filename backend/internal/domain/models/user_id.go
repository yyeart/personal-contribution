package models

import (
	"github.com/google/uuid"
	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
)

type UserID uuid.UUID

func NewUserID(id uuid.UUID) (UserID, error) {
	if id == uuid.Nil {
		return UserID{}, domainErrors.ErrInvalidUserID
	}

	return UserID(id), nil
}

func (id UserID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

func (id UserID) IsZero() bool {
	return id.UUID() == uuid.Nil
}

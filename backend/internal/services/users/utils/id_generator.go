package users_service

import "github.com/google/uuid"

type UUIDGenerator struct{}

func (UUIDGenerator) New() uuid.UUID {
	return uuid.New()
}

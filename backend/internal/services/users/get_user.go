package users_service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

func (s *UsersService) GetUser(
	ctx context.Context,
	userID uuid.UUID,
) (models.User, error) {
	user, err := s.usersRepository.GetUser(ctx, userID)
	if err != nil {
		return models.User{}, fmt.Errorf("get user from repository: %w", err)
	}

	return user, nil
}

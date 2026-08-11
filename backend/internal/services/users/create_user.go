package users_service

import (
	"context"
	"errors"
	"fmt"

	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

type CreateUserInput struct {
	Nickname string
	Email    string
	Password string
}

func (s *UsersService) CreateUser(
	ctx context.Context,
	userInput CreateUserInput,
) (models.User, error) {
	if err := validateCreateUserInput(userInput); err != nil {
		return models.User{}, err
	}

	userInput = normalizeCreateUserInput(userInput)

	passwordHash, err := s.passwordHasher.Hash(userInput.Password)
	if err != nil {
		return models.User{}, fmt.Errorf("hash password: %w", err)
	}

	user := models.NewUser(
		s.idGenerator.New(),
		userInput.Nickname,
		userInput.Email,
		passwordHash,
		s.clock.Now().UTC(),
	)
	if err := user.Validate(); err != nil {
		return models.User{}, err
	}

	if err := s.usersRepository.CreateUser(ctx, user); err != nil {
		if errors.Is(err, domainErrors.ErrConflict) {
			return models.User{}, fmt.Errorf("create user: %w", domainErrors.ErrConflict)
		}
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

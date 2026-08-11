package users_service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

type UsersService struct {
	usersRepository UsersRepository
	passwordHasher  PasswordHasher
	idGenerator     IDGenerator
	clock           Clock
}

type UsersRepository interface {
	CreateUser(ctx context.Context, user models.User) error
	GetUser(ctx context.Context, userID uuid.UUID) (models.User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type IDGenerator interface {
	New() uuid.UUID
}

type Clock interface {
	Now() time.Time
}

func NewUsersService(
	usersRepository UsersRepository,
	passwordHasher PasswordHasher,
	idGenerator IDGenerator,
	clock Clock,
) *UsersService {
	return &UsersService{
		usersRepository: usersRepository,
		passwordHasher:  passwordHasher,
		idGenerator:     idGenerator,
		clock:           clock,
	}
}

func validateCreateUserInput(input CreateUserInput) error {
	if err := models.ValidateNickname(input.Nickname); err != nil {
		return err
	}

	if err := models.ValidateEmail(input.Email); err != nil {
		return err
	}

	if len([]rune(input.Password)) < models.MinPasswordLength ||
		len([]rune(input.Password)) > models.MaxPasswordBytes {
		return fmt.Errorf("invalid password length: %w", domainErrors.ErrInvalidArgument)
	}

	return nil
}

func normalizeCreateUserInput(input CreateUserInput) CreateUserInput {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	return input
}

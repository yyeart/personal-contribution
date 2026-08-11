package users_repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

func (r *UsersRepository) GetUser(
	ctx context.Context,
	userID uuid.UUID,
) (models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT id, nickname, email, password_hash, created_at 
	FROM users 
	WHERE id=$1
	`

	row := r.pool.QueryRow(ctx, query, userID)

	var userModel UserModel
	err := row.Scan(
		&userModel.ID,
		&userModel.Nickname,
		&userModel.Email,
		&userModel.PasswordHash,
		&userModel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf(
				"user with id %d not found: %w",
				userID, domainErrors.ErrNotFound,
			)
		}

		return models.User{}, fmt.Errorf("scan user: %w", err)
	}

	userDomain := userDomainFromModel(userModel)

	return userDomain, nil
}

func (r *UsersRepository) GetUserByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT id, nickname, email, password_hash, created_at 
	FROM users 
	WHERE email=$1
	`

	row := r.pool.QueryRow(ctx, query, email)

	var userModel UserModel
	err := row.Scan(
		&userModel.ID,
		&userModel.Nickname,
		&userModel.Email,
		&userModel.PasswordHash,
		&userModel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf(
				"user with email %s not found: %w",
				email, domainErrors.ErrNotFound,
			)
		}

		return models.User{}, fmt.Errorf("scan user: %w", err)
	}

	userDomain := userDomainFromModel(userModel)

	return userDomain, nil
}

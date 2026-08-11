package users_repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

type UserModel struct {
	ID           uuid.UUID
	Nickname     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

func userDomainFromModel(user UserModel) models.User {
	return models.NewUser(
		user.ID, user.Nickname, user.Email,
		user.PasswordHash, user.CreatedAt,
	)
}

// func userDomainsFromModels(users []UserModel) []models.User {
// 	userDomains := make([]models.User, len(users))

// 	for i, user := range users {
// 		userDomains[i] = userDomainFromModel(user)
// 	}

// 	return userDomains
// }

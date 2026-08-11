package models

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
)

const (
	MinNicknameLength = 3
	MaxNicknameLength = 30
	NicknameRegexp    = `^[a-zA-Z0-9_]+$`
	MinPasswordLength = 8
	MaxPasswordBytes  = 72
)

var nicknameRegexp = regexp.MustCompile(NicknameRegexp)

type User struct {
	ID           uuid.UUID
	Nickname     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

func NewUser(
	id uuid.UUID,
	nickname, email, passwordHash string,
	createdAt time.Time,
) User {
	return User{
		ID:           id,
		Nickname:     nickname,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    createdAt,
	}
}

func (u *User) Validate() error {
	if u.ID == uuid.Nil {
		return fmt.Errorf(
			"empty user id: %w",
			domainErrors.ErrEmptyID,
		)
	}

	if err := ValidateNickname(u.Nickname); err != nil {
		return fmt.Errorf("error validating nickname: %w", err)
	}

	if err := ValidateEmail(u.Email); err != nil {
		return fmt.Errorf("error validating email: %w", err)
	}

	if u.PasswordHash == "" {
		return fmt.Errorf(
			"empty password hash: %w",
			domainErrors.ErrInvalidArgument,
		)
	}

	if u.CreatedAt.IsZero() {
		return fmt.Errorf(
			"empty created_at: %w",
			domainErrors.ErrInvalidArgument,
		)
	}

	return nil
}

func ValidateNickname(value string) error {
	nickname := strings.TrimSpace(value)
	nicknameLen := len([]rune(nickname))
	if nicknameLen < MinNicknameLength || nicknameLen > MaxNicknameLength {
		return fmt.Errorf(
			"invalid nickname len: %d: %w",
			nicknameLen, domainErrors.ErrInvalidArgument,
		)
	}

	if !nicknameRegexp.MatchString(value) {
		return fmt.Errorf(
			"invalid nickname format: %w",
			domainErrors.ErrInvalidArgument,
		)
	}

	return nil
}

func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func ValidateEmail(value string) error {
	email := NormalizeEmail(value)

	if email == "" {
		return fmt.Errorf(
			"empty email: %w",
			domainErrors.ErrInvalidArgument,
		)
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf(
			"error parsing address: %w: %w",
			err,
			domainErrors.ErrInvalidArgument,
		)
	}

	if addr.Address != email {
		return fmt.Errorf(
			"invalid email form: %w",
			domainErrors.ErrInvalidArgument,
		)
	}

	return nil
}

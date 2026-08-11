package users_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
	usersutils "github.com/yyeart/personal-contribution/backend/internal/services/users/utils"
)

type repositoryStub struct {
	user   models.User
	err    error
	userID uuid.UUID
	calls  int
}

func (r *repositoryStub) CreateUser(_ context.Context, user models.User) error {
	r.calls++
	r.user = user
	return r.err
}

func (r *repositoryStub) GetUser(_ context.Context, userID uuid.UUID) (models.User, error) {
	r.calls++
	r.userID = userID
	return r.user, r.err
}

type passwordHasherStub struct {
	hash     string
	err      error
	password string
	calls    int
}

func (h *passwordHasherStub) Hash(password string) (string, error) {
	h.calls++
	h.password = password
	return h.hash, h.err
}

type idGeneratorStub struct{ id uuid.UUID }

func (g idGeneratorStub) New() uuid.UUID { return g.id }

type clockStub struct{ now time.Time }

func (c clockStub) Now() time.Time { return c.now }

func TestCreateUserStoresCanonicalEmailAndBcryptHash(t *testing.T) {
	t.Parallel()

	repository := &repositoryStub{}
	clock := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.FixedZone("MSK", 3*60*60))
	id := uuid.New()
	service := NewUsersService(
		repository,
		usersutils.BcryptPasswordHasher{},
		idGeneratorStub{id: id},
		clockStub{now: clock},
	)

	user, err := service.CreateUser(context.Background(), CreateUserInput{
		Nickname: "alice_1",
		Email:    "  ALICE@Example.COM ",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatal(err)
	}

	if user.ID != id || repository.user.ID != id {
		t.Fatalf("unexpected user id: %s", user.ID)
	}
	if user.Email != "alice@example.com" || repository.user.Email != user.Email {
		t.Fatalf("unexpected canonical email: %q", user.Email)
	}
	if repository.user.PasswordHash == "correct horse battery staple" || repository.user.PasswordHash == "" {
		t.Fatal("repository received the plain password or an empty hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repository.user.PasswordHash), []byte("correct horse battery staple")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
	if !repository.user.CreatedAt.Equal(clock.UTC()) {
		t.Fatalf("created_at = %v, want %v", repository.user.CreatedAt, clock.UTC())
	}
	if repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.calls)
	}
}

func TestCreateUserRejectsInvalidInputBeforeHashing(t *testing.T) {
	t.Parallel()

	repository := &repositoryStub{}
	hasher := &passwordHasherStub{hash: "hash"}
	service := NewUsersService(repository, hasher, idGeneratorStub{id: uuid.New()}, clockStub{now: time.Now()})

	_, err := service.CreateUser(context.Background(), CreateUserInput{
		Nickname: "no",
		Email:    "not-an-email",
		Password: "short",
	})
	if !errors.Is(err, domainErrors.ErrInvalidArgument) {
		t.Fatalf("error = %v, want invalid argument", err)
	}
	if hasher.calls != 0 || repository.calls != 0 {
		t.Fatalf("hasher calls = %d, repository calls = %d", hasher.calls, repository.calls)
	}
}

func TestCreateUserRejectsInvalidEmailBeforeHashing(t *testing.T) {
	t.Parallel()

	hasher := &passwordHasherStub{hash: "hash"}
	repository := &repositoryStub{}
	service := NewUsersService(repository, hasher, idGeneratorStub{id: uuid.New()}, clockStub{now: time.Now()})

	_, err := service.CreateUser(context.Background(), CreateUserInput{
		Nickname: "alice",
		Email:    "not-an-email",
		Password: "password123",
	})
	if !errors.Is(err, domainErrors.ErrInvalidArgument) {
		t.Fatalf("error = %v, want invalid argument", err)
	}
	if hasher.calls != 0 || repository.calls != 0 {
		t.Fatalf("hasher calls = %d, repository calls = %d", hasher.calls, repository.calls)
	}
}

func TestCreateUserDoesNotPersistWhenHashingFails(t *testing.T) {
	t.Parallel()

	hashErr := errors.New("hash failed")
	repository := &repositoryStub{}
	service := NewUsersService(
		repository,
		&passwordHasherStub{err: hashErr},
		idGeneratorStub{id: uuid.New()},
		clockStub{now: time.Now()},
	)

	_, err := service.CreateUser(context.Background(), CreateUserInput{
		Nickname: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})
	if !errors.Is(err, hashErr) {
		t.Fatalf("error = %v, want hash error", err)
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
	}
}

func TestCreateUserValidatesDomainUserBeforeRepository(t *testing.T) {
	t.Parallel()

	repository := &repositoryStub{}
	service := NewUsersService(
		repository,
		&passwordHasherStub{hash: "hash"},
		idGeneratorStub{id: uuid.Nil},
		clockStub{now: time.Now()},
	)

	_, err := service.CreateUser(context.Background(), CreateUserInput{
		Nickname: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})
	if !errors.Is(err, domainErrors.ErrEmptyID) {
		t.Fatalf("error = %v, want empty id", err)
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
	}
}

func TestCreateUserMapsRepositoryConflict(t *testing.T) {
	t.Parallel()

	repository := &repositoryStub{err: domainErrors.ErrConflict}
	service := NewUsersService(
		repository,
		&passwordHasherStub{hash: "hash"},
		idGeneratorStub{id: uuid.New()},
		clockStub{now: time.Now()},
	)

	_, err := service.CreateUser(context.Background(), CreateUserInput{
		Nickname: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})
	if !errors.Is(err, domainErrors.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}

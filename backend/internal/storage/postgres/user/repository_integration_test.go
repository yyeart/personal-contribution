package users_repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestUsersTablePostgresConstraints(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test was not run")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close database: %v", err)
		}
	}()

	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		CREATE TEMP TABLE users (
			id UUID PRIMARY KEY,
			nickname TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)`)
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	_, err = db.ExecContext(ctx,
		`INSERT INTO users (id, nickname, email, password_hash, created_at) VALUES ($1, $2, $3, $4, $5)`,
		id, "alice", "alice@example.com", "$2a$10$hash", time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}

	var storedHash string
	if err := db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if storedHash == "alice-password" || storedHash == "" {
		t.Fatal("plain or empty password hash stored")
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO users (id, nickname, email, password_hash, created_at) VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), "alice-2", "alice@example.com", "other-hash", time.Now().UTC(),
	)
	if err == nil {
		t.Fatal("duplicate email insert unexpectedly succeeded")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("duplicate email error = %v, want PostgreSQL unique violation", err)
	}
}

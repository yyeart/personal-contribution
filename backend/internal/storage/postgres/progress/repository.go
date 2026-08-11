package progress_repository

import (
	"github.com/yyeart/personal-contribution/backend/internal/storage/postgres/pool"
)

type ProgressRepository struct {
	pool *pool.Pool
}

func NewProgressRepository(pool *pool.Pool) *ProgressRepository {
	return &ProgressRepository{
		pool: pool,
	}
}

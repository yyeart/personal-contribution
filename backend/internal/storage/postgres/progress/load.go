package progress_repository

import (
	"context"
	"fmt"

	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

func (r *ProgressRepository) Load(ctx context.Context, userID models.UserID, historyLimit int) (models.ProgressSnapshot, error) {
	if r == nil || r.pool == nil {
		return models.ProgressSnapshot{}, fmt.Errorf("%w: postgres is not configured", domainErrors.ErrDependencyUnavailable)
	}

	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.ProgressSnapshot{}, dependencyError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	scenarios, indexes, err := loadScenarios(ctx, tx, userID)
	if err != nil {
		return models.ProgressSnapshot{}, err
	}
	if err := loadRecentAttempts(ctx, tx, userID, historyLimit, scenarios, indexes); err != nil {
		return models.ProgressSnapshot{}, err
	}

	return models.ProgressSnapshot{Scenarios: scenarios}, nil
}

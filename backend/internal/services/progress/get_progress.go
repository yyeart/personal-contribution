package progress_service

import (
	"context"
	"fmt"

	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

func (s *ProgressService) Get(ctx context.Context, userID models.UserID) (models.OverallProgress, error) {
	if userID.IsZero() {
		return models.OverallProgress{}, domainErrors.ErrInvalidUserID
	}

	if s.progressRepository == nil {
		return models.OverallProgress{}, fmt.Errorf("%w: repository is not configured", domainErrors.ErrDependencyUnavailable)
	}

	snapshot, err := s.progressRepository.Load(ctx, userID, HistoryLimit)
	if err != nil {
		return models.OverallProgress{}, err
	}

	if err := validate(snapshot); err != nil {
		return models.OverallProgress{}, fmt.Errorf("%w: %w", domainErrors.ErrDataInconsistent, err)
	}

	return aggregate(snapshot), nil
}

package progress_service

import (
	"context"

	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

const HistoryLimit = 20

const recommendationLimit = 3

const experienceLevelSize = 100

type ProgressRepository interface {
	Load(context.Context, models.UserID, int) (models.ProgressSnapshot, error)
}

type ProgressService struct {
	progressRepository ProgressRepository
}

func NewProgressService(progressRepository ProgressRepository) *ProgressService {
	return &ProgressService{
		progressRepository: progressRepository,
	}
}

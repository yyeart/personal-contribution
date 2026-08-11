package progress_repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

const scenariosQuery = `
	SELECT s.id, s.slug, s.title, s.role,
		COUNT(a.id),
		COUNT(a.id) FILTER (WHERE a.status = 'finished' AND a.finished_at IS NOT NULL),
		COALESCE(BOOL_OR(a.status = 'finished' AND a.outcome = 'safe'), false),
		MAX(a.score) FILTER (WHERE a.status = 'finished' AND a.finished_at IS NOT NULL),
		(ARRAY_AGG(a.score ORDER BY a.finished_at, a.id) FILTER (WHERE a.status = 'finished' AND a.finished_at IS NOT NULL))[1],
		(ARRAY_AGG(a.score ORDER BY a.finished_at DESC, a.id DESC) FILTER (WHERE a.status = 'finished' AND a.finished_at IS NOT NULL))[1],
		(ARRAY_AGG(a.id ORDER BY a.finished_at, a.id) FILTER (WHERE a.status = 'finished' AND a.outcome = 'safe' AND a.finished_at IS NOT NULL))[1],
		(ARRAY_AGG(a.score ORDER BY a.finished_at, a.id) FILTER (WHERE a.status = 'finished' AND a.outcome = 'safe' AND a.finished_at IS NOT NULL))[1],
		(ARRAY_AGG(a.finished_at ORDER BY a.finished_at, a.id) FILTER (WHERE a.status = 'finished' AND a.outcome = 'safe' AND a.finished_at IS NOT NULL))[1],
		COUNT(a.id) FILTER (WHERE a.status = 'in_progress'),
		(MIN(a.id::text) FILTER (WHERE a.status = 'in_progress'))::uuid
	FROM scenarios s
	LEFT JOIN attempts a ON a.scenario_id = s.id AND a.user_id = $1
	WHERE s.is_active
	GROUP BY s.id, s.slug, s.title, s.role
	ORDER BY s.slug`

func loadScenarios(
	ctx context.Context,
	tx pgx.Tx,
	userID models.UserID,
) ([]models.ScenarioProgress, map[uuid.UUID]int, error) {
	rows, err := tx.Query(ctx, scenariosQuery, userID.UUID())
	if err != nil {
		return nil, nil, dependencyError(err)
	}
	defer rows.Close()

	scenarios := make([]models.ScenarioProgress, 0)
	indexes := make(map[uuid.UUID]int)

	for rows.Next() {
		scenario, err := scanProgressScenario(rows)
		if err != nil {
			return nil, nil, err
		}

		indexes[scenario.ID] = len(scenarios)
		scenarios = append(scenarios, scenario)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, dependencyError(err)
	}

	return scenarios, indexes, nil
}

func scanProgressScenario(rows pgx.Rows) (models.ScenarioProgress, error) {
	var scenario models.ScenarioProgress
	var completed, active int
	var best, initial, latest sql.NullInt64
	var firstSafeID uuid.NullUUID
	var firstSafeScore sql.NullInt64
	var firstSafeAt sql.NullTime
	var activeID uuid.NullUUID

	if err := rows.Scan(&scenario.ID, &scenario.Slug, &scenario.Title, &scenario.Role,
		&scenario.AttemptsCount, &completed, &scenario.Passed, &best, &initial, &latest,
		&firstSafeID, &firstSafeScore, &firstSafeAt, &active, &activeID); err != nil {
		return models.ScenarioProgress{}, dependencyError(err)
	}

	if err := validateProgressSnapshot(scenario.Passed, completed, active); err != nil {
		return models.ScenarioProgress{}, err
	}

	scenario.Completed = completed > 0

	var err error

	if scenario.BestScore, err = scoreFromNull(best); err != nil {
		return models.ScenarioProgress{}, err
	}

	if scenario.InitialScore, err = scoreFromNull(initial); err != nil {
		return models.ScenarioProgress{}, err
	}

	if scenario.LatestScore, err = scoreFromNull(latest); err != nil {
		return models.ScenarioProgress{}, err
	}

	if firstSafeID.Valid || firstSafeScore.Valid || firstSafeAt.Valid {
		if !firstSafeID.Valid || !firstSafeScore.Valid || !firstSafeAt.Valid {
			return models.ScenarioProgress{}, inconsistentError("first safe attempt has an invalid progress snapshot")
		}
		score, err := models.NewScore(int(firstSafeScore.Int64), 100)
		if err != nil {
			return models.ScenarioProgress{}, inconsistentError(err.Error())
		}
		firstSafe := models.AttemptResult{
			ID: firstSafeID.UUID, Score: score, Outcome: models.OutcomeSafe, CompletedAt: firstSafeAt.Time,
		}
		scenario.FirstSafeAttempt = &firstSafe
	}

	if activeID.Valid {
		id := activeID.UUID
		scenario.ActiveAttemptID = &id
	}

	return scenario, nil
}

func validateProgressSnapshot(passed bool, completed, active int) error {
	if active > 1 || passed && completed == 0 {
		return inconsistentError("attempt snapshot violates progress invariants")
	}

	return nil
}

func scoreFromNull(value sql.NullInt64) (*models.Score, error) {
	if !value.Valid {
		return nil, nil
	}

	score, err := models.NewScore(int(value.Int64), 100)
	if err != nil {
		return nil, inconsistentError(err.Error())
	}

	return &score, nil
}

const recentAttemptsQuery = `
	WITH ranked AS (
		SELECT a.scenario_id, a.id, a.score, a.outcome, a.finished_at,
			ROW_NUMBER() OVER (PARTITION BY a.scenario_id ORDER BY a.finished_at DESC, a.id DESC) AS position
		FROM attempts a
		JOIN scenarios s ON s.id = a.scenario_id
		WHERE a.user_id = $1 AND a.status = 'finished' AND a.finished_at IS NOT NULL AND s.is_active
	)
	SELECT scenario_id, id, score, outcome, finished_at
	FROM ranked
	WHERE position <= $2
	ORDER BY scenario_id, finished_at DESC, id DESC`

func loadRecentAttempts(ctx context.Context, tx pgx.Tx, userID models.UserID, limit int, scenarios []models.ScenarioProgress, indexes map[uuid.UUID]int) error {
	rows, err := tx.Query(ctx, recentAttemptsQuery, userID.UUID(), limit)
	if err != nil {
		return dependencyError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var scenarioID uuid.UUID
		var attempt models.AttemptResult
		var points sql.NullInt64
		var completedAt sql.NullTime
		if err := rows.Scan(&scenarioID, &attempt.ID, &points, &attempt.Outcome, &completedAt); err != nil {
			return dependencyError(err)
		}
		index, ok := indexes[scenarioID]
		if !ok || !points.Valid || !completedAt.Valid {
			return inconsistentError("finished attempt has an invalid progress snapshot")
		}
		score, err := models.NewScore(int(points.Int64), 100)
		if err != nil {
			return inconsistentError(err.Error())
		}
		attempt.Score = score
		attempt.CompletedAt = completedAt.Time
		scenarios[index].RecentAttempts = append(scenarios[index].RecentAttempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return dependencyError(err)
	}
	return nil
}

func dependencyError(err error) error {
	return fmt.Errorf("%w: %w", domainErrors.ErrDependencyUnavailable, err)
}

func inconsistentError(message string) error {
	return fmt.Errorf("%w: %s", domainErrors.ErrDataInconsistent, message)
}

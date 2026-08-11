package progress_repository

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
	"github.com/yyeart/personal-contribution/backend/internal/storage/postgres/pool"
)

func TestRepositoryLoadUsesAllFinishedAttemptsForDynamics(t *testing.T) {
	ctx := context.Background()
	postgres := setupTestPostgres(t, ctx)

	userID := insertTestUser(t, ctx, postgres)

	populatedScenarioID := uuid.New()
	emptyScenarioID := uuid.New()

	populatedSlug := "progress-history-" + uuid.NewString()
	emptySlug := "progress-empty-" + uuid.NewString()

	t.Cleanup(func() {
		cleanupProgressTestData(
			postgres,
			userID,
			populatedScenarioID,
			emptyScenarioID,
		)
	})

	insertScenario(t, ctx, postgres, populatedScenarioID, populatedSlug)
	insertScenario(t, ctx, postgres, emptyScenarioID, emptySlug)
	insertFinishedAttempts(t, ctx, postgres, userID, populatedScenarioID, 21)

	domainUserID := mustUserID(t, userID)

	snapshot, err := NewProgressRepository(postgres).Load(ctx, domainUserID, 20)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := scenariosBySlug(snapshot.Scenarios)

	assertPopulatedScenario(t, scenarios[populatedSlug])
	assertEmptyScenario(t, scenarios[emptySlug])
}

func setupTestPostgres(t *testing.T, ctx context.Context) *pool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	config, err := poolConfigFromURL(databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	postgres, err := pool.NewPool(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(postgres.Close)

	return postgres
}

func insertTestUser(
	t *testing.T,
	ctx context.Context,
	postgres *pool.Pool,
) uuid.UUID {
	t.Helper()

	userID := uuid.New()

	_, err := postgres.Exec(
		ctx,
		`INSERT INTO users (id, nickname, email, password_hash, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID,
		"progress-"+uuid.NewString(),
		"progress-"+uuid.NewString()+"@example.test",
		"hash",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}

	return userID
}

func cleanupProgressTestData(
	postgres *pool.Pool,
	userID uuid.UUID,
	scenarioIDs ...uuid.UUID,
) {
	ctx := context.Background()

	_, _ = postgres.Exec( //nolint:errcheck
		ctx,
		"DELETE FROM users WHERE id = $1",
		userID,
	)

	for _, scenarioID := range scenarioIDs {
		_, _ = postgres.Exec( //nolint:errcheck
			ctx,
			"DELETE FROM scenarios WHERE id = $1",
			scenarioID,
		)
	}
}

func insertFinishedAttempts(
	t *testing.T,
	ctx context.Context,
	postgres *pool.Pool,
	userID uuid.UUID,
	scenarioID uuid.UUID,
	count int,
) {
	t.Helper()

	startedAt := time.Now().UTC().Add(-time.Hour)

	for points := range count {
		finishedAt := startedAt.Add(time.Duration(points) * time.Minute)

		_, err := postgres.Exec(ctx, `
			INSERT INTO attempts (
				id,
				user_id,
				scenario_id,
				status,
				state,
				score,
				outcome,
				started_at,
				finished_at,
				revision
			)
			VALUES ($1, $2, $3, 'finished', '{}', $4, 'safe', $5, $5, 0)`,
			uuid.New(),
			userID,
			scenarioID,
			points,
			finishedAt,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func mustUserID(t *testing.T, id uuid.UUID) models.UserID {
	t.Helper()

	userID, err := models.NewUserID(id)
	if err != nil {
		t.Fatal(err)
	}

	return userID
}

func scenariosBySlug(
	scenarios []models.ScenarioProgress,
) map[string]models.ScenarioProgress {
	result := make(map[string]models.ScenarioProgress, len(scenarios))

	for _, scenario := range scenarios {
		result[scenario.Slug] = scenario
	}

	return result
}

func assertPopulatedScenario(
	t *testing.T,
	scenario models.ScenarioProgress,
) {
	t.Helper()

	if scenario.InitialScore == nil || scenario.InitialScore.Points() != 0 {
		t.Fatalf("initial score = %v", scenario.InitialScore)
	}

	if scenario.LatestScore == nil || scenario.LatestScore.Points() != 20 {
		t.Fatalf("latest score = %v", scenario.LatestScore)
	}

	if scenario.FirstSafeAttempt == nil ||
		scenario.FirstSafeAttempt.Score.Points() != 0 ||
		scenario.FirstSafeAttempt.Outcome != models.OutcomeSafe {
		t.Fatalf("first safe attempt = %+v", scenario.FirstSafeAttempt)
	}

	if len(scenario.RecentAttempts) != 20 {
		t.Fatalf("recent attempts count = %d", len(scenario.RecentAttempts))
	}

	lastRecentAttempt := scenario.RecentAttempts[len(scenario.RecentAttempts)-1]
	if lastRecentAttempt.Score.Points() != 1 {
		t.Fatalf("last recent attempt score = %v", lastRecentAttempt.Score)
	}
}

func assertEmptyScenario(
	t *testing.T,
	scenario models.ScenarioProgress,
) {
	t.Helper()

	if scenario.InitialScore != nil ||
		scenario.LatestScore != nil ||
		scenario.BestScore != nil {
		t.Fatalf("empty scenario scores = %+v", scenario)
	}
}

func insertScenario(t *testing.T, ctx context.Context, postgres *pool.Pool, id uuid.UUID, slug string) {
	t.Helper()
	doc, err := json.Marshal(map[string]any{
		"slug": slug, "role": models.RoleBuyer, "title": "Progress test", "difficulty": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.Exec(ctx, "INSERT INTO scenarios (id, doc) VALUES ($1, $2)", id, doc); err != nil {
		t.Fatal(err)
	}
}

func poolConfigFromURL(rawURL string) (pool.Config, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return pool.Config{}, err
	}
	password, _ := parsed.User.Password()
	return pool.Config{
		Host: parsed.Hostname(), Port: parsed.Port(), User: parsed.User.Username(), Password: password,
		Database: parsed.Path[1:], Timeout: time.Second,
	}, nil
}

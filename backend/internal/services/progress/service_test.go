package progress_service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
	progress_service "github.com/yyeart/personal-contribution/backend/internal/services/progress"
)

type repositoryStub struct {
	snapshot models.ProgressSnapshot
	err      error
	limit    int
}

func (r *repositoryStub) Load(_ context.Context, _ models.UserID, limit int) (models.ProgressSnapshot, error) {
	r.limit = limit
	return r.snapshot, r.err
}

func TestGetAggregatesCurrentProgress(t *testing.T) {
	t.Parallel()
	passedScore := mustScore(t, 80, 100)
	failedScore := mustScore(t, 60, 100)
	now := time.Now().UTC()
	repo := &repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{
		{ID: uuid.New(), Slug: "buyer-one", Title: "Buyer", Role: models.RoleBuyer,
			AttemptsCount: 3, Completed: true, Passed: true, BestScore: &passedScore,
			RecentAttempts: []models.AttemptResult{{ID: uuid.New(), Score: passedScore, Outcome: models.OutcomeSafe, CompletedAt: now}}},
		{ID: uuid.New(), Slug: "seller-one", Title: "Seller", Role: models.RoleSeller,
			AttemptsCount: 2, Completed: true, Passed: false, BestScore: &failedScore},
	}}}
	result, err := progress_service.NewProgressService(repo).Get(context.Background(), mustUserID(t))
	if err != nil {
		t.Fatal(err)
	}
	if repo.limit != progress_service.HistoryLimit {
		t.Fatalf("limit = %d", repo.limit)
	}
	if result.TotalScenarios != 2 || result.CompletedScenarios != 2 || result.PassedScenarios != 1 {
		t.Fatalf("unexpected totals: %+v", result)
	}
	if result.CompletionPercent != 100 || result.PassedPercent != 50 {
		t.Fatalf("unexpected percentages: %+v", result)
	}
	if len(result.Roles) != 2 || result.Roles[0].Role != models.RoleBuyer || result.Roles[1].Role != models.RoleSeller {
		t.Fatalf("unexpected roles: %+v", result.Roles)
	}
}

func TestGetEmptyCatalogReturnsZeroPercentages(t *testing.T) {
	t.Parallel()
	result, err := progress_service.NewProgressService(&repositoryStub{}).Get(context.Background(), mustUserID(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalScenarios != 0 || result.CompletionPercent != 0 || result.PassedPercent != 0 || len(result.Roles) != 2 {
		t.Fatalf("unexpected empty progress: %+v", result)
	}
}

func TestGetCalculatesScenarioDynamics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		attemptsCount int
		initial       *models.Score
		latest        *models.Score
		wantDelta     *int
		wantTrend     *models.ProgressTrend
	}{
		{
			name: "no completed attempts",
		},
		{
			name:          "one attempt",
			attemptsCount: 1,
			initial:       scorePointer(mustScore(t, 42, 100)),
			latest:        scorePointer(mustScore(t, 42, 100)),
			wantDelta:     intPointer(0),
			wantTrend:     trendPointer(models.ProgressTrendStable),
		},
		{
			name:          "improving",
			attemptsCount: 2,
			initial:       scorePointer(mustScore(t, 42, 100)),
			latest:        scorePointer(mustScore(t, 78, 100)),
			wantDelta:     intPointer(36),
			wantTrend:     trendPointer(models.ProgressTrendImproving),
		},
		{
			name:          "declining",
			attemptsCount: 2,
			initial:       scorePointer(mustScore(t, 78, 100)),
			latest:        scorePointer(mustScore(t, 42, 100)),
			wantDelta:     intPointer(-36),
			wantTrend:     trendPointer(models.ProgressTrendDeclining),
		},
		{
			name:          "stable",
			attemptsCount: 2,
			initial:       scorePointer(mustScore(t, 60, 100)),
			latest:        scorePointer(mustScore(t, 60, 100)),
			wantDelta:     intPointer(0),
			wantTrend:     trendPointer(models.ProgressTrendStable),
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{{
				ID: uuid.New(), Slug: "scenario", Title: "Scenario", Role: models.RoleBuyer,
				AttemptsCount: tt.attemptsCount,
				InitialScore:  tt.initial, LatestScore: tt.latest,
			}}}}

			result, err := progress_service.NewProgressService(repo).Get(context.Background(), mustUserID(t))
			if err != nil {
				t.Fatal(err)
			}
			scenario := result.Roles[0].Scenarios[0]
			if !equalIntPointers(scenario.ImprovementPercentPoints, tt.wantDelta) {
				t.Fatalf("improvement = %v, want %v", scenario.ImprovementPercentPoints, tt.wantDelta)
			}
			if !equalTrendPointers(scenario.Trend, tt.wantTrend) {
				t.Fatalf("trend = %v, want %v", scenario.Trend, tt.wantTrend)
			}
		})
	}
}

func TestGetRecommendationsUsesPriorityAndLimitsResults(t *testing.T) {
	t.Parallel()

	lowScore := mustScore(t, 42, 100)
	highScore := mustScore(t, 80, 100)
	activeAttemptID := uuid.New()
	repo := &repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{
		{ID: uuid.New(), Slug: "active", Title: "Active", Role: models.RoleBuyer, ActiveAttemptID: &activeAttemptID},
		{ID: uuid.New(), Slug: "failed-low", Title: "Failed low", Role: models.RoleBuyer, Completed: true, BestScore: &lowScore},
		{ID: uuid.New(), Slug: "failed-high", Title: "Failed high", Role: models.RoleBuyer, Completed: true, BestScore: &highScore},
		{ID: uuid.New(), Slug: "not-started", Title: "Not started", Role: models.RoleBuyer},
	}}}

	result, err := progress_service.NewProgressService(repo).Get(context.Background(), mustUserID(t))
	if err != nil {
		t.Fatal(err)
	}

	want := []models.Recommendation{
		{ScenarioSlug: "active", ReasonCode: "ACTIVE_ATTEMPT"},
		{ScenarioSlug: "failed-low", ReasonCode: "LOW_BEST_SCORE"},
		{ScenarioSlug: "failed-high", ReasonCode: "LOW_BEST_SCORE"},
	}
	assertRecommendations(t, result.Recommendations, want)
}

func TestGetRecommendationsBreaksScoreTiesBySlug(t *testing.T) {
	t.Parallel()

	score := mustScore(t, 42, 100)
	repo := &repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{
		{ID: uuid.New(), Slug: "zeta", Title: "Zeta", Role: models.RoleBuyer, Completed: true, BestScore: &score},
		{ID: uuid.New(), Slug: "alpha", Title: "Alpha", Role: models.RoleBuyer, Completed: true, BestScore: &score},
	}}}

	result, err := progress_service.NewProgressService(repo).Get(context.Background(), mustUserID(t))
	if err != nil {
		t.Fatal(err)
	}

	want := []models.Recommendation{
		{ScenarioSlug: "alpha", ReasonCode: "LOW_BEST_SCORE"},
		{ScenarioSlug: "zeta", ReasonCode: "LOW_BEST_SCORE"},
	}
	assertRecommendations(t, result.Recommendations, want)
}

func TestGetRecommendationsSuggestsNotStartedThenPassedScenarios(t *testing.T) {
	t.Parallel()

	lowScore := mustScore(t, 60, 100)
	highScore := mustScore(t, 80, 100)
	repo := &repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{
		{ID: uuid.New(), Slug: "passed-high", Title: "Passed high", Role: models.RoleBuyer, AttemptsCount: 1, Completed: true, Passed: true, BestScore: &highScore},
		{ID: uuid.New(), Slug: "not-started-b", Title: "Not started B", Role: models.RoleBuyer},
		{ID: uuid.New(), Slug: "not-started-a", Title: "Not started A", Role: models.RoleBuyer},
		{ID: uuid.New(), Slug: "passed-low", Title: "Passed low", Role: models.RoleBuyer, AttemptsCount: 1, Completed: true, Passed: true, BestScore: &lowScore},
	}}}

	result, err := progress_service.NewProgressService(repo).Get(context.Background(), mustUserID(t))
	if err != nil {
		t.Fatal(err)
	}

	want := []models.Recommendation{
		{ScenarioSlug: "not-started-a", ReasonCode: "NOT_STARTED"},
		{ScenarioSlug: "not-started-b", ReasonCode: "NOT_STARTED"},
		{ScenarioSlug: "passed-low", ReasonCode: "REPEAT_FOR_REINFORCEMENT"},
	}
	assertRecommendations(t, result.Recommendations, want)
}

func TestGetCalculatesExperienceAndAchievements(t *testing.T) {
	t.Parallel()

	buyerInitial := mustScore(t, 60, 100)
	buyerLatest := mustScore(t, 80, 100)
	buyerBest := mustScore(t, 100, 100)
	sellerBest := mustScore(t, 80, 100)
	repo := &repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{
		{
			ID: uuid.New(), Slug: "buyer", Title: "Buyer", Role: models.RoleBuyer,
			AttemptsCount: 2, Completed: true, Passed: true, BestScore: &buyerBest,
			InitialScore: &buyerInitial, LatestScore: &buyerLatest,
		},
		{
			ID: uuid.New(), Slug: "seller", Title: "Seller", Role: models.RoleSeller,
			AttemptsCount: 1, Completed: true, Passed: true, BestScore: &sellerBest,
		},
	}}}

	result, err := progress_service.NewProgressService(repo).Get(context.Background(), mustUserID(t))
	if err != nil {
		t.Fatal(err)
	}

	experience := result.Experience
	if experience.TotalXP != 75 || experience.Level != 1 || experience.CurrentXP != 75 || experience.NextLevelXP != 100 {
		t.Fatalf("unexpected experience: %+v", experience)
	}
	assertAchievements(t, experience.Achievements, map[string]bool{
		"FIRST_COMPLETION":  true,
		"FIRST_SAFE_RESULT": true,
		"IMPROVEMENT":       true,
		"BOTH_ROLES":        true,
		"PERFECT_SCORE":     true,
	})
}

func TestGetCalculatesExperienceAtLevelBoundary(t *testing.T) {
	t.Parallel()

	perfectScenario := func(slug string, role models.Role) models.ScenarioProgress {
		initial := mustScore(t, 60, 100)
		latest := mustScore(t, 80, 100)
		best := mustScore(t, 100, 100)
		return models.ScenarioProgress{
			ID: uuid.New(), Slug: slug, Title: slug, Role: role,
			AttemptsCount: 2, Completed: true, Passed: true, BestScore: &best,
			InitialScore: &initial, LatestScore: &latest,
		}
	}

	tests := []struct {
		name          string
		scenarios     []models.ScenarioProgress
		wantXP        int
		wantLevel     int
		wantCurrentXP int
	}{
		{name: "zero XP", wantXP: 0, wantLevel: 1, wantCurrentXP: 0},
		{
			name: "100 XP",
			scenarios: []models.ScenarioProgress{
				perfectScenario("buyer", models.RoleBuyer),
				perfectScenario("seller", models.RoleSeller),
			},
			wantXP: 100, wantLevel: 2, wantCurrentXP: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := progress_service.NewProgressService(&repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: tt.scenarios}}).Get(context.Background(), mustUserID(t))
			if err != nil {
				t.Fatal(err)
			}
			got := result.Experience
			if got.TotalXP != tt.wantXP || got.Level != tt.wantLevel || got.CurrentXP != tt.wantCurrentXP || got.NextLevelXP != 100 {
				t.Fatalf("experience = %+v", got)
			}
		})
	}
}

func TestGetRejectsInvalidSnapshotAndMissingDependency(t *testing.T) {
	t.Parallel()
	_, err := progress_service.NewProgressService(nil).Get(context.Background(), mustUserID(t))
	if !errors.Is(err, domainErrors.ErrDependencyUnavailable) {
		t.Fatalf("got %v", err)
	}
	invalid := models.ScenarioProgress{ID: uuid.New(), Slug: "broken", Title: "Broken", Role: models.RoleBuyer, Passed: true}
	_, err = progress_service.NewProgressService(&repositoryStub{snapshot: models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{invalid}}}).Get(context.Background(), mustUserID(t))
	if !errors.Is(err, domainErrors.ErrDataInconsistent) {
		t.Fatalf("got %v", err)
	}
}

func TestGetRejectsOversizedOrUnorderedHistory(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	score := mustScore(t, 80, 100)
	attempts := make([]models.AttemptResult, progress_service.HistoryLimit+1)
	for index := range attempts {
		attempts[index] = models.AttemptResult{ID: uuid.New(), Score: score, Outcome: models.OutcomeSafe, CompletedAt: now.Add(-time.Duration(index) * time.Minute)}
	}
	snapshot := models.ProgressSnapshot{Scenarios: []models.ScenarioProgress{{
		ID: uuid.New(), Slug: "scenario", Title: "Scenario", Role: models.RoleBuyer,
		RecentAttempts: attempts,
	}}}
	_, err := progress_service.NewProgressService(&repositoryStub{snapshot: snapshot}).Get(context.Background(), mustUserID(t))
	if !errors.Is(err, domainErrors.ErrDataInconsistent) {
		t.Fatalf("got %v", err)
	}
}

func mustScore(t *testing.T, points, max int) models.Score {
	t.Helper()
	score, err := models.NewScore(points, max)
	if err != nil {
		t.Fatal(err)
	}
	return score
}

func mustUserID(t *testing.T) models.UserID {
	t.Helper()
	id, err := models.NewUserID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func scorePointer(score models.Score) *models.Score { return &score }

func intPointer(value int) *int { return &value }

func trendPointer(trend models.ProgressTrend) *models.ProgressTrend { return &trend }

func equalIntPointers(actual, expected *int) bool {
	return actual == nil && expected == nil || actual != nil && expected != nil && *actual == *expected
}

func equalTrendPointers(actual, expected *models.ProgressTrend) bool {
	return actual == nil && expected == nil || actual != nil && expected != nil && *actual == *expected
}

func assertRecommendations(t *testing.T, actual, want []models.Recommendation) {
	t.Helper()
	if len(actual) != len(want) {
		t.Fatalf("recommendations count = %d, want %d: %+v", len(actual), len(want), actual)
	}
	for index, expected := range want {
		got := actual[index]
		if got.ScenarioSlug != expected.ScenarioSlug || got.ReasonCode != expected.ReasonCode {
			t.Fatalf("recommendation[%d] = %+v, want slug %q and reason %q", index, got, expected.ScenarioSlug, expected.ReasonCode)
		}
		if got.ReasonText == "" {
			t.Fatalf("recommendation[%d] has empty reason text", index)
		}
	}
}

func assertAchievements(t *testing.T, actual []models.Achievement, want map[string]bool) {
	t.Helper()
	if len(actual) != len(want) {
		t.Fatalf("achievements count = %d, want %d: %+v", len(actual), len(want), actual)
	}
	for _, achievement := range actual {
		earned, ok := want[achievement.Code]
		if !ok || achievement.Earned != earned {
			t.Fatalf("unexpected achievement: %+v", achievement)
		}
		if achievement.Title == "" || achievement.Description == "" {
			t.Fatalf("achievement has empty metadata: %+v", achievement)
		}
		delete(want, achievement.Code)
	}
	if len(want) != 0 {
		t.Fatalf("missing achievements: %+v", want)
	}
}

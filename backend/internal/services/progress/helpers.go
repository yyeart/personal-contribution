package progress_service

import (
	"cmp"
	"fmt"
	"slices"
	"sort"

	"github.com/google/uuid"
	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

func aggregate(snapshot models.ProgressSnapshot) models.OverallProgress {
	roles := map[models.Role]*models.RoleProgress{
		models.RoleBuyer:  {Role: models.RoleBuyer},
		models.RoleSeller: {Role: models.RoleSeller},
	}

	overall := models.OverallProgress{}

	for _, scenario := range snapshot.Scenarios {
		if scenario.InitialScore != nil && scenario.LatestScore != nil {
			initialPercent := scenario.InitialScore.Percent()
			latestPercent := scenario.LatestScore.Percent()

			improvement := latestPercent - initialPercent
			var trend models.ProgressTrend

			if latestPercent > initialPercent {
				trend = models.ProgressTrendImproving
			} else if latestPercent < initialPercent {
				trend = models.ProgressTrendDeclining
			} else {
				trend = models.ProgressTrendStable
			}

			scenario.ImprovementPercentPoints = &improvement
			scenario.Trend = &trend
		}

		role := roles[scenario.Role]

		role.Scenarios = append(role.Scenarios, scenario)

		role.TotalScenarios++
		overall.TotalScenarios++

		if scenario.Completed {
			role.CompletedScenarios++
			overall.CompletedScenarios++
		}

		if scenario.Passed {
			role.PassedScenarios++
			overall.PassedScenarios++
		}
	}

	overall.CompletionPercent = percentage(overall.CompletedScenarios, overall.TotalScenarios)
	overall.PassedPercent = percentage(overall.PassedScenarios, overall.TotalScenarios)

	for _, roleName := range []models.Role{models.RoleBuyer, models.RoleSeller} {
		role := roles[roleName]
		role.CompletionPercent = percentage(role.CompletedScenarios, role.TotalScenarios)
		role.PassedPercent = percentage(role.PassedScenarios, role.TotalScenarios)
		sort.Slice(role.Scenarios, func(i, j int) bool { return role.Scenarios[i].Slug < role.Scenarios[j].Slug })
		overall.Roles = append(overall.Roles, *role)
	}

	buyer := roles[models.RoleBuyer]
	seller := roles[models.RoleSeller]

	completionPercentDelta := buyer.CompletionPercent - seller.CompletionPercent
	passedPercentDelta := buyer.PassedPercent - seller.PassedPercent

	overall.RoleComparison = models.RoleComparison{
		CompletionPercentDelta: completionPercentDelta,
		PassedPercentDelta:     passedPercentDelta,
	}
	overall.Recommendations = recommendations(snapshot.Scenarios)
	overall.Experience = experience(snapshot.Scenarios)

	return overall
}

func experience(scenarios []models.ScenarioProgress) models.ExperienceProgress {
	completedScenarios := 0
	passedScenarios := 0
	improvedScenarios := 0
	perfectScenarios := 0
	passedRoles := make(map[models.Role]struct{})

	for _, scenario := range scenarios {
		if scenario.Completed {
			completedScenarios++
		}

		if scenario.Passed {
			passedScenarios++
			passedRoles[scenario.Role] = struct{}{}
		}

		if scenario.InitialScore != nil &&
			scenario.LatestScore != nil &&
			scenario.LatestScore.Percent() > scenario.InitialScore.Percent() {
			improvedScenarios++
		}

		if scenario.Passed &&
			scenario.BestScore != nil &&
			scenario.BestScore.Percent() == 100 {
			perfectScenarios++
		}
	}

	totalXP := completedScenarios*10 + passedScenarios*15 + improvedScenarios*10 + perfectScenarios*15
	return models.ExperienceProgress{
		TotalXP:     totalXP,
		Level:       totalXP/experienceLevelSize + 1,
		CurrentXP:   totalXP % experienceLevelSize,
		NextLevelXP: experienceLevelSize,
		Achievements: achievements(
			completedScenarios,
			passedScenarios,
			improvedScenarios,
			len(passedRoles) == 2,
			perfectScenarios,
		),
	}
}

func achievements(
	completedScenarios, passedScenarios, improvedScenarios int,
	bothRolesPassed bool,
	perfectScenarios int,
) []models.Achievement {
	return []models.Achievement{
		{
			Code:        "FIRST_COMPLETION",
			Title:       "Первый шаг",
			Description: "Завершён первый сценарий.",
			Earned:      completedScenarios > 0,
		},
		{
			Code:        "FIRST_SAFE_RESULT",
			Title:       "Безопасный исход",
			Description: "Получен первый безопасный результат.",
			Earned:      passedScenarios > 0,
		},
		{
			Code:        "IMPROVEMENT",
			Title:       "Работа над ошибками",
			Description: "Результат по сценарию улучшен.",
			Earned:      improvedScenarios > 0,
		},
		{
			Code:        "BOTH_ROLES",
			Title:       "Две роли",
			Description: "Есть успешно пройденные сценарии покупателя и продавца.",
			Earned:      bothRolesPassed,
		},
		{
			Code:        "PERFECT_SCORE",
			Title:       "Безупречный результат",
			Description: "Лучший score равен 100.",
			Earned:      perfectScenarios > 0,
		},
	}
}

type recommendationCandidate struct {
	recommendation models.Recommendation
	priority       int
	bestScore      int
}

func recommendations(scenarios []models.ScenarioProgress) []models.Recommendation {
	candidates := make([]recommendationCandidate, 0, len(scenarios))

	for _, scenario := range scenarios {
		candidate, ok := recommendationFor(scenario)
		if ok {
			candidates = append(candidates, candidate)
		}
	}

	slices.SortFunc(candidates, func(a, b recommendationCandidate) int {
		if n := cmp.Compare(a.priority, b.priority); n != 0 {
			return n
		}

		if n := cmp.Compare(a.bestScore, b.bestScore); n != 0 {
			return n
		}

		return cmp.Compare(
			a.recommendation.ScenarioSlug,
			b.recommendation.ScenarioSlug,
		)
	})

	if len(candidates) > recommendationLimit {
		candidates = candidates[:recommendationLimit]
	}

	result := make([]models.Recommendation, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.recommendation)
	}

	return result
}

func recommendationFor(scenario models.ScenarioProgress) (recommendationCandidate, bool) {
	if scenario.ActiveAttemptID != nil {
		return newRecommendationCandidate(scenario, "ACTIVE_ATTEMPT", "У вас есть незавершённая попытка. Продолжите сценарий.", 0), true
	}

	if !scenario.Passed && scenario.AttemptsCount > 0 && scenario.BestScore == nil {
		return newRecommendationCandidate(scenario, "NOT_PASSED", "Сценарий пока не пройден безопасно. Попробуйте ещё раз.", 1), true
	}

	if scenario.Completed && !scenario.Passed && scenario.BestScore != nil {
		return newRecommendationCandidate(scenario, "LOW_BEST_SCORE", fmt.Sprintf("Лучший результат по сценарию — %d%%. Попробуйте улучшить его.", scenario.BestScore.Percent()), 2), true
	}

	if scenario.AttemptsCount == 0 {
		return newRecommendationCandidate(scenario, "NOT_STARTED", "Сценарий ещё не пройден. Начните его.", 3), true
	}

	if scenario.Passed {
		return newRecommendationCandidate(scenario, "REPEAT_FOR_REINFORCEMENT", "Сценарий пройден безопасно. Повторите его, чтобы закрепить навык.", 4), true
	}

	return recommendationCandidate{}, false
}

func newRecommendationCandidate(scenario models.ScenarioProgress, reasonCode, reasonText string, priority int) recommendationCandidate {
	bestScore := 101 // 101 чтобы сценарий отсортировался после тех, у которых это поле задано
	if scenario.BestScore != nil {
		bestScore = scenario.BestScore.Percent()
	}

	return recommendationCandidate{
		recommendation: models.Recommendation{
			ScenarioSlug: scenario.Slug,
			ReasonCode:   reasonCode,
			ReasonText:   reasonText,
		},
		priority:  priority,
		bestScore: bestScore,
	}
}

func percentage(value, total int) int {
	if total == 0 {
		return 0
	}

	return int((int64(value)*100 + int64(total)/2) / int64(total))
}

func validate(snapshot models.ProgressSnapshot) error {
	for _, scenario := range snapshot.Scenarios {
		if err := validateScenario(scenario); err != nil {
			return err
		}
	}

	return nil
}

func validateScenario(scenario models.ScenarioProgress) error {
	if !isValidScenarioIdentity(scenario) {
		return domainErrors.ErrInvalidProgressData
	}

	if !isValidScenarioState(scenario) {
		return domainErrors.ErrInvalidProgressData
	}

	if !isValidScenarioScores(scenario) {
		return domainErrors.ErrInvalidProgressData
	}

	if len(scenario.RecentAttempts) > HistoryLimit {
		return domainErrors.ErrInvalidProgressData
	}

	if err := validateInitialLatestScore(scenario.InitialScore, scenario.LatestScore); err != nil {
		return err
	}

	return validateAttempts(scenario.RecentAttempts)
}

func isValidScenarioIdentity(scenario models.ScenarioProgress) bool {
	return scenario.ID != uuid.Nil &&
		scenario.Slug != "" &&
		scenario.Title != "" &&
		isValidRole(scenario.Role)
}

func isValidRole(role models.Role) bool {
	return role == models.RoleBuyer || role == models.RoleSeller
}

func isValidScenarioState(scenario models.ScenarioProgress) bool {
	if scenario.AttemptsCount < 0 {
		return false
	}

	if scenario.Passed && !scenario.Completed {
		return false
	}

	return scenario.BestScore == nil || scenario.Completed
}

func isValidScenarioScores(scenario models.ScenarioProgress) bool {
	if scenario.BestScore != nil && scenario.BestScore.MaxPoints() != 100 {
		return false
	}

	return scenario.AttemptsCount != 0 ||
		(scenario.InitialScore == nil && scenario.LatestScore == nil)
}

func validateInitialLatestScore(initialScore, latestScore *models.Score) error {
	if (initialScore == nil && latestScore != nil) ||
		(initialScore != nil && latestScore == nil) {
		return domainErrors.ErrInvalidProgressData
	}

	if (initialScore != nil && initialScore.MaxPoints() != 100) ||
		(latestScore != nil && latestScore.MaxPoints() != 100) {
		return domainErrors.ErrInvalidProgressData
	}

	return nil
}

func validateAttempts(attempts []models.AttemptResult) error {
	for index, attempt := range attempts {
		if attempt.ID == uuid.Nil || attempt.CompletedAt.IsZero() || attempt.Score.MaxPoints() != 100 || !validOutcome(attempt.Outcome) {
			return domainErrors.ErrInvalidProgressData
		}

		if index > 0 && attempt.CompletedAt.After(attempts[index-1].CompletedAt) {
			return domainErrors.ErrInvalidProgressData
		}
	}

	return nil
}

func validOutcome(outcome models.Outcome) bool {
	return outcome == models.OutcomeSafe || outcome == models.OutcomePartial || outcome == models.OutcomeScammed
}

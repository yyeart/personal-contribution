package httptransport

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

type ProgressService interface {
	Get(context.Context, models.UserID) (models.OverallProgress, error)
}

type ProgressHandler struct {
	progressService ProgressService
}

func NewProgressHandler(progressService ProgressService) *ProgressHandler {
	return &ProgressHandler{progressService: progressService}
}

func RegisterProgressRoutes(router gin.IRoutes, handler *ProgressHandler) {
	router.GET("/progress", handler.Get)
}

// Get godoc
//
//	@Summary		Получить прогресс пользователя
//	@Description	Возвращает общий прогресс текущего авторизованного пользователя по сценариям.
//	@Tags			progress
//	@Produce		json
//	@Security		SessionID
//	@Success		200	{object}	progressResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Failure		503	{object}	ErrorResponse
//	@Router			/v1/progress [get]
func (h *ProgressHandler) Get(c *gin.Context) {
	userID, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "unauthorized",
			Message: "authenticated user is required",
		})
		return
	}

	if h == nil || h.progressService == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Code:    "dependency_unavailable",
			Message: "progress service is unavailable",
		})
		return
	}

	result, err := h.progressService.Get(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, domainErrors.ErrDependencyUnavailable):
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{
				Code:    "dependency_unavailable",
				Message: "progress dependency is unavailable",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:    "internal_error",
				Message: "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, mapProgress(result))
}

func mapProgress(progress models.OverallProgress) progressResponse {
	roles := make([]roleProgressResponse, 0, len(progress.Roles))
	for _, role := range progress.Roles {
		roles = append(roles, mapRole(role))
	}

	recommendations := make([]recommendationResponse, 0, len(progress.Recommendations))
	for _, recommendation := range progress.Recommendations {
		recommendations = append(recommendations, mapRecommendation(recommendation))
	}

	achievements := make([]achievementResponse, 0, len(progress.Experience.Achievements))
	for _, achievement := range progress.Experience.Achievements {
		achievements = append(achievements, mapAchievement(achievement))
	}

	return progressResponse{
		TotalScenarios:     progress.TotalScenarios,
		CompletedScenarios: progress.CompletedScenarios,
		PassedScenarios:    progress.PassedScenarios,
		CompletionPercent:  progress.CompletionPercent,
		PassedPercent:      progress.PassedPercent,
		Roles:              roles,
		RoleComparison: roleComparisonResponse{
			CompletionPercentDelta: progress.RoleComparison.CompletionPercentDelta,
			PassedPercentDelta:     progress.RoleComparison.PassedPercentDelta,
		},
		Recommendations: recommendations,
		Experience: experienceResponse{
			TotalXP:      progress.Experience.TotalXP,
			Level:        progress.Experience.Level,
			CurrentXP:    progress.Experience.CurrentXP,
			NextLevelXP:  progress.Experience.NextLevelXP,
			Achievements: achievements,
		},
	}
}

func mapAchievement(achievement models.Achievement) achievementResponse {
	return achievementResponse{
		Code:        achievement.Code,
		Title:       achievement.Title,
		Description: achievement.Description,
		Earned:      achievement.Earned,
	}
}

func mapRecommendation(recommendation models.Recommendation) recommendationResponse {
	return recommendationResponse{
		ScenarioSlug: recommendation.ScenarioSlug,
		ReasonCode:   recommendation.ReasonCode,
		ReasonText:   recommendation.ReasonText,
	}
}

func mapRole(role models.RoleProgress) roleProgressResponse {
	scenarios := make([]scenarioProgressResponse, 0, len(role.Scenarios))
	for _, scenario := range role.Scenarios {
		scenarios = append(scenarios, mapScenario(scenario))
	}

	return roleProgressResponse{
		Role:               role.Role,
		TotalScenarios:     role.TotalScenarios,
		CompletedScenarios: role.CompletedScenarios,
		PassedScenarios:    role.PassedScenarios,
		CompletionPercent:  role.CompletionPercent,
		PassedPercent:      role.PassedPercent,
		Scenarios:          scenarios,
	}
}

func mapScenario(scenario models.ScenarioProgress) scenarioProgressResponse {
	attempts := make([]completedAttemptResultResponse, 0, len(scenario.RecentAttempts))
	for _, attempt := range scenario.RecentAttempts {
		attempts = append(attempts, completedAttemptResultResponse{
			AttemptID:   attempt.ID,
			Score:       mapScore(attempt.Score),
			Outcome:     attempt.Outcome,
			CompletedAt: attempt.CompletedAt,
		})
	}

	return scenarioProgressResponse{
		ScenarioSlug:             scenario.Slug,
		Title:                    scenario.Title,
		Completed:                scenario.Completed,
		Passed:                   scenario.Passed,
		AttemptsCount:            scenario.AttemptsCount,
		BestScore:                mapOptionalScore(scenario.BestScore),
		ActiveAttemptID:          scenario.ActiveAttemptID,
		RecentAttempts:           attempts,
		InitialScore:             mapOptionalScore(scenario.InitialScore),
		LatestScore:              mapOptionalScore(scenario.LatestScore),
		ImprovementPercentPoints: scenario.ImprovementPercentPoints,
		Trend:                    (*progressTrendResponse)(scenario.Trend),
		FirstSafeAttempt:         mapOptionalAttempt(scenario.FirstSafeAttempt),
	}
}

func mapOptionalAttempt(attempt *models.AttemptResult) *completedAttemptResultResponse {
	if attempt == nil {
		return nil
	}

	return &completedAttemptResultResponse{
		AttemptID:   attempt.ID,
		Score:       mapScore(attempt.Score),
		Outcome:     attempt.Outcome,
		CompletedAt: attempt.CompletedAt,
	}
}

func mapOptionalScore(score *models.Score) *scoreResponse {
	if score == nil {
		return nil
	}

	mapped := mapScore(*score)
	return &mapped
}

func mapScore(score models.Score) scoreResponse {
	return scoreResponse{
		Points:    score.Points(),
		MaxPoints: score.MaxPoints(),
		Percent:   score.Percent(),
	}
}

package httptransport

import (
	"time"

	"github.com/google/uuid"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

type scoreResponse struct {
	Points    int `json:"points"`
	MaxPoints int `json:"max_points"`
	Percent   int `json:"percent"`
}

type completedAttemptResultResponse struct {
	AttemptID   uuid.UUID      `json:"attempt_id"`
	Score       scoreResponse  `json:"score"`
	Outcome     models.Outcome `json:"outcome"`
	CompletedAt time.Time      `json:"completed_at"`
}

type progressTrendResponse string

type scenarioProgressResponse struct {
	ScenarioSlug             string                           `json:"scenario_slug"`
	Title                    string                           `json:"title"`
	Completed                bool                             `json:"completed"`
	Passed                   bool                             `json:"passed"`
	AttemptsCount            int                              `json:"attempts_count"`
	BestScore                *scoreResponse                   `json:"best_score"`
	ActiveAttemptID          *uuid.UUID                       `json:"active_attempt_id"`
	RecentAttempts           []completedAttemptResultResponse `json:"recent_attempts"`
	InitialScore             *scoreResponse                   `json:"initial_score"`
	LatestScore              *scoreResponse                   `json:"latest_score"`
	ImprovementPercentPoints *int                             `json:"improvement_percent_points"`
	Trend                    *progressTrendResponse           `json:"trend"`
	FirstSafeAttempt         *completedAttemptResultResponse  `json:"first_safe_attempt"`
}

type roleProgressResponse struct {
	Role               models.Role                `json:"role"`
	TotalScenarios     int                        `json:"total_scenarios"`
	CompletedScenarios int                        `json:"completed_scenarios"`
	PassedScenarios    int                        `json:"passed_scenarios"`
	CompletionPercent  int                        `json:"completion_percent"`
	PassedPercent      int                        `json:"passed_percent"`
	Scenarios          []scenarioProgressResponse `json:"scenarios"`
}

type roleComparisonResponse struct {
	CompletionPercentDelta int `json:"completion_percent_delta"`
	PassedPercentDelta     int `json:"passed_percent_delta"`
}

type recommendationResponse struct {
	ScenarioSlug string `json:"scenario_slug"`
	ReasonCode   string `json:"reason_code"`
	ReasonText   string `json:"reason_text"`
}

type achievementResponse struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Earned      bool   `json:"earned"`
}

type experienceResponse struct {
	TotalXP      int                   `json:"total_xp"`
	Level        int                   `json:"level"`
	CurrentXP    int                   `json:"current_xp"`
	NextLevelXP  int                   `json:"next_level_xp"`
	Achievements []achievementResponse `json:"achievements"`
}

type progressResponse struct {
	TotalScenarios     int                      `json:"total_scenarios"`
	CompletedScenarios int                      `json:"completed_scenarios"`
	PassedScenarios    int                      `json:"passed_scenarios"`
	CompletionPercent  int                      `json:"completion_percent"`
	PassedPercent      int                      `json:"passed_percent"`
	Roles              []roleProgressResponse   `json:"roles"`
	RoleComparison     roleComparisonResponse   `json:"role_comparison"`
	Recommendations    []recommendationResponse `json:"recommendations"`
	Experience         experienceResponse       `json:"experience"`
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type AttemptResult struct {
	ID          uuid.UUID
	Score       Score
	Outcome     Outcome
	CompletedAt time.Time
}

type ScenarioProgress struct {
	ID                       uuid.UUID
	Slug                     string
	Title                    string
	Role                     Role
	Completed                bool
	Passed                   bool
	AttemptsCount            int
	BestScore                *Score
	ActiveAttemptID          *uuid.UUID
	RecentAttempts           []AttemptResult
	InitialScore             *Score
	LatestScore              *Score
	ImprovementPercentPoints *int
	Trend                    *ProgressTrend
	FirstSafeAttempt         *AttemptResult
}

type ProgressTrend string

const (
	ProgressTrendImproving ProgressTrend = "improving"
	ProgressTrendStable    ProgressTrend = "stable"
	ProgressTrendDeclining ProgressTrend = "declining"
)

type RoleProgress struct {
	Role               Role
	TotalScenarios     int
	CompletedScenarios int
	PassedScenarios    int
	CompletionPercent  int
	PassedPercent      int
	Scenarios          []ScenarioProgress
}

type ProgressSnapshot struct {
	Scenarios []ScenarioProgress
}

type OverallProgress struct {
	TotalScenarios     int
	CompletedScenarios int
	PassedScenarios    int
	CompletionPercent  int
	PassedPercent      int
	Roles              []RoleProgress
	RoleComparison     RoleComparison
	Recommendations    []Recommendation
	Experience         ExperienceProgress
}

type RoleComparison struct {
	CompletionPercentDelta int
	PassedPercentDelta     int
}

type Recommendation struct {
	ScenarioSlug string
	ReasonCode   string
	ReasonText   string
}

type ExperienceProgress struct {
	TotalXP      int
	Level        int
	CurrentXP    int
	NextLevelXP  int
	Achievements []Achievement
}

type Achievement struct {
	Code        string
	Title       string
	Description string
	Earned      bool
}

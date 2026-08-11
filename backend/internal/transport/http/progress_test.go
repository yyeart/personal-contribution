package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	domainErrors "github.com/yyeart/personal-contribution/backend/internal/domain/errors"
	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

type progressStub struct {
	result models.OverallProgress
	err    error
}

func (s progressStub) Get(context.Context, models.UserID) (models.OverallProgress, error) {
	return s.result, s.err
}

func TestProgressHandlerRequiresIdentity(t *testing.T) {
	t.Parallel()
	response := performRequest(t, progressStub{}, false)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestProgressHandlerMapsSuccess(t *testing.T) {
	t.Parallel()
	response := performRequest(t, progressStub{result: models.OverallProgress{Roles: []models.RoleProgress{}}}, true)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body progressResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TotalScenarios != 0 || body.Roles == nil {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestProgressHandlerMapsRecommendations(t *testing.T) {
	t.Parallel()

	response := performRequest(t, progressStub{result: models.OverallProgress{
		Recommendations: []models.Recommendation{
			{ScenarioSlug: "active-attempt", ReasonCode: "ACTIVE_ATTEMPT", ReasonText: "Continue the active attempt."},
			{ScenarioSlug: "low-score", ReasonCode: "LOW_BEST_SCORE", ReasonText: "Improve the best score."},
		},
	}}, true)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var body struct {
		Recommendations []struct {
			ScenarioSlug string `json:"scenario_slug"`
			ReasonCode   string `json:"reason_code"`
			ReasonText   string `json:"reason_text"`
		} `json:"recommendations"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if len(body.Recommendations) != 2 {
		t.Fatalf("recommendations = %+v", body.Recommendations)
	}
	if got := body.Recommendations[0]; got.ScenarioSlug != "active-attempt" || got.ReasonCode != "ACTIVE_ATTEMPT" || got.ReasonText != "Continue the active attempt." {
		t.Fatalf("first recommendation = %+v", got)
	}
	if got := body.Recommendations[1]; got.ScenarioSlug != "low-score" || got.ReasonCode != "LOW_BEST_SCORE" || got.ReasonText != "Improve the best score." {
		t.Fatalf("second recommendation = %+v", got)
	}
}

func TestProgressHandlerMapsExperience(t *testing.T) {
	t.Parallel()

	response := performRequest(t, progressStub{result: models.OverallProgress{
		Experience: models.ExperienceProgress{
			TotalXP:     75,
			Level:       1,
			CurrentXP:   75,
			NextLevelXP: 100,
			Achievements: []models.Achievement{{
				Code:        "FIRST_COMPLETION",
				Title:       "First step",
				Description: "Completed the first scenario.",
				Earned:      true,
			}},
		},
	}}, true)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var body struct {
		TotalScenarios int `json:"total_scenarios"`
		Experience     struct {
			TotalXP      int `json:"total_xp"`
			Level        int `json:"level"`
			CurrentXP    int `json:"current_xp"`
			NextLevelXP  int `json:"next_level_xp"`
			Achievements []struct {
				Code        string `json:"code"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Earned      bool   `json:"earned"`
			} `json:"achievements"`
		} `json:"experience"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TotalScenarios != 0 {
		t.Fatalf("total_scenarios = %d", body.TotalScenarios)
	}
	if body.Experience.TotalXP != 75 || body.Experience.Level != 1 || body.Experience.CurrentXP != 75 || body.Experience.NextLevelXP != 100 {
		t.Fatalf("unexpected experience: %+v", body.Experience)
	}
	if len(body.Experience.Achievements) != 1 {
		t.Fatalf("achievements = %+v", body.Experience.Achievements)
	}
	if got := body.Experience.Achievements[0]; got.Code != "FIRST_COMPLETION" || got.Title != "First step" || got.Description != "Completed the first scenario." || !got.Earned {
		t.Fatalf("achievement = %+v", got)
	}
}

func TestProgressHandlerMapsScenarioDynamics(t *testing.T) {
	t.Parallel()
	initial, err := models.NewScore(42, 100)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := models.NewScore(78, 100)
	if err != nil {
		t.Fatal(err)
	}
	improvement := 36
	trend := models.ProgressTrendImproving
	response := performRequest(t, progressStub{result: models.OverallProgress{Roles: []models.RoleProgress{{
		Role: models.RoleBuyer,
		Scenarios: []models.ScenarioProgress{{
			ID: uuid.New(), Slug: "buyer-one", Title: "Buyer", Role: models.RoleBuyer,
			InitialScore: &initial, LatestScore: &latest, ImprovementPercentPoints: &improvement, Trend: &trend,
		}},
	}}}}, true)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var body struct {
		Roles []struct {
			Scenarios []map[string]json.RawMessage `json:"scenarios"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Roles) != 1 || len(body.Roles[0].Scenarios) != 1 {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
	scenario := body.Roles[0].Scenarios[0]
	for _, field := range []string{"initial_score", "latest_score", "improvement_percent_points", "trend"} {
		if _, ok := scenario[field]; !ok {
			t.Fatalf("missing %q in %s", field, response.Body.String())
		}
	}
	var delta int
	if err := json.Unmarshal(scenario["improvement_percent_points"], &delta); err != nil || delta != 36 {
		t.Fatalf("delta = %s, error = %v", scenario["improvement_percent_points"], err)
	}
	var gotTrend string
	if err := json.Unmarshal(scenario["trend"], &gotTrend); err != nil || gotTrend != "improving" {
		t.Fatalf("trend = %s, error = %v", scenario["trend"], err)
	}
}

func TestProgressHandlerMapsErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "dependency", err: domainErrors.ErrDependencyUnavailable, status: http.StatusServiceUnavailable},
		{name: "inconsistent", err: domainErrors.ErrDataInconsistent, status: http.StatusInternalServerError},
		{name: "unknown", err: errors.New("boom"), status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response := performRequest(t, progressStub{err: tt.err}, true)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
		})
	}
}

func performRequest(t *testing.T, service ProgressService, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if authenticated {
		userID, err := models.NewUserID(uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		router.Use(func(c *gin.Context) {
			c.Set(sessionContextKey, models.Session{ID: uuid.New(), UserID: userID.UUID()})
		})
	}
	RegisterProgressRoutes(router.Group("/v1"), NewProgressHandler(service))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/progress", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

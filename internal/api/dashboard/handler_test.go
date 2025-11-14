//nolint:noctx // Test file uses http.NewRequest for simplicity
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/service/leaderboard"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// Mock Badge Service
type mockBadgeService struct {
	userBadges   map[uint][]models.UserBadge
	badges       map[uint]*models.Badge
	badgeHolders map[uint][]models.User
}

func newMockBadgeService() *mockBadgeService {
	return &mockBadgeService{
		userBadges:   make(map[uint][]models.UserBadge),
		badges:       make(map[uint]*models.Badge),
		badgeHolders: make(map[uint][]models.User),
	}
}

func (m *mockBadgeService) GetUserBadges(ctx context.Context, userID uint) ([]models.UserBadge, error) {
	badges, exists := m.userBadges[userID]
	if !exists {
		return []models.UserBadge{}, nil
	}
	return badges, nil
}

func (m *mockBadgeService) GetBadgeCatalog(ctx context.Context) ([]models.Badge, error) {
	badges := make([]models.Badge, 0, len(m.badges))
	for _, badge := range m.badges {
		badges = append(badges, *badge)
	}
	return badges, nil
}

func (m *mockBadgeService) GetBadgeByID(ctx context.Context, badgeID uint) (*models.Badge, error) {
	badge, exists := m.badges[badgeID]
	if !exists {
		return nil, fmt.Errorf("badge not found")
	}
	return badge, nil
}

func (m *mockBadgeService) GetBadgeHolders(ctx context.Context, badgeID uint) ([]models.User, error) {
	holders, exists := m.badgeHolders[badgeID]
	if !exists {
		return []models.User{}, nil
	}
	return holders, nil
}

// Mock Leaderboard Service
type mockLeaderboardService struct {
	globalLeaderboard map[string][]leaderboard.Entry
	teamLeaderboard   map[string][]leaderboard.Entry
	userStats         map[uint]*leaderboard.UserStats
}

func newMockLeaderboardService() *mockLeaderboardService {
	return &mockLeaderboardService{
		globalLeaderboard: make(map[string][]leaderboard.Entry),
		teamLeaderboard:   make(map[string][]leaderboard.Entry),
		userStats:         make(map[uint]*leaderboard.UserStats),
	}
}

func (m *mockLeaderboardService) GetGlobalLeaderboard(ctx context.Context, period, metric string, limit int) ([]leaderboard.Entry, error) {
	key := fmt.Sprintf("%s:%s", period, metric)
	entries, exists := m.globalLeaderboard[key]
	if !exists {
		return []leaderboard.Entry{}, nil
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func (m *mockLeaderboardService) GetTeamLeaderboard(ctx context.Context, team, period, metric string, limit int) ([]leaderboard.Entry, error) {
	key := fmt.Sprintf("%s:%s:%s", team, period, metric)
	entries, exists := m.teamLeaderboard[key]
	if !exists {
		return []leaderboard.Entry{}, nil
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func (m *mockLeaderboardService) GetUserStats(ctx context.Context, userID uint, period string) (*leaderboard.UserStats, error) {
	stats, exists := m.userStats[userID]
	if !exists {
		return nil, fmt.Errorf("user stats not found")
	}
	return stats, nil
}

// Test Setup
func setupTestHandler() (*Handler, *mockBadgeService, *mockLeaderboardService) {
	badgeService := newMockBadgeService()
	leaderboardService := newMockLeaderboardService()
	log := logger.New("debug", "text", "stdout")

	handler := NewHandlerWithInterfaces(badgeService, leaderboardService, log)

	return handler, badgeService, leaderboardService
}

func setupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := router.Group("/api/v1")
	api.GET("/leaderboard", handler.GetGlobalLeaderboard)
	api.GET("/leaderboard/:team", handler.GetTeamLeaderboard)
	api.GET("/users/:id/stats", handler.GetUserStats)
	api.GET("/users/:id/badges", handler.GetUserBadges)
	api.GET("/badges", handler.GetBadgeCatalog)
	api.GET("/badges/:id", handler.GetBadgeByID)
	api.GET("/badges/:id/holders", handler.GetBadgeHolders)

	return router
}

// Tests

func TestGetGlobalLeaderboard_Success(t *testing.T) {
	handler, _, leaderboardService := setupTestHandler()
	router := setupRouter(handler)

	// Setup mock data
	entries := []leaderboard.Entry{
		{Rank: 1, UserID: 1, Username: "alice", Team: "backend", CompletedReviews: 50, EngagementScore: 95.5},
		{Rank: 2, UserID: 2, Username: "bob", Team: "frontend", CompletedReviews: 45, EngagementScore: 92.3},
	}
	leaderboardService.globalLeaderboard["month:completed_reviews"] = entries

	// Make request
	req, _ := http.NewRequest("GET", "/api/v1/leaderboard?period=month&metric=completed_reviews&limit=10", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "month", response["period"])
	assert.Equal(t, "completed_reviews", response["metric"])
	assert.Equal(t, float64(2), response["total_entries"])
}

func TestGetGlobalLeaderboard_InvalidPeriod(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/leaderboard?period=invalid&metric=completed_reviews", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid period")
}

func TestGetGlobalLeaderboard_InvalidMetric(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/leaderboard?period=month&metric=invalid", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid metric")
}

func TestGetGlobalLeaderboard_InvalidLimit(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/leaderboard?limit=abc", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid limit")
}

func TestGetTeamLeaderboard_Success(t *testing.T) {
	handler, _, leaderboardService := setupTestHandler()
	router := setupRouter(handler)

	// Setup mock data
	entries := []leaderboard.Entry{
		{Rank: 1, UserID: 1, Username: "alice", Team: "backend", CompletedReviews: 50, EngagementScore: 95.5},
		{Rank: 2, UserID: 3, Username: "charlie", Team: "backend", CompletedReviews: 40, EngagementScore: 88.2},
	}
	leaderboardService.teamLeaderboard["backend:month:completed_reviews"] = entries

	// Make request
	req, _ := http.NewRequest("GET", "/api/v1/leaderboard/backend?period=month&metric=completed_reviews&limit=10", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "backend", response["team"])
	assert.Equal(t, "month", response["period"])
	assert.Equal(t, "completed_reviews", response["metric"])
	assert.Equal(t, float64(2), response["total_entries"])
}

func TestGetTeamLeaderboard_InvalidParameters(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/leaderboard/backend?period=invalid&metric=completed_reviews", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid period")
}

func TestGetUserStats_Success(t *testing.T) {
	handler, _, leaderboardService := setupTestHandler()
	router := setupRouter(handler)

	// Setup mock data
	stats := &leaderboard.UserStats{
		UserID:            1,
		Username:          "alice",
		Team:              "backend",
		Period:            "month",
		TotalReviews:      50,
		CompletedReviews:  48,
		AvgTTFR:           3600.0,
		AvgTimeToApproval: 7200.0,
		AvgCommentCount:   5.5,
		EngagementScore:   95.5,
		GlobalRank:        1,
		TeamRank:          1,
	}
	leaderboardService.userStats[1] = stats

	// Make request
	req, _ := http.NewRequest("GET", "/api/v1/users/1/stats?period=month", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["stats"])
}

func TestGetUserStats_InvalidUserID(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/users/abc/stats?period=month", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid user ID")
}

func TestGetUserStats_InvalidPeriod(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/users/1/stats?period=invalid", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid period")
}

func TestGetUserBadges_Success(t *testing.T) {
	handler, badgeService, _ := setupTestHandler()
	router := setupRouter(handler)

	// Setup mock data
	badge := models.Badge{
		Name:        "Speed Demon",
		Description: "Complete 10 reviews in under 1 hour",
		Icon:        "speed",
	}
	badge.ID = 1

	userBadges := []models.UserBadge{
		{
			UserID:   1,
			BadgeID:  1,
			Badge:    badge,
			EarnedAt: time.Now(),
		},
	}
	badgeService.userBadges[1] = userBadges

	// Make request
	req, _ := http.NewRequest("GET", "/api/v1/users/1/badges", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, float64(1), response["user_id"])
	assert.Equal(t, float64(1), response["total_badges"])
}

func TestGetUserBadges_InvalidUserID(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/users/invalid/badges", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid user ID")
}

func TestGetBadgeCatalog_Success(t *testing.T) {
	handler, badgeService, _ := setupTestHandler()
	router := setupRouter(handler)

	// Setup mock data
	badge1 := &models.Badge{
		Name:        "Speed Demon",
		Description: "Complete 10 reviews in under 1 hour",
		Icon:        "speed",
	}
	badge1.ID = 1

	badge2 := &models.Badge{
		Name:        "Team Player",
		Description: "Review 50 MRs from team members",
		Icon:        "team",
	}
	badge2.ID = 2

	badgeService.badges[1] = badge1
	badgeService.badges[2] = badge2

	// Make request
	req, _ := http.NewRequest("GET", "/api/v1/badges", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, float64(2), response["total_badges"])
}

func TestGetBadgeByID_Success(t *testing.T) {
	handler, badgeService, _ := setupTestHandler()
	router := setupRouter(handler)

	// Setup mock data
	badge := &models.Badge{
		Name:        "Speed Demon",
		Description: "Complete 10 reviews in under 1 hour",
		Icon:        "speed",
	}
	badge.ID = 1
	badgeService.badges[1] = badge

	// Make request
	req, _ := http.NewRequest("GET", "/api/v1/badges/1", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["badge"])
}

func TestGetBadgeByID_InvalidID(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/badges/abc", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid badge ID")
}

func TestGetBadgeByID_NotFound(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/badges/999", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Badge not found")
}

func TestGetBadgeHolders_Success(t *testing.T) {
	handler, badgeService, _ := setupTestHandler()
	router := setupRouter(handler)

	// Setup mock data
	holders := []models.User{
		{Username: "alice", Email: "alice@example.com", Team: "backend"},
		{Username: "bob", Email: "bob@example.com", Team: "frontend"},
	}
	holders[0].ID = 1
	holders[1].ID = 2
	badgeService.badgeHolders[1] = holders

	// Make request
	req, _ := http.NewRequest("GET", "/api/v1/badges/1/holders?limit=10", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, float64(1), response["badge_id"])
	assert.Equal(t, float64(2), response["total_holders"])
	assert.Equal(t, float64(2), response["limited_to"])
}

func TestGetBadgeHolders_InvalidBadgeID(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/badges/abc/holders", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid badge ID")
}

func TestGetBadgeHolders_LimitTooHigh(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupRouter(handler)

	req, _ := http.NewRequest("GET", "/api/v1/badges/1/holders?limit=2000", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "limit cannot exceed 1000")
}

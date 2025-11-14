package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// Mock repositories for testing
type mockMetricsRepository struct {
	metrics []models.ReviewMetrics
}

func newMockMetricsRepository() *mockMetricsRepository {
	return &mockMetricsRepository{
		metrics: []models.ReviewMetrics{},
	}
}

func (m *mockMetricsRepository) GetByDateRange(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error) {
	// Filter by team if specified
	if team, ok := filters["team"]; ok {
		var filtered []models.ReviewMetrics
		for _, metric := range m.metrics {
			if metric.Team == team.(string) {
				filtered = append(filtered, metric)
			}
		}
		return filtered, nil
	}
	return m.metrics, nil
}

func (m *mockMetricsRepository) GetMetricsByUser(userID uint, startDate, endDate time.Time) ([]models.ReviewMetrics, error) {
	var result []models.ReviewMetrics
	for _, metric := range m.metrics {
		if metric.UserID != nil && *metric.UserID == userID {
			result = append(result, metric)
		}
	}
	return result, nil
}

type mockBadgeRepository struct {
	userBadgeCounts map[uint]int64
	userBadges      map[uint][]models.UserBadge
}

func newMockBadgeRepository() *mockBadgeRepository {
	return &mockBadgeRepository{
		userBadgeCounts: make(map[uint]int64),
		userBadges:      make(map[uint][]models.UserBadge),
	}
}

func (m *mockBadgeRepository) GetUserBadgeCount(userID uint) (int64, error) {
	count, ok := m.userBadgeCounts[userID]
	if !ok {
		return 0, nil
	}
	return count, nil
}

func (m *mockBadgeRepository) GetUserBadges(userID uint) ([]models.UserBadge, error) {
	badges, ok := m.userBadges[userID]
	if !ok {
		return []models.UserBadge{}, nil
	}
	return badges, nil
}

type mockUserRepository struct {
	users map[uint]*models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[uint]*models.User),
	}
}

func (m *mockUserRepository) GetByID(id uint) (*models.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return user, nil
}

// Test setup helper
func setupTestService() (*Service, *mockMetricsRepository, *mockBadgeRepository, *mockUserRepository) {
	metricsRepo := newMockMetricsRepository()
	badgeRepo := newMockBadgeRepository()
	userRepo := newMockUserRepository()
	log := logger.New("debug", "text", "stdout")

	service := NewServiceWithInterfaces(metricsRepo, badgeRepo, userRepo, log)

	return service, metricsRepo, badgeRepo, userRepo
}

func TestGetGlobalLeaderboard(t *testing.T) {
	service, metricsRepo, badgeRepo, userRepo := setupTestService()

	// Create test users
	user1ID := uint(1)
	user2ID := uint(2)
	user3ID := uint(3)

	userRepo.users[user1ID] = &models.User{
		ID:       user1ID,
		Username: "alice",
		Team:     "team-frontend",
	}
	userRepo.users[user2ID] = &models.User{
		ID:       user2ID,
		Username: "bob",
		Team:     "team-backend",
	}
	userRepo.users[user3ID] = &models.User{
		ID:       user3ID,
		Username: "charlie",
		Team:     "team-ops",
	}

	// Create test metrics
	ttfr1 := 60
	ttfr2 := 120
	ttfr3 := 90
	commentCount1 := 8.0
	commentCount2 := 5.0
	commentCount3 := 10.0
	engagementScore1 := 9.0
	engagementScore2 := 7.0
	engagementScore3 := 8.5

	metricsRepo.metrics = []models.ReviewMetrics{
		{
			UserID:           &user1ID,
			Team:             "team-frontend",
			CompletedReviews: 30,
			AvgTTFR:          &ttfr1,
			AvgCommentCount:  &commentCount1,
			EngagementScore:  &engagementScore1,
		},
		{
			UserID:           &user2ID,
			Team:             "team-backend",
			CompletedReviews: 50,
			AvgTTFR:          &ttfr2,
			AvgCommentCount:  &commentCount2,
			EngagementScore:  &engagementScore2,
		},
		{
			UserID:           &user3ID,
			Team:             "team-ops",
			CompletedReviews: 20,
			AvgTTFR:          &ttfr3,
			AvgCommentCount:  &commentCount3,
			EngagementScore:  &engagementScore3,
		},
	}

	// Set badge counts
	badgeRepo.userBadgeCounts[user1ID] = 3
	badgeRepo.userBadgeCounts[user2ID] = 5
	badgeRepo.userBadgeCounts[user3ID] = 2

	// Get global leaderboard sorted by completed_reviews
	leaderboard, err := service.GetGlobalLeaderboard(context.Background(), "all_time", "completed_reviews", 10)
	if err != nil {
		t.Fatalf("GetGlobalLeaderboard failed: %v", err)
	}

	if len(leaderboard) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(leaderboard))
	}

	// Check ordering (bob should be #1 with 50 reviews)
	if leaderboard[0].Username != "bob" {
		t.Errorf("Expected bob at rank 1, got %s", leaderboard[0].Username)
	}
	if leaderboard[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", leaderboard[0].Rank)
	}
	if leaderboard[0].CompletedReviews != 50 {
		t.Errorf("Expected 50 completed reviews, got %d", leaderboard[0].CompletedReviews)
	}

	// Check badge counts
	if leaderboard[0].BadgeCount != 5 {
		t.Errorf("Expected 5 badges for bob, got %d", leaderboard[0].BadgeCount)
	}
}

func TestGetTeamLeaderboard(t *testing.T) {
	service, metricsRepo, _, userRepo := setupTestService()

	// Create test users (all in same team)
	user1ID := uint(1)
	user2ID := uint(2)
	user3ID := uint(3)

	userRepo.users[user1ID] = &models.User{
		ID:       user1ID,
		Username: "alice",
		Team:     "team-frontend",
	}
	userRepo.users[user2ID] = &models.User{
		ID:       user2ID,
		Username: "bob",
		Team:     "team-frontend",
	}
	userRepo.users[user3ID] = &models.User{
		ID:       user3ID,
		Username: "charlie",
		Team:     "team-backend", // Different team
	}

	// Create test metrics
	engagementScore1 := 8.0
	engagementScore2 := 9.5
	engagementScore3 := 7.0

	metricsRepo.metrics = []models.ReviewMetrics{
		{
			UserID:           &user1ID,
			Team:             "team-frontend",
			CompletedReviews: 30,
			EngagementScore:  &engagementScore1,
		},
		{
			UserID:           &user2ID,
			Team:             "team-frontend",
			CompletedReviews: 40,
			EngagementScore:  &engagementScore2,
		},
		{
			UserID:           &user3ID,
			Team:             "team-backend",
			CompletedReviews: 50,
			EngagementScore:  &engagementScore3,
		},
	}

	// Get team leaderboard for team-frontend
	leaderboard, err := service.GetTeamLeaderboard(context.Background(), "team-frontend", "all_time", "engagement_score", 10)
	if err != nil {
		t.Fatalf("GetTeamLeaderboard failed: %v", err)
	}

	// Should only include team-frontend members
	if len(leaderboard) != 2 {
		t.Errorf("Expected 2 entries for team-frontend, got %d", len(leaderboard))
	}

	// Check ordering (bob should be #1 with 9.5 engagement)
	if leaderboard[0].Username != "bob" {
		t.Errorf("Expected bob at rank 1, got %s", leaderboard[0].Username)
	}
	if leaderboard[0].Team != "team-frontend" {
		t.Errorf("Expected team-frontend, got %s", leaderboard[0].Team)
	}
}

func TestSortLeaderboard_CompletedReviews(t *testing.T) {
	service, _, _, _ := setupTestService()

	entries := []Entry{
		{UserID: 1, Username: "alice", CompletedReviews: 30},
		{UserID: 2, Username: "bob", CompletedReviews: 50},
		{UserID: 3, Username: "charlie", CompletedReviews: 20},
	}

	service.sortLeaderboard(entries, "completed_reviews")

	// Higher is better
	if entries[0].Username != "bob" {
		t.Errorf("Expected bob first, got %s", entries[0].Username)
	}
	if entries[1].Username != "alice" {
		t.Errorf("Expected alice second, got %s", entries[1].Username)
	}
	if entries[2].Username != "charlie" {
		t.Errorf("Expected charlie third, got %s", entries[2].Username)
	}
}

func TestSortLeaderboard_AvgTTFR(t *testing.T) {
	service, _, _, _ := setupTestService()

	entries := []Entry{
		{UserID: 1, Username: "alice", AvgTTFR: 120},
		{UserID: 2, Username: "bob", AvgTTFR: 60},
		{UserID: 3, Username: "charlie", AvgTTFR: 90},
	}

	service.sortLeaderboard(entries, "avg_ttfr")

	// Lower is better for TTFR
	if entries[0].Username != "bob" {
		t.Errorf("Expected bob first (lowest TTFR), got %s", entries[0].Username)
	}
	if entries[1].Username != "charlie" {
		t.Errorf("Expected charlie second, got %s", entries[1].Username)
	}
	if entries[2].Username != "alice" {
		t.Errorf("Expected alice third, got %s", entries[2].Username)
	}
}

func TestSortLeaderboard_EngagementScore(t *testing.T) {
	service, _, _, _ := setupTestService()

	entries := []Entry{
		{UserID: 1, Username: "alice", EngagementScore: 7.5},
		{UserID: 2, Username: "bob", EngagementScore: 9.0},
		{UserID: 3, Username: "charlie", EngagementScore: 8.0},
	}

	service.sortLeaderboard(entries, "engagement_score")

	// Higher is better
	if entries[0].Username != "bob" {
		t.Errorf("Expected bob first, got %s", entries[0].Username)
	}
	if entries[1].Username != "charlie" {
		t.Errorf("Expected charlie second, got %s", entries[1].Username)
	}
	if entries[2].Username != "alice" {
		t.Errorf("Expected alice third, got %s", entries[2].Username)
	}
}

func TestGetUserRank(t *testing.T) {
	service, metricsRepo, _, userRepo := setupTestService()

	// Create test users
	user1ID := uint(1)
	user2ID := uint(2)
	user3ID := uint(3)

	userRepo.users[user1ID] = &models.User{ID: user1ID, Username: "alice", Team: "team-a"}
	userRepo.users[user2ID] = &models.User{ID: user2ID, Username: "bob", Team: "team-a"}
	userRepo.users[user3ID] = &models.User{ID: user3ID, Username: "charlie", Team: "team-a"}

	engagementScore1 := 7.0
	engagementScore2 := 9.0
	engagementScore3 := 8.0

	metricsRepo.metrics = []models.ReviewMetrics{
		{UserID: &user1ID, CompletedReviews: 30, EngagementScore: &engagementScore1},
		{UserID: &user2ID, CompletedReviews: 50, EngagementScore: &engagementScore2},
		{UserID: &user3ID, CompletedReviews: 40, EngagementScore: &engagementScore3},
	}

	// Get rank for user2 (should be rank 1 with highest engagement)
	rank, err := service.GetUserRank(context.Background(), user2ID, "all_time", "engagement_score")
	if err != nil {
		t.Fatalf("GetUserRank failed: %v", err)
	}

	if rank != 1 {
		t.Errorf("Expected rank 1 for bob, got %d", rank)
	}

	// Get rank for user3 (should be rank 2)
	rank, err = service.GetUserRank(context.Background(), user3ID, "all_time", "engagement_score")
	if err != nil {
		t.Fatalf("GetUserRank failed: %v", err)
	}

	if rank != 2 {
		t.Errorf("Expected rank 2 for charlie, got %d", rank)
	}
}

func TestAggregateMetricsByUser(t *testing.T) {
	service, _, _, _ := setupTestService()

	userID := uint(1)
	ttfr1 := 60
	ttfr2 := 120
	commentCount1 := 5.0
	commentCount2 := 7.0
	engagementScore1 := 8.0
	engagementScore2 := 9.0

	metrics := []models.ReviewMetrics{
		{
			UserID:           &userID,
			CompletedReviews: 10,
			AvgTTFR:          &ttfr1,
			AvgCommentCount:  &commentCount1,
			EngagementScore:  &engagementScore1,
		},
		{
			UserID:           &userID,
			CompletedReviews: 15,
			AvgTTFR:          &ttfr2,
			AvgCommentCount:  &commentCount2,
			EngagementScore:  &engagementScore2,
		},
	}

	aggregated := service.aggregateMetricsByUser(metrics)

	if len(aggregated) != 1 {
		t.Errorf("Expected 1 user in aggregated metrics, got %d", len(aggregated))
	}

	userMetrics, ok := aggregated[userID]
	if !ok {
		t.Fatal("User not found in aggregated metrics")
	}

	// Check totals
	expectedCompletedReviews := 25
	if userMetrics.CompletedReviews != expectedCompletedReviews {
		t.Errorf("Expected %d completed reviews, got %d", expectedCompletedReviews, userMetrics.CompletedReviews)
	}

	// Check averages
	expectedAvgTTFR := (60.0 + 120.0) / 2
	if userMetrics.AvgTTFR != expectedAvgTTFR {
		t.Errorf("Expected avg TTFR %f, got %f", expectedAvgTTFR, userMetrics.AvgTTFR)
	}

	expectedAvgCommentCount := (5.0 + 7.0) / 2
	if userMetrics.AvgCommentCount != expectedAvgCommentCount {
		t.Errorf("Expected avg comment count %f, got %f", expectedAvgCommentCount, userMetrics.AvgCommentCount)
	}

	expectedEngagementScore := (8.0 + 9.0) / 2
	if userMetrics.EngagementScore != expectedEngagementScore {
		t.Errorf("Expected engagement score %f, got %f", expectedEngagementScore, userMetrics.EngagementScore)
	}
}

func TestGetUserStats(t *testing.T) {
	service, metricsRepo, badgeRepo, userRepo := setupTestService()

	userID := uint(1)
	userRepo.users[userID] = &models.User{
		ID:       userID,
		Username: "alice",
		Team:     "team-frontend",
	}

	// Add metrics
	ttfr := 90
	timeToApproval := 120
	commentCount := 6.5
	engagementScore := 8.5

	metricsRepo.metrics = []models.ReviewMetrics{
		{
			UserID:            &userID,
			TotalReviews:      40,
			CompletedReviews:  35,
			AvgTTFR:           &ttfr,
			AvgTimeToApproval: &timeToApproval,
			AvgCommentCount:   &commentCount,
			EngagementScore:   &engagementScore,
		},
	}

	// Add badges
	badge1 := models.Badge{ID: 1, Name: "speed_demon"}
	badge2 := models.Badge{ID: 2, Name: "team_player"}
	badgeRepo.userBadges[userID] = []models.UserBadge{
		{UserID: userID, BadgeID: 1, Badge: badge1},
		{UserID: userID, BadgeID: 2, Badge: badge2},
	}

	// Get stats
	stats, err := service.GetUserStats(context.Background(), userID, "all_time")
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	// Verify stats
	if stats.Username != "alice" {
		t.Errorf("Expected username 'alice', got %s", stats.Username)
	}
	if stats.Team != "team-frontend" {
		t.Errorf("Expected team 'team-frontend', got %s", stats.Team)
	}
	if stats.TotalReviews != 40 {
		t.Errorf("Expected 40 total reviews, got %d", stats.TotalReviews)
	}
	if stats.CompletedReviews != 35 {
		t.Errorf("Expected 35 completed reviews, got %d", stats.CompletedReviews)
	}
	if stats.AvgTTFR != 90.0 {
		t.Errorf("Expected avg TTFR 90, got %f", stats.AvgTTFR)
	}
	if len(stats.Badges) != 2 {
		t.Errorf("Expected 2 badges, got %d", len(stats.Badges))
	}
}

func TestCalculatePeriodRange(t *testing.T) {
	now := time.Now()

	tests := []struct {
		period         string
		expectedDelta  time.Duration
		toleranceDelta time.Duration
	}{
		{"day", 24 * time.Hour, 1 * time.Minute},
		{"week", 7 * 24 * time.Hour, 1 * time.Minute},
		{"month", 30 * 24 * time.Hour, 1 * time.Minute},
		{"year", 365 * 24 * time.Hour, 1 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			startDate, endDate := calculatePeriodRange(tt.period)

			// End date should be approximately now
			if endDate.Sub(now) > 1*time.Second {
				t.Errorf("End date not close to now: %v", endDate.Sub(now))
			}

			// Start date should be approximately expectedDelta ago
			actualDelta := now.Sub(startDate)
			deltaError := actualDelta - tt.expectedDelta

			if deltaError < -tt.toleranceDelta || deltaError > tt.toleranceDelta {
				t.Errorf("Period '%s': expected delta ~%v, got %v (error: %v)", tt.period, tt.expectedDelta, actualDelta, deltaError)
			}
		})
	}

	// Test all_time
	t.Run("all_time", func(t *testing.T) {
		startDate, _ := calculatePeriodRange("all_time")
		expected := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		if !startDate.Equal(expected) {
			t.Errorf("Expected start date %v, got %v", expected, startDate)
		}
	})
}

func TestLeaderboard_WithLimit(t *testing.T) {
	service, metricsRepo, _, userRepo := setupTestService()

	// Create 5 users
	for i := uint(1); i <= 5; i++ {
		userRepo.users[i] = &models.User{
			ID:       i,
			Username: "user" + string(rune(i+'0')),
			Team:     "team-test",
		}

		metricsRepo.metrics = append(metricsRepo.metrics, models.ReviewMetrics{
			UserID:           &i,
			CompletedReviews: int(i * 10),
		})
	}

	// Get leaderboard with limit 3
	leaderboard, err := service.GetGlobalLeaderboard(context.Background(), "all_time", "completed_reviews", 3)
	if err != nil {
		t.Fatalf("GetGlobalLeaderboard failed: %v", err)
	}

	// Should only return top 3
	if len(leaderboard) != 3 {
		t.Errorf("Expected 3 entries (limit), got %d", len(leaderboard))
	}
}

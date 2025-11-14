package badges

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// Mock repositories for testing
type mockBadgeRepository struct {
	badges      map[uint]*models.Badge
	userBadges  map[uint]map[uint]bool // userID -> badgeID -> exists
	nextBadgeID uint
}

func newMockBadgeRepository() *mockBadgeRepository {
	return &mockBadgeRepository{
		badges:      make(map[uint]*models.Badge),
		userBadges:  make(map[uint]map[uint]bool),
		nextBadgeID: 1,
	}
}

func (m *mockBadgeRepository) GetAll() ([]models.Badge, error) {
	badges := make([]models.Badge, 0, len(m.badges))
	for _, b := range m.badges {
		badges = append(badges, *b)
	}
	return badges, nil
}

func (m *mockBadgeRepository) GetByID(id uint) (*models.Badge, error) {
	if badge, ok := m.badges[id]; ok {
		return badge, nil
	}
	return nil, nil
}

func (m *mockBadgeRepository) HasUserEarnedBadge(userID, badgeID uint) (bool, error) {
	if userBadges, ok := m.userBadges[userID]; ok {
		return userBadges[badgeID], nil
	}
	return false, nil
}

func (m *mockBadgeRepository) AwardBadge(userID, badgeID uint) error {
	if m.userBadges[userID] == nil {
		m.userBadges[userID] = make(map[uint]bool)
	}
	m.userBadges[userID][badgeID] = true
	return nil
}

func (m *mockBadgeRepository) GetUserBadges(userID uint) ([]models.UserBadge, error) {
	var result []models.UserBadge
	if userBadges, ok := m.userBadges[userID]; ok {
		for badgeID := range userBadges {
			result = append(result, models.UserBadge{
				UserID:   userID,
				BadgeID:  badgeID,
				EarnedAt: time.Now(),
			})
		}
	}
	return result, nil
}

func (m *mockBadgeRepository) GetUsersWithBadge(badgeID uint) ([]models.User, error) {
	var users []models.User
	for userID, badges := range m.userBadges {
		if badges[badgeID] {
			users = append(users, models.User{ID: userID})
		}
	}
	return users, nil
}

func (m *mockBadgeRepository) GetBadgeHoldersCount(badgeID uint) (int64, error) {
	count := int64(0)
	for _, badges := range m.userBadges {
		if badges[badgeID] {
			count++
		}
	}
	return count, nil
}

type mockMetricsRepository struct {
	metrics []models.ReviewMetrics
}

func newMockMetricsRepository() *mockMetricsRepository {
	return &mockMetricsRepository{
		metrics: []models.ReviewMetrics{},
	}
}

func (m *mockMetricsRepository) GetByDateRange(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error) {
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

type mockReviewRepository struct{}

func newMockReviewRepository() *mockReviewRepository {
	return &mockReviewRepository{}
}

type mockUserRepository struct {
	users []models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: []models.User{},
	}
}

func (m *mockUserRepository) List(team, role string) ([]models.User, error) {
	return m.users, nil
}

func (m *mockUserRepository) GetByID(id uint) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

// Test setup helper
func setupTestService() (*Service, *mockBadgeRepository, *mockMetricsRepository, *mockUserRepository) {
	badgeRepo := newMockBadgeRepository()
	metricsRepo := newMockMetricsRepository()
	reviewRepo := newMockReviewRepository()
	userRepo := newMockUserRepository()
	log := logger.New("debug", "text", "stdout")

	service := NewServiceWithInterfaces(badgeRepo, metricsRepo, reviewRepo, userRepo, log)

	return service, badgeRepo, metricsRepo, userRepo
}

func TestEvaluateMetricCriteria(t *testing.T) {
	service, _, _, _ := setupTestService()

	tests := []struct {
		name        string
		operator    string
		threshold   float64
		actualValue float64
		expected    bool
		expectError bool
	}{
		{"Less than - true", "<", 100, 50, true, false},
		{"Less than - false", "<", 100, 150, false, false},
		{"Less than or equal - true (less)", "<=", 100, 50, true, false},
		{"Less than or equal - true (equal)", "<=", 100, 100, true, false},
		{"Less than or equal - false", "<=", 100, 150, false, false},
		{"Greater than - true", ">", 100, 150, true, false},
		{"Greater than - false", ">", 100, 50, false, false},
		{"Greater than or equal - true (greater)", ">=", 100, 150, true, false},
		{"Greater than or equal - true (equal)", ">=", 100, 100, true, false},
		{"Greater than or equal - false", ">=", 100, 50, false, false},
		{"Equal - true", "==", 100, 100, true, false},
		{"Equal - false", "==", 100, 50, false, false},
		{"Invalid operator", "!=", 100, 50, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.evaluateMetricCriteria(tt.operator, tt.threshold, tt.actualValue)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculatePeriodRange(t *testing.T) {
	service, _, _, _ := setupTestService()

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
			startDate, endDate := service.calculatePeriodRange(tt.period)

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
		startDate, _ := service.calculatePeriodRange("all_time")
		expected := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		if !startDate.Equal(expected) {
			t.Errorf("Expected start date %v, got %v", expected, startDate)
		}
	})

	// Test empty string (should default to all_time)
	t.Run("empty", func(t *testing.T) {
		startDate, _ := service.calculatePeriodRange("")
		expected := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		if !startDate.Equal(expected) {
			t.Errorf("Expected start date %v, got %v", expected, startDate)
		}
	})
}

func TestAggregateUserMetrics(t *testing.T) {
	service, _, metricsRepo, _ := setupTestService()

	userID := uint(1)
	ttfr1 := 120
	ttfr2 := 180
	commentCount1 := 5.0
	commentCount2 := 7.0
	engagementScore1 := 8.0
	engagementScore2 := 9.0

	// Add test metrics
	metricsRepo.metrics = []models.ReviewMetrics{
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

	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	metrics, err := service.aggregateUserMetrics(userID, startDate, endDate)
	if err != nil {
		t.Fatalf("aggregateUserMetrics failed: %v", err)
	}

	// Check completed_reviews (should be sum)
	expectedCompleted := 25.0
	if metrics["completed_reviews"] != expectedCompleted {
		t.Errorf("Expected completed_reviews %v, got %v", expectedCompleted, metrics["completed_reviews"])
	}

	// Check avg_ttfr (should be average)
	expectedAvgTTFR := (120.0 + 180.0) / 2
	if metrics["avg_ttfr"] != expectedAvgTTFR {
		t.Errorf("Expected avg_ttfr %v, got %v", expectedAvgTTFR, metrics["avg_ttfr"])
	}

	// Check avg_comment_count (should be average)
	expectedAvgCommentCount := (5.0 + 7.0) / 2
	if metrics["avg_comment_count"] != expectedAvgCommentCount {
		t.Errorf("Expected avg_comment_count %v, got %v", expectedAvgCommentCount, metrics["avg_comment_count"])
	}

	// Check engagement_score (should be average)
	expectedEngagementScore := (8.0 + 9.0) / 2
	if metrics["engagement_score"] != expectedEngagementScore {
		t.Errorf("Expected engagement_score %v, got %v", expectedEngagementScore, metrics["engagement_score"])
	}
}

func TestCheckCriteria_LessThan(t *testing.T) {
	service, _, metricsRepo, _ := setupTestService()

	userID := uint(1)
	ttfr := 60 // 60 minutes

	metricsRepo.metrics = []models.ReviewMetrics{
		{
			UserID:  &userID,
			AvgTTFR: &ttfr,
		},
	}

	// Criteria: avg_ttfr < 120
	criteria := &models.BadgeCriteria{
		Metric:   "avg_ttfr",
		Operator: "<",
		Value:    120.0,
		Period:   "all_time",
	}

	result, err := service.checkCriteria(context.Background(), criteria, userID)
	if err != nil {
		t.Fatalf("checkCriteria failed: %v", err)
	}

	if !result {
		t.Error("Expected user to qualify (60 < 120)")
	}
}

func TestCheckCriteria_GreaterThanOrEqual(t *testing.T) {
	service, _, metricsRepo, _ := setupTestService()

	userID := uint(1)
	commentCount := 7.5

	metricsRepo.metrics = []models.ReviewMetrics{
		{
			UserID:          &userID,
			AvgCommentCount: &commentCount,
		},
	}

	// Criteria: avg_comment_count >= 5
	criteria := &models.BadgeCriteria{
		Metric:   "avg_comment_count",
		Operator: ">=",
		Value:    5.0,
		Period:   "all_time",
	}

	result, err := service.checkCriteria(context.Background(), criteria, userID)
	if err != nil {
		t.Fatalf("checkCriteria failed: %v", err)
	}

	if !result {
		t.Error("Expected user to qualify (7.5 >= 5)")
	}
}

func TestCheckCriteria_NoData(t *testing.T) {
	service, _, metricsRepo, _ := setupTestService()

	userID := uint(999) // User with no metrics

	metricsRepo.metrics = []models.ReviewMetrics{} // Empty

	criteria := &models.BadgeCriteria{
		Metric:   "avg_ttfr",
		Operator: "<",
		Value:    120.0,
		Period:   "all_time",
	}

	result, err := service.checkCriteria(context.Background(), criteria, userID)
	if err != nil {
		t.Fatalf("checkCriteria failed: %v", err)
	}

	if result {
		t.Error("Expected user to NOT qualify (no data)")
	}
}

func TestEvaluateBadge(t *testing.T) {
	service, badgeRepo, metricsRepo, _ := setupTestService()

	userID := uint(1)
	ttfr := 60

	metricsRepo.metrics = []models.ReviewMetrics{
		{
			UserID:  &userID,
			AvgTTFR: &ttfr,
		},
	}

	// Create badge with criteria
	badge := &models.Badge{
		ID:          1,
		Name:        "speed_demon",
		Description: "Fast reviewer",
		Icon:        "⚡",
		Criteria:    json.RawMessage(`{"metric":"avg_ttfr","operator":"<","value":120}`),
	}
	badgeRepo.badges[badge.ID] = badge

	result, err := service.EvaluateBadge(context.Background(), badge, userID)
	if err != nil {
		t.Fatalf("EvaluateBadge failed: %v", err)
	}

	if !result {
		t.Error("Expected user to qualify for speed_demon badge")
	}
}

func TestAwardBadge(t *testing.T) {
	service, badgeRepo, _, _ := setupTestService()

	userID := uint(1)
	badge := &models.Badge{
		ID:   1,
		Name: "test_badge",
	}
	badgeRepo.badges[badge.ID] = badge

	err := service.AwardBadge(context.Background(), userID, badge)
	if err != nil {
		t.Fatalf("AwardBadge failed: %v", err)
	}

	// Verify badge was awarded
	hasEarned, err := badgeRepo.HasUserEarnedBadge(userID, badge.ID)
	if err != nil {
		t.Fatalf("HasUserEarnedBadge failed: %v", err)
	}

	if !hasEarned {
		t.Error("Expected user to have earned the badge")
	}
}

func TestGetUserBadges(t *testing.T) {
	service, badgeRepo, _, _ := setupTestService()

	userID := uint(1)
	badge1 := &models.Badge{ID: 1, Name: "badge1"}
	badge2 := &models.Badge{ID: 2, Name: "badge2"}

	badgeRepo.badges[badge1.ID] = badge1
	badgeRepo.badges[badge2.ID] = badge2

	// Award badges
	_ = badgeRepo.AwardBadge(userID, badge1.ID)
	_ = badgeRepo.AwardBadge(userID, badge2.ID)

	userBadges, err := service.GetUserBadges(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserBadges failed: %v", err)
	}

	if len(userBadges) != 2 {
		t.Errorf("Expected 2 badges, got %d", len(userBadges))
	}
}

func TestGetBadgeCatalog(t *testing.T) {
	service, badgeRepo, _, _ := setupTestService()

	badge1 := &models.Badge{ID: 1, Name: "badge1"}
	badge2 := &models.Badge{ID: 2, Name: "badge2"}
	badge3 := &models.Badge{ID: 3, Name: "badge3"}

	badgeRepo.badges[badge1.ID] = badge1
	badgeRepo.badges[badge2.ID] = badge2
	badgeRepo.badges[badge3.ID] = badge3

	badges, err := service.GetBadgeCatalog(context.Background())
	if err != nil {
		t.Fatalf("GetBadgeCatalog failed: %v", err)
	}

	if len(badges) != 3 {
		t.Errorf("Expected 3 badges, got %d", len(badges))
	}
}

func TestEvaluateTopRanking(t *testing.T) {
	service, _, metricsRepo, _ := setupTestService()

	user1 := uint(1)
	user2 := uint(2)
	user3 := uint(3)

	// User 1: 30 completed reviews
	// User 2: 50 completed reviews (top 1)
	// User 3: 20 completed reviews
	metricsRepo.metrics = []models.ReviewMetrics{
		{UserID: &user1, CompletedReviews: 30},
		{UserID: &user2, CompletedReviews: 50},
		{UserID: &user3, CompletedReviews: 20},
	}

	// Check if user2 is in top 1 for completed_reviews
	result, err := service.evaluateTopRanking(context.Background(), "completed_reviews", 1, "all_time", user2)
	if err != nil {
		t.Fatalf("evaluateTopRanking failed: %v", err)
	}

	if !result {
		t.Error("Expected user2 to be in top 1")
	}

	// Check if user1 is in top 2
	result, err = service.evaluateTopRanking(context.Background(), "completed_reviews", 2, "all_time", user1)
	if err != nil {
		t.Fatalf("evaluateTopRanking failed: %v", err)
	}

	if !result {
		t.Error("Expected user1 to be in top 2")
	}

	// Check if user3 is NOT in top 2
	result, err = service.evaluateTopRanking(context.Background(), "completed_reviews", 2, "all_time", user3)
	if err != nil {
		t.Fatalf("evaluateTopRanking failed: %v", err)
	}

	if result {
		t.Error("Expected user3 to NOT be in top 2")
	}
}

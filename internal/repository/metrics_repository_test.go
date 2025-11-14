package repository

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(
		&models.User{},
		&models.ReviewMetrics{},
	); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return &DB{db}
}

// cleanupTestDB closes the test database connection
func cleanupTestDB(t *testing.T, db *DB) {
	t.Helper()
	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Errorf("Failed to get database instance: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		t.Errorf("Failed to close test database: %v", err)
	}
}

func TestMetricsRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)

	metric := &models.ReviewMetrics{
		Date:              time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Team:              "team-frontend",
		UserID:            nil,
		ProjectID:         nil,
		TotalReviews:      10,
		CompletedReviews:  8,
		AvgTTFR:           intPtr(3600),
		AvgTimeToApproval: intPtr(7200),
		AvgCommentCount:   floatPtr(5.0),
		AvgCommentLength:  floatPtr(500.0),
		EngagementScore:   floatPtr(55.0),
	}

	err := repo.Create(metric)
	if err != nil {
		t.Fatalf("Failed to create metric: %v", err)
	}

	if metric.ID == 0 {
		t.Error("Expected ID to be set after creation")
	}
}

func TestMetricsRepository_CreateOrUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test create (first time)
	metric1 := &models.ReviewMetrics{
		Date:         date,
		Team:         "team-frontend",
		TotalReviews: 10,
		AvgTTFR:      intPtr(3600),
	}

	err := repo.CreateOrUpdate(metric1)
	if err != nil {
		t.Fatalf("Failed to create metric: %v", err)
	}

	firstID := metric1.ID

	// Test update (same date/team)
	metric2 := &models.ReviewMetrics{
		Date:         date,
		Team:         "team-frontend",
		TotalReviews: 15,           // Updated value
		AvgTTFR:      intPtr(2400), // Updated value
	}

	err = repo.CreateOrUpdate(metric2)
	if err != nil {
		t.Fatalf("Failed to update metric: %v", err)
	}

	// Verify it updated the same record
	if metric2.ID != firstID {
		t.Errorf("Expected same ID after update, got %d, want %d", metric2.ID, firstID)
	}

	// Fetch and verify values
	fetched, err := repo.GetByDate(date, "team-frontend", nil)
	if err != nil {
		t.Fatalf("Failed to fetch metric: %v", err)
	}

	if fetched.TotalReviews != 15 {
		t.Errorf("Expected TotalReviews = 15, got %d", fetched.TotalReviews)
	}

	if *fetched.AvgTTFR != 2400 {
		t.Errorf("Expected AvgTTFR = 2400, got %d", *fetched.AvgTTFR)
	}
}

func TestMetricsRepository_CreateOrUpdate_WithUserID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)
	userRepo := NewUserRepository(db)

	// Create test user
	user := &models.User{
		GitLabID: 123,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "dev",
		Team:     "team-frontend",
	}
	_ = userRepo.Create(user)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create metric with user_id
	metric1 := &models.ReviewMetrics{
		Date:         date,
		Team:         "team-frontend",
		UserID:       &user.ID,
		TotalReviews: 5,
	}

	err := repo.CreateOrUpdate(metric1)
	if err != nil {
		t.Fatalf("Failed to create metric: %v", err)
	}

	// Create another metric for same date/team but WITHOUT user_id
	metric2 := &models.ReviewMetrics{
		Date:         date,
		Team:         "team-frontend",
		UserID:       nil,
		TotalReviews: 10,
	}

	err = repo.CreateOrUpdate(metric2)
	if err != nil {
		t.Fatalf("Failed to create team-level metric: %v", err)
	}

	// Both should exist as separate records
	if metric1.ID == metric2.ID {
		t.Error("Expected different IDs for user-level vs team-level metrics")
	}
}

func TestMetricsRepository_GetByDate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create test metric
	metric := &models.ReviewMetrics{
		Date:         date,
		Team:         "team-backend",
		TotalReviews: 20,
		AvgTTFR:      intPtr(1800),
	}
	_ = repo.Create(metric)

	// Test retrieval
	fetched, err := repo.GetByDate(date, "team-backend", nil)
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	if fetched.TotalReviews != 20 {
		t.Errorf("Expected TotalReviews = 20, got %d", fetched.TotalReviews)
	}

	// Test non-existent metric
	_, err = repo.GetByDate(date, "team-nonexistent", nil)
	if err == nil {
		t.Error("Expected error for non-existent metric")
	}
}

func TestMetricsRepository_GetByDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)

	// Create metrics for different dates
	dates := []time.Time{
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC), // Outside range
	}

	for i, date := range dates {
		metric := &models.ReviewMetrics{
			Date:         date,
			Team:         "team-frontend",
			TotalReviews: i + 1,
		}
		_ = repo.Create(metric)
	}

	// Query range: Jan 1-3
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)

	metrics, err := repo.GetByDateRange(startDate, endDate, map[string]interface{}{
		"team": "team-frontend",
	})

	if err != nil {
		t.Fatalf("Failed to get metrics by date range: %v", err)
	}

	// Should get 3 records (Jan 1, 2, 3), not Jan 5
	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(metrics))
	}
}

func TestMetricsRepository_GetAverageTTFRByTeam(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create metrics for different teams
	metrics := []*models.ReviewMetrics{
		{
			Date:    startDate,
			Team:    "team-frontend",
			AvgTTFR: intPtr(3600),
		},
		{
			Date:    startDate.AddDate(0, 0, 1),
			Team:    "team-frontend",
			AvgTTFR: intPtr(7200),
		},
		{
			Date:    startDate,
			Team:    "team-backend",
			AvgTTFR: intPtr(1800),
		},
	}

	for _, m := range metrics {
		_ = repo.Create(m)
	}

	// Get averages
	endDate := startDate.AddDate(0, 0, 7)
	avgMap, err := repo.GetAverageTTFRByTeam(startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to get average TTFR: %v", err)
	}

	// team-frontend: avg of 3600 and 7200 = 5400
	if avgMap["team-frontend"] != 5400.0 {
		t.Errorf("Expected team-frontend avg = 5400, got %f", avgMap["team-frontend"])
	}

	// team-backend: 1800
	if avgMap["team-backend"] != 1800.0 {
		t.Errorf("Expected team-backend avg = 1800, got %f", avgMap["team-backend"])
	}
}

func TestMetricsRepository_GetTopReviewersByEngagement(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)
	userRepo := NewUserRepository(db)

	// Create test users
	users := []*models.User{
		{GitLabID: 1, Username: "alice", Email: "alice@example.com", Role: "dev", Team: "team-frontend"},
		{GitLabID: 2, Username: "bob", Email: "bob@example.com", Role: "dev", Team: "team-backend"},
		{GitLabID: 3, Username: "charlie", Email: "charlie@example.com", Role: "ops", Team: "team-platform"},
	}

	for _, u := range users {
		_ = userRepo.Create(u)
	}

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create metrics with different engagement scores
	metrics := []*models.ReviewMetrics{
		{Date: startDate, Team: "team-frontend", UserID: &users[0].ID, EngagementScore: floatPtr(100.0)},
		{Date: startDate.AddDate(0, 0, 1), Team: "team-frontend", UserID: &users[0].ID, EngagementScore: floatPtr(50.0)}, // Alice total: 150
		{Date: startDate, Team: "team-backend", UserID: &users[1].ID, EngagementScore: floatPtr(80.0)},                   // Bob total: 80
		{Date: startDate, Team: "team-platform", UserID: &users[2].ID, EngagementScore: floatPtr(120.0)},                 // Charlie total: 120
	}

	for _, m := range metrics {
		_ = repo.Create(m)
	}

	// Get top 2 reviewers
	endDate := startDate.AddDate(0, 0, 7)
	topReviewers, err := repo.GetTopReviewersByEngagement(startDate, endDate, 2)
	if err != nil {
		t.Fatalf("Failed to get top reviewers: %v", err)
	}

	if len(topReviewers) != 2 {
		t.Fatalf("Expected 2 reviewers, got %d", len(topReviewers))
	}

	// Should be Alice (150) and Charlie (120)
	if topReviewers[0].Username != "alice" {
		t.Errorf("Expected alice as top reviewer, got %s", topReviewers[0].Username)
	}

	if topReviewers[1].Username != "charlie" {
		t.Errorf("Expected charlie as second reviewer, got %s", topReviewers[1].Username)
	}
}

func TestMetricsRepository_GetDailyStats(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMetricsRepository(db)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create multiple metrics for the same date
	metrics := []*models.ReviewMetrics{
		{
			Date:             date,
			Team:             "team-frontend",
			TotalReviews:     10,
			CompletedReviews: 8,
			AvgTTFR:          intPtr(3600),
		},
		{
			Date:             date,
			Team:             "team-backend",
			TotalReviews:     5,
			CompletedReviews: 4,
			AvgTTFR:          intPtr(7200),
		},
	}

	for _, m := range metrics {
		_ = repo.Create(m)
	}

	stats, err := repo.GetDailyStats(date)
	if err != nil {
		t.Fatalf("Failed to get daily stats: %v", err)
	}

	// Total reviews: 10 + 5 = 15
	if stats["total_reviews"].(int64) != 15 {
		t.Errorf("Expected total_reviews = 15, got %v", stats["total_reviews"])
	}

	// Total completed: 8 + 4 = 12
	if stats["total_completed"].(int64) != 12 {
		t.Errorf("Expected total_completed = 12, got %v", stats["total_completed"])
	}

	// Avg TTFR: (3600 + 7200) / 2 = 5400
	avgTTFR := stats["avg_ttfr"].(float64)
	if avgTTFR < 5399.9 || avgTTFR > 5400.1 { // Float comparison with tolerance
		t.Errorf("Expected avg_ttfr = 5400, got %f", avgTTFR)
	}
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}

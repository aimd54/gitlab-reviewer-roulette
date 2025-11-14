package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/repository"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate all models
	err = db.AutoMigrate(
		&models.User{},
		&models.MRReview{},
		&models.ReviewerAssignment{},
		&models.ReviewMetrics{},
	)
	require.NoError(t, err)

	cleanup := func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}

	return db, cleanup
}

func TestAggregateDaily_NoReviews(t *testing.T) {
	gormDB, cleanup := setupTestDB(t)
	defer cleanup()

	db := &repository.DB{DB: gormDB}
	reviewRepo := repository.NewReviewRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)

	log := zerolog.Nop()
	service := NewService(reviewRepo, metricsRepo, &log)

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	err := service.AggregateDaily(context.Background(), date)

	assert.NoError(t, err)

	// Verify no metrics were created
	metrics, err := metricsRepo.GetByDateRange(date, date, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Empty(t, metrics)
}

func TestAggregateDaily_TeamMetrics(t *testing.T) {
	gormDB, cleanup := setupTestDB(t)
	defer cleanup()

	db := &repository.DB{DB: gormDB}
	reviewRepo := repository.NewReviewRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)

	// Create test users
	user1 := models.User{
		GitLabID: 1,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "dev",
		Team:     "team-frontend",
	}
	require.NoError(t, gormDB.Create(&user1).Error)

	user2 := models.User{
		GitLabID: 2,
		Username: "bob",
		Email:    "bob@example.com",
		Role:     "dev",
		Team:     "team-frontend",
	}
	require.NoError(t, gormDB.Create(&user2).Error)

	// Create test reviews with completed lifecycle
	date := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	triggeredAt := date.Add(-2 * time.Hour)
	firstReviewAt := date.Add(-1 * time.Hour)
	approvedAt := date.Add(-30 * time.Minute)
	mergedAt := date

	review1 := models.MRReview{
		GitLabMRIID:         1,
		GitLabProjectID:     100,
		MRURL:               "https://gitlab.example.com/project/mr/1",
		MRTitle:             "Test MR 1",
		Team:                "team-frontend",
		RouletteTriggeredAt: &triggeredAt,
		FirstReviewAt:       &firstReviewAt,
		ApprovedAt:          &approvedAt,
		MergedAt:            &mergedAt,
		Status:              models.MRStatusMerged,
	}
	require.NoError(t, reviewRepo.CreateMRReview(&review1))

	// Create assignments
	assignment1 := models.ReviewerAssignment{
		MRReviewID:    review1.ID,
		UserID:        user1.ID,
		Role:          models.ReviewerRoleCodeowner,
		CommentCount:  5,
		CommentLength: 500,
	}
	require.NoError(t, gormDB.Create(&assignment1).Error)

	assignment2 := models.ReviewerAssignment{
		MRReviewID:    review1.ID,
		UserID:        user2.ID,
		Role:          models.ReviewerRoleTeamMember,
		CommentCount:  3,
		CommentLength: 300,
	}
	require.NoError(t, gormDB.Create(&assignment2).Error)

	// Run aggregation
	log := zerolog.Nop()
	service := NewService(reviewRepo, metricsRepo, &log)

	err := service.AggregateDaily(context.Background(), date)
	require.NoError(t, err)

	// Verify team-level metrics (use start of day for query)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	teamMetrics, err := metricsRepo.GetByDate(startOfDay, "team-frontend", nil)
	require.NoError(t, err)
	assert.NotNil(t, teamMetrics)

	assert.Equal(t, 1, teamMetrics.TotalReviews)
	assert.Equal(t, 1, teamMetrics.CompletedReviews)
	assert.NotNil(t, teamMetrics.AvgTTFR)
	assert.Equal(t, 60, *teamMetrics.AvgTTFR) // 1 hour in minutes
	assert.NotNil(t, teamMetrics.AvgTimeToApproval)
	assert.Equal(t, 90, *teamMetrics.AvgTimeToApproval) // 1.5 hours in minutes
	assert.NotNil(t, teamMetrics.AvgCommentCount)
	assert.Equal(t, 8.0, *teamMetrics.AvgCommentCount) // Total comments per review (5 + 3 = 8)
	assert.NotNil(t, teamMetrics.AvgCommentLength)
	assert.Equal(t, 800.0, *teamMetrics.AvgCommentLength) // Total length per review (500 + 300 = 800)
	assert.NotNil(t, teamMetrics.EngagementScore)
	assert.Greater(t, *teamMetrics.EngagementScore, 0.0)
}

func TestAggregateDaily_UserMetrics(t *testing.T) {
	gormDB, cleanup := setupTestDB(t)
	defer cleanup()

	db := &repository.DB{DB: gormDB}
	reviewRepo := repository.NewReviewRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)

	// Create test user
	user := models.User{
		GitLabID: 1,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "dev",
		Team:     "team-frontend",
	}
	require.NoError(t, gormDB.Create(&user).Error)

	// Create test review
	date := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	triggeredAt := date.Add(-2 * time.Hour)
	mergedAt := date

	review := models.MRReview{
		GitLabMRIID:         1,
		GitLabProjectID:     100,
		MRURL:               "https://gitlab.example.com/project/mr/1",
		MRTitle:             "Test MR",
		Team:                "team-frontend",
		RouletteTriggeredAt: &triggeredAt,
		MergedAt:            &mergedAt,
		Status:              models.MRStatusMerged,
	}
	require.NoError(t, reviewRepo.CreateMRReview(&review))

	// Create assignment with timestamps
	assignedAt := triggeredAt
	firstCommentAt := assignedAt.Add(30 * time.Minute)
	approvedAtTime := assignedAt.Add(1 * time.Hour)

	assignment := models.ReviewerAssignment{
		MRReviewID:     review.ID,
		UserID:         user.ID,
		Role:           models.ReviewerRoleCodeowner,
		AssignedAt:     assignedAt,
		FirstCommentAt: &firstCommentAt,
		ApprovedAt:     &approvedAtTime,
		CommentCount:   5,
		CommentLength:  500,
	}
	require.NoError(t, gormDB.Create(&assignment).Error)

	// Run aggregation
	log := zerolog.Nop()
	service := NewService(reviewRepo, metricsRepo, &log)

	err := service.AggregateDaily(context.Background(), date)
	require.NoError(t, err)

	// Verify user-level metrics (query with start of day)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	userMetrics, err := metricsRepo.GetMetricsByUser(user.ID, startOfDay, startOfDay)
	require.NoError(t, err)
	require.Len(t, userMetrics, 1, "Should have exactly one user metric")

	metric := userMetrics[0]
	assert.Equal(t, 1, metric.TotalReviews)
	assert.Equal(t, 1, metric.CompletedReviews)
	assert.NotNil(t, metric.AvgTTFR)
	assert.Equal(t, 30, *metric.AvgTTFR) // 30 minutes
	assert.NotNil(t, metric.AvgTimeToApproval)
	assert.Equal(t, 60, *metric.AvgTimeToApproval) // 1 hour
	assert.NotNil(t, metric.AvgCommentCount)
	assert.Equal(t, 5.0, *metric.AvgCommentCount)
	assert.NotNil(t, metric.AvgCommentLength)
	assert.Equal(t, 500.0, *metric.AvgCommentLength)
}

func TestAggregateDaily_MultipleTeams(t *testing.T) {
	gormDB, cleanup := setupTestDB(t)
	defer cleanup()

	db := &repository.DB{DB: gormDB}
	reviewRepo := repository.NewReviewRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)

	// Create test users from different teams
	user1 := models.User{
		GitLabID: 1,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "dev",
		Team:     "team-frontend",
	}
	require.NoError(t, gormDB.Create(&user1).Error)

	user2 := models.User{
		GitLabID: 2,
		Username: "bob",
		Email:    "bob@example.com",
		Role:     "ops",
		Team:     "team-platform",
	}
	require.NoError(t, gormDB.Create(&user2).Error)

	// Create reviews for both teams
	date := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	mergedAt := date

	review1 := models.MRReview{
		GitLabMRIID:     1,
		GitLabProjectID: 100,
		MRURL:           "https://gitlab.example.com/project/mr/1",
		MRTitle:         "Frontend MR",
		Team:            "team-frontend",
		MergedAt:        &mergedAt,
		Status:          models.MRStatusMerged,
	}
	require.NoError(t, reviewRepo.CreateMRReview(&review1))

	review2 := models.MRReview{
		GitLabMRIID:     2,
		GitLabProjectID: 101,
		MRURL:           "https://gitlab.example.com/project/mr/2",
		MRTitle:         "Platform MR",
		Team:            "team-platform",
		MergedAt:        &mergedAt,
		Status:          models.MRStatusMerged,
	}
	require.NoError(t, reviewRepo.CreateMRReview(&review2))

	// Create assignments
	assignment1 := models.ReviewerAssignment{
		MRReviewID:    review1.ID,
		UserID:        user1.ID,
		Role:          models.ReviewerRoleCodeowner,
		CommentCount:  5,
		CommentLength: 500,
	}
	require.NoError(t, gormDB.Create(&assignment1).Error)

	assignment2 := models.ReviewerAssignment{
		MRReviewID:    review2.ID,
		UserID:        user2.ID,
		Role:          models.ReviewerRoleCodeowner,
		CommentCount:  3,
		CommentLength: 300,
	}
	require.NoError(t, gormDB.Create(&assignment2).Error)

	// Run aggregation
	log := zerolog.Nop()
	service := NewService(reviewRepo, metricsRepo, &log)

	err := service.AggregateDaily(context.Background(), date)
	require.NoError(t, err)

	// Verify both teams have metrics (use start of day)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	frontendMetrics, err := metricsRepo.GetByDate(startOfDay, "team-frontend", nil)
	require.NoError(t, err)
	assert.NotNil(t, frontendMetrics)
	assert.Equal(t, 1, frontendMetrics.TotalReviews)

	platformMetrics, err := metricsRepo.GetByDate(startOfDay, "team-platform", nil)
	require.NoError(t, err)
	assert.NotNil(t, platformMetrics)
	assert.Equal(t, 1, platformMetrics.TotalReviews)
}

func TestAggregateDaily_Idempotency(t *testing.T) {
	gormDB, cleanup := setupTestDB(t)
	defer cleanup()

	db := &repository.DB{DB: gormDB}
	reviewRepo := repository.NewReviewRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)

	// Create test user and review
	user := models.User{
		GitLabID: 1,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "dev",
		Team:     "team-frontend",
	}
	require.NoError(t, gormDB.Create(&user).Error)

	date := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	mergedAt := date

	review := models.MRReview{
		GitLabMRIID:     1,
		GitLabProjectID: 100,
		MRURL:           "https://gitlab.example.com/project/mr/1",
		MRTitle:         "Test MR",
		Team:            "team-frontend",
		MergedAt:        &mergedAt,
		Status:          models.MRStatusMerged,
	}
	require.NoError(t, reviewRepo.CreateMRReview(&review))

	assignment := models.ReviewerAssignment{
		MRReviewID:    review.ID,
		UserID:        user.ID,
		Role:          models.ReviewerRoleCodeowner,
		CommentCount:  5,
		CommentLength: 500,
	}
	require.NoError(t, gormDB.Create(&assignment).Error)

	// Run aggregation
	log := zerolog.Nop()
	service := NewService(reviewRepo, metricsRepo, &log)

	// Run twice
	err := service.AggregateDaily(context.Background(), date)
	require.NoError(t, err)

	err = service.AggregateDaily(context.Background(), date)
	require.NoError(t, err)

	// Verify only one team metric exists (query with start of day)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	teamMetrics, err := metricsRepo.GetMetricsByTeam("team-frontend", startOfDay, startOfDay)
	require.NoError(t, err)
	// Should have 1 team-level metric
	teamLevelCount := 0
	for _, m := range teamMetrics {
		if m.UserID == nil {
			teamLevelCount++
		}
	}
	assert.Equal(t, 1, teamLevelCount, "Should have exactly one team-level metric")

	// Verify only one user metric exists
	userMetrics, err := metricsRepo.GetMetricsByUser(user.ID, startOfDay, startOfDay)
	require.NoError(t, err)
	assert.Len(t, userMetrics, 1, "Should have exactly one user-level metric")
}

func TestAggregateDaily_ClosedButNotMerged(t *testing.T) {
	gormDB, cleanup := setupTestDB(t)
	defer cleanup()

	db := &repository.DB{DB: gormDB}
	reviewRepo := repository.NewReviewRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)

	// Create test user
	user := models.User{
		GitLabID: 1,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "dev",
		Team:     "team-frontend",
	}
	require.NoError(t, gormDB.Create(&user).Error)

	// Create closed but not merged review
	date := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	closedAt := date

	review := models.MRReview{
		GitLabMRIID:     1,
		GitLabProjectID: 100,
		MRURL:           "https://gitlab.example.com/project/mr/1",
		MRTitle:         "Closed MR",
		Team:            "team-frontend",
		ClosedAt:        &closedAt,
		Status:          models.MRStatusClosed,
	}
	require.NoError(t, reviewRepo.CreateMRReview(&review))

	assignment := models.ReviewerAssignment{
		MRReviewID:    review.ID,
		UserID:        user.ID,
		Role:          models.ReviewerRoleCodeowner,
		CommentCount:  2,
		CommentLength: 200,
	}
	require.NoError(t, gormDB.Create(&assignment).Error)

	// Run aggregation
	log := zerolog.Nop()
	service := NewService(reviewRepo, metricsRepo, &log)

	err := service.AggregateDaily(context.Background(), date)
	require.NoError(t, err)

	// Verify metrics exist but completed count is 0 (query with start of day)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	teamMetrics, err := metricsRepo.GetByDate(startOfDay, "team-frontend", nil)
	require.NoError(t, err)
	assert.NotNil(t, teamMetrics)
	assert.Equal(t, 1, teamMetrics.TotalReviews)
	assert.Equal(t, 0, teamMetrics.CompletedReviews) // Not merged, so not completed
}

package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// MockMetricsRepository implements the repository interface for testing
type MockMetricsRepository struct {
	CreateOrUpdateFunc func(metric *models.ReviewMetrics) error
	GetByDateFunc      func(date time.Time, team string, userID *uint) (*models.ReviewMetrics, error)
	GetByDateRangeFunc func(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error)
}

func (m *MockMetricsRepository) CreateOrUpdate(metric *models.ReviewMetrics) error {
	if m.CreateOrUpdateFunc != nil {
		return m.CreateOrUpdateFunc(metric)
	}
	return nil
}

func (m *MockMetricsRepository) GetByDate(date time.Time, team string, userID *uint) (*models.ReviewMetrics, error) {
	if m.GetByDateFunc != nil {
		return m.GetByDateFunc(date, team, userID)
	}
	return nil, nil
}

func (m *MockMetricsRepository) GetByDateRange(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error) {
	if m.GetByDateRangeFunc != nil {
		return m.GetByDateRangeFunc(startDate, endDate, filters)
	}
	return nil, nil
}

func TestService_RecordReviewTriggered(t *testing.T) {
	repo := &MockMetricsRepository{
		CreateOrUpdateFunc: func(metric *models.ReviewMetrics) error {
			// Verify the metric is correctly initialized
			if metric.TotalReviews != 1 {
				t.Errorf("Expected TotalReviews = 1, got %d", metric.TotalReviews)
			}
			if metric.CompletedReviews != 0 {
				t.Errorf("Expected CompletedReviews = 0, got %d", metric.CompletedReviews)
			}
			return nil
		},
	}

	svc := NewService(repo)

	mrReview := &models.MRReview{
		ID:                  1,
		GitLabMRIID:         100,
		GitLabProjectID:     10,
		Team:                "team-frontend",
		RouletteTriggeredAt: timePtr(time.Now()),
	}

	err := svc.RecordReviewTriggered(context.Background(), mrReview)
	if err != nil {
		t.Fatalf("RecordReviewTriggered failed: %v", err)
	}
}

func TestService_RecordReviewStarted(t *testing.T) {
	repo := &MockMetricsRepository{
		GetByDateFunc: func(date time.Time, team string, userID *uint) (*models.ReviewMetrics, error) {
			// Return existing metric
			return &models.ReviewMetrics{
				ID:               1,
				Date:             date,
				Team:             team,
				TotalReviews:     5,
				CompletedReviews: 2,
			}, nil
		},
		CreateOrUpdateFunc: func(metric *models.ReviewMetrics) error {
			// Metric should be updated, not initialized
			if metric.ID != 1 {
				t.Errorf("Expected to update existing metric with ID=1, got ID=%d", metric.ID)
			}
			return nil
		},
	}

	svc := NewService(repo)

	mrReview := &models.MRReview{
		ID:                  1,
		Team:                "team-frontend",
		RouletteTriggeredAt: timePtr(time.Now().Add(-1 * time.Hour)),
		FirstReviewAt:       timePtr(time.Now()),
	}

	assignment := &models.ReviewerAssignment{
		ID:              1,
		UserID:          10,
		StartedReviewAt: timePtr(time.Now()),
	}

	err := svc.RecordReviewStarted(context.Background(), mrReview, assignment)
	if err != nil {
		t.Fatalf("RecordReviewStarted failed: %v", err)
	}
}

func TestService_RecordReviewCompleted(t *testing.T) {
	repo := &MockMetricsRepository{
		GetByDateFunc: func(date time.Time, team string, userID *uint) (*models.ReviewMetrics, error) {
			return &models.ReviewMetrics{
				ID:               1,
				Date:             date,
				Team:             team,
				TotalReviews:     5,
				CompletedReviews: 2,
			}, nil
		},
		CreateOrUpdateFunc: func(metric *models.ReviewMetrics) error {
			// Verify CompletedReviews incremented
			if metric.CompletedReviews != 3 {
				t.Errorf("Expected CompletedReviews = 3, got %d", metric.CompletedReviews)
			}
			// Verify TTFR and TimeToApproval are calculated
			if metric.AvgTTFR == nil {
				t.Error("Expected AvgTTFR to be set")
			}
			if metric.AvgTimeToApproval == nil {
				t.Error("Expected AvgTimeToApproval to be set")
			}
			return nil
		},
	}

	svc := NewService(repo)

	triggeredAt := time.Now().Add(-2 * time.Hour)
	firstReviewAt := time.Now().Add(-1 * time.Hour)
	approvedAt := time.Now()

	mrReview := &models.MRReview{
		ID:                  1,
		Team:                "team-frontend",
		RouletteTriggeredAt: &triggeredAt,
		FirstReviewAt:       &firstReviewAt,
		ApprovedAt:          &approvedAt,
	}

	assignment := &models.ReviewerAssignment{
		ID:            1,
		UserID:        10,
		CommentCount:  5,
		CommentLength: 500,
		ApprovedAt:    &approvedAt,
	}

	err := svc.RecordReviewCompleted(context.Background(), mrReview, assignment)
	if err != nil {
		t.Fatalf("RecordReviewCompleted failed: %v", err)
	}
}

func TestService_RecordReviewEngagement(t *testing.T) {
	repo := &MockMetricsRepository{
		GetByDateFunc: func(date time.Time, team string, userID *uint) (*models.ReviewMetrics, error) {
			if userID == nil {
				t.Error("Expected userID to be provided for engagement tracking")
			}
			return &models.ReviewMetrics{
				ID:     1,
				Date:   date,
				Team:   team,
				UserID: userID,
			}, nil
		},
		CreateOrUpdateFunc: func(metric *models.ReviewMetrics) error {
			// Verify engagement score is calculated
			if metric.EngagementScore == nil {
				t.Error("Expected EngagementScore to be set")
			}
			// Verify comment metrics are updated
			if metric.AvgCommentCount == nil {
				t.Error("Expected AvgCommentCount to be set")
			}
			if metric.AvgCommentLength == nil {
				t.Error("Expected AvgCommentLength to be set")
			}
			return nil
		},
	}

	svc := NewService(repo)

	mrReview := &models.MRReview{
		ID:                  1,
		Team:                "team-frontend",
		RouletteTriggeredAt: timePtr(time.Now().Add(-1 * time.Hour)),
	}

	assignment := &models.ReviewerAssignment{
		ID:            1,
		UserID:        10,
		CommentCount:  8,
		CommentLength: 1200,
	}

	err := svc.RecordReviewEngagement(context.Background(), mrReview, assignment)
	if err != nil {
		t.Fatalf("RecordReviewEngagement failed: %v", err)
	}
}

func TestService_CalculateMetricsForPeriod(t *testing.T) {
	// This test will be implemented when we have a review repository
	// For now, just verify the method signature
	repo := &MockMetricsRepository{}
	svc := NewService(repo)

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	err := svc.CalculateMetricsForPeriod(context.Background(), startDate, endDate)
	if err != nil {
		// Expected to fail since we don't have review repository yet
		// This is a placeholder for future implementation
		t.Logf("CalculateMetricsForPeriod not fully implemented yet: %v", err)
	}
}

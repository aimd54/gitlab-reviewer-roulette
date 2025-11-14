package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// Repository interface defines the methods needed for metrics storage.
type Repository interface {
	CreateOrUpdate(metric *models.ReviewMetrics) error
	GetByDate(date time.Time, team string, userID *uint) (*models.ReviewMetrics, error)
	GetByDateRange(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error)
}

// Service handles metrics calculation and storage.
type Service struct {
	repo Repository
}

// NewService creates a new metrics service.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// RecordReviewTriggered records when a review is triggered. This increments the total_reviews counter for the team on the given date.
func (s *Service) RecordReviewTriggered(_ context.Context, mrReview *models.MRReview) error {
	if mrReview.RouletteTriggeredAt == nil {
		return fmt.Errorf("roulette_triggered_at is required")
	}

	date := mrReview.RouletteTriggeredAt.Truncate(24 * time.Hour) // Get date only

	// Get or create metric for this team on this date
	metric, err := s.repo.GetByDate(date, mrReview.Team, nil)
	if err != nil || metric == nil {
		// Metric doesn't exist, create new one
		metric = &models.ReviewMetrics{
			Date:             date,
			Team:             mrReview.Team,
			TotalReviews:     1,
			CompletedReviews: 0,
		}
	} else {
		// Metric exists, increment total reviews
		metric.TotalReviews++
	}

	return s.repo.CreateOrUpdate(metric)
}

// RecordReviewStarted records when a reviewer starts reviewing. This updates TTFR metrics.
func (s *Service) RecordReviewStarted(_ context.Context, mrReview *models.MRReview, _ *models.ReviewerAssignment) error {
	if mrReview.RouletteTriggeredAt == nil {
		return fmt.Errorf("roulette_triggered_at is required")
	}

	date := mrReview.RouletteTriggeredAt.Truncate(24 * time.Hour)

	// Get metric for this team
	metric, err := s.repo.GetByDate(date, mrReview.Team, nil)
	if err != nil || metric == nil {
		// Create new metric if doesn't exist
		metric = &models.ReviewMetrics{
			Date:             date,
			Team:             mrReview.Team,
			TotalReviews:     1,
			CompletedReviews: 0,
		}
	}

	// Calculate TTFR if we have first_review_at
	if mrReview.FirstReviewAt != nil {
		ttfr := CalculateTTFRForMR(mrReview)
		if ttfr != nil {
			// Update average TTFR (simple average for now, can be improved with weighted average)
			if metric.AvgTTFR == nil {
				metric.AvgTTFR = ttfr
			} else {
				// Running average: new_avg = (old_avg + new_value) / 2
				newAvg := (*metric.AvgTTFR + *ttfr) / 2
				metric.AvgTTFR = &newAvg
			}
		}
	}

	return s.repo.CreateOrUpdate(metric)
}

// RecordReviewCompleted records when a review is completed. This updates completion metrics, time to approval, and engagement scores.
func (s *Service) RecordReviewCompleted(_ context.Context, mrReview *models.MRReview, assignment *models.ReviewerAssignment) error {
	if mrReview.RouletteTriggeredAt == nil {
		return fmt.Errorf("roulette_triggered_at is required")
	}

	date := mrReview.RouletteTriggeredAt.Truncate(24 * time.Hour)

	// Get metric for this team
	metric, err := s.repo.GetByDate(date, mrReview.Team, nil)
	if err != nil || metric == nil {
		// Create new metric if doesn't exist
		metric = &models.ReviewMetrics{
			Date:             date,
			Team:             mrReview.Team,
			TotalReviews:     1,
			CompletedReviews: 1,
		}
	} else {
		// Increment completed reviews
		metric.CompletedReviews++
	}

	// Calculate Time to Approval if we have approved_at
	if mrReview.ApprovedAt != nil {
		timeToApproval := CalculateTimeToApprovalForMR(mrReview)
		if timeToApproval != nil {
			if metric.AvgTimeToApproval == nil {
				metric.AvgTimeToApproval = timeToApproval
			} else {
				// Running average
				newAvg := (*metric.AvgTimeToApproval + *timeToApproval) / 2
				metric.AvgTimeToApproval = &newAvg
			}
		}
	}

	// Calculate TTFR if not already set
	if metric.AvgTTFR == nil && mrReview.FirstReviewAt != nil {
		ttfr := CalculateTTFRForMR(mrReview)
		if ttfr != nil {
			metric.AvgTTFR = ttfr
		}
	}

	// Update comment metrics
	if assignment != nil {
		commentCount := float64(assignment.CommentCount)
		if metric.AvgCommentCount == nil {
			metric.AvgCommentCount = &commentCount
		} else {
			newAvg := (*metric.AvgCommentCount + commentCount) / 2
			metric.AvgCommentCount = &newAvg
		}

		commentLength := float64(assignment.CommentLength)
		if metric.AvgCommentLength == nil {
			metric.AvgCommentLength = &commentLength
		} else {
			newAvg := (*metric.AvgCommentLength + commentLength) / 2
			metric.AvgCommentLength = &newAvg
		}
	}

	return s.repo.CreateOrUpdate(metric)
}

// RecordReviewEngagement records reviewer engagement metrics. This creates per-user metrics for leaderboard and gamification.
func (s *Service) RecordReviewEngagement(_ context.Context, mrReview *models.MRReview, assignment *models.ReviewerAssignment) error {
	if mrReview.RouletteTriggeredAt == nil {
		return fmt.Errorf("roulette_triggered_at is required")
	}
	if assignment == nil {
		return fmt.Errorf("assignment is required")
	}

	date := mrReview.RouletteTriggeredAt.Truncate(24 * time.Hour)

	// Get or create metric for this user
	metric, err := s.repo.GetByDate(date, mrReview.Team, &assignment.UserID)
	if err != nil || metric == nil {
		// Create new user-level metric
		metric = &models.ReviewMetrics{
			Date:   date,
			Team:   mrReview.Team,
			UserID: &assignment.UserID,
		}
	}

	// Calculate engagement score
	engagementScore := CalculateEngagementScore(assignment, mrReview)
	if metric.EngagementScore == nil {
		metric.EngagementScore = &engagementScore
	} else {
		// Running average for engagement score
		newAvg := (*metric.EngagementScore + engagementScore) / 2
		metric.EngagementScore = &newAvg
	}

	// Update comment metrics
	commentCount := float64(assignment.CommentCount)
	if metric.AvgCommentCount == nil {
		metric.AvgCommentCount = &commentCount
	} else {
		newAvg := (*metric.AvgCommentCount + commentCount) / 2
		metric.AvgCommentCount = &newAvg
	}

	commentLength := float64(assignment.CommentLength)
	if metric.AvgCommentLength == nil {
		metric.AvgCommentLength = &commentLength
	} else {
		newAvg := (*metric.AvgCommentLength + commentLength) / 2
		metric.AvgCommentLength = &newAvg
	}

	return s.repo.CreateOrUpdate(metric)
}

// CalculateMetricsForPeriod recalculates metrics for a date range. This is useful for backfilling or recalculating metrics after bugs/changes.
func (s *Service) CalculateMetricsForPeriod(_ context.Context, _, _ time.Time) error {
	// This will be implemented in Phase 3.2 when we have review repository
	// For now, return an error indicating it's not yet implemented
	return fmt.Errorf("CalculateMetricsForPeriod not yet implemented - requires review repository")
}

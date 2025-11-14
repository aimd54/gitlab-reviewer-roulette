// Package aggregator provides daily batch aggregation of review metrics.
package aggregator

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/repository"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/service/metrics"
)

// Service aggregates metrics from completed reviews.
type Service struct {
	reviewRepo  *repository.ReviewRepository
	metricsRepo *repository.MetricsRepository
	log         *zerolog.Logger
}

// NewService creates a new aggregator service.
func NewService(reviewRepo *repository.ReviewRepository, metricsRepo *repository.MetricsRepository, log *zerolog.Logger) *Service {
	return &Service{
		reviewRepo:  reviewRepo,
		metricsRepo: metricsRepo,
		log:         log,
	}
}

// AggregateDaily aggregates metrics for a specific date.
func (s *Service) AggregateDaily(ctx context.Context, date time.Time) error {
	// Normalize to start of day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	s.log.Info().
		Time("date", startOfDay).
		Msg("Starting daily metrics aggregation")

	// Get all completed reviews for this day
	reviews, err := s.reviewRepo.GetCompletedReviewsByDateRange(startOfDay, endOfDay)
	if err != nil {
		return fmt.Errorf("failed to get completed reviews: %w", err)
	}

	s.log.Debug().
		Int("review_count", len(reviews)).
		Msg("Found completed reviews")

	if len(reviews) == 0 {
		s.log.Info().Msg("No completed reviews found for date")
		return nil
	}

	// Group reviews by team
	teamReviews := make(map[string][]models.MRReview)
	for _, review := range reviews {
		teamReviews[review.Team] = append(teamReviews[review.Team], review)
	}

	// Aggregate metrics for each team
	for team, reviews := range teamReviews {
		if err := s.aggregateTeamMetrics(ctx, startOfDay, team, reviews); err != nil {
			s.log.Error().
				Err(err).
				Str("team", team).
				Msg("Failed to aggregate team metrics")
			continue
		}
	}

	// Aggregate user-level metrics
	for _, review := range reviews {
		if err := s.aggregateUserMetrics(ctx, startOfDay, review); err != nil {
			s.log.Error().
				Err(err).
				Uint("review_id", review.ID).
				Msg("Failed to aggregate user metrics")
			continue
		}
	}

	s.log.Info().
		Time("date", startOfDay).
		Int("teams", len(teamReviews)).
		Int("reviews", len(reviews)).
		Msg("Daily metrics aggregation completed")

	return nil
}

// aggregateTeamMetrics calculates and stores team-level metrics.
func (s *Service) aggregateTeamMetrics(_ context.Context, date time.Time, team string, reviews []models.MRReview) error {
	// Calculate metrics
	var totalTTFR, totalTimeToApproval float64
	var ttfrCount, approvalCount int
	var totalCommentCount, totalCommentLength int
	var completedCount int

	for _, review := range reviews {
		// Count completed reviews (merged)
		if review.Status == models.MRStatusMerged {
			completedCount++
		}

		// Calculate TTFR
		if review.FirstReviewAt != nil && review.RouletteTriggeredAt != nil {
			ttfr := review.FirstReviewAt.Sub(*review.RouletteTriggeredAt).Seconds()
			if ttfr >= 0 {
				totalTTFR += ttfr
				ttfrCount++
			}
		}

		// Calculate time to approval
		if review.ApprovedAt != nil && review.RouletteTriggeredAt != nil {
			approvalTime := review.ApprovedAt.Sub(*review.RouletteTriggeredAt).Seconds()
			if approvalTime >= 0 {
				totalTimeToApproval += approvalTime
				approvalCount++
			}
		}

		// Get assignments for comment metrics
		assignments, err := s.reviewRepo.GetAssignmentsByMRReviewID(review.ID)
		if err != nil {
			s.log.Warn().Err(err).Uint("review_id", review.ID).Msg("Failed to get assignments")
			continue
		}

		for _, assignment := range assignments {
			totalCommentCount += assignment.CommentCount
			totalCommentLength += assignment.CommentLength
		}
	}

	// Calculate averages
	var avgTTFR, avgTimeToApproval float64
	if ttfrCount > 0 {
		avgTTFR = totalTTFR / float64(ttfrCount)
	}
	if approvalCount > 0 {
		avgTimeToApproval = totalTimeToApproval / float64(approvalCount)
	}

	avgCommentCount := 0.0
	avgCommentLength := 0.0
	if len(reviews) > 0 {
		avgCommentCount = float64(totalCommentCount) / float64(len(reviews))
		avgCommentLength = float64(totalCommentLength) / float64(len(reviews))
	}

	// Calculate engagement score
	// For aggregated data, we'll use a simple formula: (avgCommentCount * 10) + (avgCommentLength / 100)
	var engagementScore float64
	if len(reviews) > 0 {
		engagementScore = (avgCommentCount * 10.0) + (avgCommentLength / 100.0)
	}

	// Convert seconds to minutes for storage
	var avgTTFRMinutes, avgTimeToApprovalMinutes *int
	if ttfrCount > 0 {
		minutes := int(avgTTFR / 60)
		avgTTFRMinutes = &minutes
	}
	if approvalCount > 0 {
		minutes := int(avgTimeToApproval / 60)
		avgTimeToApprovalMinutes = &minutes
	}

	// Store metrics
	metric := &models.ReviewMetrics{
		Date:              date,
		Team:              team,
		TotalReviews:      len(reviews),
		CompletedReviews:  completedCount,
		AvgTTFR:           avgTTFRMinutes,
		AvgTimeToApproval: avgTimeToApprovalMinutes,
		AvgCommentCount:   &avgCommentCount,
		AvgCommentLength:  &avgCommentLength,
		EngagementScore:   &engagementScore,
	}

	if err := s.metricsRepo.CreateOrUpdate(metric); err != nil {
		return fmt.Errorf("failed to save team metrics: %w", err)
	}

	s.log.Debug().
		Str("team", team).
		Int("total_reviews", len(reviews)).
		Int("completed", completedCount).
		Float64("avg_ttfr", avgTTFR).
		Float64("engagement", engagementScore).
		Msg("Team metrics aggregated")

	return nil
}

// aggregateUserMetrics calculates and stores user-level metrics.
func (s *Service) aggregateUserMetrics(_ context.Context, date time.Time, review models.MRReview) error {
	assignments, err := s.reviewRepo.GetAssignmentsByMRReviewID(review.ID)
	if err != nil {
		return fmt.Errorf("failed to get assignments: %w", err)
	}

	for _, assignment := range assignments {
		// Calculate metrics for this user
		var avgTTFR, avgTimeToApproval float64

		// TTFR from user's first comment
		if assignment.FirstCommentAt != nil && assignment.AssignedAt.Unix() > 0 {
			ttfr := assignment.FirstCommentAt.Sub(assignment.AssignedAt).Seconds()
			if ttfr >= 0 {
				avgTTFR = ttfr
			}
		}

		// Time to approval
		if assignment.ApprovedAt != nil && assignment.AssignedAt.Unix() > 0 {
			approvalTime := assignment.ApprovedAt.Sub(assignment.AssignedAt).Seconds()
			if approvalTime >= 0 {
				avgTimeToApproval = approvalTime
			}
		}

		// Engagement score - use the actual assignment object
		engagementScore := metrics.CalculateEngagementScore(&assignment, &review)

		// Convert seconds to minutes for storage
		var avgTTFRMinutes, avgTimeToApprovalMinutes *int
		if avgTTFR > 0 {
			minutes := int(avgTTFR / 60)
			avgTTFRMinutes = &minutes
		}
		if avgTimeToApproval > 0 {
			minutes := int(avgTimeToApproval / 60)
			avgTimeToApprovalMinutes = &minutes
		}

		commentCount := float64(assignment.CommentCount)
		commentLength := float64(assignment.CommentLength)
		completedReviews := 0
		if review.Status == models.MRStatusMerged {
			completedReviews = 1
		}

		// Store user-level metrics
		metric := &models.ReviewMetrics{
			Date:              date,
			Team:              review.Team,
			UserID:            &assignment.UserID,
			ProjectID:         &review.GitLabProjectID,
			TotalReviews:      1,
			CompletedReviews:  completedReviews,
			AvgTTFR:           avgTTFRMinutes,
			AvgTimeToApproval: avgTimeToApprovalMinutes,
			AvgCommentCount:   &commentCount,
			AvgCommentLength:  &commentLength,
			EngagementScore:   &engagementScore,
		}

		if err := s.metricsRepo.CreateOrUpdate(metric); err != nil {
			s.log.Warn().
				Err(err).
				Uint("user_id", assignment.UserID).
				Msg("Failed to save user metrics")
			continue
		}

		s.log.Debug().
			Uint("user_id", assignment.UserID).
			Str("team", review.Team).
			Float64("engagement", engagementScore).
			Msg("User metrics aggregated")
	}

	return nil
}

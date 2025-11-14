// Package metrics provides metrics calculation functions for review analytics.
package metrics

import (
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// CalculateTTFR calculates Time To First Review in seconds. Returns nil if firstReviewAt is nil (review hasn't started).
func CalculateTTFR(triggeredAt time.Time, firstReviewAt *time.Time) *int {
	if firstReviewAt == nil {
		return nil
	}

	duration := firstReviewAt.Sub(triggeredAt)
	seconds := int(duration.Seconds())

	// Handle negative durations (clock skew) by returning 0
	if seconds < 0 {
		seconds = 0
	}

	return &seconds
}

// CalculateTimeToApproval calculates time to approval in seconds. Returns nil if approvedAt is nil (not yet approved).
func CalculateTimeToApproval(triggeredAt time.Time, approvedAt *time.Time) *int {
	if approvedAt == nil {
		return nil
	}

	duration := approvedAt.Sub(triggeredAt)
	seconds := int(duration.Seconds())

	// Handle negative durations (clock skew) by returning 0
	if seconds < 0 {
		seconds = 0
	}

	return &seconds
}

// CalculateEngagementScore calculates reviewer engagement based on comments. Formula: (comment_count * 10) + (comment_length / 100).
func CalculateEngagementScore(assignment *models.ReviewerAssignment, _ *models.MRReview) float64 {
	if assignment == nil {
		return 0.0
	}

	score := 0.0

	// Comment count contribution (10 points per comment)
	score += float64(assignment.CommentCount) * 10.0

	// Comment length contribution (1 point per 100 characters)
	score += float64(assignment.CommentLength) / 100.0

	// TODO: Add response time bonus
	// If first_comment_at is within 1 hour of assignment: +10 bonus
	// If within 4 hours: +5 bonus

	return score
}

// CalculateTTFRForMR is a helper function that wraps CalculateTTFR for MR reviews.
func CalculateTTFRForMR(mrReview *models.MRReview) *int {
	if mrReview == nil || mrReview.RouletteTriggeredAt == nil {
		return nil
	}
	return CalculateTTFR(*mrReview.RouletteTriggeredAt, mrReview.FirstReviewAt)
}

// CalculateTimeToApprovalForMR is a helper function that wraps CalculateTimeToApproval for MR reviews.
func CalculateTimeToApprovalForMR(mrReview *models.MRReview) *int {
	if mrReview == nil || mrReview.RouletteTriggeredAt == nil {
		return nil
	}
	return CalculateTimeToApproval(*mrReview.RouletteTriggeredAt, mrReview.ApprovedAt)
}

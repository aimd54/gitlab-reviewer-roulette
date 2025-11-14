package metrics

import (
	"testing"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

func TestCalculateTTFR(t *testing.T) {
	tests := []struct {
		name           string
		triggeredAt    time.Time
		firstReviewAt  *time.Time
		expectedResult *int
		description    string
	}{
		{
			name:           "normal case - 2 hours",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			firstReviewAt:  timePtr(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
			expectedResult: intPtr(7200), // 2 hours = 7200 seconds
			description:    "Normal case: 2 hour difference",
		},
		{
			name:           "same timestamp",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			firstReviewAt:  timePtr(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
			expectedResult: intPtr(0),
			description:    "Instant review",
		},
		{
			name:           "nil first_review_at",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			firstReviewAt:  nil,
			expectedResult: nil,
			description:    "Review hasn't started yet",
		},
		{
			name:           "one day difference",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			firstReviewAt:  timePtr(time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)),
			expectedResult: intPtr(86400), // 24 hours = 86400 seconds
			description:    "Review after 1 day",
		},
		{
			name:           "fast review - 30 minutes",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			firstReviewAt:  timePtr(time.Date(2025, 1, 1, 10, 30, 0, 0, time.UTC)),
			expectedResult: intPtr(1800), // 30 minutes = 1800 seconds
			description:    "Quick review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTTFR(tt.triggeredAt, tt.firstReviewAt)

			if tt.expectedResult == nil {
				if result != nil {
					t.Errorf("Expected nil, got %d", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %d, got nil", *tt.expectedResult)
				} else if *result != *tt.expectedResult {
					t.Errorf("Expected %d seconds, got %d seconds", *tt.expectedResult, *result)
				}
			}
		})
	}
}

func TestCalculateTimeToApproval(t *testing.T) {
	tests := []struct {
		name           string
		triggeredAt    time.Time
		approvedAt     *time.Time
		expectedResult *int
		description    string
	}{
		{
			name:           "normal case - 1 day",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			approvedAt:     timePtr(time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)),
			expectedResult: intPtr(86400), // 1 day = 86400 seconds
			description:    "Approved after 1 day",
		},
		{
			name:           "fast approval - 1 hour",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			approvedAt:     timePtr(time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)),
			expectedResult: intPtr(3600), // 1 hour = 3600 seconds
			description:    "Quick approval",
		},
		{
			name:           "nil approved_at",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			approvedAt:     nil,
			expectedResult: nil,
			description:    "Not yet approved",
		},
		{
			name:           "same timestamp",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			approvedAt:     timePtr(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
			expectedResult: intPtr(0),
			description:    "Instant approval",
		},
		{
			name:           "slow approval - 1 week",
			triggeredAt:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			approvedAt:     timePtr(time.Date(2025, 1, 8, 10, 0, 0, 0, time.UTC)),
			expectedResult: intPtr(604800), // 7 days = 604800 seconds
			description:    "Approval after 1 week",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTimeToApproval(tt.triggeredAt, tt.approvedAt)

			if tt.expectedResult == nil {
				if result != nil {
					t.Errorf("Expected nil, got %d", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %d, got nil", *tt.expectedResult)
				} else if *result != *tt.expectedResult {
					t.Errorf("Expected %d seconds, got %d seconds", *tt.expectedResult, *result)
				}
			}
		})
	}
}

func TestCalculateEngagementScore(t *testing.T) {
	tests := []struct {
		name               string
		assignment         *models.ReviewerAssignment
		mrReview           *models.MRReview
		expectedScoreRange [2]float64 // min, max
		description        string
	}{
		{
			name: "high engagement - many comments, long length",
			assignment: &models.ReviewerAssignment{
				CommentCount:  10,
				CommentLength: 2000,
			},
			mrReview: &models.MRReview{
				RouletteTriggeredAt: timePtr(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
			},
			expectedScoreRange: [2]float64{100, 200},
			description:        "Highly engaged reviewer",
		},
		{
			name: "low engagement - single short comment",
			assignment: &models.ReviewerAssignment{
				CommentCount:  1,
				CommentLength: 50,
			},
			mrReview: &models.MRReview{
				RouletteTriggeredAt: timePtr(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
			},
			expectedScoreRange: [2]float64{10, 20},
			description:        "Minimal engagement",
		},
		{
			name: "no comments - zero score",
			assignment: &models.ReviewerAssignment{
				CommentCount:  0,
				CommentLength: 0,
			},
			mrReview: &models.MRReview{
				RouletteTriggeredAt: timePtr(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
			},
			expectedScoreRange: [2]float64{0, 0},
			description:        "No engagement",
		},
		{
			name: "medium engagement",
			assignment: &models.ReviewerAssignment{
				CommentCount:  5,
				CommentLength: 500,
			},
			mrReview: &models.MRReview{
				RouletteTriggeredAt: timePtr(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
			},
			expectedScoreRange: [2]float64{50, 60},
			description:        "Moderate engagement",
		},
		{
			name: "very long comments",
			assignment: &models.ReviewerAssignment{
				CommentCount:  2,
				CommentLength: 5000,
			},
			mrReview: &models.MRReview{
				RouletteTriggeredAt: timePtr(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
			},
			expectedScoreRange: [2]float64{60, 80},
			description:        "Thorough reviewer with detailed comments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateEngagementScore(tt.assignment, tt.mrReview)

			if score < tt.expectedScoreRange[0] || score > tt.expectedScoreRange[1] {
				t.Errorf("Expected score between %.2f and %.2f, got %.2f",
					tt.expectedScoreRange[0], tt.expectedScoreRange[1], score)
			}
		})
	}
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}

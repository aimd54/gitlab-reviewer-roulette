package scheduler

import (
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/mattermost"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// buildPendingMRs transforms MRReview models into Mattermost PendingMR format.
func buildPendingMRs(reviews []models.MRReview) []mattermost.PendingMR {
	pendingMRs := make([]mattermost.PendingMR, 0, len(reviews))

	for _, review := range reviews {
		// Get author username with nil check
		author := "unknown"
		if review.MRAuthor != nil {
			author = review.MRAuthor.Username
		}

		// Skip if roulette was never triggered (shouldn't happen in practice)
		if review.RouletteTriggeredAt == nil || review.RouletteTriggeredAt.IsZero() {
			continue
		}

		// Create PendingMR with age closure
		triggeredAt := *review.RouletteTriggeredAt
		pendingMR := mattermost.PendingMR{
			Title:     review.MRTitle,
			URL:       review.MRURL,
			Author:    author,
			CreatedAt: review.RouletteTriggeredAt.Format(time.RFC3339),
			Team:      review.Team,
			Age: func() time.Duration {
				return time.Since(triggeredAt)
			},
		}

		pendingMRs = append(pendingMRs, pendingMR)
	}

	return pendingMRs
}

package leaderboard

import (
	"context"
	"fmt"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// UserStats represents comprehensive statistics for a user.
type UserStats struct {
	UserID            uint           `json:"user_id"`
	Username          string         `json:"username"`
	Team              string         `json:"team"`
	Period            string         `json:"period"`
	TotalReviews      int            `json:"total_reviews"`
	CompletedReviews  int            `json:"completed_reviews"`
	AvgTTFR           float64        `json:"avg_ttfr"`             // in minutes
	AvgTimeToApproval float64        `json:"avg_time_to_approval"` // in minutes
	AvgCommentCount   float64        `json:"avg_comment_count"`
	EngagementScore   float64        `json:"engagement_score"`
	Badges            []models.Badge `json:"badges"`
	GlobalRank        int            `json:"global_rank"`
	TeamRank          int            `json:"team_rank"`
}

// GetUserStats returns comprehensive statistics for a user.
func (s *Service) GetUserStats(ctx context.Context, userID uint, period string) (*UserStats, error) {
	// Get user info
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Calculate date range
	startDate, endDate := calculatePeriodRange(period)

	// Get user metrics
	metrics, err := s.metricsRepo.GetMetricsByUser(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	// Aggregate metrics
	stats := &UserStats{
		UserID:   userID,
		Username: user.Username,
		Team:     user.Team,
		Period:   period,
	}

	var (
		totalTTFR            float64
		totalTimeToApproval  float64
		totalCommentCount    float64
		totalEngagementScore float64
		metricsCount         int
	)

	for _, m := range metrics {
		stats.TotalReviews += m.TotalReviews
		stats.CompletedReviews += m.CompletedReviews

		if m.AvgTTFR != nil {
			totalTTFR += float64(*m.AvgTTFR)
		}
		if m.AvgTimeToApproval != nil {
			totalTimeToApproval += float64(*m.AvgTimeToApproval)
		}
		if m.AvgCommentCount != nil {
			totalCommentCount += *m.AvgCommentCount
		}
		if m.EngagementScore != nil {
			totalEngagementScore += *m.EngagementScore
		}
		metricsCount++
	}

	// Calculate averages
	if metricsCount > 0 {
		stats.AvgTTFR = totalTTFR / float64(metricsCount)
		stats.AvgTimeToApproval = totalTimeToApproval / float64(metricsCount)
		stats.AvgCommentCount = totalCommentCount / float64(metricsCount)
		stats.EngagementScore = totalEngagementScore / float64(metricsCount)
	}

	// Get user badges
	userBadges, err := s.badgeRepo.GetUserBadges(userID)
	if err != nil {
		s.log.Warn().Err(err).Uint("user_id", userID).Msg("Failed to get user badges")
	} else {
		// Extract badge details
		for _, ub := range userBadges {
			if ub.Badge.ID != 0 {
				stats.Badges = append(stats.Badges, ub.Badge)
			}
		}
	}

	// Get global rank
	globalRank, err := s.GetUserRank(ctx, userID, period, "engagement_score")
	if err != nil {
		s.log.Warn().Err(err).Uint("user_id", userID).Msg("Failed to get global rank")
		stats.GlobalRank = 0
	} else {
		stats.GlobalRank = globalRank
	}

	// Get team rank
	teamRank, err := s.getUserTeamRank(ctx, userID, user.Team, period, "engagement_score")
	if err != nil {
		s.log.Warn().Err(err).Uint("user_id", userID).Str("team", user.Team).Msg("Failed to get team rank")
		stats.TeamRank = 0
	} else {
		stats.TeamRank = teamRank
	}

	return stats, nil
}

// getUserTeamRank returns the rank of a user within their team.
func (s *Service) getUserTeamRank(ctx context.Context, userID uint, team, period, metric string) (int, error) {
	// Get team leaderboard (no limit)
	leaderboard, err := s.GetTeamLeaderboard(ctx, team, period, metric, 0)
	if err != nil {
		return 0, err
	}

	// Find user in leaderboard
	for _, entry := range leaderboard {
		if entry.UserID == userID {
			return entry.Rank, nil
		}
	}

	// User not found in leaderboard
	return 0, fmt.Errorf("user not found in team leaderboard")
}

// Package leaderboard provides leaderboard and ranking services.
package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/repository"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// MetricsRepository interface for metrics operations.
type MetricsRepository interface {
	GetByDateRange(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error)
	GetMetricsByUser(userID uint, startDate, endDate time.Time) ([]models.ReviewMetrics, error)
}

// BadgeRepository interface for badge operations.
type BadgeRepository interface {
	GetUserBadgeCount(userID uint) (int64, error)
	GetUserBadges(userID uint) ([]models.UserBadge, error)
}

// UserRepository interface for user operations.
type UserRepository interface {
	GetByID(id uint) (*models.User, error)
}

// Entry represents a single entry in a leaderboard.
type Entry struct {
	UserID           uint    `json:"user_id"`
	Username         string  `json:"username"`
	Team             string  `json:"team"`
	CompletedReviews int     `json:"completed_reviews"`
	AvgTTFR          float64 `json:"avg_ttfr"` // in minutes
	AvgCommentCount  float64 `json:"avg_comment_count"`
	EngagementScore  float64 `json:"engagement_score"`
	BadgeCount       int     `json:"badge_count"`
	Rank             int     `json:"rank"`
}

// Service handles leaderboard generation and user statistics.
type Service struct {
	metricsRepo MetricsRepository
	badgeRepo   BadgeRepository
	userRepo    UserRepository
	log         *logger.Logger
}

// NewService creates a new leaderboard service with concrete repository types.
func NewService(
	metricsRepo *repository.MetricsRepository,
	badgeRepo *repository.BadgeRepository,
	userRepo *repository.UserRepository,
	log *logger.Logger,
) *Service {
	return &Service{
		metricsRepo: metricsRepo,
		badgeRepo:   badgeRepo,
		userRepo:    userRepo,
		log:         log,
	}
}

// NewServiceWithInterfaces creates a new leaderboard service with interface dependencies (useful for testing).
func NewServiceWithInterfaces(
	metricsRepo MetricsRepository,
	badgeRepo BadgeRepository,
	userRepo UserRepository,
	log *logger.Logger,
) *Service {
	return &Service{
		metricsRepo: metricsRepo,
		badgeRepo:   badgeRepo,
		userRepo:    userRepo,
		log:         log,
	}
}

// GetGlobalLeaderboard returns the global leaderboard for a given period and metric.
func (s *Service) GetGlobalLeaderboard(ctx context.Context, period, metric string, limit int) ([]Entry, error) {
	return s.getLeaderboard(ctx, "", period, metric, limit)
}

// GetTeamLeaderboard returns the leaderboard for a specific team.
func (s *Service) GetTeamLeaderboard(ctx context.Context, team, period, metric string, limit int) ([]Entry, error) {
	return s.getLeaderboard(ctx, team, period, metric, limit)
}

// getLeaderboard is the internal method that builds leaderboards.
//
//nolint:revive,unparam // ctx reserved for future context-aware operations (tracing, cancellation)
func (s *Service) getLeaderboard(ctx context.Context, team, period, metric string, limit int) ([]Entry, error) {
	// Calculate date range
	startDate, endDate := calculatePeriodRange(period)

	// Build filters
	filters := make(map[string]interface{})
	if team != "" {
		filters["team"] = team
	}

	// Get metrics from database
	metrics, err := s.metricsRepo.GetByDateRange(startDate, endDate, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Aggregate metrics by user
	userMetrics := s.aggregateMetricsByUser(metrics)

	// Get badge counts for all users
	badgeCounts := make(map[uint]int)
	for userID := range userMetrics {
		count, err := s.badgeRepo.GetUserBadgeCount(userID)
		if err != nil {
			s.log.Warn().Err(err).Uint("user_id", userID).Msg("Failed to get badge count")
			badgeCounts[userID] = 0
		} else {
			badgeCounts[userID] = int(count)
		}
	}

	// Build leaderboard entries
	entries := make([]Entry, 0, len(userMetrics))
	for userID, aggMetrics := range userMetrics {
		// Get user info
		user, err := s.userRepo.GetByID(userID)
		if err != nil {
			s.log.Warn().Err(err).Uint("user_id", userID).Msg("Failed to get user")
			continue
		}

		entry := Entry{
			UserID:           userID,
			Username:         user.Username,
			Team:             user.Team,
			CompletedReviews: aggMetrics.CompletedReviews,
			AvgTTFR:          aggMetrics.AvgTTFR,
			AvgCommentCount:  aggMetrics.AvgCommentCount,
			EngagementScore:  aggMetrics.EngagementScore,
			BadgeCount:       badgeCounts[userID],
		}

		entries = append(entries, entry)
	}

	// Sort entries by the specified metric
	s.sortLeaderboard(entries, metric)

	// Assign ranks
	for i := range entries {
		entries[i].Rank = i + 1
	}

	// Apply limit
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

// aggregateMetricsByUser aggregates metrics by user ID.
func (s *Service) aggregateMetricsByUser(metrics []models.ReviewMetrics) map[uint]aggregatedMetrics {
	userMetrics := make(map[uint]aggregatedMetrics)

	for _, m := range metrics {
		if m.UserID == nil {
			continue
		}

		userID := *m.UserID
		agg := userMetrics[userID]

		// Aggregate totals
		agg.CompletedReviews += m.CompletedReviews
		agg.MetricsCount++

		// Aggregate averages
		if m.AvgTTFR != nil {
			agg.TotalTTFR += float64(*m.AvgTTFR)
		}
		if m.AvgCommentCount != nil {
			agg.TotalCommentCount += *m.AvgCommentCount
		}
		if m.EngagementScore != nil {
			agg.TotalEngagementScore += *m.EngagementScore
		}

		userMetrics[userID] = agg
	}

	// Calculate averages
	for userID, agg := range userMetrics {
		if agg.MetricsCount > 0 {
			agg.AvgTTFR = agg.TotalTTFR / float64(agg.MetricsCount)
			agg.AvgCommentCount = agg.TotalCommentCount / float64(agg.MetricsCount)
			agg.EngagementScore = agg.TotalEngagementScore / float64(agg.MetricsCount)
			userMetrics[userID] = agg
		}
	}

	return userMetrics
}

// sortLeaderboard sorts leaderboard entries by the specified metric.
func (s *Service) sortLeaderboard(entries []Entry, metric string) {
	switch metric {
	case "completed_reviews":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].CompletedReviews > entries[j].CompletedReviews
		})
	case "engagement_score":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].EngagementScore > entries[j].EngagementScore
		})
	case "avg_ttfr":
		// Lower is better for TTFR
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].AvgTTFR < entries[j].AvgTTFR
		})
	case "avg_comment_count":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].AvgCommentCount > entries[j].AvgCommentCount
		})
	default:
		// Default to completed_reviews
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].CompletedReviews > entries[j].CompletedReviews
		})
	}
}

// GetUserRank returns the rank of a user for a specific metric in a period.
func (s *Service) GetUserRank(ctx context.Context, userID uint, period, metric string) (int, error) {
	// Get global leaderboard (no limit)
	leaderboard, err := s.GetGlobalLeaderboard(ctx, period, metric, 0)
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
	return 0, fmt.Errorf("user not found in leaderboard")
}

// aggregatedMetrics holds aggregated metrics for a user.
type aggregatedMetrics struct {
	CompletedReviews     int
	TotalTTFR            float64
	TotalCommentCount    float64
	TotalEngagementScore float64
	MetricsCount         int
	AvgTTFR              float64
	AvgCommentCount      float64
	EngagementScore      float64
}

// calculatePeriodRange calculates the start and end dates for a period.
func calculatePeriodRange(period string) (startDate, endDate time.Time) {
	now := time.Now()
	endDate = now

	switch period {
	case "day":
		startDate = now.Add(-24 * time.Hour)
	case "week":
		startDate = now.Add(-7 * 24 * time.Hour)
	case "month":
		startDate = now.Add(-30 * 24 * time.Hour)
	case "year":
		startDate = now.Add(-365 * 24 * time.Hour)
	case "all_time", "":
		// All time: use a very old date
		startDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		// Default to all time
		startDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	return startDate, endDate
}

package badges

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// checkCriteria evaluates badge criteria against user metrics.
func (s *Service) checkCriteria(ctx context.Context, criteria *models.BadgeCriteria, userID uint) (bool, error) {
	// Calculate date range based on period
	startDate, endDate := s.calculatePeriodRange(criteria.Period)

	// Get user metrics for the period
	userMetrics, err := s.aggregateUserMetrics(userID, startDate, endDate)
	if err != nil {
		return false, fmt.Errorf("failed to aggregate user metrics: %w", err)
	}

	// Get the metric value
	metricValue, exists := userMetrics[criteria.Metric]
	if !exists {
		// User has no data for this metric in the period
		return false, nil
	}

	// Handle "top" operator specially (requires ranking)
	if criteria.Operator == "top" {
		// Convert value to int
		topN, ok := criteria.Value.(float64) // JSON numbers are float64
		if !ok {
			return false, fmt.Errorf("invalid value type for 'top' operator: %T", criteria.Value)
		}
		return s.evaluateTopRanking(ctx, criteria.Metric, int(topN), criteria.Period, userID)
	}

	// Convert threshold value to float64 for comparison
	threshold, ok := criteria.Value.(float64)
	if !ok {
		return false, fmt.Errorf("invalid value type: expected float64, got %T", criteria.Value)
	}

	// Evaluate based on operator
	return s.evaluateMetricCriteria(criteria.Operator, threshold, metricValue)
}

// evaluateMetricCriteria compares a metric value against criteria using the specified operator.
func (s *Service) evaluateMetricCriteria(operator string, threshold, actualValue float64) (bool, error) {
	switch operator {
	case "<":
		return actualValue < threshold, nil
	case "<=":
		return actualValue <= threshold, nil
	case ">":
		return actualValue > threshold, nil
	case ">=":
		return actualValue >= threshold, nil
	case "==":
		return actualValue == threshold, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// evaluateTopRanking checks if a user is in the top N for a metric.
//
//nolint:revive,unparam // ctx reserved for future context-aware operations
func (s *Service) evaluateTopRanking(ctx context.Context, metric string, topN int, period string, userID uint) (bool, error) {
	// Calculate date range
	startDate, endDate := s.calculatePeriodRange(period)

	// Get all metrics for the period
	allMetrics, err := s.metricsRepo.GetByDateRange(startDate, endDate, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Aggregate metrics by user
	userAggregates, err := s.aggregateMetricsByUser(allMetrics, metric)
	if err != nil {
		return false, err
	}

	// Create and sort rankings
	rankings := s.sortUserRankings(userAggregates)

	// Check if userID is in top N
	for i := 0; i < topN && i < len(rankings); i++ {
		if rankings[i].userID == userID {
			return true, nil
		}
	}

	return false, nil
}

// getMetricValue extracts the value for a specific metric from a review metric.
func getMetricValue(m *models.ReviewMetrics, metric string) (float64, error) {
	switch metric {
	case "completed_reviews":
		return float64(m.CompletedReviews), nil
	case "engagement_score":
		if m.EngagementScore != nil {
			return *m.EngagementScore, nil
		}
		return 0, nil
	case "avg_ttfr":
		if m.AvgTTFR != nil {
			return float64(*m.AvgTTFR), nil
		}
		return 0, nil
	case "avg_comment_count":
		if m.AvgCommentCount != nil {
			return *m.AvgCommentCount, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("top ranking not supported for metric: %s", metric)
	}
}

// aggregateMetricsByUser aggregates metric values by user ID.
func (s *Service) aggregateMetricsByUser(allMetrics []models.ReviewMetrics, metric string) (map[uint]float64, error) {
	userAggregates := make(map[uint]float64)

	for _, m := range allMetrics {
		if m.UserID == nil {
			continue
		}

		value, err := getMetricValue(&m, metric)
		if err != nil {
			return nil, err
		}

		userAggregates[*m.UserID] += value
	}

	return userAggregates, nil
}

// sortUserRankings creates a sorted list of user rankings.
func (s *Service) sortUserRankings(userAggregates map[uint]float64) []userRank {
	rankings := make([]userRank, 0, len(userAggregates))
	for uid, val := range userAggregates {
		rankings = append(rankings, userRank{userID: uid, value: val})
	}

	// Sort by value descending (higher is better)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].value > rankings[j].value
	})

	return rankings
}

// userRank represents a user's ranking for a specific metric.
type userRank struct {
	userID uint
	value  float64
}

// calculatePeriodRange calculates the start and end dates for a period.
func (s *Service) calculatePeriodRange(period string) (startDate, endDate time.Time) {
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

// aggregateUserMetrics calculates aggregated metrics for a user in a time period.
func (s *Service) aggregateUserMetrics(userID uint, startDate, endDate time.Time) (map[string]float64, error) {
	metrics := make(map[string]float64)

	// Get metrics from database
	userMetrics, err := s.metricsRepo.GetMetricsByUser(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	if len(userMetrics) == 0 {
		// No metrics for this user in the period
		return metrics, nil
	}

	// Aggregate metrics across the period
	var (
		totalTTFR             float64
		totalCommentCount     float64
		totalCommentLength    float64
		totalEngagementScore  float64
		totalCompletedReviews int
		metricsCount          int
	)

	for _, m := range userMetrics {
		if m.AvgTTFR != nil {
			totalTTFR += float64(*m.AvgTTFR)
		}
		if m.AvgCommentCount != nil {
			totalCommentCount += *m.AvgCommentCount
		}
		if m.AvgCommentLength != nil {
			totalCommentLength += *m.AvgCommentLength
		}
		if m.EngagementScore != nil {
			totalEngagementScore += *m.EngagementScore
		}
		totalCompletedReviews += m.CompletedReviews
		metricsCount++
	}

	// Calculate averages
	if metricsCount > 0 {
		metrics["avg_ttfr"] = totalTTFR / float64(metricsCount)
		metrics["avg_comment_count"] = totalCommentCount / float64(metricsCount)
		metrics["avg_comment_length"] = totalCommentLength / float64(metricsCount)
		metrics["engagement_score"] = totalEngagementScore / float64(metricsCount)
	}

	// Totals
	metrics["completed_reviews"] = float64(totalCompletedReviews)

	// Calculate external reviews (reviews for other teams)
	// This would require additional data from review_metrics table
	// For now, use completed_reviews as a placeholder
	metrics["external_reviews"] = float64(totalCompletedReviews) // TODO: Implement actual external reviews logic

	return metrics, nil
}

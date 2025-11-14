package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// MetricsRepository handles database operations for review metrics.
type MetricsRepository struct {
	db *DB
}

// NewMetricsRepository creates a new metrics repository.
func NewMetricsRepository(db *DB) *MetricsRepository {
	return &MetricsRepository{db: db}
}

// Create creates a new review metric record.
func (r *MetricsRepository) Create(metric *models.ReviewMetrics) error {
	return r.db.Create(metric).Error
}

// CreateOrUpdate creates or updates a review metrics record. This ensures idempotency for daily aggregations.
func (r *MetricsRepository) CreateOrUpdate(metric *models.ReviewMetrics) error {
	// Try to find existing record
	var existing models.ReviewMetrics
	query := r.db.Where("date = ? AND team = ?", metric.Date, metric.Team)

	if metric.UserID != nil {
		query = query.Where("user_id = ?", *metric.UserID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	if metric.ProjectID != nil {
		query = query.Where("project_id = ?", *metric.ProjectID)
	} else {
		query = query.Where("project_id IS NULL")
	}

	err := query.First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Record doesn't exist, create new one
		return r.Create(metric)
	}

	if err != nil {
		return err
	}

	// Record exists, update it
	metric.ID = existing.ID
	return r.db.Save(metric).Error
}

// GetByDate retrieves metrics for a specific date with optional filters.
func (r *MetricsRepository) GetByDate(date time.Time, team string, userID *uint) (*models.ReviewMetrics, error) {
	var metric models.ReviewMetrics
	query := r.db.Where("date = ? AND team = ?", date, team)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	err := query.First(&metric).Error
	if err != nil {
		return nil, err
	}

	return &metric, nil
}

// GetByDateRange retrieves metrics within a date range with optional filters.
func (r *MetricsRepository) GetByDateRange(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error) {
	var metrics []models.ReviewMetrics
	query := r.db.Where("date BETWEEN ? AND ?", startDate, endDate)

	// Apply filters
	if team, ok := filters["team"].(string); ok && team != "" {
		query = query.Where("team = ?", team)
	}

	if userID, ok := filters["user_id"].(*uint); ok && userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	if projectID, ok := filters["project_id"].(*uint); ok && projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}

	err := query.Order("date DESC").Find(&metrics).Error
	return metrics, err
}

// GetAverageTTFRByTeam calculates average TTFR (in seconds) by team for a date range.
func (r *MetricsRepository) GetAverageTTFRByTeam(startDate, endDate time.Time) (map[string]float64, error) {
	type Result struct {
		Team    string
		AvgTTFR float64
	}

	var results []Result
	err := r.db.Model(&models.ReviewMetrics{}).
		Select("team, AVG(avg_ttfr) as avg_ttfr").
		Where("date BETWEEN ? AND ? AND avg_ttfr IS NOT NULL", startDate, endDate).
		Group("team").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert to map
	avgMap := make(map[string]float64)
	for _, result := range results {
		avgMap[result.Team] = result.AvgTTFR
	}

	return avgMap, nil
}

// GetTopReviewersByEngagement returns top N reviewers by engagement score for a date range.
func (r *MetricsRepository) GetTopReviewersByEngagement(startDate, endDate time.Time, limit int) ([]models.User, error) {
	type Result struct {
		UserID               uint
		TotalEngagementScore float64
	}

	var results []Result
	err := r.db.Model(&models.ReviewMetrics{}).
		Select("user_id, SUM(engagement_score) as total_engagement_score").
		Where("date BETWEEN ? AND ? AND user_id IS NOT NULL", startDate, endDate).
		Group("user_id").
		Order("total_engagement_score DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Fetch user details
	var users []models.User
	for _, result := range results {
		var user models.User
		if err := r.db.First(&user, result.UserID).Error; err == nil {
			users = append(users, user)
		}
	}

	return users, nil
}

// GetMetricsByTeam retrieves all metrics for a specific team within a date range.
func (r *MetricsRepository) GetMetricsByTeam(team string, startDate, endDate time.Time) ([]models.ReviewMetrics, error) {
	var metrics []models.ReviewMetrics
	err := r.db.Where("team = ? AND date BETWEEN ? AND ?", team, startDate, endDate).
		Order("date DESC").
		Find(&metrics).Error

	return metrics, err
}

// GetMetricsByUser retrieves all metrics for a specific user within a date range.
func (r *MetricsRepository) GetMetricsByUser(userID uint, startDate, endDate time.Time) ([]models.ReviewMetrics, error) {
	var metrics []models.ReviewMetrics
	err := r.db.Where("user_id = ? AND date BETWEEN ? AND ?", userID, startDate, endDate).
		Order("date DESC").
		Find(&metrics).Error

	return metrics, err
}

// DeleteOldMetrics deletes metrics older than the specified retention period. Used for data cleanup if retention policy is configured.
func (r *MetricsRepository) DeleteOldMetrics(retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	return r.db.Where("date < ?", cutoffDate).Delete(&models.ReviewMetrics{}).Error
}

// GetDailyStats retrieves aggregated stats for a specific date.
func (r *MetricsRepository) GetDailyStats(date time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total reviews across all teams
	var totalReviews int64
	if err := r.db.Model(&models.ReviewMetrics{}).
		Where("date = ?", date).
		Select("SUM(total_reviews)").
		Scan(&totalReviews).Error; err != nil {
		return nil, err
	}
	stats["total_reviews"] = totalReviews

	// Average TTFR across all teams
	var avgTTFR float64
	if err := r.db.Model(&models.ReviewMetrics{}).
		Where("date = ? AND avg_ttfr IS NOT NULL", date).
		Select("AVG(avg_ttfr)").
		Scan(&avgTTFR).Error; err != nil {
		return nil, err
	}
	stats["avg_ttfr"] = avgTTFR

	// Total completed reviews
	var totalCompleted int64
	if err := r.db.Model(&models.ReviewMetrics{}).
		Where("date = ?", date).
		Select("SUM(completed_reviews)").
		Scan(&totalCompleted).Error; err != nil {
		return nil, err
	}
	stats["total_completed"] = totalCompleted

	return stats, nil
}

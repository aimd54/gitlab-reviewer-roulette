package repository

import (
	"fmt"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// ReviewRepository handles review-related database operations.
type ReviewRepository struct {
	db *DB
}

// NewReviewRepository creates a new review repository.
func NewReviewRepository(db *DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// CreateMRReview creates a new MR review record.
func (r *ReviewRepository) CreateMRReview(review *models.MRReview) error {
	if err := r.db.Create(review).Error; err != nil {
		return fmt.Errorf("failed to create MR review: %w", err)
	}
	return nil
}

// GetMRReview retrieves an MR review by project ID and MR IID.
func (r *ReviewRepository) GetMRReview(projectID, mrIID int) (*models.MRReview, error) {
	var review models.MRReview
	err := r.db.Where("gitlab_project_id = ? AND gitlab_mr_iid = ?", projectID, mrIID).
		Preload("Assignments").
		Preload("Assignments.User").
		First(&review).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get MR review for project %d, MR %d: %w", projectID, mrIID, err)
	}
	return &review, nil
}

// GetMRReviewByID retrieves an MR review by ID.
func (r *ReviewRepository) GetMRReviewByID(id uint) (*models.MRReview, error) {
	var review models.MRReview
	err := r.db.Preload("Assignments").Preload("Assignments.User").First(&review, id).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get MR review by id %d: %w", id, err)
	}
	return &review, nil
}

// UpdateMRReview updates an MR review.
func (r *ReviewRepository) UpdateMRReview(review *models.MRReview) error {
	if err := r.db.Save(review).Error; err != nil {
		return fmt.Errorf("failed to update MR review: %w", err)
	}
	return nil
}

// CreateOrUpdateMRReview creates or updates an MR review.
func (r *ReviewRepository) CreateOrUpdateMRReview(review *models.MRReview) error {
	existing, err := r.GetMRReview(review.GitLabProjectID, review.GitLabMRIID)
	if err != nil {
		// Doesn't exist, create it
		return r.CreateMRReview(review)
	}

	// Exists, update it - preserve fields that shouldn't be overwritten
	review.ID = existing.ID
	review.CreatedAt = existing.CreatedAt
	// Preserve BotCommentID if not set in the new review
	if review.BotCommentID == nil && existing.BotCommentID != nil {
		review.BotCommentID = existing.BotCommentID
	}
	return r.UpdateMRReview(review)
}

// CreateAssignment creates a new reviewer assignment.
func (r *ReviewRepository) CreateAssignment(assignment *models.ReviewerAssignment) error {
	if err := r.db.Create(assignment).Error; err != nil {
		return fmt.Errorf("failed to create reviewer assignment: %w", err)
	}
	return nil
}

// UpdateAssignment updates a reviewer assignment.
func (r *ReviewRepository) UpdateAssignment(assignment *models.ReviewerAssignment) error {
	if err := r.db.Save(assignment).Error; err != nil {
		return fmt.Errorf("failed to update reviewer assignment: %w", err)
	}
	return nil
}

// GetAssignmentsByMRReviewID retrieves all assignments for an MR review.
func (r *ReviewRepository) GetAssignmentsByMRReviewID(mrReviewID uint) ([]models.ReviewerAssignment, error) {
	var assignments []models.ReviewerAssignment
	err := r.db.Where("mr_review_id = ?", mrReviewID).Preload("User").Find(&assignments).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments for MR review %d: %w", mrReviewID, err)
	}
	return assignments, nil
}

// GetActiveAssignmentsByUserID retrieves active assignments for a user.
func (r *ReviewRepository) GetActiveAssignmentsByUserID(userID uint) ([]models.ReviewerAssignment, error) {
	var assignments []models.ReviewerAssignment
	err := r.db.Joins("JOIN mr_reviews ON mr_reviews.id = reviewer_assignments.mr_review_id").
		Where("reviewer_assignments.user_id = ?", userID).
		Where("mr_reviews.status IN ?", []string{models.MRStatusPending, models.MRStatusInReview, models.MRStatusApproved}).
		Preload("MRReview").
		Find(&assignments).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active assignments for user %d: %w", userID, err)
	}
	return assignments, nil
}

// CountActiveReviewsByUserID counts active reviews for a user.
func (r *ReviewRepository) CountActiveReviewsByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.ReviewerAssignment{}).
		Joins("JOIN mr_reviews ON mr_reviews.id = reviewer_assignments.mr_review_id").
		Where("reviewer_assignments.user_id = ?", userID).
		Where("mr_reviews.status IN ?", []string{models.MRStatusPending, models.MRStatusInReview, models.MRStatusApproved}).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count active reviews for user %d: %w", userID, err)
	}
	return count, nil
}

// GetRecentAssignmentsByUserID retrieves recent assignments for a user within a time window.
func (r *ReviewRepository) GetRecentAssignmentsByUserID(userID uint, since time.Time) ([]models.ReviewerAssignment, error) {
	var assignments []models.ReviewerAssignment
	err := r.db.Where("user_id = ? AND assigned_at >= ?", userID, since).
		Preload("MRReview").
		Find(&assignments).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent assignments for user %d: %w", userID, err)
	}
	return assignments, nil
}

// ListPendingMRReviews lists all MR reviews in pending or in_review status.
func (r *ReviewRepository) ListPendingMRReviews() ([]models.MRReview, error) {
	var reviews []models.MRReview
	err := r.db.Where("status IN ?", []string{models.MRStatusPending, models.MRStatusInReview}).
		Preload("Assignments").
		Preload("Assignments.User").
		Order("roulette_triggered_at ASC").
		Find(&reviews).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list pending MR reviews: %w", err)
	}
	return reviews, nil
}

// ListMRReviewsByStatus lists MR reviews by status.
func (r *ReviewRepository) ListMRReviewsByStatus(status string) ([]models.MRReview, error) {
	var reviews []models.MRReview
	err := r.db.Where("status = ?", status).
		Preload("Assignments").
		Preload("Assignments.User").
		Find(&reviews).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list MR reviews by status %s: %w", status, err)
	}
	return reviews, nil
}

// DeleteAssignmentsByMRReviewID deletes all assignments for an MR review.
func (r *ReviewRepository) DeleteAssignmentsByMRReviewID(mrReviewID uint) error {
	if err := r.db.Where("mr_review_id = ?", mrReviewID).Delete(&models.ReviewerAssignment{}).Error; err != nil {
		return fmt.Errorf("failed to delete assignments for MR review %d: %w", mrReviewID, err)
	}
	return nil
}

// GetMRReviewStats retrieves statistics for MR reviews.
func (r *ReviewRepository) GetMRReviewStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total reviews
	var totalCount int64
	r.db.Model(&models.MRReview{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(&totalCount)
	stats["total_reviews"] = totalCount

	// Reviews by status
	var statusCounts []struct {
		Status string
		Count  int64
	}
	r.db.Model(&models.MRReview{}).
		Select("status, count(*) as count").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("status").
		Scan(&statusCounts)
	stats["by_status"] = statusCounts

	// Average TTFR
	var avgTTFR float64
	r.db.Model(&models.MRReview{}).
		Select("AVG(EXTRACT(EPOCH FROM (first_review_at - roulette_triggered_at))/60) as avg_ttfr").
		Where("first_review_at IS NOT NULL AND roulette_triggered_at IS NOT NULL").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Scan(&avgTTFR)
	stats["avg_ttfr_minutes"] = avgTTFR

	return stats, nil
}

// GetByProjectAndMR retrieves an MR review by project ID and MR IID.
func (r *ReviewRepository) GetByProjectAndMR(projectID, mrIID int) (*models.MRReview, error) {
	var review models.MRReview
	err := r.db.Where("gitlab_project_id = ? AND gitlab_mr_iid = ?", projectID, mrIID).First(&review).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get MR review by project %d and MR %d: %w", projectID, mrIID, err)
	}
	return &review, nil
}

// GetCompletedReviewsByDateRange retrieves all completed reviews within a date range.
func (r *ReviewRepository) GetCompletedReviewsByDateRange(startDate, endDate time.Time) ([]models.MRReview, error) {
	var reviews []models.MRReview
	err := r.db.Where("(merged_at BETWEEN ? AND ?) OR (closed_at BETWEEN ? AND ?)",
		startDate, endDate, startDate, endDate).
		Where("status IN ?", []string{models.MRStatusMerged, models.MRStatusClosed}).
		Find(&reviews).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get completed reviews: %w", err)
	}
	return reviews, nil
}

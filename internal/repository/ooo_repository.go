package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// OOORepository handles out-of-office status operations.
type OOORepository struct {
	db *gorm.DB
}

// NewOOORepository creates a new OOO repository instance.
func NewOOORepository(db *DB) *OOORepository {
	return &OOORepository{
		db: db.DB,
	}
}

// IsUserOOO checks if a user is currently out of office.
func (r *OOORepository) IsUserOOO(userID uint) (bool, error) {
	var count int64
	now := time.Now()

	err := r.db.Model(&models.OOOStatus{}).
		Where("user_id = ? AND start_date <= ? AND end_date >= ?", userID, now, now).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check OOO status for user %d: %w", userID, err)
	}

	return count > 0, nil
}

// GetActiveOOO retrieves all active OOO entries for a user.
func (r *OOORepository) GetActiveOOO(userID uint) ([]models.OOOStatus, error) {
	var statuses []models.OOOStatus
	now := time.Now()

	err := r.db.
		Where("user_id = ? AND start_date <= ? AND end_date >= ?", userID, now, now).
		Find(&statuses).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active OOO entries for user %d: %w", userID, err)
	}

	return statuses, nil
}

// CreateOOO creates a new out-of-office entry.
func (r *OOORepository) CreateOOO(status *models.OOOStatus) error {
	if err := r.db.Create(status).Error; err != nil {
		return fmt.Errorf("failed to create OOO entry for user %d: %w", status.UserID, err)
	}
	return nil
}

// DeleteOOO deletes an out-of-office entry by ID.
func (r *OOORepository) DeleteOOO(id uint) error {
	result := r.db.Delete(&models.OOOStatus{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete OOO entry %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("OOO entry %d not found", id)
	}
	return nil
}

// GetOOOByID retrieves a specific OOO entry by ID.
func (r *OOORepository) GetOOOByID(id uint) (*models.OOOStatus, error) {
	var status models.OOOStatus
	err := r.db.First(&status, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OOO entry %d not found", id)
		}
		return nil, fmt.Errorf("failed to get OOO entry %d: %w", id, err)
	}
	return &status, nil
}

// GetAllOOOForUser retrieves all OOO entries (past, present, future) for a user.
func (r *OOORepository) GetAllOOOForUser(userID uint) ([]models.OOOStatus, error) {
	var statuses []models.OOOStatus
	err := r.db.
		Where("user_id = ?", userID).
		Order("start_date DESC").
		Find(&statuses).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get all OOO entries for user %d: %w", userID, err)
	}

	return statuses, nil
}

// GetAllActive retrieves all currently active OOO statuses across all users.
func (r *OOORepository) GetAllActive() ([]models.OOOStatus, error) {
	var statuses []models.OOOStatus
	now := time.Now()

	err := r.db.Preload("User").
		Where("start_date <= ? AND end_date >= ?", now, now).
		Find(&statuses).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get all active OOO statuses: %w", err)
	}

	return statuses, nil
}

// GetActiveByUserID retrieves the active OOO status for a specific user.
func (r *OOORepository) GetActiveByUserID(userID uint) (*models.OOOStatus, error) {
	var status models.OOOStatus
	now := time.Now()

	err := r.db.
		Where("user_id = ? AND start_date <= ? AND end_date >= ?", userID, now, now).
		First(&status).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No active OOO is not an error
		}
		return nil, fmt.Errorf("failed to get active OOO status for user %d: %w", userID, err)
	}

	return &status, nil
}

// Package repository provides data access layer for the application.
package repository

import (
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// BadgeRepository handles badge-related database operations.
type BadgeRepository struct {
	db *DB
}

// NewBadgeRepository creates a new badge repository.
func NewBadgeRepository(db *DB) *BadgeRepository {
	return &BadgeRepository{db: db}
}

// Create creates a new badge in the database.
func (r *BadgeRepository) Create(badge *models.Badge) error {
	return r.db.Create(badge).Error
}

// GetByID retrieves a badge by its ID.
func (r *BadgeRepository) GetByID(id uint) (*models.Badge, error) {
	var badge models.Badge
	err := r.db.First(&badge, id).Error
	if err != nil {
		return nil, err
	}
	return &badge, nil
}

// GetByName retrieves a badge by its name.
func (r *BadgeRepository) GetByName(name string) (*models.Badge, error) {
	var badge models.Badge
	err := r.db.Where("name = ?", name).First(&badge).Error
	if err != nil {
		return nil, err
	}
	return &badge, nil
}

// GetAll retrieves all badges from the database.
func (r *BadgeRepository) GetAll() ([]models.Badge, error) {
	var badges []models.Badge
	err := r.db.Order("created_at ASC").Find(&badges).Error
	return badges, err
}

// Update updates an existing badge in the database.
func (r *BadgeRepository) Update(badge *models.Badge) error {
	return r.db.Save(badge).Error
}

// Delete deletes a badge by its ID.
func (r *BadgeRepository) Delete(id uint) error {
	return r.db.Delete(&models.Badge{}, id).Error
}

// AwardBadge awards a badge to a user.
// Returns nil if successful, error if badge already awarded or database error.
func (r *BadgeRepository) AwardBadge(userID, badgeID uint) error {
	// Check if already awarded
	exists, err := r.HasUserEarnedBadge(userID, badgeID)
	if err != nil {
		return err
	}
	if exists {
		// Idempotent: already awarded, return success
		return nil
	}

	userBadge := &models.UserBadge{
		UserID:   userID,
		BadgeID:  badgeID,
		EarnedAt: time.Now(),
	}
	return r.db.Create(userBadge).Error
}

// GetUserBadges retrieves all badges earned by a user with badge details preloaded.
func (r *BadgeRepository) GetUserBadges(userID uint) ([]models.UserBadge, error) {
	var userBadges []models.UserBadge
	err := r.db.
		Where("user_id = ?", userID).
		Preload("Badge").
		Preload("User").
		Order("earned_at DESC").
		Find(&userBadges).Error
	return userBadges, err
}

// HasUserEarnedBadge checks if a user has earned a specific badge.
func (r *BadgeRepository) HasUserEarnedBadge(userID, badgeID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.UserBadge{}).
		Where("user_id = ? AND badge_id = ?", userID, badgeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUsersWithBadge retrieves all users who have earned a specific badge.
func (r *BadgeRepository) GetUsersWithBadge(badgeID uint) ([]models.User, error) {
	var users []models.User
	err := r.db.
		Joins("JOIN user_badges ON user_badges.user_id = users.id").
		Where("user_badges.badge_id = ?", badgeID).
		Order("user_badges.earned_at DESC").
		Find(&users).Error
	return users, err
}

// GetBadgeHoldersCount returns the number of users who have earned a specific badge.
func (r *BadgeRepository) GetBadgeHoldersCount(badgeID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserBadge{}).
		Where("badge_id = ?", badgeID).
		Count(&count).Error
	return count, err
}

// RevokeUserBadge revokes a badge from a user.
func (r *BadgeRepository) RevokeUserBadge(userID, badgeID uint) error {
	return r.db.
		Where("user_id = ? AND badge_id = ?", userID, badgeID).
		Delete(&models.UserBadge{}).Error
}

// GetUserBadgeCount returns the total number of badges a user has earned.
func (r *BadgeRepository) GetUserBadgeCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserBadge{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// GetRecentlyAwardedBadges retrieves badges awarded within a time period.
func (r *BadgeRepository) GetRecentlyAwardedBadges(since time.Time) ([]models.UserBadge, error) {
	var userBadges []models.UserBadge
	err := r.db.
		Where("earned_at >= ?", since).
		Preload("Badge").
		Preload("User").
		Order("earned_at DESC").
		Find(&userBadges).Error
	return userBadges, err
}

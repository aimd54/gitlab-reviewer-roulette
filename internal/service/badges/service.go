// Package badges provides badge evaluation and management services.
package badges

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	prommetrics "github.com/aimd54/gitlab-reviewer-roulette/internal/metrics"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/repository"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// BadgeRepository interface for badge operations.
type BadgeRepository interface {
	GetAll() ([]models.Badge, error)
	GetByID(id uint) (*models.Badge, error)
	HasUserEarnedBadge(userID, badgeID uint) (bool, error)
	AwardBadge(userID, badgeID uint) error
	GetUserBadges(userID uint) ([]models.UserBadge, error)
	GetUsersWithBadge(badgeID uint) ([]models.User, error)
	GetBadgeHoldersCount(badgeID uint) (int64, error)
}

// MetricsRepository interface for metrics operations.
type MetricsRepository interface {
	GetByDateRange(startDate, endDate time.Time, filters map[string]interface{}) ([]models.ReviewMetrics, error)
	GetMetricsByUser(userID uint, startDate, endDate time.Time) ([]models.ReviewMetrics, error)
}

// ReviewRepository interface for review operations.
type ReviewRepository interface {
	// Add methods as needed
}

// UserRepository interface for user operations.
type UserRepository interface {
	List(team, role string) ([]models.User, error)
	GetByID(id uint) (*models.User, error)
}

// Service handles badge evaluation and awarding.
type Service struct {
	badgeRepo   BadgeRepository
	metricsRepo MetricsRepository
	reviewRepo  ReviewRepository
	userRepo    UserRepository
	log         *logger.Logger
}

// NewService creates a new badge service.
func NewService(
	badgeRepo *repository.BadgeRepository,
	metricsRepo *repository.MetricsRepository,
	reviewRepo *repository.ReviewRepository,
	userRepo *repository.UserRepository,
	log *logger.Logger,
) *Service {
	return &Service{
		badgeRepo:   badgeRepo,
		metricsRepo: metricsRepo,
		reviewRepo:  reviewRepo,
		userRepo:    userRepo,
		log:         log,
	}
}

// NewServiceWithInterfaces creates a new badge service with interface dependencies (useful for testing).
func NewServiceWithInterfaces(
	badgeRepo BadgeRepository,
	metricsRepo MetricsRepository,
	reviewRepo ReviewRepository,
	userRepo UserRepository,
	log *logger.Logger,
) *Service {
	return &Service{
		badgeRepo:   badgeRepo,
		metricsRepo: metricsRepo,
		reviewRepo:  reviewRepo,
		userRepo:    userRepo,
		log:         log,
	}
}

// EvaluateAllBadges evaluates all badges for all users.
// This is typically run as a scheduled job.
// Returns the number of badges awarded.
func (s *Service) EvaluateAllBadges(ctx context.Context) (int, error) {
	s.log.Info().Msg("Starting badge evaluation for all users")
	start := time.Now()

	// Get all badges
	badges, err := s.badgeRepo.GetAll()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get badges")
		return 0, fmt.Errorf("failed to get badges: %w", err)
	}

	// Get all users
	users, err := s.userRepo.List("", "") // Get all users (empty filters)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get users")
		return 0, fmt.Errorf("failed to get users: %w", err)
	}

	awardsCount := 0

	// Evaluate each badge for each user
	for _, badge := range badges {
		for _, user := range users {
			// Check if user already has this badge
			hasEarned, err := s.badgeRepo.HasUserEarnedBadge(user.ID, badge.ID)
			if err != nil {
				s.log.Error().
					Err(err).
					Uint("user_id", user.ID).
					Uint("badge_id", badge.ID).
					Msg("Failed to check if user has badge")
				continue
			}

			if hasEarned {
				// User already has this badge, skip
				continue
			}

			// Evaluate badge criteria
			qualifies, err := s.EvaluateBadge(ctx, &badge, user.ID)
			if err != nil {
				s.log.Error().
					Err(err).
					Uint("user_id", user.ID).
					Str("badge", badge.Name).
					Msg("Failed to evaluate badge")
				continue
			}

			if qualifies {
				// Award badge
				err = s.AwardBadge(ctx, user.ID, &badge)
				if err != nil {
					s.log.Error().
						Err(err).
						Uint("user_id", user.ID).
						Str("badge", badge.Name).
						Msg("Failed to award badge")
					continue
				}

				awardsCount++
				s.log.Info().
					Uint("user_id", user.ID).
					Str("username", user.Username).
					Str("badge", badge.Name).
					Msg("Badge awarded")
			}
		}
	}

	duration := time.Since(start)
	s.log.Info().
		Int("badges_evaluated", len(badges)).
		Int("users_evaluated", len(users)).
		Int("badges_awarded", awardsCount).
		Dur("duration", duration).
		Msg("Badge evaluation complete")

	return awardsCount, nil
}

// EvaluateUserBadges evaluates all badges for a specific user and returns newly earned badges.
func (s *Service) EvaluateUserBadges(ctx context.Context, userID uint) ([]models.Badge, error) {
	s.log.Debug().Uint("user_id", userID).Msg("Evaluating badges for user")

	// Get all badges
	badges, err := s.badgeRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get badges: %w", err)
	}

	var newlyEarned []models.Badge

	for _, badge := range badges {
		// Check if user already has this badge
		hasEarned, err := s.badgeRepo.HasUserEarnedBadge(userID, badge.ID)
		if err != nil {
			s.log.Error().
				Err(err).
				Uint("user_id", userID).
				Uint("badge_id", badge.ID).
				Msg("Failed to check if user has badge")
			continue
		}

		if hasEarned {
			continue
		}

		// Evaluate badge criteria
		qualifies, err := s.EvaluateBadge(ctx, &badge, userID)
		if err != nil {
			s.log.Error().
				Err(err).
				Uint("user_id", userID).
				Str("badge", badge.Name).
				Msg("Failed to evaluate badge")
			continue
		}

		if qualifies {
			// Award badge
			err = s.AwardBadge(ctx, userID, &badge)
			if err != nil {
				s.log.Error().
					Err(err).
					Uint("user_id", userID).
					Str("badge", badge.Name).
					Msg("Failed to award badge")
				continue
			}

			newlyEarned = append(newlyEarned, badge)
		}
	}

	return newlyEarned, nil
}

// EvaluateBadge evaluates if a user qualifies for a specific badge.
func (s *Service) EvaluateBadge(ctx context.Context, badge *models.Badge, userID uint) (bool, error) {
	// Parse badge criteria
	var criteria models.BadgeCriteria
	err := json.Unmarshal(badge.Criteria, &criteria)
	if err != nil {
		return false, fmt.Errorf("failed to parse badge criteria: %w", err)
	}

	// Evaluate criteria
	return s.checkCriteria(ctx, &criteria, userID)
}

// AwardBadge awards a badge to a user.
//
//nolint:revive // ctx reserved for future context-aware operations (tracing, cancellation)
func (s *Service) AwardBadge(ctx context.Context, userID uint, badge *models.Badge) error {
	err := s.badgeRepo.AwardBadge(userID, badge.ID)
	if err != nil {
		return err
	}

	// Get user to record team name in metrics
	user, userErr := s.userRepo.GetByID(userID)
	team := "unknown"
	if userErr == nil && user != nil {
		team = user.Team
	}

	// Record badge awarded metric
	prommetrics.RecordBadgeAwarded(badge.Name, team)

	// Update active holders count
	count, _ := s.badgeRepo.GetBadgeHoldersCount(badge.ID)
	prommetrics.SetActiveBadgeHolders(badge.Name, int(count))

	return nil
}

// GetUserBadges retrieves all badges earned by a user.
//
//nolint:revive // ctx reserved for future context-aware operations (tracing, cancellation)
func (s *Service) GetUserBadges(ctx context.Context, userID uint) ([]models.UserBadge, error) {
	return s.badgeRepo.GetUserBadges(userID)
}

// GetBadgeCatalog retrieves all available badges.
//
//nolint:revive // ctx reserved for future context-aware operations (tracing, cancellation)
func (s *Service) GetBadgeCatalog(ctx context.Context) ([]models.Badge, error) {
	return s.badgeRepo.GetAll()
}

// GetBadgeByID retrieves a badge by its ID.
//
//nolint:revive // ctx reserved for future context-aware operations (tracing, cancellation)
func (s *Service) GetBadgeByID(ctx context.Context, badgeID uint) (*models.Badge, error) {
	return s.badgeRepo.GetByID(badgeID)
}

// GetBadgeHolders retrieves users who have earned a specific badge.
//
//nolint:revive // ctx reserved for future context-aware operations (tracing, cancellation)
func (s *Service) GetBadgeHolders(ctx context.Context, badgeID uint) ([]models.User, error) {
	return s.badgeRepo.GetUsersWithBadge(badgeID)
}

// GetBadgeHoldersCount retrieves the count of users who have earned a badge.
//
//nolint:revive // ctx reserved for future context-aware operations (tracing, cancellation)
func (s *Service) GetBadgeHoldersCount(ctx context.Context, badgeID uint) (int64, error) {
	return s.badgeRepo.GetBadgeHoldersCount(badgeID)
}

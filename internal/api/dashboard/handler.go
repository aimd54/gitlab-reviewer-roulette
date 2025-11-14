// Package dashboard provides REST API handlers for the gamification dashboard.
// It exposes endpoints for leaderboards, user statistics, badges, and badge holders.
package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/service/badges"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/service/leaderboard"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// BadgeService interface for badge operations.
type BadgeService interface {
	GetUserBadges(ctx context.Context, userID uint) ([]models.UserBadge, error)
	GetBadgeCatalog(ctx context.Context) ([]models.Badge, error)
	GetBadgeByID(ctx context.Context, badgeID uint) (*models.Badge, error)
	GetBadgeHolders(ctx context.Context, badgeID uint) ([]models.User, error)
}

// LeaderboardService interface for leaderboard operations.
type LeaderboardService interface {
	GetGlobalLeaderboard(ctx context.Context, period, metric string, limit int) ([]leaderboard.Entry, error)
	GetTeamLeaderboard(ctx context.Context, team, period, metric string, limit int) ([]leaderboard.Entry, error)
	GetUserStats(ctx context.Context, userID uint, period string) (*leaderboard.UserStats, error)
}

// Handler handles dashboard API requests.
type Handler struct {
	badgeService       BadgeService
	leaderboardService LeaderboardService
	log                *logger.Logger
}

// NewHandler creates a new dashboard handler.
func NewHandler(badgeService *badges.Service, leaderboardService *leaderboard.Service, log *logger.Logger) *Handler {
	return &Handler{
		badgeService:       badgeService,
		leaderboardService: leaderboardService,
		log:                log,
	}
}

// NewHandlerWithInterfaces creates a new dashboard handler with interface dependencies (useful for testing).
func NewHandlerWithInterfaces(badgeService BadgeService, leaderboardService LeaderboardService, log *logger.Logger) *Handler {
	return &Handler{
		badgeService:       badgeService,
		leaderboardService: leaderboardService,
		log:                log,
	}
}

// GetGlobalLeaderboard returns the global leaderboard.
// GET /api/v1/leaderboard?period=month&metric=completed_reviews&limit=10.
func (h *Handler) GetGlobalLeaderboard(c *gin.Context) {
	period := c.DefaultQuery("period", "all_time")
	metric := c.DefaultQuery("metric", "completed_reviews")
	limit, err := h.parseLimit(c, 10)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate parameters
	if err := h.validatePeriod(period); err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validateMetric(metric); err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := context.Background()
	entries, err := h.leaderboardService.GetGlobalLeaderboard(ctx, period, metric, limit)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get global leaderboard")
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve leaderboard")
		return
	}

	h.log.Info().
		Str("period", period).
		Str("metric", metric).
		Int("limit", limit).
		Int("entries", len(entries)).
		Msg("Retrieved global leaderboard")

	c.JSON(http.StatusOK, gin.H{
		"leaderboard":   entries,
		"period":        period,
		"metric":        metric,
		"total_entries": len(entries),
		"generated_at":  time.Now().UTC(),
	})
}

// GetTeamLeaderboard returns the leaderboard for a specific team.
// GET /api/v1/leaderboard/:team?period=month&metric=completed_reviews&limit=10.
func (h *Handler) GetTeamLeaderboard(c *gin.Context) {
	team := c.Param("team")
	if team == "" {
		h.errorResponse(c, http.StatusBadRequest, "team parameter is required")
		return
	}

	period := c.DefaultQuery("period", "all_time")
	metric := c.DefaultQuery("metric", "completed_reviews")
	limit, err := h.parseLimit(c, 10)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate parameters
	if err := h.validatePeriod(period); err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validateMetric(metric); err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := context.Background()
	entries, err := h.leaderboardService.GetTeamLeaderboard(ctx, team, period, metric, limit)
	if err != nil {
		h.log.Error().Err(err).Str("team", team).Msg("Failed to get team leaderboard")
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve team leaderboard")
		return
	}

	h.log.Info().
		Str("team", team).
		Str("period", period).
		Str("metric", metric).
		Int("limit", limit).
		Int("entries", len(entries)).
		Msg("Retrieved team leaderboard")

	c.JSON(http.StatusOK, gin.H{
		"team":          team,
		"leaderboard":   entries,
		"period":        period,
		"metric":        metric,
		"total_entries": len(entries),
		"generated_at":  time.Now().UTC(),
	})
}

// GetUserStats returns statistics for a specific user.
// GET /api/v1/users/:id/stats?period=month.
func (h *Handler) GetUserStats(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	period := c.DefaultQuery("period", "all_time")
	if err := h.validatePeriod(period); err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := context.Background()
	stats, err := h.leaderboardService.GetUserStats(ctx, userID, period)
	if err != nil {
		h.log.Error().Err(err).Uint("user_id", userID).Msg("Failed to get user stats")
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve user statistics")
		return
	}

	h.log.Info().
		Uint("user_id", userID).
		Str("period", period).
		Msg("Retrieved user stats")

	c.JSON(http.StatusOK, gin.H{
		"stats":        stats,
		"generated_at": time.Now().UTC(),
	})
}

// GetUserBadges returns badges earned by a specific user.
// GET /api/v1/users/:id/badges.
func (h *Handler) GetUserBadges(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := context.Background()
	userBadges, err := h.badgeService.GetUserBadges(ctx, userID)
	if err != nil {
		h.log.Error().Err(err).Uint("user_id", userID).Msg("Failed to get user badges")
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve user badges")
		return
	}

	h.log.Info().
		Uint("user_id", userID).
		Int("badge_count", len(userBadges)).
		Msg("Retrieved user badges")

	c.JSON(http.StatusOK, gin.H{
		"user_id":      userID,
		"badges":       userBadges,
		"total_badges": len(userBadges),
		"generated_at": time.Now().UTC(),
	})
}

// GetBadgeCatalog returns all available badges with holder counts.
// GET /api/v1/badges.
func (h *Handler) GetBadgeCatalog(c *gin.Context) {
	ctx := context.Background()
	catalogBadges, err := h.badgeService.GetBadgeCatalog(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get badge catalog")
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve badge catalog")
		return
	}

	h.log.Info().
		Int("badge_count", len(catalogBadges)).
		Msg("Retrieved badge catalog")

	c.JSON(http.StatusOK, gin.H{
		"badges":       catalogBadges,
		"total_badges": len(catalogBadges),
		"generated_at": time.Now().UTC(),
	})
}

// GetBadgeByID returns details for a specific badge.
// GET /api/v1/badges/:id.
func (h *Handler) GetBadgeByID(c *gin.Context) {
	badgeID, err := h.parseBadgeID(c)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := context.Background()
	badge, err := h.badgeService.GetBadgeByID(ctx, badgeID)
	if err != nil {
		h.log.Error().Err(err).Uint("badge_id", badgeID).Msg("Failed to get badge details")
		h.errorResponse(c, http.StatusNotFound, "Badge not found")
		return
	}

	h.log.Info().
		Uint("badge_id", badgeID).
		Str("badge_name", badge.Name).
		Msg("Retrieved badge details")

	c.JSON(http.StatusOK, gin.H{
		"badge":        badge,
		"generated_at": time.Now().UTC(),
	})
}

// GetBadgeHolders returns users who have earned a specific badge.
// GET /api/v1/badges/:id/holders?limit=50.
func (h *Handler) GetBadgeHolders(c *gin.Context) {
	badgeID, err := h.parseBadgeID(c)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	limit, err := h.parseLimit(c, 50)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := context.Background()
	holders, err := h.badgeService.GetBadgeHolders(ctx, badgeID)
	if err != nil {
		h.log.Error().Err(err).Uint("badge_id", badgeID).Msg("Failed to get badge holders")
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve badge holders")
		return
	}

	// Apply limit
	totalHolders := len(holders)
	if limit > 0 && len(holders) > limit {
		holders = holders[:limit]
	}

	h.log.Info().
		Uint("badge_id", badgeID).
		Int("holder_count", len(holders)).
		Int("limit", limit).
		Msg("Retrieved badge holders")

	c.JSON(http.StatusOK, gin.H{
		"badge_id":      badgeID,
		"holders":       holders,
		"total_holders": totalHolders,
		"limited_to":    len(holders),
		"generated_at":  time.Now().UTC(),
	})
}

// Helper functions

// parseUserID extracts and validates the user ID from the URL parameter.
func (h *Handler) parseUserID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %s", idStr)
	}
	return uint(id), nil
}

// parseBadgeID extracts and validates the badge ID from the URL parameter.
func (h *Handler) parseBadgeID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid badge ID: %s", idStr)
	}
	return uint(id), nil
}

// parseLimit extracts and validates the limit query parameter.
func (h *Handler) parseLimit(c *gin.Context, defaultLimit int) (int, error) {
	limitStr := c.Query("limit")
	if limitStr == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, fmt.Errorf("invalid limit parameter: %s", limitStr)
	}

	if limit < 1 {
		return 0, fmt.Errorf("limit must be greater than 0")
	}

	if limit > 1000 {
		return 0, fmt.Errorf("limit cannot exceed 1000")
	}

	return limit, nil
}

// validatePeriod validates the period parameter.
func (h *Handler) validatePeriod(period string) error {
	validPeriods := map[string]bool{
		"day":      true,
		"week":     true,
		"month":    true,
		"year":     true,
		"all_time": true,
	}

	if !validPeriods[period] {
		return fmt.Errorf("invalid period: %s (valid: day, week, month, year, all_time)", period)
	}
	return nil
}

// validateMetric validates the metric parameter.
func (h *Handler) validateMetric(metric string) error {
	validMetrics := map[string]bool{
		"completed_reviews": true,
		"engagement_score":  true,
		"avg_ttfr":          true,
		"avg_comment_count": true,
	}

	if !validMetrics[metric] {
		return fmt.Errorf("invalid metric: %s (valid: completed_reviews, engagement_score, avg_ttfr, avg_comment_count)", metric)
	}
	return nil
}

// errorResponse sends a standardized error response.
func (h *Handler) errorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"error":     message,
		"timestamp": time.Now().UTC(),
	})
}

// Package scheduler provides daily notification scheduling for pending merge requests.
package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/config"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/mattermost"
	prommetrics "github.com/aimd54/gitlab-reviewer-roulette/internal/metrics"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/repository"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/service/badges"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// Service handles daily notification scheduling.
type Service struct {
	config           *config.Config
	reviewRepo       *repository.ReviewRepository
	badgeService     *badges.Service
	mattermostClient *mattermost.Client
	log              *logger.Logger
	cron             *cron.Cron
}

// NewService creates a new scheduler service.
func NewService(
	cfg *config.Config,
	reviewRepo *repository.ReviewRepository,
	badgeService *badges.Service,
	mattermostClient *mattermost.Client,
	log *logger.Logger,
) *Service {
	return &Service{
		config:           cfg,
		reviewRepo:       reviewRepo,
		badgeService:     badgeService,
		mattermostClient: mattermostClient,
		log:              log,
	}
}

// Start initializes and starts the cron scheduler.
func (s *Service) Start() error {
	// Validate configuration
	if !s.config.Scheduler.Enabled {
		s.log.Info().Msg("Scheduler is disabled in configuration")
		return nil
	}

	// Load timezone
	location, err := time.LoadLocation(s.config.Scheduler.Timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", s.config.Scheduler.Timezone, err)
	}

	// Create cron scheduler with timezone
	s.cron = cron.New(cron.WithLocation(location))

	// Build cron expression
	cronExpr, err := s.buildCronExpression()
	if err != nil {
		return fmt.Errorf("failed to build cron expression: %w", err)
	}

	// Register daily notification job
	_, err = s.cron.AddFunc(cronExpr, func() {
		s.runDailyNotifications(context.Background())
	})
	if err != nil {
		return fmt.Errorf("failed to register daily notification job: %w", err)
	}

	// Register badge evaluation job if configured
	if s.config.Scheduler.BadgeEvaluationTime != "" && s.badgeService != nil {
		_, err = s.cron.AddFunc(s.config.Scheduler.BadgeEvaluationTime, func() {
			s.runBadgeEvaluation(context.Background())
		})
		if err != nil {
			return fmt.Errorf("failed to register badge evaluation job: %w", err)
		}
		s.log.Info().
			Str("schedule", s.config.Scheduler.BadgeEvaluationTime).
			Msg("Badge evaluation job registered")
	}

	// Start the scheduler
	s.cron.Start()

	// Log scheduler start with next run time
	entries := s.cron.Entries()
	nextRun := ""
	if len(entries) > 0 {
		nextRun = entries[0].Next.Format(time.RFC3339)
	}

	s.log.Info().
		Str("schedule", cronExpr).
		Str("timezone", s.config.Scheduler.Timezone).
		Str("time", s.config.Scheduler.Time).
		Bool("skip_weekends", s.config.Scheduler.SkipWeekends).
		Str("next_run", nextRun).
		Msg("Scheduler started successfully")

	return nil
}

// Stop gracefully shuts down the scheduler.
func (s *Service) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		s.log.Info().Msg("Scheduler stopped")
	}
}

// buildCronExpression generates a cron expression from config.
func (s *Service) buildCronExpression() (string, error) {
	// Parse time string (format: "HH:MM")
	parts := strings.Split(s.config.Scheduler.Time, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid time format %q, expected HH:MM", s.config.Scheduler.Time)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return "", fmt.Errorf("invalid hour %q", parts[0])
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return "", fmt.Errorf("invalid minute %q", parts[1])
	}

	// Build cron expression
	// Format: "minute hour day month weekday"
	if s.config.Scheduler.SkipWeekends {
		// Monday-Friday only (1-5)
		return fmt.Sprintf("%d %d * * 1-5", minute, hour), nil
	}

	// Every day
	return fmt.Sprintf("%d %d * * *", minute, hour), nil
}

// runDailyNotifications executes the daily notification job.
func (s *Service) runDailyNotifications(_ context.Context) {
	start := time.Now()

	// Track job duration and update last run timestamp on exit
	defer func() {
		duration := time.Since(start).Seconds()
		prommetrics.ObserveSchedulerJobDuration(duration)
		prommetrics.SetSchedulerLastRun()
	}()

	s.log.Info().Msg("Running daily notification job")

	// Query pending MRs
	queryStart := time.Now()
	reviews, err := s.reviewRepo.ListPendingMRReviews()
	queryDuration := time.Since(queryStart)

	if err != nil {
		s.log.Error().
			Err(err).
			Dur("query_duration", queryDuration).
			Msg("Failed to list pending MR reviews")
		prommetrics.RecordSchedulerJobRun("error")
		prommetrics.RecordSchedulerNotificationFailed("query_error")
		return
	}

	s.log.Info().
		Int("count", len(reviews)).
		Dur("query_duration", queryDuration).
		Msg("Found pending MR reviews")

	// Build pending MRs for Mattermost
	pendingMRs := buildPendingMRs(reviews)

	// Filter out very recent MRs (< 4 hours old)
	filtered := filterRecentMRs(pendingMRs, 4*time.Hour)

	s.log.Info().
		Int("total", len(pendingMRs)).
		Int("filtered", len(filtered)).
		Msg("Filtered recent MRs")

	// Send notification
	if len(filtered) == 0 {
		s.log.Debug().Msg("No pending MRs to notify about")
		prommetrics.RecordSchedulerJobRun("success")
		return
	}

	// Send to Mattermost
	sendStart := time.Now()
	err = s.mattermostClient.SendDailyReviewReminder(filtered)
	sendDuration := time.Since(sendStart)

	if err != nil {
		s.log.Error().
			Err(err).
			Dur("send_duration", sendDuration).
			Msg("Failed to send daily review reminder")
		prommetrics.RecordSchedulerJobRun("error")
		prommetrics.RecordSchedulerNotificationFailed("mattermost_error")
		return
	}

	// Success - record metrics
	prommetrics.RecordSchedulerJobRun("success")
	prommetrics.RecordSchedulerNotificationSent("all") // Could be team-specific if needed
	prommetrics.SetSchedulerPendingMRs("all", len(filtered))

	s.log.Info().
		Int("mr_count", len(filtered)).
		Dur("send_duration", sendDuration).
		Dur("total_duration", time.Since(start)).
		Msg("Successfully sent daily notification")
}

// filterRecentMRs filters out MRs that are too recent.
func filterRecentMRs(pendingMRs []mattermost.PendingMR, minAge time.Duration) []mattermost.PendingMR {
	var filtered []mattermost.PendingMR

	for _, mr := range pendingMRs {
		age := mr.Age()
		if age >= minAge {
			filtered = append(filtered, mr)
		}
	}

	return filtered
}

// runBadgeEvaluation executes the daily badge evaluation job.
func (s *Service) runBadgeEvaluation(ctx context.Context) {
	start := time.Now()

	// Track job duration and update last run timestamp on exit
	defer func() {
		duration := time.Since(start).Seconds()
		prommetrics.ObserveBadgeEvaluationDuration(duration)
	}()

	s.log.Info().Msg("Running badge evaluation job")

	// Run badge evaluation for all users
	awardsCount, err := s.badgeService.EvaluateAllBadges(ctx)
	if err != nil {
		s.log.Error().
			Err(err).
			Dur("duration", time.Since(start)).
			Msg("Badge evaluation job failed")
		prommetrics.RecordBadgeEvaluationRun("error")
		return
	}

	duration := time.Since(start)
	prommetrics.RecordBadgeEvaluationRun("success")

	s.log.Info().
		Int("badges_awarded", awardsCount).
		Dur("duration", duration).
		Msg("Badge evaluation job completed successfully")
}

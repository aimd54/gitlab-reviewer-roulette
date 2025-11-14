// Package metrics provides Prometheus exporters for application metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for the reviewer roulette bot.
var (
	// Counters.
	RouletteTriggersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "roulette_triggers_total",
			Help: "Total number of roulette commands triggered",
		},
		[]string{"team", "status"},
	)

	ReviewsCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reviews_completed_total",
			Help: "Total number of reviews completed (approved or merged)",
		},
		[]string{"team", "user", "role"},
	)

	ReviewsAbandonedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reviews_abandoned_total",
			Help: "Total number of reviews abandoned (closed without merge)",
		},
		[]string{"team"},
	)

	// Gauges.
	ActiveReviews = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_reviews",
			Help: "Current number of active reviews",
		},
		[]string{"team", "user"},
	)

	AvailableReviewers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "available_reviewers",
			Help: "Current number of available reviewers",
		},
		[]string{"team", "role"},
	)

	// Histograms.
	ReviewTTFRSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "review_ttfr_seconds",
			Help:    "Time to first review in seconds",
			Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1min to ~17hours
		},
		[]string{"team"},
	)

	ReviewTimeToApprovalSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "review_time_to_approval_seconds",
			Help:    "Time from trigger to approval in seconds",
			Buckets: prometheus.ExponentialBuckets(300, 2, 10), // 5min to ~85hours
		},
		[]string{"team"},
	)

	ReviewCommentCount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "review_comment_count",
			Help:    "Number of comments per review",
			Buckets: prometheus.LinearBuckets(0, 5, 10), // 0 to 45 comments
		},
		[]string{"team"},
	)

	ReviewCommentLength = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "review_comment_length",
			Help:    "Total length of comments per review",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10), // 100 to ~100k chars
		},
		[]string{"team"},
	)

	// Summary.
	ReviewerEngagementScore = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "reviewer_engagement_score",
			Help:       "Reviewer engagement score based on comments and thoroughness",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"team", "user"},
	)

	// Scheduler metrics.
	SchedulerJobsRunTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scheduler_jobs_run_total",
			Help: "Total scheduler job executions",
		},
		[]string{"status"},
	)

	SchedulerNotificationsSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scheduler_notifications_sent_total",
			Help: "Total successful daily notifications sent",
		},
		[]string{"team"},
	)

	SchedulerNotificationsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scheduler_notifications_failed_total",
			Help: "Total failed notification attempts",
		},
		[]string{"reason"},
	)

	SchedulerPendingMRsCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scheduler_pending_mrs_count",
			Help: "Number of pending MRs in last notification",
		},
		[]string{"team"},
	)

	SchedulerLastRunTimestamp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "scheduler_last_run_timestamp",
			Help: "Unix timestamp of last scheduler run",
		},
	)

	SchedulerJobDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "scheduler_job_duration_seconds",
			Help:    "Time taken to execute scheduler notification job",
			Buckets: prometheus.ExponentialBuckets(1, 2, 8), // 1s to ~128s
		},
	)

	// Badge gamification metrics.
	BadgesAwardedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "badges_awarded_total",
			Help: "Total number of badges awarded",
		},
		[]string{"badge_name", "team"},
	)

	ActiveBadgeHolders = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_badge_holders",
			Help: "Current number of users holding each badge",
		},
		[]string{"badge_name"},
	)

	BadgeEvaluationJobsRunTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "badge_evaluation_jobs_run_total",
			Help: "Total badge evaluation job executions",
		},
		[]string{"status"},
	)

	BadgeEvaluationDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "badge_evaluation_duration_seconds",
			Help:    "Time taken to execute badge evaluation job",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~1024s
		},
	)
)

// RecordRouletteTrigger records a roulette command trigger.
func RecordRouletteTrigger(team, status string) {
	RouletteTriggersTotal.WithLabelValues(team, status).Inc()
}

// RecordReviewCompleted records a completed review.
func RecordReviewCompleted(team, user, role string) {
	ReviewsCompletedTotal.WithLabelValues(team, user, role).Inc()
}

// RecordReviewAbandoned records an abandoned review.
func RecordReviewAbandoned(team string) {
	ReviewsAbandonedTotal.WithLabelValues(team).Inc()
}

// SetActiveReviews sets the current number of active reviews for a user.
func SetActiveReviews(team, user string, count int) {
	ActiveReviews.WithLabelValues(team, user).Set(float64(count))
}

// SetAvailableReviewers sets the current number of available reviewers.
func SetAvailableReviewers(team, role string, count int) {
	AvailableReviewers.WithLabelValues(team, role).Set(float64(count))
}

// ObserveTTFR observes time to first review.
func ObserveTTFR(team string, seconds float64) {
	ReviewTTFRSeconds.WithLabelValues(team).Observe(seconds)
}

// ObserveTimeToApproval observes time to approval.
func ObserveTimeToApproval(team string, seconds float64) {
	ReviewTimeToApprovalSeconds.WithLabelValues(team).Observe(seconds)
}

// ObserveCommentCount observes comment count.
func ObserveCommentCount(team string, count float64) {
	ReviewCommentCount.WithLabelValues(team).Observe(count)
}

// ObserveCommentLength observes comment length.
func ObserveCommentLength(team string, length float64) {
	ReviewCommentLength.WithLabelValues(team).Observe(length)
}

// ObserveEngagementScore observes engagement score.
func ObserveEngagementScore(team, user string, score float64) {
	ReviewerEngagementScore.WithLabelValues(team, user).Observe(score)
}

// RecordSchedulerJobRun records a scheduler job execution.
func RecordSchedulerJobRun(status string) {
	SchedulerJobsRunTotal.WithLabelValues(status).Inc()
}

// RecordSchedulerNotificationSent records a successful notification sent.
func RecordSchedulerNotificationSent(team string) {
	SchedulerNotificationsSentTotal.WithLabelValues(team).Inc()
}

// RecordSchedulerNotificationFailed records a failed notification attempt.
func RecordSchedulerNotificationFailed(reason string) {
	SchedulerNotificationsFailedTotal.WithLabelValues(reason).Inc()
}

// SetSchedulerPendingMRs sets the number of pending MRs in the last notification.
func SetSchedulerPendingMRs(team string, count int) {
	SchedulerPendingMRsCount.WithLabelValues(team).Set(float64(count))
}

// SetSchedulerLastRun sets the timestamp of the last scheduler run.
func SetSchedulerLastRun() {
	SchedulerLastRunTimestamp.SetToCurrentTime()
}

// ObserveSchedulerJobDuration observes the duration of a scheduler job.
func ObserveSchedulerJobDuration(seconds float64) {
	SchedulerJobDurationSeconds.Observe(seconds)
}

// RecordBadgeAwarded records a badge award event.
func RecordBadgeAwarded(badgeName, team string) {
	BadgesAwardedTotal.WithLabelValues(badgeName, team).Inc()
}

// SetActiveBadgeHolders sets the number of holders for a badge.
func SetActiveBadgeHolders(badgeName string, count int) {
	ActiveBadgeHolders.WithLabelValues(badgeName).Set(float64(count))
}

// RecordBadgeEvaluationRun records a badge evaluation job execution.
func RecordBadgeEvaluationRun(status string) {
	BadgeEvaluationJobsRunTotal.WithLabelValues(status).Inc()
}

// ObserveBadgeEvaluationDuration observes the duration of a badge evaluation job.
func ObserveBadgeEvaluationDuration(seconds float64) {
	BadgeEvaluationDurationSeconds.Observe(seconds)
}

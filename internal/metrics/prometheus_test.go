package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordRouletteTrigger(t *testing.T) {
	// Reset the counter before test
	RouletteTriggersTotal.Reset()

	// Record some triggers
	RecordRouletteTrigger("team-frontend", "success")
	RecordRouletteTrigger("team-frontend", "success")
	RecordRouletteTrigger("team-backend", "success")

	// Verify counter increased
	count := testutil.ToFloat64(RouletteTriggersTotal.WithLabelValues("team-frontend", "success"))
	if count != 2 {
		t.Errorf("Expected team-frontend success count = 2, got %f", count)
	}

	count = testutil.ToFloat64(RouletteTriggersTotal.WithLabelValues("team-backend", "success"))
	if count != 1 {
		t.Errorf("Expected team-backend success count = 1, got %f", count)
	}
}

func TestRecordReviewCompleted(t *testing.T) {
	// Reset the counter before test
	ReviewsCompletedTotal.Reset()

	// Record some completions
	RecordReviewCompleted("team-frontend", "alice", "codeowner")
	RecordReviewCompleted("team-frontend", "bob", "team_member")

	// Verify counter increased
	count := testutil.ToFloat64(ReviewsCompletedTotal.WithLabelValues("team-frontend", "alice", "codeowner"))
	if count != 1 {
		t.Errorf("Expected alice codeowner count = 1, got %f", count)
	}
}

func TestRecordReviewAbandoned(t *testing.T) {
	// Reset the counter before test
	ReviewsAbandonedTotal.Reset()

	// Record some abandonments
	RecordReviewAbandoned("team-frontend")
	RecordReviewAbandoned("team-frontend")

	// Verify counter increased
	count := testutil.ToFloat64(ReviewsAbandonedTotal.WithLabelValues("team-frontend"))
	if count != 2 {
		t.Errorf("Expected abandoned count = 2, got %f", count)
	}
}

func TestSetActiveReviews(t *testing.T) {
	// Set active reviews for users
	SetActiveReviews("team-frontend", "alice", 3)
	SetActiveReviews("team-frontend", "bob", 1)

	// Verify gauge values
	count := testutil.ToFloat64(ActiveReviews.WithLabelValues("team-frontend", "alice"))
	if count != 3 {
		t.Errorf("Expected alice active reviews = 3, got %f", count)
	}

	count = testutil.ToFloat64(ActiveReviews.WithLabelValues("team-frontend", "bob"))
	if count != 1 {
		t.Errorf("Expected bob active reviews = 1, got %f", count)
	}
}

func TestSetAvailableReviewers(t *testing.T) {
	// Set available reviewers
	SetAvailableReviewers("team-frontend", "dev", 5)
	SetAvailableReviewers("team-frontend", "ops", 3)

	// Verify gauge values
	count := testutil.ToFloat64(AvailableReviewers.WithLabelValues("team-frontend", "dev"))
	if count != 5 {
		t.Errorf("Expected dev available = 5, got %f", count)
	}

	count = testutil.ToFloat64(AvailableReviewers.WithLabelValues("team-frontend", "ops"))
	if count != 3 {
		t.Errorf("Expected ops available = 3, got %f", count)
	}
}

func TestObserveTTFR(t *testing.T) {
	// Observe some TTFR values
	ObserveTTFR("team-frontend", 3600) // 1 hour
	ObserveTTFR("team-frontend", 7200) // 2 hours

	// Verify histogram has observations
	// Note: We can't easily check histogram values without scraping,
	// so we just ensure it doesn't panic
}

func TestObserveTimeToApproval(t *testing.T) {
	// Observe some approval times
	ObserveTimeToApproval("team-frontend", 86400) // 1 day

	// Verify it doesn't panic
}

func TestObserveCommentCount(t *testing.T) {
	// Observe some comment counts
	ObserveCommentCount("team-frontend", 5)
	ObserveCommentCount("team-frontend", 10)

	// Verify it doesn't panic
}

func TestObserveCommentLength(t *testing.T) {
	// Observe some comment lengths
	ObserveCommentLength("team-frontend", 500)
	ObserveCommentLength("team-frontend", 1000)

	// Verify it doesn't panic
}

func TestObserveEngagementScore(t *testing.T) {
	// Observe some engagement scores
	ObserveEngagementScore("team-frontend", "alice", 75.5)
	ObserveEngagementScore("team-frontend", "bob", 50.0)

	// Verify it doesn't panic
}

func TestMetricsRegistration(t *testing.T) {
	// Verify all metrics are registered
	metrics := []prometheus.Collector{
		RouletteTriggersTotal,
		ReviewsCompletedTotal,
		ReviewsAbandonedTotal,
		ActiveReviews,
		AvailableReviewers,
		ReviewTTFRSeconds,
		ReviewTimeToApprovalSeconds,
		ReviewCommentCount,
		ReviewCommentLength,
		ReviewerEngagementScore,
	}

	for i, metric := range metrics {
		if metric == nil {
			t.Errorf("Metric %d is nil", i)
		}
	}
}

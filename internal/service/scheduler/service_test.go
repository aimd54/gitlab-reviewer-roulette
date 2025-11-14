package scheduler

import (
	"testing"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/config"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/mattermost"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

func TestBuildCronExpression(t *testing.T) {
	tests := []struct {
		name         string
		time         string
		skipWeekends bool
		want         string
		wantErr      bool
	}{
		{
			name:         "daily at 9am",
			time:         "09:00",
			skipWeekends: false,
			want:         "0 9 * * *",
			wantErr:      false,
		},
		{
			name:         "weekdays at 9am",
			time:         "09:00",
			skipWeekends: true,
			want:         "0 9 * * 1-5",
			wantErr:      false,
		},
		{
			name:         "daily at 14:30",
			time:         "14:30",
			skipWeekends: false,
			want:         "30 14 * * *",
			wantErr:      false,
		},
		{
			name:         "invalid format no colon",
			time:         "0900",
			skipWeekends: false,
			want:         "",
			wantErr:      true,
		},
		{
			name:         "invalid hour",
			time:         "25:00",
			skipWeekends: false,
			want:         "",
			wantErr:      true,
		},
		{
			name:         "invalid minute",
			time:         "09:60",
			skipWeekends: false,
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Scheduler: config.SchedulerConfig{
					Time:         tt.time,
					SkipWeekends: tt.skipWeekends,
				},
			}

			s := &Service{config: cfg}

			got, err := s.buildCronExpression()

			if (err != nil) != tt.wantErr {
				t.Errorf("buildCronExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("buildCronExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildPendingMRs(t *testing.T) {
	yesterday := time.Now().Add(-24 * time.Hour)
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	zeroTime := time.Time{}

	alice := &models.User{
		Username: "alice",
	}

	tests := []struct {
		name    string
		reviews []models.MRReview
		want    int // Expected count of pending MRs
	}{
		{
			name: "normal case with author",
			reviews: []models.MRReview{
				{
					MRTitle:             "Fix bug in login",
					MRURL:               "https://gitlab.com/project/mr/1",
					Team:                "backend",
					RouletteTriggeredAt: &yesterday,
					MRAuthor:            alice,
				},
			},
			want: 1,
		},
		{
			name: "nil author",
			reviews: []models.MRReview{
				{
					MRTitle:             "Add feature",
					MRURL:               "https://gitlab.com/project/mr/2",
					Team:                "frontend",
					RouletteTriggeredAt: &yesterday,
					MRAuthor:            nil,
				},
			},
			want: 1,
		},
		{
			name: "zero roulette triggered at",
			reviews: []models.MRReview{
				{
					MRTitle:             "Never triggered",
					MRURL:               "https://gitlab.com/project/mr/3",
					Team:                "backend",
					RouletteTriggeredAt: &zeroTime, // Zero time
					MRAuthor:            alice,
				},
			},
			want: 0, // Should be skipped
		},
		{
			name: "multiple reviews",
			reviews: []models.MRReview{
				{
					MRTitle:             "Fix #1",
					MRURL:               "https://gitlab.com/project/mr/1",
					Team:                "backend",
					RouletteTriggeredAt: &yesterday,
					MRAuthor:            alice,
				},
				{
					MRTitle:             "Fix #2",
					MRURL:               "https://gitlab.com/project/mr/2",
					Team:                "frontend",
					RouletteTriggeredAt: &twoDaysAgo,
					MRAuthor:            nil,
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPendingMRs(tt.reviews)

			if len(got) != tt.want {
				t.Errorf("buildPendingMRs() returned %d pending MRs, want %d", len(got), tt.want)
			}

			// Validate structure for non-zero results
			for i, pendingMR := range got {
				if pendingMR.Title == "" {
					t.Errorf("pendingMR[%d].Title is empty", i)
				}
				if pendingMR.URL == "" {
					t.Errorf("pendingMR[%d].URL is empty", i)
				}
				if pendingMR.Age == nil {
					t.Errorf("pendingMR[%d].Age closure is nil", i)
				} else {
					age := pendingMR.Age()
					if age <= 0 {
						t.Errorf("pendingMR[%d].Age() returned %v, expected positive duration", i, age)
					}
				}
			}
		})
	}
}

func TestBuildPendingMRs_AuthorHandling(t *testing.T) {
	yesterday := time.Now().Add(-24 * time.Hour)

	review := models.MRReview{
		MRTitle:             "Test MR",
		MRURL:               "https://gitlab.com/project/mr/1",
		Team:                "backend",
		RouletteTriggeredAt: &yesterday,
		MRAuthor:            nil,
	}

	result := buildPendingMRs([]models.MRReview{review})

	if len(result) != 1 {
		t.Fatalf("Expected 1 pending MR, got %d", len(result))
	}

	if result[0].Author != "unknown" {
		t.Errorf("Expected author 'unknown' for nil MRAuthor, got %q", result[0].Author)
	}
}

func TestFilterRecentMRs(t *testing.T) {
	// Create pending MRs with different ages
	oldMR := mattermost.PendingMR{
		Title:  "Old MR",
		URL:    "https://gitlab.com/project/mr/1",
		Author: "alice",
		Team:   "backend",
		Age: func() time.Duration {
			return 6 * time.Hour
		},
	}

	recentMR := mattermost.PendingMR{
		Title:  "Recent MR",
		URL:    "https://gitlab.com/project/mr/2",
		Author: "bob",
		Team:   "frontend",
		Age: func() time.Duration {
			return 2 * time.Hour
		},
	}

	veryOldMR := mattermost.PendingMR{
		Title:  "Very Old MR",
		URL:    "https://gitlab.com/project/mr/3",
		Author: "charlie",
		Team:   "backend",
		Age: func() time.Duration {
			return 50 * time.Hour
		},
	}

	tests := []struct {
		name       string
		pendingMRs []mattermost.PendingMR
		minAge     time.Duration
		wantCount  int
	}{
		{
			name:       "filter 4 hour minimum",
			pendingMRs: []mattermost.PendingMR{oldMR, recentMR, veryOldMR},
			minAge:     4 * time.Hour,
			wantCount:  2, // oldMR and veryOldMR
		},
		{
			name:       "no filter with zero min age",
			pendingMRs: []mattermost.PendingMR{oldMR, recentMR, veryOldMR},
			minAge:     0,
			wantCount:  3,
		},
		{
			name:       "all filtered with very high min age",
			pendingMRs: []mattermost.PendingMR{oldMR, recentMR},
			minAge:     100 * time.Hour,
			wantCount:  0,
		},
		{
			name:       "empty input",
			pendingMRs: []mattermost.PendingMR{},
			minAge:     4 * time.Hour,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterRecentMRs(tt.pendingMRs, tt.minAge)

			if len(got) != tt.wantCount {
				t.Errorf("filterRecentMRs() returned %d MRs, want %d", len(got), tt.wantCount)
			}

			// Verify all filtered MRs meet the minimum age requirement
			for i, mr := range got {
				age := mr.Age()
				if age < tt.minAge {
					t.Errorf("filtered MR[%d] has age %v, which is less than minAge %v", i, age, tt.minAge)
				}
			}
		})
	}
}

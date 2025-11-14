package models

import (
	"time"
)

// MRReview represents a merge request review tracking.
type MRReview struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	GitLabMRIID         int        `gorm:"column:gitlab_mr_iid;not null" json:"gitlab_mr_iid"`
	GitLabProjectID     int        `gorm:"column:gitlab_project_id;not null" json:"gitlab_project_id"`
	MRURL               string     `gorm:"type:text;not null" json:"mr_url"`
	MRTitle             string     `gorm:"type:text" json:"mr_title"`
	MRAuthorID          *uint      `gorm:"index" json:"mr_author_id"`
	MRAuthor            *User      `gorm:"foreignKey:MRAuthorID" json:"mr_author,omitempty"`
	Team                string     `gorm:"size:100" json:"team"`
	RouletteTriggeredAt *time.Time `json:"roulette_triggered_at"`
	RouletteTriggeredBy *uint      `json:"roulette_triggered_by"`
	TriggeredBy         *User      `gorm:"foreignKey:RouletteTriggeredBy" json:"triggered_by,omitempty"`
	BotCommentID        *int       `gorm:"index" json:"bot_comment_id"` // GitLab note ID for updating the bot's comment
	FirstReviewAt       *time.Time `json:"first_review_at"`
	ApprovedAt          *time.Time `json:"approved_at"`
	MergedAt            *time.Time `json:"merged_at"`
	ClosedAt            *time.Time `json:"closed_at"`
	Status              string     `gorm:"size:50;index" json:"status"` // 'pending', 'in_review', 'approved', 'merged', 'closed'
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// Relationships
	Assignments []ReviewerAssignment `gorm:"foreignKey:MRReviewID" json:"assignments,omitempty"`
}

// TableName specifies the table name for MRReview model.
func (MRReview) TableName() string {
	return "mr_reviews"
}

// ReviewerAssignment represents a reviewer assigned to an MR.
type ReviewerAssignment struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	MRReviewID      uint       `gorm:"not null;index" json:"mr_review_id"`
	MRReview        MRReview   `gorm:"foreignKey:MRReviewID" json:"mr_review,omitempty"`
	UserID          uint       `gorm:"not null;index" json:"user_id"`
	User            User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role            string     `gorm:"size:50" json:"role"` // 'codeowner', 'team_member', 'external'
	AssignedAt      time.Time  `json:"assigned_at"`
	StartedReviewAt *time.Time `json:"started_review_at"` // when they add themselves as reviewer
	FirstCommentAt  *time.Time `json:"first_comment_at"`
	ApprovedAt      *time.Time `json:"approved_at"`
	CommentCount    int        `gorm:"default:0" json:"comment_count"`
	CommentLength   int        `gorm:"column:comment_total_length;default:0" json:"comment_total_length"`
}

// TableName specifies the table name for ReviewerAssignment model.
func (ReviewerAssignment) TableName() string {
	return "reviewer_assignments"
}

// ReviewMetrics represents aggregated review metrics.
type ReviewMetrics struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Date              time.Time `gorm:"type:date;not null" json:"date"`
	Team              string    `gorm:"size:100" json:"team"`
	UserID            *uint     `gorm:"index" json:"user_id"`
	User              *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ProjectID         *int      `json:"project_id"`
	TotalReviews      int       `gorm:"default:0" json:"total_reviews"`
	CompletedReviews  int       `gorm:"default:0" json:"completed_reviews"`
	AvgTTFR           *int      `json:"avg_ttfr"`             // Average Time To First Review in minutes
	AvgTimeToApproval *int      `json:"avg_time_to_approval"` // in minutes
	AvgCommentCount   *float64  `gorm:"type:decimal(10,2)" json:"avg_comment_count"`
	AvgCommentLength  *float64  `gorm:"type:decimal(10,2)" json:"avg_comment_length"`
	EngagementScore   *float64  `gorm:"type:decimal(10,2)" json:"engagement_score"`
	CreatedAt         time.Time `json:"created_at"`
}

// TableName specifies the table name for ReviewMetrics model.
func (ReviewMetrics) TableName() string {
	return "review_metrics"
}

// MRStatus constants.
const (
	MRStatusPending  = "pending"
	MRStatusInReview = "in_review"
	MRStatusApproved = "approved"
	MRStatusMerged   = "merged"
	MRStatusClosed   = "closed"
)

// ReviewerRole constants.
const (
	ReviewerRoleCodeowner  = "codeowner"
	ReviewerRoleTeamMember = "team_member"
	ReviewerRoleExternal   = "external"
)

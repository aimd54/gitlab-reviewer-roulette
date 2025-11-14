package models

import (
	"time"
)

// User represents a GitLab user in the system.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GitLabID  int       `gorm:"column:gitlab_id;uniqueIndex;not null" json:"gitlab_id"`
	Username  string    `gorm:"uniqueIndex;not null;size:255" json:"username"`
	Email     string    `gorm:"size:255" json:"email"`
	Role      string    `gorm:"size:50" json:"role"` // 'dev' or 'ops'
	Team      string    `gorm:"size:100" json:"team"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for User model.
func (User) TableName() string {
	return "users"
}

// OOOStatus represents out-of-office status for a user.
type OOOStatus struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	StartDate time.Time `gorm:"not null" json:"start_date"`
	EndDate   time.Time `gorm:"not null" json:"end_date"`
	Reason    string    `gorm:"type:text" json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for OOOStatus model.
func (OOOStatus) TableName() string {
	return "ooo_status"
}

// IsActive checks if the OOO status is currently active.
func (o *OOOStatus) IsActive() bool {
	now := time.Now()
	return now.After(o.StartDate) && now.Before(o.EndDate)
}

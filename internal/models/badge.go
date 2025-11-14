// Package models defines domain models for the reviewer roulette system.
package models

import (
	"encoding/json"
	"time"
)

// Badge represents a badge that can be earned by users.
type Badge struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	Name        string          `gorm:"uniqueIndex;not null;size:100" json:"name"`
	Description string          `gorm:"type:text" json:"description"`
	Icon        string          `gorm:"size:50" json:"icon"`
	Criteria    json.RawMessage `gorm:"type:jsonb" json:"criteria"` // JSON structure for criteria
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TableName specifies the table name for Badge model.
func (Badge) TableName() string {
	return "badges"
}

// BadgeCriteria represents the criteria for earning a badge.
type BadgeCriteria struct {
	Metric   string      `json:"metric"`
	Operator string      `json:"operator"` // "<", ">", ">=", "<=", "==", "top"
	Value    interface{} `json:"value"`
	Period   string      `json:"period,omitempty"` // "day", "week", "month", "year"
}

// UserBadge represents a badge earned by a user.
type UserBadge struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	UserID   uint      `gorm:"not null;index" json:"user_id"`
	User     User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	BadgeID  uint      `gorm:"not null;index" json:"badge_id"`
	Badge    Badge     `gorm:"foreignKey:BadgeID" json:"badge,omitempty"`
	EarnedAt time.Time `gorm:"not null" json:"earned_at"`
}

// TableName specifies the table name for UserBadge model.
func (UserBadge) TableName() string {
	return "user_badges"
}

// Configuration represents a configuration key-value pair.
type Configuration struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Key       string          `gorm:"uniqueIndex;not null;size:255" json:"key"`
	Value     json.RawMessage `gorm:"type:jsonb;not null" json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// TableName specifies the table name for Configuration model.
func (Configuration) TableName() string {
	return "configuration"
}

// Package mattermost provides webhook client for sending notifications to Mattermost.
package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/config"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// Client handles Mattermost webhook notifications.
type Client struct {
	webhookURL string
	channel    string
	enabled    bool
	log        *logger.Logger
}

// NewClient creates a new Mattermost client.
func NewClient(cfg *config.MattermostConfig, log *logger.Logger) *Client {
	return &Client{
		webhookURL: cfg.WebhookURL,
		channel:    cfg.Channel,
		enabled:    cfg.Enabled,
		log:        log,
	}
}

// Message represents a Mattermost message payload.
type Message struct {
	Channel     string       `json:"channel,omitempty"`
	Username    string       `json:"username,omitempty"`
	Text        string       `json:"text,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a message attachment.
type Attachment struct {
	Fallback   string  `json:"fallback,omitempty"`
	Color      string  `json:"color,omitempty"`
	Pretext    string  `json:"pretext,omitempty"`
	AuthorName string  `json:"author_name,omitempty"`
	AuthorLink string  `json:"author_link,omitempty"`
	AuthorIcon string  `json:"author_icon,omitempty"`
	Title      string  `json:"title,omitempty"`
	TitleLink  string  `json:"title_link,omitempty"`
	Text       string  `json:"text,omitempty"`
	Fields     []Field `json:"fields,omitempty"`
	ImageURL   string  `json:"image_url,omitempty"`
	ThumbURL   string  `json:"thumb_url,omitempty"`
	Footer     string  `json:"footer,omitempty"`
	FooterIcon string  `json:"footer_icon,omitempty"`
}

// Field represents a message field.
type Field struct {
	Short bool   `json:"short"`
	Title string `json:"title"`
	Value string `json:"value"`
}

// SendMessage sends a message to Mattermost.
func (c *Client) SendMessage(msg *Message) error {
	if !c.enabled {
		c.log.Debug().Msg("Mattermost is disabled, skipping message")
		return nil
	}

	if msg.Channel == "" {
		msg.Channel = c.channel
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message to Mattermost: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mattermost returned status %d", resp.StatusCode)
	}

	c.log.Debug().
		Str("channel", msg.Channel).
		Msg("Sent message to Mattermost")

	return nil
}

// SendSimpleMessage sends a simple text message.
func (c *Client) SendSimpleMessage(text string) error {
	return c.SendMessage(&Message{
		Text: text,
	})
}

// SendDailyReviewReminder sends a daily reminder about pending reviews.
func (c *Client) SendDailyReviewReminder(pendingMRs []PendingMR) error {
	if len(pendingMRs) == 0 {
		c.log.Debug().Msg("No pending MRs, skipping daily reminder")
		return nil
	}

	text := fmt.Sprintf("### 📋 Daily Review Reminder\n\nThere are **%d** merge requests pending review:\n\n", len(pendingMRs))

	for _, mr := range pendingMRs {
		age := mr.Age()
		ageStr := fmt.Sprintf("%.1f hours", age.Hours())
		if age.Hours() > 24 {
			ageStr = fmt.Sprintf("%.1f days", age.Hours()/24)
		}

		// Add warning icon for old MRs
		icon := "•"
		if age.Hours() > 48 {
			icon = "⚠️"
		}

		text += fmt.Sprintf("%s [%s](%s) by @%s (%s old)\n", icon, mr.Title, mr.URL, mr.Author, ageStr)
	}

	text += "\n_Please review these merge requests when you have time!_ 🙏"

	return c.SendMessage(&Message{
		Username: "Reviewer Roulette Bot",
		Text:     text,
	})
}

// PendingMR represents a pending merge request for daily reminders.
type PendingMR struct {
	Title     string
	URL       string
	Author    string
	CreatedAt string
	Team      string
	Age       func() time.Duration
}

// SendRouletteResult sends the roulette selection result.
func (c *Client) SendRouletteResult(_, _ int, mrURL string, selections []ReviewerSelection) error {
	if !c.enabled {
		return nil
	}

	text := "🎲 **Reviewer Roulette Results**\n\n"

	for _, sel := range selections {
		roleEmoji := "👤"
		roleName := sel.Role
		switch sel.Role {
		case "codeowner":
			roleEmoji = "👑"
			roleName = "Code Owner"
		case "team_member":
			roleEmoji = "🤝"
			roleName = "Team Member"
		case "external":
			roleEmoji = "🌍"
			roleName = "External Reviewer"
		}

		activeReviews := ""
		if sel.ActiveReviews > 0 {
			activeReviews = fmt.Sprintf(" (%d active reviews)", sel.ActiveReviews)
		}

		text += fmt.Sprintf("%s **%s**: @%s%s\n", roleEmoji, roleName, sel.Username, activeReviews)
	}

	text += fmt.Sprintf("\n[View Merge Request](%s)", mrURL)

	return c.SendSimpleMessage(text)
}

// ReviewerSelection represents a selected reviewer.
type ReviewerSelection struct {
	Username      string
	Role          string
	ActiveReviews int
	Team          string
}

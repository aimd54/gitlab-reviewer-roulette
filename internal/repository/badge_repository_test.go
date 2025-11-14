package repository

import (
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// setupBadgeTestDB creates an in-memory SQLite database for testing.
func setupBadgeTestDB(t *testing.T) *DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Enable foreign key constraints (SQLite default is off)
	db.Exec("PRAGMA foreign_keys = ON")

	// Auto-migrate tables
	err = db.AutoMigrate(
		&models.User{},
		&models.Badge{},
		&models.UserBadge{},
	)
	if err != nil {
		t.Fatalf("Failed to auto-migrate tables: %v", err)
	}

	return &DB{db}
}

// createTestBadge creates a test badge in the database.
func createTestBadge(t *testing.T, repo *BadgeRepository, name, description, icon string) *models.Badge {
	t.Helper()

	badge := &models.Badge{
		Name:        name,
		Description: description,
		Icon:        icon,
		Criteria:    json.RawMessage(`{"metric":"completed_reviews","operator":">=","value":10}`),
	}

	err := repo.Create(badge)
	if err != nil {
		t.Fatalf("Failed to create test badge: %v", err)
	}

	return badge
}

// createTestUser creates a test user in the database.
func createTestUser(t *testing.T, db *DB, username, team string) *models.User {
	t.Helper()

	user := &models.User{
		GitLabID: int(time.Now().UnixNano()), // Unique ID
		Username: username,
		Email:    username + "@example.com",
		Team:     team,
		Role:     "developer",
	}

	err := db.Create(user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

func TestBadgeRepository_Create(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	badge := &models.Badge{
		Name:        "speed_demon",
		Description: "Fast reviewer",
		Icon:        "⚡",
		Criteria:    json.RawMessage(`{"metric":"avg_ttfr","operator":"<","value":120}`),
	}

	err := repo.Create(badge)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if badge.ID == 0 {
		t.Error("Expected badge ID to be set after creation")
	}

	if badge.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestBadgeRepository_GetByID(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test badge
	created := createTestBadge(t, repo, "test_badge", "Test", "🎯")

	// Retrieve by ID
	retrieved, err := repo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}

	if retrieved.Name != "test_badge" {
		t.Errorf("Expected name 'test_badge', got %q", retrieved.Name)
	}

	// Test non-existent ID
	_, err = repo.GetByID(999)
	if err == nil {
		t.Error("Expected error for non-existent badge ID")
	}
}

func TestBadgeRepository_GetByName(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test badge
	createTestBadge(t, repo, "speed_demon", "Fast reviewer", "⚡")

	// Retrieve by name
	badge, err := repo.GetByName("speed_demon")
	if err != nil {
		t.Fatalf("GetByName() failed: %v", err)
	}

	if badge.Description != "Fast reviewer" {
		t.Errorf("Expected description 'Fast reviewer', got %q", badge.Description)
	}

	// Test non-existent name
	_, err = repo.GetByName("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent badge name")
	}
}

func TestBadgeRepository_GetAll(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create multiple badges
	createTestBadge(t, repo, "badge1", "First", "1️⃣")
	createTestBadge(t, repo, "badge2", "Second", "2️⃣")
	createTestBadge(t, repo, "badge3", "Third", "3️⃣")

	badges, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}

	if len(badges) != 3 {
		t.Errorf("Expected 3 badges, got %d", len(badges))
	}

	// Verify order (should be by created_at ASC)
	if badges[0].Name != "badge1" {
		t.Errorf("Expected first badge to be 'badge1', got %q", badges[0].Name)
	}
}

func TestBadgeRepository_Update(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create badge
	badge := createTestBadge(t, repo, "test_badge", "Original", "🔵")

	// Update
	badge.Description = "Updated description"
	badge.Icon = "🟢"

	err := repo.Update(badge)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(badge.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated badge: %v", err)
	}

	if retrieved.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got %q", retrieved.Description)
	}

	if retrieved.Icon != "🟢" {
		t.Errorf("Expected icon '🟢', got %q", retrieved.Icon)
	}
}

func TestBadgeRepository_Delete(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create badge
	badge := createTestBadge(t, repo, "test_badge", "Test", "🗑️")

	// Delete
	err := repo.Delete(badge.ID)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(badge.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted badge")
	}
}

func TestBadgeRepository_AwardBadge(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user := createTestUser(t, db, "alice", "team-frontend")
	badge := createTestBadge(t, repo, "test_badge", "Test", "🏆")

	// Award badge
	err := repo.AwardBadge(user.ID, badge.ID)
	if err != nil {
		t.Fatalf("AwardBadge() failed: %v", err)
	}

	// Verify awarded
	hasEarned, err := repo.HasUserEarnedBadge(user.ID, badge.ID)
	if err != nil {
		t.Fatalf("HasUserEarnedBadge() failed: %v", err)
	}

	if !hasEarned {
		t.Error("Expected user to have earned the badge")
	}
}

func TestBadgeRepository_AwardBadge_Idempotent(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user := createTestUser(t, db, "bob", "team-backend")
	badge := createTestBadge(t, repo, "test_badge", "Test", "🏆")

	// Award badge twice
	err := repo.AwardBadge(user.ID, badge.ID)
	if err != nil {
		t.Fatalf("First AwardBadge() failed: %v", err)
	}

	err = repo.AwardBadge(user.ID, badge.ID)
	if err != nil {
		t.Fatalf("Second AwardBadge() failed: %v", err)
	}

	// Verify only one entry exists
	userBadges, err := repo.GetUserBadges(user.ID)
	if err != nil {
		t.Fatalf("GetUserBadges() failed: %v", err)
	}

	if len(userBadges) != 1 {
		t.Errorf("Expected 1 user badge entry, got %d", len(userBadges))
	}
}

func TestBadgeRepository_GetUserBadges(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user := createTestUser(t, db, "charlie", "team-ops")
	badge1 := createTestBadge(t, repo, "badge1", "First", "1️⃣")
	badge2 := createTestBadge(t, repo, "badge2", "Second", "2️⃣")

	// Award badges
	_ = repo.AwardBadge(user.ID, badge1.ID)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	_ = repo.AwardBadge(user.ID, badge2.ID)

	// Get user badges
	userBadges, err := repo.GetUserBadges(user.ID)
	if err != nil {
		t.Fatalf("GetUserBadges() failed: %v", err)
	}

	if len(userBadges) != 2 {
		t.Errorf("Expected 2 badges, got %d", len(userBadges))
	}

	// Verify order (DESC by earned_at, so badge2 should be first)
	if userBadges[0].Badge.Name != "badge2" {
		t.Errorf("Expected first badge to be 'badge2', got %q", userBadges[0].Badge.Name)
	}

	// Verify relationships preloaded
	if userBadges[0].User.Username != "charlie" {
		t.Errorf("Expected user username 'charlie', got %q", userBadges[0].User.Username)
	}
}

func TestBadgeRepository_HasUserEarnedBadge(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user := createTestUser(t, db, "dave", "team-frontend")
	badge := createTestBadge(t, repo, "test_badge", "Test", "🎯")

	// Check before awarding
	hasEarned, err := repo.HasUserEarnedBadge(user.ID, badge.ID)
	if err != nil {
		t.Fatalf("HasUserEarnedBadge() failed: %v", err)
	}

	if hasEarned {
		t.Error("Expected user to not have earned the badge yet")
	}

	// Award badge
	_ = repo.AwardBadge(user.ID, badge.ID)

	// Check after awarding
	hasEarned, err = repo.HasUserEarnedBadge(user.ID, badge.ID)
	if err != nil {
		t.Fatalf("HasUserEarnedBadge() after award failed: %v", err)
	}

	if !hasEarned {
		t.Error("Expected user to have earned the badge")
	}
}

func TestBadgeRepository_GetUsersWithBadge(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user1 := createTestUser(t, db, "alice", "team-frontend")
	user2 := createTestUser(t, db, "bob", "team-backend")
	user3 := createTestUser(t, db, "charlie", "team-ops")
	badge := createTestBadge(t, repo, "test_badge", "Test", "🏅")

	// Award badge to user1 and user3
	_ = repo.AwardBadge(user1.ID, badge.ID)
	_ = repo.AwardBadge(user3.ID, badge.ID)

	// Get users with badge
	users, err := repo.GetUsersWithBadge(badge.ID)
	if err != nil {
		t.Fatalf("GetUsersWithBadge() failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// Verify user2 is not in the list
	for _, user := range users {
		if user.ID == user2.ID {
			t.Error("Expected user2 to not have the badge")
		}
	}
}

func TestBadgeRepository_GetBadgeHoldersCount(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user1 := createTestUser(t, db, "alice", "team-frontend")
	user2 := createTestUser(t, db, "bob", "team-backend")
	badge := createTestBadge(t, repo, "test_badge", "Test", "📊")

	// Check count before awarding
	count, err := repo.GetBadgeHoldersCount(badge.ID)
	if err != nil {
		t.Fatalf("GetBadgeHoldersCount() failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Award badge to users
	_ = repo.AwardBadge(user1.ID, badge.ID)
	_ = repo.AwardBadge(user2.ID, badge.ID)

	// Check count after awarding
	count, err = repo.GetBadgeHoldersCount(badge.ID)
	if err != nil {
		t.Fatalf("GetBadgeHoldersCount() after awards failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestBadgeRepository_RevokeUserBadge(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user := createTestUser(t, db, "eve", "team-ops")
	badge := createTestBadge(t, repo, "test_badge", "Test", "❌")

	// Award badge
	_ = repo.AwardBadge(user.ID, badge.ID)

	// Verify awarded
	hasEarned, _ := repo.HasUserEarnedBadge(user.ID, badge.ID)
	if !hasEarned {
		t.Fatal("Expected badge to be awarded before revocation")
	}

	// Revoke badge
	err := repo.RevokeUserBadge(user.ID, badge.ID)
	if err != nil {
		t.Fatalf("RevokeUserBadge() failed: %v", err)
	}

	// Verify revoked
	hasEarned, _ = repo.HasUserEarnedBadge(user.ID, badge.ID)
	if hasEarned {
		t.Error("Expected badge to be revoked")
	}
}

func TestBadgeRepository_GetUserBadgeCount(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user := createTestUser(t, db, "frank", "team-frontend")
	badge1 := createTestBadge(t, repo, "badge1", "First", "1️⃣")
	badge2 := createTestBadge(t, repo, "badge2", "Second", "2️⃣")
	badge3 := createTestBadge(t, repo, "badge3", "Third", "3️⃣")

	// Check count before awarding
	count, err := repo.GetUserBadgeCount(user.ID)
	if err != nil {
		t.Fatalf("GetUserBadgeCount() failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Award badges
	_ = repo.AwardBadge(user.ID, badge1.ID)
	_ = repo.AwardBadge(user.ID, badge2.ID)
	_ = repo.AwardBadge(user.ID, badge3.ID)

	// Check count after awarding
	count, err = repo.GetUserBadgeCount(user.ID)
	if err != nil {
		t.Fatalf("GetUserBadgeCount() after awards failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestBadgeRepository_GetRecentlyAwardedBadges(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create test data
	user1 := createTestUser(t, db, "grace", "team-backend")
	user2 := createTestUser(t, db, "heidi", "team-ops")
	badge := createTestBadge(t, repo, "test_badge", "Test", "🆕")

	// Award badge to user1 (should be in results)
	_ = repo.AwardBadge(user1.ID, badge.ID)

	// Award badge to user2 with manual timestamp in the past (simulate old award)
	userBadge := &models.UserBadge{
		UserID:   user2.ID,
		BadgeID:  badge.ID,
		EarnedAt: time.Now().Add(-48 * time.Hour), // 2 days ago
	}
	_ = db.Create(userBadge)

	// Get recently awarded badges (last 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	recentBadges, err := repo.GetRecentlyAwardedBadges(since)
	if err != nil {
		t.Fatalf("GetRecentlyAwardedBadges() failed: %v", err)
	}

	// Should only get user1's badge
	if len(recentBadges) != 1 {
		t.Errorf("Expected 1 recently awarded badge, got %d", len(recentBadges))
	}

	if len(recentBadges) > 0 && recentBadges[0].User.Username != "grace" {
		t.Errorf("Expected user 'grace', got %q", recentBadges[0].User.Username)
	}
}

func TestBadgeRepository_ForeignKeyConstraints(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Try to award badge with non-existent user
	err := repo.AwardBadge(999, 1)
	if err == nil {
		t.Error("Expected error when awarding badge to non-existent user")
	}

	// Try to award non-existent badge
	user := createTestUser(t, db, "test_user", "team-test")
	err = repo.AwardBadge(user.ID, 999)
	if err == nil {
		t.Error("Expected error when awarding non-existent badge")
	}
}

func TestBadgeRepository_UniqueConstraint(t *testing.T) {
	db := setupBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	// Create first badge
	badge1 := &models.Badge{
		Name:        "unique_badge",
		Description: "First",
		Icon:        "🔒",
		Criteria:    json.RawMessage(`{}`),
	}
	err := repo.Create(badge1)
	if err != nil {
		t.Fatalf("Failed to create first badge: %v", err)
	}

	// Try to create badge with same name
	badge2 := &models.Badge{
		Name:        "unique_badge", // Duplicate name
		Description: "Second",
		Icon:        "🔓",
		Criteria:    json.RawMessage(`{}`),
	}
	err = repo.Create(badge2)
	if err == nil {
		t.Error("Expected error when creating badge with duplicate name")
	}
}

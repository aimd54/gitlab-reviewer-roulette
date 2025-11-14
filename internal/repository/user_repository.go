package repository

import (
	"fmt"
	"time"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
)

// UserRepository handles user-related database operations.
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user.
func (r *UserRepository) Create(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByGitLabID retrieves a user by GitLab ID.
func (r *UserRepository) GetByGitLabID(gitlabID int) (*models.User, error) {
	var user models.User
	if err := r.db.Where("gitlab_id = ?", gitlabID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to get user by gitlab_id %d: %w", gitlabID, err)
	}
	return &user, nil
}

// GetByUsername retrieves a user by username.
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to get user by username %s: %w", username, err)
	}
	return &user, nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get user by id %d: %w", id, err)
	}
	return &user, nil
}

// Update updates a user.
func (r *UserRepository) Update(user *models.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// List retrieves all users with optional filters.
func (r *UserRepository) List(team, role string) ([]models.User, error) {
	query := r.db.Model(&models.User{})

	if team != "" {
		query = query.Where("team = ?", team)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// GetByTeam retrieves all users in a team.
func (r *UserRepository) GetByTeam(team string) ([]models.User, error) {
	return r.List(team, "")
}

// GetByRole retrieves all users with a specific role.
func (r *UserRepository) GetByRole(role string) ([]models.User, error) {
	return r.List("", role)
}

// GetByTeamAndRole retrieves users by team and role.
func (r *UserRepository) GetByTeamAndRole(team, role string) ([]models.User, error) {
	return r.List(team, role)
}

// CreateOrUpdate creates a user if it doesn't exist, or updates if it does.
// It first checks by gitlab_id, then falls back to username lookup.
// This handles cases where a user was created with gitlab_id=0.
func (r *UserRepository) CreateOrUpdate(user *models.User) error {
	var existing models.User

	// First try to find by GitLab ID
	err := r.db.Where("gitlab_id = ?", user.GitLabID).First(&existing).Error
	if err == nil {
		// User exists by GitLab ID, update it
		existing.Username = user.Username
		existing.Email = user.Email
		existing.Role = user.Role
		existing.Team = user.Team
		existing.UpdatedAt = time.Now()
		return r.Update(&existing)
	}

	// GitLab ID not found, try by username (handles gitlab_id=0 case)
	err = r.db.Where("username = ?", user.Username).First(&existing).Error
	if err == nil {
		// User exists by username, update including GitLab ID
		existing.GitLabID = user.GitLabID
		existing.Email = user.Email
		existing.Role = user.Role
		existing.Team = user.Team
		existing.UpdatedAt = time.Now()
		return r.Update(&existing)
	}

	// User doesn't exist, create it
	return r.Create(user)
}

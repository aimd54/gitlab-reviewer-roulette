package mocks

import "github.com/aimd54/gitlab-reviewer-roulette/internal/models"

// MockUserRepository is a simple mock for user repository
type MockUserRepository struct {
	GetByUsernameFunc func(username string) (*models.User, error)
	GetByGitLabIDFunc func(gitlabID int) (*models.User, error)
	GetByTeamFunc     func(team string) ([]models.User, error)
	GetAllFunc        func() ([]models.User, error)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	if m.GetByUsernameFunc != nil {
		return m.GetByUsernameFunc(username)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByGitLabID(gitlabID int) (*models.User, error) {
	if m.GetByGitLabIDFunc != nil {
		return m.GetByGitLabIDFunc(gitlabID)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByTeam(team string) ([]models.User, error) {
	if m.GetByTeamFunc != nil {
		return m.GetByTeamFunc(team)
	}
	return []models.User{}, nil
}

func (m *MockUserRepository) GetAll() ([]models.User, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []models.User{}, nil
}

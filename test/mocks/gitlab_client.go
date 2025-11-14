package mocks

import "github.com/aimd54/gitlab-reviewer-roulette/internal/gitlab"

// MockGitLabClient is a simple mock for GitLab client
type MockGitLabClient struct {
	GetCodeownersFunc     func(projectID int, ref string) (string, error)
	GetUserStatusFunc     func(userID int) (*gitlab.UserStatus, error)
	PostCommentFunc       func(projectID, mrIID int, comment string) (int, error)
	UpdateCommentFunc     func(projectID, mrIID, noteID int, comment string) error
	GetMRChangedFilesFunc func(projectID, mrIID int) ([]string, error)
}

func (m *MockGitLabClient) GetCodeowners(projectID int, ref string) (string, error) {
	if m.GetCodeownersFunc != nil {
		return m.GetCodeownersFunc(projectID, ref)
	}
	return "", nil
}

func (m *MockGitLabClient) GetUserStatus(userID int) (*gitlab.UserStatus, error) {
	if m.GetUserStatusFunc != nil {
		return m.GetUserStatusFunc(userID)
	}
	return nil, nil
}

func (m *MockGitLabClient) PostComment(projectID, mrIID int, comment string) (int, error) {
	if m.PostCommentFunc != nil {
		return m.PostCommentFunc(projectID, mrIID, comment)
	}
	return 1, nil // Return a dummy note ID
}

func (m *MockGitLabClient) UpdateComment(projectID, mrIID, noteID int, comment string) error {
	if m.UpdateCommentFunc != nil {
		return m.UpdateCommentFunc(projectID, mrIID, noteID, comment)
	}
	return nil
}

func (m *MockGitLabClient) GetMRChangedFiles(projectID, mrIID int) ([]string, error) {
	if m.GetMRChangedFilesFunc != nil {
		return m.GetMRChangedFilesFunc(projectID, mrIID)
	}
	return []string{}, nil
}

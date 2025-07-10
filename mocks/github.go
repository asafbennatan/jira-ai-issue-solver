package mocks

import (
	"jira-ai-issue-solver/models"
)

// MockGitHubService is a mock implementation of the GitHubService interface
type MockGitHubService struct {
	CloneRepositoryFunc      func(repoURL, directory string) error
	CreateBranchFunc         func(directory, branchName string) error
	CommitChangesFunc        func(directory, message string) error
	PushChangesFunc          func(directory, branchName string) error
	CreatePullRequestFunc    func(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error)
	ForkRepositoryFunc       func(owner, repo string) (string, error)
	CheckForkExistsFunc      func(owner, repo string) (exists bool, cloneURL string, err error)
	ResetForkFunc            func(forkCloneURL, directory string) error
	SyncForkWithUpstreamFunc func(owner, repo string) error
	SwitchToTargetBranchFunc func(directory string) error
}

// CloneRepository is the mock implementation of GitHubService's CloneRepository method
func (m *MockGitHubService) CloneRepository(repoURL, directory string) error {
	if m.CloneRepositoryFunc != nil {
		return m.CloneRepositoryFunc(repoURL, directory)
	}
	return nil
}

// CreateBranch is the mock implementation of GitHubService's CreateBranch method
func (m *MockGitHubService) CreateBranch(directory, branchName string) error {
	if m.CreateBranchFunc != nil {
		return m.CreateBranchFunc(directory, branchName)
	}
	return nil
}

// CommitChanges is the mock implementation of GitHubService's CommitChanges method
func (m *MockGitHubService) CommitChanges(directory, message string) error {
	if m.CommitChangesFunc != nil {
		return m.CommitChangesFunc(directory, message)
	}
	return nil
}

// PushChanges is the mock implementation of GitHubService's PushChanges method
func (m *MockGitHubService) PushChanges(directory, branchName string) error {
	if m.PushChangesFunc != nil {
		return m.PushChangesFunc(directory, branchName)
	}
	return nil
}

// CreatePullRequest is the mock implementation of GitHubService's CreatePullRequest method
func (m *MockGitHubService) CreatePullRequest(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
	if m.CreatePullRequestFunc != nil {
		return m.CreatePullRequestFunc(owner, repo, title, body, head, base)
	}
	return nil, nil
}

// ForkRepository is the mock implementation of GitHubService's ForkRepository method
func (m *MockGitHubService) ForkRepository(owner, repo string) (string, error) {
	if m.ForkRepositoryFunc != nil {
		return m.ForkRepositoryFunc(owner, repo)
	}
	return "", nil
}

// CheckForkExists is the mock implementation of GitHubService's CheckForkExists method
func (m *MockGitHubService) CheckForkExists(owner, repo string) (exists bool, cloneURL string, err error) {
	if m.CheckForkExistsFunc != nil {
		return m.CheckForkExistsFunc(owner, repo)
	}
	return false, "", nil
}

// ResetFork is the mock implementation of GitHubService's ResetFork method
func (m *MockGitHubService) ResetFork(forkCloneURL, directory string) error {
	if m.ResetForkFunc != nil {
		return m.ResetForkFunc(forkCloneURL, directory)
	}
	return nil
}

// SyncForkWithUpstream is the mock implementation of GitHubService's SyncForkWithUpstream method
func (m *MockGitHubService) SyncForkWithUpstream(owner, repo string) error {
	if m.SyncForkWithUpstreamFunc != nil {
		return m.SyncForkWithUpstreamFunc(owner, repo)
	}
	return nil
}

// SwitchToTargetBranch is the mock implementation of GitHubService's SwitchToTargetBranch method
func (m *MockGitHubService) SwitchToTargetBranch(directory string) error {
	if m.SwitchToTargetBranchFunc != nil {
		return m.SwitchToTargetBranchFunc(directory)
	}
	return nil
}

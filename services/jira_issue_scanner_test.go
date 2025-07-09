package services

import (
	"testing"
	"time"

	"jira-ai-issue-solver/models"
)

// Mock services for testing
type mockJiraService struct{}
type mockGitHubService struct{}
type mockClaudeService struct{}

func (m *mockJiraService) GetTicket(key string) (*models.JiraTicketResponse, error) { return nil, nil }
func (m *mockJiraService) UpdateTicketLabels(key string, addLabels, removeLabels []string) error {
	return nil
}
func (m *mockJiraService) UpdateTicketStatus(key string, status string) error { return nil }
func (m *mockJiraService) AddComment(key string, comment string) error        { return nil }
func (m *mockJiraService) SearchTickets(jql string) (*models.JiraSearchResponse, error) {
	return &models.JiraSearchResponse{
		Total:  0,
		Issues: []models.JiraIssue{},
	}, nil
}

func (m *mockGitHubService) CloneRepository(repoURL, directory string) error { return nil }
func (m *mockGitHubService) CreateBranch(directory, branchName string) error { return nil }
func (m *mockGitHubService) CommitChanges(directory, message string) error   { return nil }
func (m *mockGitHubService) PushChanges(directory, branchName string) error  { return nil }
func (m *mockGitHubService) CreatePullRequest(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
	return nil, nil
}
func (m *mockGitHubService) ForkRepository(owner, repo string) (string, error) { return "", nil }
func (m *mockGitHubService) CheckForkExists(owner, repo string) (exists bool, cloneURL string, err error) {
	return false, "", nil
}
func (m *mockGitHubService) ResetFork(forkCloneURL, directory string) error { return nil }
func (m *mockGitHubService) SyncForkWithUpstream(owner, repo string) error  { return nil }

func (m *mockClaudeService) GenerateCode(prompt string, repoDir string) (interface{}, error) {
	return nil, nil
}

func (m *mockClaudeService) GenerateDocumentation(repoDir string) error {
	return nil
}

func TestJiraIssueScannerService_StartStop(t *testing.T) {
	// Create mock services
	mockJiraService := &mockJiraService{}
	mockGitHubService := &mockGitHubService{}
	mockClaudeService := &mockClaudeService{}

	// Create config with short interval for testing
	config := &models.Config{}
	config.Jira.IntervalSeconds = 1 // 1 second for testing
	config.TempDir = "/tmp/test"

	// Create scanner service
	scanner := NewJiraIssueScannerService(mockJiraService, mockGitHubService, mockClaudeService, config)

	// Start the scanner
	scanner.Start()

	// Wait a bit to ensure it starts
	time.Sleep(100 * time.Millisecond)

	// Stop the scanner
	scanner.Stop()

	// Wait a bit to ensure it stops
	time.Sleep(100 * time.Millisecond)
}

func TestJiraIssueScannerService_ScanForTickets(t *testing.T) {
	// Create mock services
	mockJiraService := &mockJiraService{}
	mockGitHubService := &mockGitHubService{}
	mockClaudeService := &mockClaudeService{}

	// Create config
	config := &models.Config{}
	config.Jira.IntervalSeconds = 300
	config.Jira.StatusTransitions.Todo = "To Do"
	config.TempDir = "/tmp/test"

	// Create scanner service
	scanner := &JiraIssueScannerServiceImpl{
		jiraService:   mockJiraService,
		githubService: mockGitHubService,
		aiService:     mockClaudeService,
		config:        config,
	}

	// Test scanning for tickets
	scanner.scanForTickets()
}

// Note: The JQL query now only filters by assignee and status for simpler logic.

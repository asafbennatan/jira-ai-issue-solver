package services

import (
	"testing"
	"time"

	"jira-ai-issue-solver/mocks"
	"jira-ai-issue-solver/models"

	"go.uber.org/zap"
)

func TestJiraIssueScannerService_StartStop(t *testing.T) {
	// Create test logger
	logger := zap.NewNop()

	// Create mock services with stubbed methods
	mockJiraService := &mocks.MockJiraService{
		SearchTicketsFunc: func(jql string) (*models.JiraSearchResponse, error) {
			return &models.JiraSearchResponse{
				Total:  0,
				Issues: []models.JiraIssue{},
			}, nil
		},
	}
	mockGitHubService := &mocks.MockGitHubService{}
	mockClaudeService := &mocks.MockClaudeService{}

	// Create config with short interval for testing
	config := &models.Config{}
	config.Jira.IntervalSeconds = 1 // 1 second for testing
	config.TempDir = "/tmp/test"

	// Create scanner service
	scanner := NewJiraIssueScannerService(mockJiraService, mockGitHubService, mockClaudeService, config, logger)

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
	// Create test logger
	logger := zap.NewNop()

	// Create mock services with stubbed methods
	mockJiraService := &mocks.MockJiraService{
		SearchTicketsFunc: func(jql string) (*models.JiraSearchResponse, error) {
			return &models.JiraSearchResponse{
				Total:  1,
				Issues: []models.JiraIssue{{Key: "TEST-1"}},
			}, nil
		},
		GetTicketFunc: func(key string) (*models.JiraTicketResponse, error) {
			return &models.JiraTicketResponse{
				Key: key,
				Fields: models.JiraFields{
					Summary:     "Test ticket",
					Description: "Test description",
					Components:  []models.JiraComponent{{ID: "1", Name: "frontend"}},
				},
			}, nil
		},
		UpdateTicketLabelsFunc: func(key string, addLabels, removeLabels []string) error {
			return nil
		},
		UpdateTicketStatusFunc: func(key string, status string) error {
			return nil
		},
		AddCommentFunc: func(key string, comment string) error {
			return nil
		},
		GetFieldIDByNameFunc: func(fieldName string) (string, error) {
			return "customfield_10001", nil
		},
	}
	mockGitHubService := &mocks.MockGitHubService{
		CreatePullRequestFunc: func(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
			return &models.GitHubCreatePRResponse{
				ID:      1,
				Number:  1,
				State:   "open",
				Title:   title,
				Body:    body,
				HTMLURL: "https://github.com/example/repo/pull/1",
			}, nil
		},
		ForkRepositoryFunc: func(owner, repo string) (string, error) {
			return "https://github.com/mockuser/frontend.git", nil
		},
		CheckForkExistsFunc: func(owner, repo string) (exists bool, cloneURL string, err error) {
			return true, "https://github.com/mockuser/frontend.git", nil
		},
	}
	mockClaudeService := &mocks.MockClaudeService{
		GenerateCodeFunc: func(prompt string, repoDir string) (*models.ClaudeResponse, error) {
			return nil, nil
		},
	}

	// Create config
	config := &models.Config{}
	config.Jira.IntervalSeconds = 300
	config.Jira.StatusTransitions.Todo = "To Do"
	config.TempDir = "/tmp/test"

	// Create a mock ticket processor with a no-op ProcessTicket
	mockTicketProcessor := &mocks.MockTicketProcessor{
		ProcessTicketFunc: func(key string) error {
			return nil
		},
	}

	// Create scanner service with injected mock ticket processor
	scanner := &JiraIssueScannerServiceImpl{
		jiraService:     mockJiraService,
		githubService:   mockGitHubService,
		aiService:       mockClaudeService,
		ticketProcessor: mockTicketProcessor,
		config:          config,
		logger:          logger,
	}

	// Test scanning for tickets
	scanner.scanForTickets()
}

// Note: The JQL query now only filters by assignee and status for simpler logic.

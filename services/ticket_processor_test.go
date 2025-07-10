package services

import (
	"testing"

	"jira-ai-issue-solver/mocks"
	"jira-ai-issue-solver/models"

	"go.uber.org/zap"
)

func TestTicketProcessor_ProcessTicket(t *testing.T) {
	// Create test logger
	logger := zap.NewNop()

	// Create mock services
	mockJiraService := &mocks.MockJiraService{
		GetTicketFunc: func(key string) (*models.JiraTicketResponse, error) {
			return &models.JiraTicketResponse{
				Key: key,
				Fields: models.JiraFields{
					Summary:     "Test ticket",
					Description: "Test description",
					Components: []models.JiraComponent{
						{
							ID:   "1",
							Name: "frontend",
						},
					},
				},
			}, nil
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
	mockClaudeService := &mocks.MockClaudeService{}

	// Create config
	config := &models.Config{}
	config.Jira.IntervalSeconds = 300
	config.Jira.StatusTransitions.Todo = "To Do"
	config.Jira.StatusTransitions.InProgress = "In Progress"
	config.Jira.StatusTransitions.InReview = "In Review"
	config.ComponentToRepo = map[string]string{
		"frontend": "https://github.com/example/frontend.git",
	}
	config.TempDir = "/tmp/test"

	// Create ticket processor
	processor := NewTicketProcessor(mockJiraService, mockGitHubService, mockClaudeService, config, logger)

	// Test processing a ticket
	err := processor.ProcessTicket("TEST-123")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

func TestTicketProcessor_CreatePullRequestHeadFormat(t *testing.T) {
	// Create test logger
	logger := zap.NewNop()

	// Test that the pull request creation uses the correct head format
	config := &models.Config{}
	config.GitHub.BotUsername = "test-bot"
	config.GitHub.BotEmail = "test@example.com"
	config.GitHub.PersonalAccessToken = "test-token"
	config.GitHub.PRLabel = "ai-pr"
	config.TempDir = "/tmp"
	config.Jira.DisableErrorComments = true
	config.ComponentToRepo = map[string]string{
		"frontend": "https://github.com/example/frontend.git",
	}

	// Create mock services with captured values
	var capturedHead, capturedCommitMessage, capturedPRTitle string

	mockGitHub := &mocks.MockGitHubService{
		CommitChangesFunc: func(directory, message string) error {
			capturedCommitMessage = message
			return nil
		},
		CreatePullRequestFunc: func(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
			capturedHead = head
			capturedPRTitle = title
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
			return "https://github.com/test-bot/repo.git", nil
		},
		CheckForkExistsFunc: func(owner, repo string) (exists bool, cloneURL string, err error) {
			return true, "https://github.com/test-bot/repo.git", nil
		},
	}
	mockJira := &mocks.MockJiraService{
		GetTicketFunc: func(key string) (*models.JiraTicketResponse, error) {
			return &models.JiraTicketResponse{
				Key: key,
				Fields: models.JiraFields{
					Summary:     "Test ticket",
					Description: "Test description",
					Components: []models.JiraComponent{
						{
							ID:   "1",
							Name: "frontend",
						},
					},
				},
			}, nil
		},
		GetFieldIDByNameFunc: func(fieldName string) (string, error) {
			return "customfield_10001", nil
		},
	}
	mockAI := &mocks.MockClaudeService{}

	processor := NewTicketProcessor(mockJira, mockGitHub, mockAI, config, logger)

	// Process a ticket
	err := processor.ProcessTicket("TEST-123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the head parameter was formatted correctly
	expectedHead := "test-bot:TEST-123"
	if capturedHead != expectedHead {
		t.Errorf("Expected head to be '%s', got '%s'", expectedHead, capturedHead)
	}

	// Verify that the commit message was formatted correctly
	expectedCommitMessage := "TEST-123: Test ticket"
	if capturedCommitMessage != expectedCommitMessage {
		t.Errorf("Expected commit message to be '%s', got '%s'", expectedCommitMessage, capturedCommitMessage)
	}

	// Verify that the PR title was formatted correctly
	expectedPRTitle := "TEST-123: Test ticket"
	if capturedPRTitle != expectedPRTitle {
		t.Errorf("Expected PR title to be '%s', got '%s'", expectedPRTitle, capturedPRTitle)
	}
}

func TestTicketProcessor_ConfigurableStatusTransitions(t *testing.T) {
	// Create mock services with captured statuses
	var capturedStatuses []string

	mockJiraService := &mocks.MockJiraService{
		GetTicketFunc: func(key string) (*models.JiraTicketResponse, error) {
			return &models.JiraTicketResponse{
				Key: key,
				Fields: models.JiraFields{
					Summary:     "Test ticket",
					Description: "Test description",
					Components: []models.JiraComponent{
						{
							ID:   "1",
							Name: "frontend",
						},
					},
				},
			}, nil
		},
		UpdateTicketStatusFunc: func(key string, status string) error {
			capturedStatuses = append(capturedStatuses, status)
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
	mockClaudeService := &mocks.MockClaudeService{}

	// Create config with custom status transitions
	config := &models.Config{}
	config.Jira.IntervalSeconds = 300
	config.Jira.StatusTransitions.Todo = "To Do"
	config.Jira.StatusTransitions.InProgress = "Development"
	config.Jira.StatusTransitions.InReview = "Code Review"
	config.ComponentToRepo = map[string]string{
		"frontend": "https://github.com/example/frontend.git",
	}
	config.TempDir = "/tmp/test"

	// Create ticket processor
	processor := NewTicketProcessor(mockJiraService, mockGitHubService, mockClaudeService, config, zap.NewNop())

	// Test processing a ticket
	err := processor.ProcessTicket("TEST-123")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Verify that the correct status transitions were used
	expectedStatuses := []string{"Development", "Code Review"}
	if len(capturedStatuses) != len(expectedStatuses) {
		t.Errorf("Expected %d status updates, got %d", len(expectedStatuses), len(capturedStatuses))
	}

	for i, expectedStatus := range expectedStatuses {
		if i >= len(capturedStatuses) {
			t.Errorf("Missing status update at index %d", i)
			continue
		}
		if capturedStatuses[i] != expectedStatus {
			t.Errorf("Expected status at index %d to be '%s', got '%s'", i, expectedStatus, capturedStatuses[i])
		}
	}
}

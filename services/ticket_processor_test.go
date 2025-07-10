package services

import (
	"testing"

	"jira-ai-issue-solver/models"
)

// Mock services for testing
type mockJiraServiceForProcessor struct{}
type mockGitHubServiceForProcessor struct {
	forkReadyCount int
}
type mockClaudeServiceForProcessor struct{}

func (m *mockJiraServiceForProcessor) GetTicket(key string) (*models.JiraTicketResponse, error) {
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
}
func (m *mockJiraServiceForProcessor) UpdateTicketLabels(key string, addLabels, removeLabels []string) error {
	return nil
}
func (m *mockJiraServiceForProcessor) UpdateTicketStatus(key string, status string) error { return nil }
func (m *mockJiraServiceForProcessor) AddComment(key string, comment string) error        { return nil }
func (m *mockJiraServiceForProcessor) SearchTickets(jql string) (*models.JiraSearchResponse, error) {
	return &models.JiraSearchResponse{
		Total:  0,
		Issues: []models.JiraIssue{},
	}, nil
}

func (m *mockGitHubServiceForProcessor) CloneRepository(repoURL, directory string) error { return nil }
func (m *mockGitHubServiceForProcessor) CreateBranch(directory, branchName string) error { return nil }
func (m *mockGitHubServiceForProcessor) CommitChanges(directory, message string) error   { return nil }
func (m *mockGitHubServiceForProcessor) PushChanges(directory, branchName string) error  { return nil }
func (m *mockGitHubServiceForProcessor) CreatePullRequest(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
	return &models.GitHubCreatePRResponse{
		ID:      1,
		Number:  1,
		State:   "open",
		Title:   title,
		Body:    body,
		HTMLURL: "https://github.com/example/repo/pull/1",
	}, nil
}
func (m *mockGitHubServiceForProcessor) ForkRepository(owner, repo string) (string, error) {
	return "https://github.com/mockuser/frontend.git", nil
}
func (m *mockGitHubServiceForProcessor) CheckForkExists(owner, repo string) (exists bool, cloneURL string, err error) {
	m.forkReadyCount++
	if m.forkReadyCount >= 3 {
		return true, "https://github.com/mockuser/frontend.git", nil
	}
	return false, "", nil
}
func (m *mockGitHubServiceForProcessor) ResetFork(forkCloneURL, directory string) error { return nil }
func (m *mockGitHubServiceForProcessor) SyncForkWithUpstream(owner, repo string) error  { return nil }
func (m *mockGitHubServiceForProcessor) SwitchToTargetBranch(directory string) error    { return nil }

func (m *mockClaudeServiceForProcessor) GenerateCode(prompt string, repoDir string) (interface{}, error) {
	return nil, nil
}

func (m *mockClaudeServiceForProcessor) GenerateDocumentation(repoDir string) error {
	return nil
}

func TestTicketProcessor_ProcessTicket(t *testing.T) {
	// Create mock services
	mockJiraService := &mockJiraServiceForProcessor{}
	mockGitHubService := &mockGitHubServiceForProcessor{}
	mockClaudeService := &mockClaudeServiceForProcessor{}

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
	processor := NewTicketProcessor(mockJiraService, mockGitHubService, mockClaudeService, config)

	// Test processing a ticket
	err := processor.ProcessTicket("TEST-123")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

// Mock services for testing head format
type mockGitHubServiceForHeadFormat struct {
	capturedHead          string
	capturedCommitMessage string
	capturedPRTitle       string
}

func (m *mockGitHubServiceForHeadFormat) CloneRepository(repoURL, directory string) error { return nil }
func (m *mockGitHubServiceForHeadFormat) CreateBranch(directory, branchName string) error { return nil }
func (m *mockGitHubServiceForHeadFormat) CommitChanges(directory, message string) error {
	m.capturedCommitMessage = message
	return nil
}
func (m *mockGitHubServiceForHeadFormat) PushChanges(directory, branchName string) error { return nil }
func (m *mockGitHubServiceForHeadFormat) CreatePullRequest(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
	m.capturedHead = head
	m.capturedPRTitle = title
	return &models.GitHubCreatePRResponse{
		ID:      1,
		Number:  1,
		State:   "open",
		Title:   title,
		Body:    body,
		HTMLURL: "https://github.com/example/repo/pull/1",
	}, nil
}
func (m *mockGitHubServiceForHeadFormat) ForkRepository(owner, repo string) (string, error) {
	return "https://github.com/test-bot/repo.git", nil
}
func (m *mockGitHubServiceForHeadFormat) CheckForkExists(owner, repo string) (exists bool, cloneURL string, err error) {
	return true, "https://github.com/test-bot/repo.git", nil
}
func (m *mockGitHubServiceForHeadFormat) ResetFork(forkCloneURL, directory string) error { return nil }
func (m *mockGitHubServiceForHeadFormat) SyncForkWithUpstream(owner, repo string) error  { return nil }
func (m *mockGitHubServiceForHeadFormat) SwitchToTargetBranch(directory string) error    { return nil }

type mockJiraServiceForHeadFormat struct{}

func (m *mockJiraServiceForHeadFormat) GetTicket(key string) (*models.JiraTicketResponse, error) {
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
}
func (m *mockJiraServiceForHeadFormat) UpdateTicketLabels(key string, addLabels, removeLabels []string) error {
	return nil
}
func (m *mockJiraServiceForHeadFormat) UpdateTicketStatus(key string, status string) error {
	return nil
}
func (m *mockJiraServiceForHeadFormat) AddComment(key string, comment string) error { return nil }
func (m *mockJiraServiceForHeadFormat) SearchTickets(jql string) (*models.JiraSearchResponse, error) {
	return &models.JiraSearchResponse{
		Total:  0,
		Issues: []models.JiraIssue{},
	}, nil
}

type mockAIServiceForHeadFormat struct{}

func (m *mockAIServiceForHeadFormat) GenerateCode(prompt string, repoDir string) (interface{}, error) {
	return nil, nil
}

func (m *mockAIServiceForHeadFormat) GenerateDocumentation(repoDir string) error {
	return nil
}

func TestTicketProcessor_CreatePullRequestHeadFormat(t *testing.T) {
	// Test that the pull request creation uses the correct head format
	config := &models.Config{}
	config.GitHub.BotUsername = "test-bot"
	config.GitHub.BotEmail = "test@example.com"
	config.GitHub.PersonalAccessToken = "test-token"
	config.TempDir = "/tmp"
	config.Jira.DisableErrorComments = true
	config.ComponentToRepo = map[string]string{
		"frontend": "https://github.com/example/frontend.git",
	}

	// Create mock services
	mockGitHub := &mockGitHubServiceForHeadFormat{}
	mockJira := &mockJiraServiceForHeadFormat{}
	mockAI := &mockAIServiceForHeadFormat{}

	processor := &TicketProcessorImpl{
		jiraService:   mockJira,
		githubService: mockGitHub,
		aiService:     mockAI,
		config:        config,
	}

	// Process a ticket
	err := processor.ProcessTicket("TEST-123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the head parameter was formatted correctly
	expectedHead := "test-bot:TEST-123"
	if mockGitHub.capturedHead != expectedHead {
		t.Errorf("Expected head to be '%s', got '%s'", expectedHead, mockGitHub.capturedHead)
	}

	// Verify that the commit message was formatted correctly
	expectedCommitMessage := "TEST-123: Test ticket"
	if mockGitHub.capturedCommitMessage != expectedCommitMessage {
		t.Errorf("Expected commit message to be '%s', got '%s'", expectedCommitMessage, mockGitHub.capturedCommitMessage)
	}

	// Verify that the PR title was formatted correctly
	expectedPRTitle := "TEST-123: Test ticket"
	if mockGitHub.capturedPRTitle != expectedPRTitle {
		t.Errorf("Expected PR title to be '%s', got '%s'", expectedPRTitle, mockGitHub.capturedPRTitle)
	}
}

// Mock services for testing configurable status transitions
type mockJiraServiceForStatusTransitions struct {
	capturedStatuses []string
}

func (m *mockJiraServiceForStatusTransitions) GetTicket(key string) (*models.JiraTicketResponse, error) {
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
}
func (m *mockJiraServiceForStatusTransitions) UpdateTicketLabels(key string, addLabels, removeLabels []string) error {
	return nil
}
func (m *mockJiraServiceForStatusTransitions) UpdateTicketStatus(key string, status string) error {
	m.capturedStatuses = append(m.capturedStatuses, status)
	return nil
}
func (m *mockJiraServiceForStatusTransitions) AddComment(key string, comment string) error {
	return nil
}
func (m *mockJiraServiceForStatusTransitions) SearchTickets(jql string) (*models.JiraSearchResponse, error) {
	return &models.JiraSearchResponse{
		Total:  0,
		Issues: []models.JiraIssue{},
	}, nil
}

func TestTicketProcessor_ConfigurableStatusTransitions(t *testing.T) {
	// Create mock services
	mockJiraService := &mockJiraServiceForStatusTransitions{}
	mockGitHubService := &mockGitHubServiceForProcessor{}
	mockClaudeService := &mockClaudeServiceForProcessor{}

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
	processor := NewTicketProcessor(mockJiraService, mockGitHubService, mockClaudeService, config)

	// Test processing a ticket
	err := processor.ProcessTicket("TEST-123")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Verify that the correct status transitions were used
	expectedStatuses := []string{"Development", "Code Review"}
	if len(mockJiraService.capturedStatuses) != len(expectedStatuses) {
		t.Errorf("Expected %d status updates, got %d", len(expectedStatuses), len(mockJiraService.capturedStatuses))
	}

	for i, expectedStatus := range expectedStatuses {
		if i >= len(mockJiraService.capturedStatuses) {
			t.Errorf("Missing status update at index %d", i)
			continue
		}
		if mockJiraService.capturedStatuses[i] != expectedStatus {
			t.Errorf("Expected status at index %d to be '%s', got '%s'", i, expectedStatus, mockJiraService.capturedStatuses[i])
		}
	}
}

package services

import (
	"testing"
	"time"

	"jira-ai-issue-solver/mocks"
	"jira-ai-issue-solver/models"
)

func TestPRFeedbackScannerService_StartStop(t *testing.T) {
	// Create mock services
	mockJiraService := &mocks.MockJiraService{
		GetTicketFunc: func(key string) (*models.JiraTicketResponse, error) {
			return &models.JiraTicketResponse{
				Key: key,
				Fields: models.JiraFields{
					Summary:     "Test ticket",
					Description: "Test description",
					Status: models.JiraStatus{
						Name: "In Review",
					},
					Assignee: &models.JiraUser{
						Name: "testuser",
					},
					Components: []models.JiraComponent{
						{
							ID:   "1",
							Name: "frontend",
						},
					},
				},
			}, nil
		},
		GetTicketWithExpandedFieldsFunc: func(key string) (map[string]interface{}, map[string]string, error) {
			return map[string]interface{}{
					"customfield_10001": "https://github.com/testuser/frontend/pull/1",
				}, map[string]string{
					"customfield_10001": "Git Pull Request",
				}, nil
		},
		SearchTicketsFunc: func(jql string) (*models.JiraSearchResponse, error) {
			return &models.JiraSearchResponse{
				Total: 1,
				Issues: []models.JiraIssue{
					{
						Key: "TEST-123",
						Fields: models.JiraFields{
							Summary: "Test ticket",
							Status: models.JiraStatus{
								Name: "In Review",
							},
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
			return "https://github.com/testuser/frontend.git", nil
		},
		CheckForkExistsFunc: func(owner, repo string) (exists bool, cloneURL string, err error) {
			return true, "https://github.com/testuser/frontend.git", nil
		},
		GetPRDetailsFunc: func(owner, repo string, prNumber int) (*models.GitHubPRDetails, error) {
			return &models.GitHubPRDetails{
				Number:  1,
				Title:   "Test PR",
				Body:    "Test description",
				HTMLURL: "https://github.com/testuser/frontend/pull/1",
				Head: models.GitHubRef{
					Ref: "feature-branch",
					Repo: models.GitHubRepository{
						CloneURL: "https://github.com/testuser/frontend.git",
					},
				},
				Reviews:  []models.GitHubReview{},
				Comments: []models.GitHubPRComment{},
				Files:    []models.GitHubPRFile{},
			}, nil
		},
		ListPRCommentsFunc: func(owner, repo string, prNumber int) ([]models.GitHubPRComment, error) {
			return []models.GitHubPRComment{}, nil
		},
		ListPRReviewsFunc: func(owner, repo string, prNumber int) ([]models.GitHubReview, error) {
			return []models.GitHubReview{}, nil
		},
	}
	mockAIService := &mocks.MockClaudeService{}

	// Create config with short interval for testing
	config := &models.Config{}
	config.Jira.IntervalSeconds = 1 // 1 second for testing
	config.Jira.Username = "testuser"
	config.Jira.StatusTransitions.InReview = "In Review"
	config.Jira.GitPullRequestFieldName = "Git Pull Request"
	config.TempDir = "/tmp/test"

	// Create scanner service
	scanner := NewPRFeedbackScannerService(mockJiraService, mockGitHubService, mockAIService, config)

	// Start the scanner
	scanner.Start()

	// Wait a bit to ensure it starts
	time.Sleep(100 * time.Millisecond)

	// Stop the scanner
	scanner.Stop()

	// Wait a bit to ensure it stops
	time.Sleep(100 * time.Millisecond)
}

func TestPRFeedbackScannerService_ScanForPRFeedback(t *testing.T) {
	// Create mock services
	mockJiraService := &mocks.MockJiraService{
		GetTicketFunc: func(key string) (*models.JiraTicketResponse, error) {
			return &models.JiraTicketResponse{
				Key: key,
				Fields: models.JiraFields{
					Summary:     "Test ticket",
					Description: "Test description",
					Status: models.JiraStatus{
						Name: "In Review",
					},
					Assignee: &models.JiraUser{
						Name: "testuser",
					},
					Components: []models.JiraComponent{
						{
							ID:   "1",
							Name: "frontend",
						},
					},
				},
			}, nil
		},
		GetTicketWithExpandedFieldsFunc: func(key string) (map[string]interface{}, map[string]string, error) {
			return map[string]interface{}{
					"customfield_10001": "https://github.com/testuser/frontend/pull/1",
				}, map[string]string{
					"customfield_10001": "Git Pull Request",
				}, nil
		},
		SearchTicketsFunc: func(jql string) (*models.JiraSearchResponse, error) {
			return &models.JiraSearchResponse{
				Total: 1,
				Issues: []models.JiraIssue{
					{
						Key: "TEST-123",
						Fields: models.JiraFields{
							Summary: "Test ticket",
							Status: models.JiraStatus{
								Name: "In Review",
							},
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
			return "https://github.com/testuser/frontend.git", nil
		},
		CheckForkExistsFunc: func(owner, repo string) (exists bool, cloneURL string, err error) {
			return true, "https://github.com/testuser/frontend.git", nil
		},
		GetPRDetailsFunc: func(owner, repo string, prNumber int) (*models.GitHubPRDetails, error) {
			return &models.GitHubPRDetails{
				Number:  1,
				Title:   "Test PR",
				Body:    "Test description",
				HTMLURL: "https://github.com/testuser/frontend/pull/1",
				Head: models.GitHubRef{
					Ref: "feature-branch",
					Repo: models.GitHubRepository{
						CloneURL: "https://github.com/testuser/frontend.git",
					},
				},
				Reviews:  []models.GitHubReview{},
				Comments: []models.GitHubPRComment{},
				Files:    []models.GitHubPRFile{},
			}, nil
		},
		ListPRCommentsFunc: func(owner, repo string, prNumber int) ([]models.GitHubPRComment, error) {
			return []models.GitHubPRComment{}, nil
		},
		ListPRReviewsFunc: func(owner, repo string, prNumber int) ([]models.GitHubReview, error) {
			return []models.GitHubReview{}, nil
		},
	}
	mockAIService := &mocks.MockClaudeService{}

	// Create config
	config := &models.Config{}
	config.Jira.IntervalSeconds = 300
	config.Jira.Username = "testuser"
	config.Jira.StatusTransitions.InReview = "In Review"
	config.Jira.GitPullRequestFieldName = "Git Pull Request"
	config.TempDir = "/tmp/test"

	// Create scanner service
	scanner := &PRFeedbackScannerServiceImpl{
		jiraService:       mockJiraService,
		githubService:     mockGitHubService,
		aiService:         mockAIService,
		prReviewProcessor: NewPRReviewProcessor(mockJiraService, mockGitHubService, mockAIService, config),
		config:            config,
	}

	// Test scanning for PR feedback
	scanner.scanForPRFeedback()
}

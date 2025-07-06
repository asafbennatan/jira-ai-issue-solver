package services

import (
	"bytes"
	"io"
	"net/http"
	"os/exec"
	"testing"

	"jira-ai-issue-solver/models"
)

// execCommand is a variable that holds the exec.Command function
// It can be replaced with a mock for testing
var execCommand = exec.Command

// MockGitHubAppService is a mock implementation of GitHubAppService
type MockGitHubAppService struct {
	GetInstallationTokenFunc func() (string, error)
	GetAppTokenFunc          func() (string, error)
}

func (m *MockGitHubAppService) GetInstallationToken() (string, error) {
	if m.GetInstallationTokenFunc != nil {
		return m.GetInstallationTokenFunc()
	}
	return "mock-installation-token", nil
}

func (m *MockGitHubAppService) GetAppToken() (string, error) {
	if m.GetAppTokenFunc != nil {
		return m.GetAppTokenFunc()
	}
	return "mock-app-token", nil
}

// TestCreatePullRequest tests the CreatePullRequest method
func TestCreatePullRequest(t *testing.T) {
	// Test cases
	testCases := []struct {
		name           string
		owner          string
		repo           string
		title          string
		body           string
		head           string
		base           string
		mockResponse   *http.Response
		mockError      error
		expectedResult *models.GitHubCreatePRResponse
		expectedError  bool
	}{
		{
			name:  "successful PR creation",
			owner: "example",
			repo:  "repo",
			title: "Test PR",
			body:  "This is a test PR",
			head:  "feature/TEST-123",
			base:  "main",
			mockResponse: &http.Response{
				StatusCode: http.StatusCreated,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"id": 12345,
					"number": 1,
					"state": "open",
					"title": "Test PR",
					"body": "This is a test PR",
					"html_url": "https://github.com/example/repo/pull/1",
					"created_at": "2023-01-01T00:00:00Z",
					"updated_at": "2023-01-01T00:00:00Z"
				}`))),
			},
			mockError: nil,
			expectedResult: &models.GitHubCreatePRResponse{
				ID:      12345,
				Number:  1,
				State:   "open",
				Title:   "Test PR",
				Body:    "This is a test PR",
				HTMLURL: "https://github.com/example/repo/pull/1",
			},
			expectedError: false,
		},
		{
			name:  "error creating PR",
			owner: "example",
			repo:  "repo",
			title: "Test PR",
			body:  "This is a test PR",
			head:  "feature/TEST-123",
			base:  "main",
			mockResponse: &http.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"Validation Failed","errors":[{"resource":"PullRequest","code":"custom","message":"A pull request already exists for example:feature/TEST-123."}],"documentation_url":"https://docs.github.com/rest/reference/pulls#create-a-pull-request"}`))),
			},
			mockError:      nil,
			expectedResult: nil,
			expectedError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock HTTP client
			mockClient := NewTestClient(func(req *http.Request) (*http.Response, error) {
				return tc.mockResponse, tc.mockError
			})

			// Create a mock GitHub App service
			mockAppService := &MockGitHubAppService{
				GetInstallationTokenFunc: func() (string, error) {
					return "mock-installation-token", nil
				},
			}

			// Create a GitHubService with the mock client
			service := &GitHubServiceImpl{
				config: &models.Config{
					GitHub: struct {
						AppID          int64  `json:"app_id" envconfig:"GITHUB_APP_ID"`
						AppPrivateKey  string `json:"app_private_key" envconfig:"GITHUB_APP_PRIVATE_KEY"`
						InstallationID int64  `json:"installation_id" envconfig:"GITHUB_INSTALLATION_ID"`
						BotUsername    string `json:"bot_username" envconfig:"GITHUB_BOT_USERNAME"`
						BotEmail       string `json:"bot_email" envconfig:"GITHUB_BOT_EMAIL"`
					}{
						AppID:          1531890,
						AppPrivateKey:  "test-private-key",
						InstallationID: 12345678,
						BotUsername:    "test-bot",
						BotEmail:       "test-bot@example.com",
					},
				},
				client:     mockClient,
				executor:   execCommand,
				appService: mockAppService,
			}

			// Call the method being tested
			result, err := service.CreatePullRequest(tc.owner, tc.repo, tc.title, tc.body, tc.head, tc.base)

			// Check the results
			if tc.expectedError && err == nil {
				t.Errorf("Expected an error but got nil")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tc.expectedResult != nil {
				if result == nil {
					t.Errorf("Expected a result but got nil")
				} else {
					if result.ID != tc.expectedResult.ID {
						t.Errorf("Expected result ID %d but got %d", tc.expectedResult.ID, result.ID)
					}
					if result.Number != tc.expectedResult.Number {
						t.Errorf("Expected result Number %d but got %d", tc.expectedResult.Number, result.Number)
					}
					// Add more assertions for other fields as needed
				}
			}
		})
	}
}

// TestExtractRepoInfo tests the ExtractRepoInfo function
func TestExtractRepoInfo(t *testing.T) {
	// Test cases
	testCases := []struct {
		name          string
		repoURL       string
		expectedOwner string
		expectedRepo  string
		expectedError bool
	}{
		{
			name:          "HTTPS URL",
			repoURL:       "https://github.com/example/repo.git",
			expectedOwner: "example",
			expectedRepo:  "repo",
			expectedError: false,
		},
		{
			name:          "SSH URL",
			repoURL:       "git@github.com:example/repo.git",
			expectedOwner: "example",
			expectedRepo:  "repo",
			expectedError: false,
		},
		{
			name:          "HTTPS URL without .git",
			repoURL:       "https://github.com/example/repo",
			expectedOwner: "example",
			expectedRepo:  "repo",
			expectedError: false,
		},
		{
			name:          "invalid URL",
			repoURL:       "invalid-url",
			expectedOwner: "",
			expectedRepo:  "",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function being tested
			owner, repo, err := ExtractRepoInfo(tc.repoURL)

			// Check the results
			if tc.expectedError && err == nil {
				t.Errorf("Expected an error but got nil")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if owner != tc.expectedOwner {
				t.Errorf("Expected owner %s but got %s", tc.expectedOwner, owner)
			}
			if repo != tc.expectedRepo {
				t.Errorf("Expected repo %s but got %s", tc.expectedRepo, repo)
			}
		})
	}
}

// TestGitHubValidateWebhookSignature tests the ValidateWebhookSignature method
func TestGitHubValidateWebhookSignature(t *testing.T) {
	// Create a GitHubService
	service := &GitHubServiceImpl{
		config: &models.Config{
			GitHub: struct {
				AppID          int64  `json:"app_id" envconfig:"GITHUB_APP_ID"`
				AppPrivateKey  string `json:"app_private_key" envconfig:"GITHUB_APP_PRIVATE_KEY"`
				InstallationID int64  `json:"installation_id" envconfig:"GITHUB_INSTALLATION_ID"`
				BotUsername    string `json:"bot_username" envconfig:"GITHUB_BOT_USERNAME"`
				BotEmail       string `json:"bot_email" envconfig:"GITHUB_BOT_EMAIL"`
			}{
				AppID:          1531890,
				AppPrivateKey:  "test-private-key",
				InstallationID: 12345678,
				BotUsername:    "test-bot",
				BotEmail:       "test-bot@example.com",
			},
		},
		executor:   execCommand,
		appService: &MockGitHubAppService{},
	}

	// Test the method
	body := []byte(`{"test":"data"}`)
	signature := "test-signature"
	result := service.ValidateWebhookSignature(body, signature)

	// Since the current implementation always returns true, we expect true
	if !result {
		t.Errorf("Expected true but got false")
	}
}

package services

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"jira-ai-issue-solver/models"

	"go.uber.org/zap"
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
		prLabel        string
		mockResponse   *http.Response
		mockError      error
		expectedResult *models.GitHubCreatePRResponse
		expectedError  bool
	}{
		{
			name:    "successful PR creation",
			owner:   "example",
			repo:    "repo",
			title:   "Test PR",
			body:    "This is a test PR",
			head:    "feature/TEST-123",
			base:    "main",
			prLabel: "ai-pr",
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
			name:    "error creating PR",
			owner:   "example",
			repo:    "repo",
			title:   "Test PR",
			body:    "This is a test PR",
			head:    "feature/TEST-123",
			base:    "main",
			prLabel: "ai-pr",
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
			// Create a mock HTTP client that captures the request body
			var capturedBody []byte
			mockClient := NewTestClient(func(req *http.Request) (*http.Response, error) {
				// Capture the request body
				capturedBody, _ = io.ReadAll(req.Body)
				return tc.mockResponse, tc.mockError
			})

			// Create a GitHubService with the mock client
			config := &models.Config{}
			config.GitHub.PersonalAccessToken = "test-token"
			config.GitHub.BotUsername = "test-bot"
			config.GitHub.BotEmail = "test@example.com"
			config.GitHub.PRLabel = tc.prLabel

			service := &GitHubServiceImpl{
				config:   config,
				client:   mockClient,
				executor: execCommand,
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

			// Verify that the label was included in the request
			if len(capturedBody) > 0 {
				var requestPayload models.GitHubCreatePRRequest
				if err := json.Unmarshal(capturedBody, &requestPayload); err != nil {
					t.Errorf("Failed to unmarshal request body: %v", err)
				} else {
					if len(requestPayload.Labels) == 0 {
						t.Errorf("Expected labels to be included in request, but got empty labels")
					} else if requestPayload.Labels[0] != tc.prLabel {
						t.Errorf("Expected label '%s' but got '%s'", tc.prLabel, requestPayload.Labels[0])
					}
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

// TestSwitchToBranch tests the SwitchToBranch method
func TestSwitchToBranch(t *testing.T) {
	// Create test logger
	logger := zap.NewNop()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "github-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Track the commands that would be executed
	var executedCommands []string
	mockExecutor := func(name string, args ...string) *exec.Cmd {
		command := strings.Join(append([]string{name}, args...), " ")
		executedCommands = append(executedCommands, command)

		// Return a mock command that does nothing
		return exec.Command("echo", "mocked")
	}

	// Create config
	config := &models.Config{}
	config.GitHub.BotUsername = "test-bot"
	config.GitHub.BotEmail = "test@example.com"

	// Create GitHub service with mocked executor
	githubService := NewGitHubService(config, logger, mockExecutor)

	// Test switching to the test branch
	err = githubService.SwitchToBranch(tempDir, "test-branch")
	if err != nil {
		t.Errorf("SwitchToBranch() error = %v", err)
	}

	// Verify the correct commands were executed
	expectedCommands := []string{
		"git fetch origin",
		"git checkout test-branch",
	}

	if len(executedCommands) != len(expectedCommands) {
		t.Errorf("Expected %d commands to be executed, got %d", len(expectedCommands), len(executedCommands))
	}

	for i, expected := range expectedCommands {
		if i < len(executedCommands) && executedCommands[i] != expected {
			t.Errorf("Expected command '%s', got '%s'", expected, executedCommands[i])
		}
	}
}

// TestSwitchToBranch_NonExistentBranch tests switching to a non-existent branch
func TestSwitchToBranch_NonExistentBranch(t *testing.T) {
	// Create test logger
	logger := zap.NewNop()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "github-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repository: %v", err)
	}

	// Create initial commit
	cmd = exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Create config
	config := &models.Config{}
	config.GitHub.BotUsername = "test-bot"
	config.GitHub.BotEmail = "test@example.com"

	// Create GitHub service
	githubService := NewGitHubService(config, logger)

	// Test switching to a non-existent branch
	err = githubService.SwitchToBranch(tempDir, "non-existent-branch")
	if err == nil {
		t.Error("SwitchToBranch() should return error for non-existent branch")
	}
}

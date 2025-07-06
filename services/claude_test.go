package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"jira-ai-issue-solver/models"
)

// TestPreparePrompt tests the PreparePrompt function
func TestPreparePrompt(t *testing.T) {
	// Create a test ticket
	ticket := &models.JiraTicketResponse{
		ID:   "12345",
		Key:  "TEST-123",
		Self: "https://jira.example.com/rest/api/2/issue/12345",
		Fields: models.JiraFields{
			Summary:     "Test ticket",
			Description: "This is a test ticket",
			Status: models.JiraStatus{
				ID:   "1",
				Name: "Open",
			},
			Project: models.JiraProject{
				ID:   "10000",
				Key:  "TEST",
				Name: "Test Project",
				Properties: map[string]string{
					"ai.bot.github.repo": "https://github.com/example/repo.git",
				},
			},
			Labels: []string{"good-for-ai"},
			Comment: models.JiraComments{
				Comments: []models.JiraComment{
					{
						ID:   "comment-1",
						Body: "This is a test comment",
						Author: models.JiraUser{
							DisplayName: "Test User",
						},
						Created: time.Now(),
					},
				},
			},
		},
	}

	// Call the function being tested
	prompt := PreparePrompt(ticket)

	// Check the results
	if prompt == "" {
		t.Errorf("Expected a non-empty prompt but got an empty string")
	}

	// Check that the prompt contains the ticket summary and description
	if !bytes.Contains([]byte(prompt), []byte(ticket.Fields.Summary)) {
		t.Errorf("Expected prompt to contain the ticket summary")
	}
	if !bytes.Contains([]byte(prompt), []byte(ticket.Fields.Description)) {
		t.Errorf("Expected prompt to contain the ticket description")
	}

	// Check that the prompt contains the comment
	if !bytes.Contains([]byte(prompt), []byte(ticket.Fields.Comment.Comments[0].Body)) {
		t.Errorf("Expected prompt to contain the comment body")
	}
}

// TestGenerateCode tests the GenerateCode method
func TestGenerateCode(t *testing.T) {
	// Create a temporary directory to simulate a repo
	tempDir, err := os.MkdirTemp("", "test-repo")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository in the temporary directory
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tempDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Configure git user for the commit
	configNameCmd := exec.Command("git", "config", "user.name", "Test User")
	configNameCmd.Dir = tempDir
	if err := configNameCmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	configEmailCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configEmailCmd.Dir = tempDir
	if err := configEmailCmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	// Test case 1: Successful code generation
	t.Run("Successful code generation", func(t *testing.T) {
		// Create a mock Claude response
		expectedResponse := &ClaudeResponse{
			Type:          "completion",
			Subtype:       "text",
			IsError:       false,
			DurationMs:    1000,
			DurationApiMs: 900,
			NumTurns:      1,
			Result:        "Generated code here",
			SessionID:     "test-session",
			TotalCostUsd:  0.01,
			Usage: ClaudeUsage{
				InputTokens:  100,
				OutputTokens: 200,
				ServiceTier:  "claude-3-opus-20240229",
			},
		}
		responseJSON, _ := json.Marshal(expectedResponse)

		// Mock the Claude CLI execution
		execCommand = func(name string, args ...string) *exec.Cmd {
			// Create a mock command that returns the JSON response
			cmd := exec.Command("echo", string(responseJSON))
			return cmd
		}
		defer func() { execCommand = exec.Command }()

		// Create a test config
		config := &models.Config{}
		config.Claude.Path = "claude"
		config.Claude.Timeout = 60
		config.Claude.DangerouslySkipPermissions = true
		config.Claude.AllowedTools = "filesystem"
		config.Claude.DisallowedTools = "execution"

		// Create a Claude service
		service := NewClaudeService(config, execCommand)

		// Call the function being tested
		response, err := service.GenerateCode("Test prompt", tempDir)
		if err != nil {
			t.Fatalf("GenerateCode returned an error: %v", err)
		}

		// Compare the response with expected
		if response.Result != expectedResponse.Result {
			t.Errorf("Expected result %s, got %s", expectedResponse.Result, response.Result)
		}
		if response.Type != expectedResponse.Type {
			t.Errorf("Expected type %s, got %s", expectedResponse.Type, response.Type)
		}
		if response.IsError != expectedResponse.IsError {
			t.Errorf("Expected IsError %v, got %v", expectedResponse.IsError, response.IsError)
		}
		if response.Usage.InputTokens != expectedResponse.Usage.InputTokens {
			t.Errorf("Expected InputTokens %d, got %d", expectedResponse.Usage.InputTokens, response.Usage.InputTokens)
		}
		if response.Usage.OutputTokens != expectedResponse.Usage.OutputTokens {
			t.Errorf("Expected OutputTokens %d, got %d", expectedResponse.Usage.OutputTokens, response.Usage.OutputTokens)
		}
		if response.TotalCostUsd != expectedResponse.TotalCostUsd {
			t.Errorf("Expected TotalCostUsd %f, got %f", expectedResponse.TotalCostUsd, response.TotalCostUsd)
		}
	})

	// Test case 2: Error response from Claude
	t.Run("Error response from Claude", func(t *testing.T) {
		// Create a mock Claude error response
		errorResponse := &ClaudeResponse{
			Type:          "completion",
			Subtype:       "text",
			IsError:       true,
			DurationMs:    1000,
			DurationApiMs: 900,
			NumTurns:      1,
			Result:        "Error: something went wrong",
			SessionID:     "test-session",
			TotalCostUsd:  0.01,
			Usage: ClaudeUsage{
				InputTokens:  100,
				OutputTokens: 200,
				ServiceTier:  "claude-3-opus-20240229",
			},
		}
		responseJSON, _ := json.Marshal(errorResponse)

		// Mock the Claude CLI execution
		execCommand = func(name string, args ...string) *exec.Cmd {
			// Create a mock command that returns the error JSON response
			cmd := exec.Command("echo", string(responseJSON))
			return cmd
		}
		defer func() { execCommand = exec.Command }()

		// Create a test config
		config := &models.Config{}
		config.Claude.Path = "claude"
		config.Claude.Timeout = 60
		config.Claude.DangerouslySkipPermissions = true
		config.Claude.AllowedTools = "filesystem"
		config.Claude.DisallowedTools = "execution"

		// Create a Claude service
		service := NewClaudeService(config, execCommand)

		// Call the function being tested
		_, err := service.GenerateCode("Test prompt", tempDir)
		if err == nil {
			t.Fatalf("Expected an error but got nil")
		}
	})

	// Test case 3: Timeout
	t.Run("Timeout", func(t *testing.T) {
		// Mock the Claude CLI execution to simulate a timeout
		execCommand = func(name string, args ...string) *exec.Cmd {
			// Create a command that sleeps for 2 seconds
			cmd := exec.Command("sleep", "2")
			return cmd
		}
		defer func() { execCommand = exec.Command }()

		// Create a test config with a very short timeout
		config := &models.Config{}
		config.Claude.Path = "claude"
		config.Claude.Timeout = 1 // 1 second timeout
		config.Claude.DangerouslySkipPermissions = true
		config.Claude.AllowedTools = "filesystem"
		config.Claude.DisallowedTools = "execution"

		// Create a Claude service
		service := NewClaudeService(config, execCommand)

		// Call the function being tested
		_, err := service.GenerateCode("Test prompt", tempDir)
		if err == nil {
			t.Fatalf("Expected a timeout error but got nil")
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected a timeout error, got: %v", err)
		}
	})
}

// TestHelperProcess is not a real test. It's used as a helper process for mocking exec.Command.
// This implementation is based on the template in mocks/exec_command.go
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Check if we should sleep to simulate a long-running process
	if sleepTime := os.Getenv("SLEEP"); sleepTime != "" {
		sleepSeconds, _ := time.ParseDuration(sleepTime + "s")
		time.Sleep(sleepSeconds)
	}

	// Check if we should exit with a non-zero status
	if exitStatus := os.Getenv("EXIT_STATUS"); exitStatus != "" {
		os.Exit(1)
	}

	// Print stdout if provided
	if stdout := os.Getenv("STDOUT"); stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}

	// Print stderr if provided
	if stderr := os.Getenv("STDERR"); stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}

	os.Exit(0)
}

// helperProcess is a separate function that can be called as a helper process
func helperProcess() {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Check if we should sleep to simulate a long-running process
	if sleepTime := os.Getenv("SLEEP"); sleepTime != "" {
		sleepSeconds, _ := time.ParseDuration(sleepTime + "s")
		time.Sleep(sleepSeconds)
	}

	// Check if we should exit with a non-zero status
	if exitStatus := os.Getenv("EXIT_STATUS"); exitStatus != "" {
		os.Exit(1)
	}

	// Print stdout if provided
	if stdout := os.Getenv("STDOUT"); stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}

	// Print stderr if provided
	if stderr := os.Getenv("STDERR"); stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}

	os.Exit(0)
}

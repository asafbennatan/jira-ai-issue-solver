package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"jira-ai-issue-solver/models"
)

// ClaudeService defines the interface for interacting with Claude CLI
type ClaudeService interface {
	// GenerateCode generates code using Claude CLI
	GenerateCode(prompt string, repoDir string) (*ClaudeResponse, error)
}

// ClaudeServiceImpl implements the ClaudeService interface
type ClaudeServiceImpl struct {
	config   *models.Config
	executor models.CommandExecutor
}

// NewClaudeService creates a new ClaudeService
func NewClaudeService(config *models.Config, executor ...models.CommandExecutor) ClaudeService {
	commandExecutor := exec.Command
	if len(executor) > 0 {
		commandExecutor = executor[0]
	}
	return &ClaudeServiceImpl{
		config:   config,
		executor: commandExecutor,
	}
}

// ClaudeUsage represents the usage information in the Claude CLI response
type ClaudeUsage struct {
	InputTokens              int            `json:"input_tokens"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens"`
	OutputTokens             int            `json:"output_tokens"`
	ServerToolUse            map[string]int `json:"server_tool_use"`
	ServiceTier              string         `json:"service_tier"`
}

// ClaudeResponse represents the JSON response from Claude CLI
type ClaudeResponse struct {
	Type          string      `json:"type"`
	Subtype       string      `json:"subtype"`
	IsError       bool        `json:"is_error"`
	DurationMs    int         `json:"duration_ms"`
	DurationApiMs int         `json:"duration_api_ms"`
	NumTurns      int         `json:"num_turns"`
	Result        string      `json:"result"`
	SessionID     string      `json:"session_id"`
	TotalCostUsd  float64     `json:"total_cost_usd"`
	Usage         ClaudeUsage `json:"usage"`
}

// GenerateCode generates code using Claude CLI
func (s *ClaudeServiceImpl) GenerateCode(prompt string, repoDir string) (*ClaudeResponse, error) {
	// Build command arguments based on configuration
	args := []string{"--output-format", "json", "-p", prompt}

	// Add dangerously-skip-permissions flag if configured
	if s.config.Claude.DangerouslySkipPermissions {
		args = append([]string{"--dangerously-skip-permissions"}, args...)
	}

	// Add allowedTools if configured
	if s.config.Claude.AllowedTools != "" {
		args = append([]string{"--allowedTools", s.config.Claude.AllowedTools}, args...)
	}

	// Add disallowedTools if configured
	if s.config.Claude.DisallowedTools != "" {
		args = append([]string{"--disallowedTools", s.config.Claude.DisallowedTools}, args...)
	}

	// Prepare the command with the built arguments
	cmd := s.executor(s.config.Claude.Path, args...)
	cmd.Dir = repoDir

	// Set environment variables
	cmd.Env = os.Environ()

	// Create buffers for stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout
	timeout := time.Duration(s.config.Claude.Timeout) * time.Second

	// Create a channel to signal command completion
	done := make(chan error, 1)

	// Run the command in a goroutine
	go func() {
		err := cmd.Run()
		if err != nil {
			done <- fmt.Errorf("failed to run Claude CLI: %w, stderr: %s", err, stderr.String())
			return
		}

		done <- nil
	}()

	// Wait for the command to complete or timeout
	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
	case <-time.After(timeout):
		// Kill the process if it times out
		if err := cmd.Process.Kill(); err != nil {
			return nil, fmt.Errorf("failed to kill Claude CLI process: %w", err)
		}
		return nil, fmt.Errorf("claude CLI timed out after %d seconds", s.config.Claude.Timeout)
	}

	// Parse the JSON response
	var response ClaudeResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to parse Claude CLI response: %w, stdout: %s", err, stdout.String())
	}

	// Check if there was an error
	if response.IsError {
		return nil, fmt.Errorf("claude CLI returned an error: %s", response.Result)
	}

	return &response, nil
}

// PreparePrompt prepares a prompt for Claude CLI based on the Jira ticket
func PreparePrompt(ticket *models.JiraTicketResponse) string {
	var sb strings.Builder

	sb.WriteString("# Task\n\n")
	sb.WriteString(fmt.Sprintf("## %s\n\n", ticket.Fields.Summary))
	sb.WriteString(fmt.Sprintf("%s\n\n", ticket.Fields.Description))

	// Add comments if available
	if len(ticket.Fields.Comment.Comments) > 0 {
		sb.WriteString("## Comments\n\n")
		for _, comment := range ticket.Fields.Comment.Comments {
			sb.WriteString(fmt.Sprintf("**%s** (%s):\n%s\n\n",
				comment.Author.DisplayName,
				comment.Created.Format("2006-01-02 15:04:05"),
				comment.Body))
		}
	}

	sb.WriteString("# Instructions\n\n")
	sb.WriteString("1. Analyze the task description and comments.\n")
	sb.WriteString("2. Implement the necessary changes to fulfill the requirements.\n")
	sb.WriteString("3. Write tests for the implemented functionality if appropriate.\n")
	sb.WriteString("4. Update documentation if necessary.\n")
	sb.WriteString("5. Provide a summary of the changes made.\n\n")

	sb.WriteString("# Output Format\n\n")
	sb.WriteString("Please provide your response in the following format:\n\n")
	sb.WriteString("```\n")
	sb.WriteString("## Summary\n")
	sb.WriteString("<A brief summary of the changes made>\n\n")
	sb.WriteString("## Changes Made\n")
	sb.WriteString("<List of files modified and a description of the changes>\n\n")
	sb.WriteString("## Testing\n")
	sb.WriteString("<Description of how the changes were tested>\n")
	sb.WriteString("```\n")

	return sb.String()
}

// PreparePromptForPRFeedback prepares a prompt for Claude CLI based on PR feedback
func PreparePromptForPRFeedback(pr *models.GitHubPullRequest, review *models.GitHubReview, repoDir string) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Pull Request Feedback\n\n")
	sb.WriteString(fmt.Sprintf("## PR: %s\n\n", pr.Title))
	sb.WriteString(fmt.Sprintf("%s\n\n", pr.Body))

	sb.WriteString("## Review Feedback\n\n")
	sb.WriteString(fmt.Sprintf("**%s**:\n%s\n\n", review.User.Login, review.Body))

	// Get the diff of the PR
	cmd := exec.Command("git", "diff", "origin/main...HEAD")
	cmd.Dir = repoDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get PR diff: %w, stderr: %s", err, stderr.String())
	}

	sb.WriteString("## Current Changes\n\n")
	sb.WriteString("```diff\n")
	sb.WriteString(stdout.String())
	sb.WriteString("\n```\n\n")

	sb.WriteString("# Instructions\n\n")
	sb.WriteString("1. Analyze the PR feedback and the current changes.\n")
	sb.WriteString("2. Implement the necessary changes to address the feedback.\n")
	sb.WriteString("3. Update tests if necessary.\n")
	sb.WriteString("4. Update documentation if necessary.\n")
	sb.WriteString("5. Provide a summary of the changes made.\n\n")

	sb.WriteString("# Output Format\n\n")
	sb.WriteString("Please provide your response in the following format:\n\n")
	sb.WriteString("```\n")
	sb.WriteString("## Summary\n")
	sb.WriteString("<A brief summary of the changes made>\n\n")
	sb.WriteString("## Changes Made\n")
	sb.WriteString("<List of files modified and a description of the changes>\n\n")
	sb.WriteString("## Feedback Addressed\n")
	sb.WriteString("<Description of how the feedback was addressed>\n")
	sb.WriteString("```\n")

	return sb.String(), nil
}

// GetChangedFiles gets a list of files changed in the current branch
func GetChangedFiles(repoDir string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "origin/main...HEAD")
	cmd.Dir = repoDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w, stderr: %s", err, stderr.String())
	}

	files := strings.Split(strings.TrimSpace(stdout.String()), "\n")

	// Filter out empty strings
	var result []string
	for _, file := range files {
		if file != "" {
			// Get the absolute path
			absPath := filepath.Join(repoDir, file)
			result = append(result, absPath)
		}
	}

	return result, nil
}

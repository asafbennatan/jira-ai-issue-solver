package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"jira-ai-issue-solver/models"
	"jira-ai-issue-solver/services"
)

// TestJiraIntegration tests the Jira integration
func TestJiraIntegration(config *models.Config) error {
	log.Println("Testing Jira integration...")

	jiraService := services.NewJiraService(config)

	// Test getting a ticket
	ticketKey := os.Getenv("TEST_JIRA_TICKET_KEY")
	if ticketKey == "" {
		return fmt.Errorf("TEST_JIRA_TICKET_KEY environment variable not set")
	}

	ticket, err := jiraService.GetTicket(ticketKey)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	log.Printf("Successfully retrieved ticket %s: %s", ticketKey, ticket.Fields.Summary)

	// Test adding a comment
	if os.Getenv("TEST_ADD_COMMENT") == "true" {
		err = jiraService.AddComment(ticketKey, "Test comment from Jira AI Issue Solver")
		if err != nil {
			return fmt.Errorf("failed to add comment: %w", err)
		}

		log.Printf("Successfully added comment to ticket %s", ticketKey)
	}

	// Test webhook registration
	if os.Getenv("TEST_REGISTER_WEBHOOK") == "true" {
		serverURL := os.Getenv("TEST_SERVER_URL")
		if serverURL == "" {
			return fmt.Errorf("TEST_SERVER_URL environment variable not set")
		}

		err = jiraService.RegisterOrRefreshWebhook(serverURL)
		if err != nil {
			return fmt.Errorf("failed to register webhook: %w", err)
		}

		log.Printf("Successfully registered webhook for %s", serverURL)
	}

	return nil
}

// TestGitHubIntegration tests the GitHub integration
func TestGitHubIntegration(config *models.Config) error {
	log.Println("Testing GitHub integration...")

	githubService := services.NewGitHubService(config)

	// Test cloning a repository
	repoURL := os.Getenv("TEST_GITHUB_REPO_URL")
	if repoURL == "" {
		return fmt.Errorf("TEST_GITHUB_REPO_URL environment variable not set")
	}

	repoDir := filepath.Join(config.TempDir, "test-repo")

	err := githubService.CloneRepository(repoURL, repoDir)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Printf("Successfully cloned repository to %s", repoDir)

	// Test creating a branch
	if os.Getenv("TEST_CREATE_BRANCH") == "true" {
		branchName := "test-branch"

		err = githubService.CreateBranch(repoDir, branchName)
		if err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}

		log.Printf("Successfully created branch %s", branchName)

		// Create a test file
		testFile := filepath.Join(repoDir, "test.txt")
		err = os.WriteFile(testFile, []byte("Test file created by Jira AI Issue Solver"), 0644)
		if err != nil {
			return fmt.Errorf("failed to create test file: %w", err)
		}

		// Commit the changes
		err = githubService.CommitChanges(repoDir, "Test commit")
		if err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}

		log.Printf("Successfully committed changes")

		// Push the changes
		if os.Getenv("TEST_PUSH_CHANGES") == "true" {
			err = githubService.PushChanges(repoDir, branchName)
			if err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}

			log.Printf("Successfully pushed changes")

			// Create a pull request
			if os.Getenv("TEST_CREATE_PR") == "true" {
				owner, repo, err := services.ExtractRepoInfo(repoURL)
				if err != nil {
					return fmt.Errorf("failed to extract repo info: %w", err)
				}

				pr, err := githubService.CreatePullRequest(
					owner,
					repo,
					"Test PR",
					"This is a test PR created by Jira AI Issue Solver",
					branchName,
					"main",
				)
				if err != nil {
					return fmt.Errorf("failed to create pull request: %w", err)
				}

				log.Printf("Successfully created pull request: %s", pr.HTMLURL)
			}
		}
	}

	return nil
}

// TestClaudeIntegration tests the Claude CLI integration
func TestClaudeIntegration(config *models.Config) error {
	log.Println("Testing Claude CLI integration...")

	// Check for Claude CLI configuration options in environment variables
	if dangerouslySkipPermissions := os.Getenv("TEST_CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS"); dangerouslySkipPermissions == "true" {
		config.Claude.DangerouslySkipPermissions = true
		log.Println("Using --dangerously-skip-permissions flag")
	}

	if allowedTools := os.Getenv("TEST_CLAUDE_ALLOWED_TOOLS"); allowedTools != "" {
		config.Claude.AllowedTools = allowedTools
		log.Printf("Using --allowedTools with value: %s", allowedTools)
	}

	if disallowedTools := os.Getenv("TEST_CLAUDE_DISALLOWED_TOOLS"); disallowedTools != "" {
		config.Claude.DisallowedTools = disallowedTools
		log.Printf("Using --disallowedTools with value: %s", disallowedTools)
	}

	claudeService := services.NewClaudeService(config)

	// Test generating code
	prompt := `# Task

## Test Task

This is a test task for the Claude CLI integration.

# Instructions

1. Write a simple "Hello, World!" program in Go.
2. Provide a brief explanation of how it works.

# Output Format

Please provide your response in the following format:

` + "```" + `
## Summary
<A brief summary of the changes made>

## Changes Made
<List of files modified and a description of the changes>

## Testing
<Description of how the changes were tested>
` + "```"

	repoDir := os.Getenv("TEST_REPO_DIR")
	if repoDir == "" {
		// Use a temporary directory
		repoDir = filepath.Join(config.TempDir, "test-claude")
		err := os.MkdirAll(repoDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create test directory: %w", err)
		}
	}

	response, err := claudeService.GenerateCode(prompt, repoDir)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	log.Printf("Successfully generated code:\n%s", response.Result)
	log.Printf("Claude usage - Input tokens: %d, Output tokens: %d, Cost: $%.4f, Duration: %dms",
		response.Usage.InputTokens, response.Usage.OutputTokens, response.TotalCostUsd, response.DurationMs)

	return nil
}

// RunTests runs all tests
func RunTests(config *models.Config) {
	// Test Jira integration
	if os.Getenv("TEST_JIRA") == "true" {
		err := TestJiraIntegration(config)
		if err != nil {
			log.Printf("Jira integration test failed: %v", err)
		} else {
			log.Println("Jira integration test passed")
		}
	}

	// Test GitHub integration
	if os.Getenv("TEST_GITHUB") == "true" {
		err := TestGitHubIntegration(config)
		if err != nil {
			log.Printf("GitHub integration test failed: %v", err)
		} else {
			log.Println("GitHub integration test passed")
		}
	}

	// Test Claude CLI integration
	if os.Getenv("TEST_CLAUDE") == "true" {
		err := TestClaudeIntegration(config)
		if err != nil {
			log.Printf("Claude CLI integration test failed: %v", err)
		} else {
			log.Println("Claude CLI integration test passed")
		}
	}
}

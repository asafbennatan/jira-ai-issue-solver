package services

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"jira-ai-issue-solver/models"
)

// GeminiService interface for code generation using Gemini CLI
type GeminiService interface {
	AIService
	// GenerateCodeGemini generates code using Gemini CLI and returns GeminiResponse
	GenerateCodeGemini(prompt string, repoDir string) (*GeminiResponse, error)
}

// GeminiServiceImpl implements the GeminiService interface
type GeminiServiceImpl struct {
	config   *models.Config
	executor models.CommandExecutor
}

// NewGeminiService creates a new GeminiService
func NewGeminiService(config *models.Config, executor ...models.CommandExecutor) GeminiService {
	commandExecutor := exec.Command
	if len(executor) > 0 {
		commandExecutor = executor[0]
	}
	return &GeminiServiceImpl{
		config:   config,
		executor: commandExecutor,
	}
}

// GeminiResponse represents the response from Gemini CLI
type GeminiResponse struct {
	Type         string         `json:"type"`
	IsError      bool           `json:"is_error"`
	Result       string         `json:"result"`
	SessionID    string         `json:"session_id"`
	TotalCostUsd float64        `json:"total_cost_usd"`
	Usage        GeminiUsage    `json:"usage"`
	Message      *GeminiMessage `json:"message"`
}

// GeminiUsage represents usage information
type GeminiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// GeminiMessage represents the message structure from Gemini
type GeminiMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Model   string `json:"model"`
	Content string `json:"content"`
}

// GenerateCode implements the AIService interface
func (s *GeminiServiceImpl) GenerateCode(prompt string, repoDir string) (interface{}, error) {
	return s.GenerateCodeGemini(prompt, repoDir)
}

// GenerateDocumentation implements the AIService interface
func (s *GeminiServiceImpl) GenerateDocumentation(repoDir string) error {
	// Check if GEMINI.md already exists
	geminiPath := filepath.Join(repoDir, "GEMINI.md")
	if _, err := os.Stat(geminiPath); err == nil {
		log.Printf("GEMINI.md already exists, skipping generation")
		return nil
	}

	log.Printf("GEMINI.md not found, generating documentation...")

	// Create prompt for generating GEMINI.md
	prompt := `Create a comprehensive GEMINI.md file that serves as an index and guide to all markdown documentation in this repository.

## Requirements:
1. **File Structure**: Create a well-organized document with clear sections and subsections
2. **File Index**: List all markdown files found in the repository (including nested folders) with:
   - Proper headlines for each file
   - Brief descriptions of what each file contains
   - Links to the actual files rather than copying their content
3. **Organization**: Group files logically (e.g., by directory, by purpose)
4. **Navigation**: Include a table of contents at the top
5. **Context**: Provide context about how the files relate to each other

## Format:
- Use clear, descriptive headlines for each file entry
- Include a brief description (1-2 sentences) explaining what each file covers
- Use relative links to the actual markdown files
- Organize files in a logical structure
- Make it easy for users to find relevant documentation

## Example structure:
# GEMINI.md

## Table of Contents
- [Getting Started](#getting-started)
- [Documentation](#documentation)
- [Contributing](#contributing)

## Getting Started
- [README.md](./README.md) - Main project overview and setup instructions
- [INSTALL.md](./docs/INSTALL.md) - Detailed installation guide

## Documentation
- [API.md](./docs/API.md) - API reference and usage examples
- [ARCHITECTURE.md](./docs/ARCHITECTURE.md) - System architecture overview

## Contributing
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Guidelines for contributors
- [STYLE.md](./docs/STYLE.md) - Code style and formatting guidelines

Search the entire repository for all .md files and create a comprehensive index following this structure.`

	// Generate the documentation using Gemini
	_, err := s.GenerateCodeGemini(prompt, repoDir)
	if err != nil {
		return fmt.Errorf("failed to generate GEMINI.md: %w", err)
	}

	// After running the CLI, check if GEMINI.md exists. If not, return an error.
	if _, err := os.Stat(geminiPath); os.IsNotExist(err) {
		return fmt.Errorf("GEMINI.md was not created by Gemini CLI")
	}

	log.Printf("Successfully generated GEMINI.md")
	return nil
}

// GenerateCodeGemini generates code using Gemini CLI
func (s *GeminiServiceImpl) GenerateCodeGemini(prompt string, repoDir string) (*GeminiResponse, error) {
	// Build command arguments based on configuration
	log.Printf("repoDir: %s", repoDir)
	log.Printf("Generating code with prompt: %s", prompt)

	args := []string{"--debug", "--y"}
	// Add model if configured
	if s.config.Gemini.Model != "" {
		args = append(args, "-m", s.config.Gemini.Model)
	}
	// Add all files flag if configured
	if s.config.Gemini.AllFiles {
		args = append(args, "-a")
	}
	// Add sandbox flag if configured
	if s.config.Gemini.Sandbox {
		args = append(args, "-s")
	}
	// Add prompt
	args = append(args, "-p", prompt)

	// Prepare the command with the built arguments
	cmd := s.executor(s.config.Gemini.CLIPath, args...)
	cmd.Dir = repoDir

	// Print the actual command being executed
	log.Printf("=== Executing Gemini CLI ===")
	log.Printf("Command: %s %s", s.config.Gemini.CLIPath, strings.Join(args, " "))
	log.Printf("Directory: %s", repoDir)
	log.Printf("===========================")

	// Set environment variables
	cmd.Env = os.Environ()

	// Set Gemini API key if configured
	if s.config.Gemini.APIKey != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("GEMINI_API_KEY=%s", s.config.Gemini.APIKey))
	}

	// Create pipes for stdout and stderr to read in real-time
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Set timeout
	timeout := time.Duration(s.config.Gemini.Timeout) * time.Second

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Gemini CLI: %w", err)
	}

	// Create channels for communication
	done := make(chan error, 1)
	resultChan := make(chan *GeminiResponse, 1)
	errorChan := make(chan error, 1)

	// Read stderr in a goroutine
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			log.Printf("=== Gemini stderr ===\n%s\n===================", scanner.Text())
		}
	}()

	// Wait for command completion in a goroutine
	go func() {
		err := cmd.Wait()
		done <- err
	}()

	// Stream stdout in real-time and process output
	go func() {
		log.Printf("Starting to stream Gemini output...")
		var finalResponse *GeminiResponse
		scanner := bufio.NewScanner(stdoutPipe)

		// Stream output in real-time while command is running
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			// Log each line for debugging in real-time
			log.Printf("=== Gemini Output Line ===")
			log.Printf("Content: %s", line)
			log.Printf("========================")

		}

		// Create response from accumulated output
		finalResponse = &GeminiResponse{
			Type:    "assistant",
			IsError: false,
			Result:  "done",
			Message: &GeminiMessage{
				Type:    "message",
				Role:    "assistant",
				Model:   s.config.Gemini.Model,
				Content: "done",
			},
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading Gemini output: %w", err)
			return
		}

		if finalResponse == nil {
			errorChan <- fmt.Errorf("no valid response found in Gemini output")
			return
		}

		log.Printf("Capturing final Gemini response...")
		log.Printf("Output processing complete. Final response captured.")
		resultChan <- finalResponse
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
			return nil, fmt.Errorf("failed to kill Gemini CLI process: %w", err)
		}
		return nil, fmt.Errorf("gemini CLI timed out after %d seconds", s.config.Gemini.Timeout)
	}

	// Wait for the result or error from the processing goroutine
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(5 * time.Second): // Additional timeout for result processing
		return nil, fmt.Errorf("timeout waiting for output processing result")
	}
}

// PreparePrompt prepares a prompt for Gemini CLI based on the Jira ticket
func PreparePromptForGemini(ticket *models.JiraTicketResponse) string {
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
	sb.WriteString("1. First, examine any relevant *.md files (README.md, CONTRIBUTING.md, etc.) in the repository (these might be nested so search the entire repo!) to understand the project structure, testing conventions, and how to run tests.\n")
	sb.WriteString("2. Analyze the task description and comments.\n")
	sb.WriteString("3. Implement the necessary changes to fulfill the requirements.\n")
	sb.WriteString("4. Write tests for the implemented functionality if appropriate.\n")
	sb.WriteString("5. Update documentation if necessary.\n")
	sb.WriteString("6. Make sure the project builds successfully before running tests.\n")
	sb.WriteString("7. Review the markdown files (README.md, CONTRIBUTING.md, etc.) to understand how tests should be run for this project. These files might be nested inside directories, so search the entire repository structure.\n")
	sb.WriteString("8. Verify your changes by running the relevant tests to ensure they work correctly.\n")
	sb.WriteString("9. Provide a summary of the changes made.\n")
	sb.WriteString("10. IMPORTANT: Do NOT perform any git operations (commit, push, pull, etc.). Git handling is managed by the system.\n\n")

	return sb.String()
}

// PreparePromptForPRFeedbackGemini prepares a prompt for Gemini CLI based on PR feedback
func PreparePromptForPRFeedbackGemini(pr *models.GitHubPullRequest, review *models.GitHubReview, repoDir string) (string, error) {
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
	sb.WriteString("5. Make sure the project builds successfully before running tests.\n")
	sb.WriteString("6. Review the markdown files (README.md, CONTRIBUTING.md, etc.) to understand how tests should be run for this project. These files might be nested inside directories, so search the entire repository structure.\n")
	sb.WriteString("7. Verify your changes by running the relevant tests to ensure they work correctly.\n")
	sb.WriteString("8. Provide a summary of the changes made.\n")
	sb.WriteString("9. IMPORTANT: Do NOT perform any git operations (commit, push, pull, etc.). Git handling is managed by the system.\n\n")

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

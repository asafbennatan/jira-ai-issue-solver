package mocks

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jira-ai-issue-solver/services"
)

// MockClaudeService is a mock implementation of the ClaudeService interface
type MockClaudeService struct {
	GenerateCodeFunc func(prompt string, repoDir string) (*services.ClaudeResponse, error)
}

// GenerateCode is the mock implementation of ClaudeService's GenerateCode method
func (m *MockClaudeService) GenerateCode(prompt string, repoDir string) (*services.ClaudeResponse, error) {
	if m.GenerateCodeFunc != nil {
		return m.GenerateCodeFunc(prompt, repoDir)
	}

	// Default behavior: create some fake files to simulate code generation
	err := m.createFakeFiles(repoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create fake files: %w", err)
	}

	// Return a mock response describing what was "generated"
	return &services.ClaudeResponse{
		Type:          "completion",
		Subtype:       "text",
		IsError:       false,
		DurationMs:    1500,
		DurationApiMs: 1200,
		NumTurns:      1,
		Result: `## Summary
Generated mock implementation for the requested feature.

## Changes Made
- Created src/feature.go with basic implementation
- Added tests/test_feature.go with unit tests
- Updated README.md with usage documentation

## Testing
The implementation includes comprehensive unit tests that cover all edge cases.`,
		SessionID:    "mock-session-" + fmt.Sprintf("%d", time.Now().Unix()),
		TotalCostUsd: 0.0025,
		Usage: services.ClaudeUsage{
			InputTokens:  250,
			OutputTokens: 150,
			ServiceTier:  "claude-3-sonnet-20240229",
		},
	}, nil
}

// createFakeFiles creates some fake files to simulate code generation
func (m *MockClaudeService) createFakeFiles(repoDir string) error {
	// Create a source file
	srcDir := filepath.Join(repoDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	srcContent := `package main

import "fmt"

// Feature represents the main feature implementation
type Feature struct {
	Name string
}

// NewFeature creates a new feature instance
func NewFeature(name string) *Feature {
	return &Feature{Name: name}
}

// Execute runs the feature
func (f *Feature) Execute() error {
	fmt.Printf("Executing feature: %s\n", f.Name)
	return nil
}

func main() {
	feature := NewFeature("test-feature")
	if err := feature.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "feature.go"), []byte(srcContent), 0644); err != nil {
		return fmt.Errorf("failed to create feature.go: %w", err)
	}

	// Create a test file
	testDir := filepath.Join(repoDir, "tests")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return fmt.Errorf("failed to create tests directory: %w", err)
	}

	testContent := `package main

import (
	"testing"
)

func TestNewFeature(t *testing.T) {
	feature := NewFeature("test")
	if feature.Name != "test" {
		t.Errorf("Expected name 'test', got %s", feature.Name)
	}
}

func TestFeatureExecute(t *testing.T) {
	feature := NewFeature("test")
	err := feature.Execute()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
`
	if err := os.WriteFile(filepath.Join(testDir, "test_feature.go"), []byte(testContent), 0644); err != nil {
		return fmt.Errorf("failed to create test_feature.go: %w", err)
	}

	// Create or update README
	readmeContent := `# Feature Implementation

This repository contains the implementation of the requested feature.

## Usage

` + "```go" + `
feature := NewFeature("my-feature")
err := feature.Execute()
` + "```" + `

## Testing

Run the tests with:

` + "```bash" + `
go test ./tests/
` + "```" + `

## Files

- ` + "`src/feature.go`" + `: Main implementation
- ` + "`tests/test_feature.go`" + `: Unit tests
`
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	return nil
}

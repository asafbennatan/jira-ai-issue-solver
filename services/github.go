package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"jira-ai-issue-solver/models"
)

// GitHubService defines the interface for interacting with GitHub
type GitHubService interface {
	// CloneRepository clones a repository to a local directory
	CloneRepository(repoURL, directory string) error

	// CreateBranch creates a new branch in a local repository
	CreateBranch(directory, branchName string) error

	// CommitChanges commits changes to a local repository
	CommitChanges(directory, message string) error

	// PushChanges pushes changes to a remote repository
	PushChanges(directory, branchName string) error

	// CreatePullRequest creates a pull request
	CreatePullRequest(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error)

	// ValidateWebhookSignature validates the signature of a webhook
	ValidateWebhookSignature(body []byte, signature string) bool

	// ForkRepository forks a repository and returns the clone URL of the fork
	ForkRepository(owner, repo string) (string, error)

	// CheckForkExists checks if a fork already exists for the given repository
	CheckForkExists(owner, repo string) (exists bool, cloneURL string, err error)

	// ResetFork resets a fork to match the original repository
	ResetFork(forkCloneURL, directory string) error

	// SyncForkWithUpstream syncs a fork with its upstream repository
	SyncForkWithUpstream(owner, repo string) error
}

// GitHubServiceImpl implements the GitHubService interface
type GitHubServiceImpl struct {
	config   *models.Config
	client   *http.Client
	executor models.CommandExecutor
}

// NewGitHubService creates a new GitHubService
func NewGitHubService(config *models.Config, executor ...models.CommandExecutor) GitHubService {
	commandExecutor := exec.Command
	if len(executor) > 0 {
		commandExecutor = executor[0]
	}
	return &GitHubServiceImpl{
		config:   config,
		client:   &http.Client{},
		executor: commandExecutor,
	}
}

// CloneRepository clones a repository to a local directory
func (s *GitHubServiceImpl) CloneRepository(repoURL, directory string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if the directory is already a git repository
	if _, err := os.Stat(filepath.Join(directory, ".git")); err == nil {
		// Directory is already a git repository, pull the latest changes
		cmd := s.executor("git", "pull")
		cmd.Dir = directory

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to pull repository: %w, stderr: %s", err, stderr.String())
		}

		return nil
	}

	// Clone the repository
	cmd := s.executor("git", "clone", repoURL, directory)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
	}

	// Configure git user
	cmd = s.executor("git", "config", "user.name", s.config.GitHub.Username)
	cmd.Dir = directory

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git user name: %w", err)
	}

	cmd = s.executor("git", "config", "user.email", s.config.GitHub.Email)
	cmd.Dir = directory

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git user email: %w", err)
	}

	return nil
}

// CreateBranch creates a new branch in a local repository based on the latest origin/main
func (s *GitHubServiceImpl) CreateBranch(directory, branchName string) error {
	// Fetch the latest changes from origin
	cmd := s.executor("git", "fetch", "origin")
	cmd.Dir = directory

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch origin: %w, stderr: %s", err, stderr.String())
	}

	// Checkout main/master branch
	cmd = s.executor("git", "checkout", "main")
	cmd.Dir = directory

	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Try master branch
		cmd = s.executor("git", "checkout", "master")
		cmd.Dir = directory

		stderr.Reset()
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout main/master branch: %w, stderr: %s", err, stderr.String())
		}
	}

	// Reset to origin/main or origin/master to ensure we're up to date
	cmd = s.executor("git", "reset", "--hard", "origin/main")
	cmd.Dir = directory

	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Try with master branch
		cmd = s.executor("git", "reset", "--hard", "origin/master")
		cmd.Dir = directory

		stderr.Reset()
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to reset to origin/main or origin/master: %w, stderr: %s", err, stderr.String())
		}
	}

	// Create a new branch from the current state
	cmd = s.executor("git", "checkout", "-b", branchName)
	cmd.Dir = directory

	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// CommitChanges commits changes to a local repository
func (s *GitHubServiceImpl) CommitChanges(directory, message string) error {
	// Add all changes
	cmd := s.executor("git", "add", ".")
	cmd.Dir = directory

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add changes: %w, stderr: %s", err, stderr.String())
	}

	// Check if there are changes to commit
	cmd = s.executor("git", "status", "--porcelain")
	cmd.Dir = directory

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}

	if stdout.Len() == 0 {
		// No changes to commit
		return nil
	}

	// Commit changes
	cmd = s.executor("git", "commit", "-m", message)
	cmd.Dir = directory

	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// PushChanges pushes changes to a remote repository
func (s *GitHubServiceImpl) PushChanges(directory, branchName string) error {
	cmd := s.executor("git", "push", "-u", "origin", branchName)
	cmd.Dir = directory

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push changes: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// CreatePullRequest creates a pull request
func (s *GitHubServiceImpl) CreatePullRequest(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)

	payload := models.GitHubCreatePRRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.config.GitHub.APIToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create pull request: %s, status code: %d", string(body), resp.StatusCode)
	}

	var prResponse models.GitHubCreatePRResponse
	if err := json.NewDecoder(resp.Body).Decode(&prResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &prResponse, nil
}

// ValidateWebhookSignature validates the signature of a webhook
func (s *GitHubServiceImpl) ValidateWebhookSignature(body []byte, signature string) bool {
	// Implement webhook signature validation if needed
	// For simplicity, we're not implementing this now
	return true
}

// CheckForkExists checks if a fork already exists for the given repository
func (s *GitHubServiceImpl) CheckForkExists(owner, repo string) (exists bool, cloneURL string, err error) {
	// Check if the fork already exists by listing the bot's repositories
	url := fmt.Sprintf("https://api.github.com/users/%s/repos", s.config.GitHub.BotUsername)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Use the bot's token
	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.config.GitHub.BotToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, "", fmt.Errorf("failed to list repositories: %s, status code: %d", string(body), resp.StatusCode)
	}

	var repos []struct {
		Name     string `json:"name"`
		CloneURL string `json:"clone_url"`
		Fork     bool   `json:"fork"`
		Source   struct {
			FullName string `json:"full_name"`
		} `json:"source"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return false, "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if any of the repositories is a fork of the target repository
	targetFullName := fmt.Sprintf("%s/%s", owner, repo)
	for _, r := range repos {
		if r.Fork && r.Source.FullName == targetFullName {
			return true, r.CloneURL, nil
		}
	}

	return false, "", nil
}

// ResetFork resets a fork to match the original repository and sets up upstream
func (s *GitHubServiceImpl) ResetFork(forkCloneURL, directory string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if the directory is already a git repository
	if _, err := os.Stat(filepath.Join(directory, ".git")); err == nil {
		// Directory is already a git repository, fetch and reset
		// Fetch the upstream repository
		cmd := s.executor("git", "fetch", "origin")
		cmd.Dir = directory

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to fetch origin: %w, stderr: %s", err, stderr.String())
		}

		// Reset to origin/main or origin/master
		cmd = s.executor("git", "reset", "--hard", "origin/main")
		cmd.Dir = directory

		stderr.Reset()
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// Try with master branch
			cmd = s.executor("git", "reset", "--hard", "origin/master")
			cmd.Dir = directory

			stderr.Reset()
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to reset to origin/main or origin/master: %w, stderr: %s", err, stderr.String())
			}
		}

		// Clean the repository
		cmd = s.executor("git", "clean", "-fdx")
		cmd.Dir = directory

		stderr.Reset()
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clean repository: %w, stderr: %s", err, stderr.String())
		}

		return nil
	}

	// Clone the repository
	return s.CloneRepository(forkCloneURL, directory)
}

// ForkRepository forks a repository and returns the clone URL of the fork
func (s *GitHubServiceImpl) ForkRepository(owner, repo string) (string, error) {
	// Create a new fork
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/forks", owner, repo)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Use the bot's token
	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.config.GitHub.BotToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fork repository: %s, status code: %d", string(body), resp.StatusCode)
	}

	var forkResponse struct {
		HTMLURL  string `json:"html_url"`
		CloneURL string `json:"clone_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&forkResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return forkResponse.CloneURL, nil
}

// SyncForkWithUpstream syncs a fork with its upstream repository
func (s *GitHubServiceImpl) SyncForkWithUpstream(owner, repo string) error {
	// Get the fork details to sync with upstream
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", s.config.GitHub.BotUsername, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.config.GitHub.BotToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get fork details: %s, status code: %d", string(body), resp.StatusCode)
	}

	var forkDetails struct {
		Source struct {
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
			Name string `json:"name"`
		} `json:"source"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&forkDetails); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Sync the fork with upstream
	syncURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/merge-upstream", s.config.GitHub.BotUsername, repo)
	syncBody := map[string]string{
		"branch": "main",
	}

	jsonBody, err := json.Marshal(syncBody)
	if err != nil {
		return fmt.Errorf("failed to marshal sync request: %w", err)
	}

	req, err = http.NewRequest("POST", syncURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.config.GitHub.BotToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err = s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send sync request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to sync fork: %s, status code: %d", string(body), resp.StatusCode)
	}

	return nil
}

// ExtractRepoInfo extracts owner and repo from a repository URL
func ExtractRepoInfo(repoURL string) (owner, repo string, err error) {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(repoURL, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(repoURL, "git@github.com:"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub SSH URL: %s", repoURL)
		}
		owner = parts[0]
		repo = strings.TrimSuffix(parts[1], ".git")
		return owner, repo, nil
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	if strings.HasPrefix(repoURL, "https://github.com/") {
		parts := strings.Split(strings.TrimPrefix(repoURL, "https://github.com/"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub HTTPS URL: %s", repoURL)
		}
		owner = parts[0]
		repo = strings.TrimSuffix(parts[1], ".git")
		return owner, repo, nil
	}

	return "", "", fmt.Errorf("unsupported repository URL format: %s", repoURL)
}

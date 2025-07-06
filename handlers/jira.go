package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"jira-ai-issue-solver/models"
	"jira-ai-issue-solver/services"
)

// JiraWebhookHandler handles webhooks from Jira
type JiraWebhookHandler struct {
	jiraService   services.JiraService
	githubService services.GitHubService
	claudeService services.ClaudeService
	config        *models.Config
}

// NewJiraWebhookHandler creates a new JiraWebhookHandler
func NewJiraWebhookHandler(
	jiraService services.JiraService,
	githubService services.GitHubService,
	claudeService services.ClaudeService,
	config *models.Config,
) *JiraWebhookHandler {
	return &JiraWebhookHandler{
		jiraService:   jiraService,
		githubService: githubService,
		claudeService: claudeService,
		config:        config,
	}
}

// HandleWebhook handles a webhook from Jira
func (h *JiraWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the webhook signature
	signature := r.Header.Get("X-Jira-Signature")
	if !h.jiraService.ValidateWebhookSignature(body, signature) {
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}

	// Parse the webhook payload
	var webhook models.JiraWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse webhook payload: %v", err), http.StatusBadRequest)
		return
	}

	// Check if this is an issue update event
	if webhook.WebhookEvent != "jira:issue_updated" {
		// Not an issue update event, ignore
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check if the issue has the "good-for-ai" label and doesn't have the "ai-in-progress" label
	hasGoodForAILabel := false
	hasAIInProgressLabel := false

	for _, label := range webhook.Issue.Fields.Labels {
		if label == string(models.LabelGoodForAI) {
			hasGoodForAILabel = true
		}
		if label == string(models.LabelAIInProgress) {
			hasAIInProgressLabel = true
		}
	}

	if !hasGoodForAILabel || hasAIInProgressLabel {
		// Not a ticket for AI to process, ignore
		w.WriteHeader(http.StatusOK)
		return
	}

	// Add the "ai-in-progress" label to prevent duplicate processing
	err = h.jiraService.UpdateTicketLabels(webhook.Issue.Key, []string{string(models.LabelAIInProgress)}, nil)
	if err != nil {
		log.Printf("failed to add ai-in-progress label: %v", err)
		http.Error(w, fmt.Sprintf("failed to add ai-in-progress label: %v", err), http.StatusInternalServerError)
		return
	}

	// Update the ticket status to "In Progress"
	err = h.jiraService.UpdateTicketStatus(webhook.Issue.Key, string(models.StatusInProgress))
	if err != nil {
		log.Printf("failed to update ticket status: %v", err)
		// Continue processing even if status update fails
	}

	// Process the ticket asynchronously
	go h.processTicket(webhook.Issue.Key)

	// Return success
	w.WriteHeader(http.StatusOK)
}

// processTicket processes a Jira ticket
func (h *JiraWebhookHandler) processTicket(ticketKey string) {
	log.Printf("Processing ticket %s", ticketKey)

	// Get the ticket details
	ticket, err := h.jiraService.GetTicket(ticketKey)
	if err != nil {
		log.Printf("failed to get ticket details: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to get ticket details: %v", err))
		return
	}

	// Get the repository URL from the project properties
	if ticket.Fields.Project.Properties == nil {
		log.Printf("project properties is nil")
		h.handleFailure(ticketKey, "Project properties is nil")
		return
	}

	repoURL, ok := ticket.Fields.Project.Properties["ai.bot.github.repo"]
	if !ok || repoURL == "" {
		log.Printf("repository URL not found in project properties")
		h.handleFailure(ticketKey, "Repository URL not found in project properties")
		return
	}

	// Create a temporary directory for the repository
	repoDir := filepath.Join(h.config.TempDir, ticketKey)

	// Extract owner and repo from the repository URL
	owner, repo, err := services.ExtractRepoInfo(repoURL)
	if err != nil {
		log.Printf("failed to extract repo info: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to extract repo info: %v", err))
		return
	}

	// Check if a fork already exists
	exists, forkURL, err := h.githubService.CheckForkExists(owner, repo)
	if err != nil {
		log.Printf("failed to check if fork exists: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to check if fork exists: %v", err))
		return
	}

	if exists {
		// Fork exists, sync it with upstream
		err = h.githubService.SyncForkWithUpstream(owner, repo)
		if err != nil {
			log.Printf("failed to sync fork with upstream: %v", err)
			h.handleFailure(ticketKey, fmt.Sprintf("Failed to sync fork with upstream: %v", err))
			return
		}
	} else {
		// Fork doesn't exist, create a new one
		forkURL, err = h.githubService.ForkRepository(owner, repo)
		if err != nil {
			log.Printf("failed to fork repository: %v", err)
			h.handleFailure(ticketKey, fmt.Sprintf("Failed to fork repository: %v", err))
			return
		}
	}

	// Wait for the fork to be ready (GitHub may take a moment to complete the fork)
	//TODO: fix this sleeping to get more reliable results
	time.Sleep(5 * time.Second)

	// Clone the fork to the local directory
	err = h.githubService.CloneRepository(forkURL, repoDir)
	if err != nil {
		log.Printf("failed to clone fork: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to clone fork: %v", err))
		return
	}

	// Configure git to use the bot credentials
	cmd := exec.Command("git", "config", "user.name", h.config.GitHub.BotUsername)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		log.Printf("failed to configure git user name: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to configure git user name: %v", err))
		return
	}

	cmd = exec.Command("git", "config", "user.email", h.config.GitHub.BotEmail)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		log.Printf("failed to configure git user email: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to configure git user email: %v", err))
		return
	}

	// Create a new branch
	branchName := fmt.Sprintf("feature/%s-%s", ticketKey, strings.ToLower(strings.ReplaceAll(ticket.Fields.Summary, " ", "-")))
	err = h.githubService.CreateBranch(repoDir, branchName)
	if err != nil {
		log.Printf("failed to create branch: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to create branch: %v", err))
		return
	}

	// Prepare the prompt for Claude
	prompt := services.PreparePrompt(ticket)

	// Generate code using Claude
	response, err := h.claudeService.GenerateCode(prompt, repoDir)
	if err != nil {
		log.Printf("failed to generate code: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to generate code: %v", err))
		return
	}

	// Log Claude usage statistics
	log.Printf("Claude usage - Input tokens: %d, Output tokens: %d, Cost: $%.4f, Duration: %dms",
		response.Usage.InputTokens, response.Usage.OutputTokens, response.TotalCostUsd, response.DurationMs)

	// Commit the changes
	commitMessage := fmt.Sprintf("feat(%s): %s", ticketKey, ticket.Fields.Summary)
	err = h.githubService.CommitChanges(repoDir, commitMessage)
	if err != nil {
		log.Printf("failed to commit changes: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to commit changes: %v", err))
		return
	}

	// Push the changes
	err = h.githubService.PushChanges(repoDir, branchName)
	if err != nil {
		log.Printf("failed to push changes: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to push changes: %v", err))
		return
	}

	// Create a pull request
	prTitle := fmt.Sprintf("%s: %s", ticketKey, ticket.Fields.Summary)
	prBody := fmt.Sprintf("## AI-generated PR for %s\n\n%s\n\n## Claude Output\n\n```\n%s\n```",
		ticketKey,
		ticket.Fields.Description,
		response.Result)

	// The head should be in the format "username:branch" (from the fork)
	head := fmt.Sprintf("%s:%s", h.config.GitHub.BotUsername, branchName)
	// The base should be the original repository's main branch
	base := "main"

	pr, err := h.githubService.CreatePullRequest(owner, repo, prTitle, prBody, head, base)
	if err != nil {
		log.Printf("failed to create pull request: %v", err)
		h.handleFailure(ticketKey, fmt.Sprintf("Failed to create pull request: %v", err))
		return
	}

	// Update the ticket with the PR link
	comment := fmt.Sprintf("AI has created a pull request: %s", pr.HTMLURL)
	err = h.jiraService.AddComment(ticketKey, comment)
	if err != nil {
		log.Printf("failed to add comment: %v", err)
		// Continue even if comment fails
	}

	// Update the ticket labels
	err = h.jiraService.UpdateTicketLabels(ticketKey, []string{string(models.LabelAIPRCreated)}, []string{string(models.LabelAIInProgress)})
	if err != nil {
		log.Printf("failed to update ticket labels: %v", err)
		// Continue even if label update fails
	}

	// Update the ticket status to "In Review"
	err = h.jiraService.UpdateTicketStatus(ticketKey, string(models.StatusInReview))
	if err != nil {
		log.Printf("failed to update ticket status: %v", err)
		// Continue even if status update fails
	}

	// Clean up the temporary directory
	err = os.RemoveAll(repoDir)
	if err != nil {
		log.Printf("failed to clean up temporary directory: %v", err)
		// Continue even if cleanup fails
	}

	log.Printf("Successfully processed ticket %s", ticketKey)
}

// handleFailure handles a failure in processing a ticket
func (h *JiraWebhookHandler) handleFailure(ticketKey, errorMessage string) {
	// Add a comment to the ticket
	err := h.jiraService.AddComment(ticketKey, fmt.Sprintf("AI failed to process this ticket: %s", errorMessage))
	if err != nil {
		log.Printf("failed to add comment: %v", err)
	}

	// Update the ticket labels
	err = h.jiraService.UpdateTicketLabels(ticketKey, []string{string(models.LabelAIFailed)}, []string{string(models.LabelAIInProgress)})
	if err != nil {
		log.Printf("failed to update ticket labels: %v", err)
	}
}

// StartJanitor starts a janitor process to clean up stuck tickets
func (h *JiraWebhookHandler) StartJanitor() {
	go func() {
		for {
			// Sleep for 1 hour
			time.Sleep(1 * time.Hour)

			// TODO: Implement janitor logic to clean up stuck tickets
			// This would involve searching for tickets with the models.LabelAIInProgress label
			// that haven't been updated in a long time, and marking them as failed
		}
	}()
}

package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"jira-ai-issue-solver/models"
)

// TicketProcessor defines the interface for processing Jira tickets
type TicketProcessor interface {
	// ProcessTicket processes a single Jira ticket
	ProcessTicket(ticketKey string) error
}

// TicketProcessorImpl implements the TicketProcessor interface
type TicketProcessorImpl struct {
	jiraService   JiraService
	githubService GitHubService
	aiService     AIService
	config        *models.Config
}

// NewTicketProcessor creates a new TicketProcessor
func NewTicketProcessor(
	jiraService JiraService,
	githubService GitHubService,
	aiService AIService,
	config *models.Config,
) TicketProcessor {
	return &TicketProcessorImpl{
		jiraService:   jiraService,
		githubService: githubService,
		aiService:     aiService,
		config:        config,
	}
}

// ProcessTicket processes a Jira ticket
func (p *TicketProcessorImpl) ProcessTicket(ticketKey string) error {
	log.Printf("Processing ticket %s", ticketKey)

	// Get the ticket details
	ticket, err := p.jiraService.GetTicket(ticketKey)
	if err != nil {
		log.Printf("failed to get ticket details: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to get ticket details: %v", err))
		return err
	}

	// Get the repository URL from the component mapping
	if len(ticket.Fields.Components) == 0 {
		log.Printf("no components found on ticket")
		p.handleFailure(ticketKey, "No components found on ticket")
		return fmt.Errorf("no components found on ticket")
	}

	// Use the first component to find the repository
	firstComponent := ticket.Fields.Components[0].Name
	repoURL, ok := p.config.ComponentToRepo[firstComponent]
	if !ok || repoURL == "" {
		log.Printf("no repository mapping found for component: %s", firstComponent)
		p.handleFailure(ticketKey, fmt.Sprintf("No repository mapping found for component: %s", firstComponent))
		return fmt.Errorf("no repository mapping found for component: %s", firstComponent)
	}
	log.Printf("found repository mapping for component: %s : repo url: %s", firstComponent, repoURL)

	// Update the ticket status to the configured "In Progress" status
	err = p.jiraService.UpdateTicketStatus(ticketKey, p.config.Jira.StatusTransitions.InProgress)
	if err != nil {
		log.Printf("failed to update ticket status: %v", err)
		// Continue processing even if status update fails
	}

	// Extract owner and repo from the repository URL
	owner, repo, err := ExtractRepoInfo(repoURL)
	if err != nil {
		log.Printf("failed to extract repo info: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to extract repo info: %v", err))
		return err
	}
	log.Printf("owner: %s, repo: %s", owner, repo)

	// Check if a fork already exists
	exists, forkURL, err := p.githubService.CheckForkExists(owner, repo)
	if err != nil {
		log.Printf("failed to check if fork exists: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to check if fork exists: %v", err))
		return err
	}

	if !exists {
		// Create a fork
		forkURL, err = p.githubService.ForkRepository(owner, repo)
		if err != nil {
			log.Printf("failed to create fork: %v", err)
			p.handleFailure(ticketKey, fmt.Sprintf("Failed to create fork: %v", err))
			return err
		}
		log.Printf("fork created successfully, waiting for fork to be ready...")

		// Wait for the fork to be ready by checking if it exists
		for i := 0; i < 10; i++ { // Try up to 10 times (50 seconds total)
			exists, forkURL, err = p.githubService.CheckForkExists(owner, repo)
			if err != nil {
				log.Printf("failed to check fork readiness (attempt %d): %v", i+1, err)
				time.Sleep(5 * time.Second)
				continue
			}

			if exists {
				log.Printf("fork is ready after %d attempts", i+1)
				break
			}

			log.Printf("fork not ready yet, waiting... (attempt %d)", i+1)
			time.Sleep(5 * time.Second)
		}

		if !exists {
			log.Printf("fork failed to become ready after multiple attempts")
			p.handleFailure(ticketKey, "Fork failed to become ready after multiple attempts")
			return fmt.Errorf("fork failed to become ready after multiple attempts")
		}
	}

	// Clone the repository
	repoDir := strings.Join([]string{p.config.TempDir, ticketKey}, "/")
	err = p.githubService.CloneRepository(forkURL, repoDir)
	if err != nil {
		log.Printf("failed to clone repository: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to clone repository: %v", err))
		return err
	}

	// Switch to the target branch if we're not already on it
	err = p.githubService.SwitchToTargetBranch(repoDir)
	if err != nil {
		log.Printf("failed to switch to target branch: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to switch to target branch: %v", err))
		return err
	}

	// Create a new branch
	branchName := ticketKey
	err = p.githubService.CreateBranch(repoDir, branchName)
	if err != nil {
		log.Printf("failed to create branch: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to create branch: %v", err))
		return err
	}

	// Generate documentation file (CLAUDE.md or GEMINI.md) if it doesn't exist
	err = p.aiService.GenerateDocumentation(repoDir)
	if err != nil {
		log.Printf("failed to generate documentation: %v", err)
		// Continue processing even if documentation generation fails
	}

	// Generate a prompt for Claude CLI
	prompt := p.generatePrompt(ticket)

	// Run AI service to generate code changes
	_, err = p.aiService.GenerateCode(prompt, repoDir)
	if err != nil {
		log.Printf("failed to generate code changes: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to generate code changes: %v", err))
		return err
	}

	// Commit the changes
	err = p.githubService.CommitChanges(repoDir, fmt.Sprintf("%s: %s", ticketKey, ticket.Fields.Summary))
	if err != nil {
		log.Printf("failed to commit changes: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to commit changes: %v", err))
		return err
	}

	// Push the changes
	err = p.githubService.PushChanges(repoDir, branchName)
	if err != nil {
		log.Printf("failed to push changes: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to push changes: %v", err))
		return err
	}

	// Create a pull request
	prTitle := fmt.Sprintf("%s: %s", ticketKey, ticket.Fields.Summary)
	prBody := fmt.Sprintf("This PR addresses the issue described in %s.\n\n**Summary:** %s\n\n**Description:** %s",
		ticketKey, ticket.Fields.Summary, ticket.Fields.Description)

	// When creating a pull request from a fork, the head parameter should be in the format "forkOwner:branchName"
	head := fmt.Sprintf("%s:%s", p.config.GitHub.BotUsername, branchName)
	pr, err := p.githubService.CreatePullRequest(owner, repo, prTitle, prBody, head, p.config.GitHub.TargetBranch)
	if err != nil {
		log.Printf("failed to create pull request: %v", err)
		p.handleFailure(ticketKey, fmt.Sprintf("Failed to create pull request: %v", err))
		return err
	}

	// Update the Git Pull Request field if configured
	if p.config.Jira.GitPullRequestFieldName != "" {
		err = p.jiraService.UpdateTicketFieldByName(ticketKey, p.config.Jira.GitPullRequestFieldName, pr.HTMLURL)
		if err != nil {
			log.Printf("failed to update Git Pull Request field: %v", err)
			// Continue even if field update fails
		} else {
			log.Printf("Successfully updated Git Pull Request field with URL: %s", pr.HTMLURL)
		}
	}

	// Add a comment to the ticket with the PR link
	comment := fmt.Sprintf("AI has created a pull request to address this issue: %s", pr.HTMLURL)
	err = p.jiraService.AddComment(ticketKey, comment)
	if err != nil {
		log.Printf("failed to add comment: %v", err)
		// Continue even if comment fails
	}

	// Update the ticket status to the configured "In Review" status
	err = p.jiraService.UpdateTicketStatus(ticketKey, p.config.Jira.StatusTransitions.InReview)
	if err != nil {
		log.Printf("failed to update ticket status: %v", err)
		// Continue even if status update fails
	}

	log.Printf("Successfully processed ticket %s", ticketKey)
	return nil
}

// handleFailure handles a failure in processing a ticket
func (p *TicketProcessorImpl) handleFailure(ticketKey, errorMessage string) {
	// Add a comment to the ticket only if error comments are not disabled
	if !p.config.Jira.DisableErrorComments {
		err := p.jiraService.AddComment(ticketKey, fmt.Sprintf("AI failed to process this ticket: %s", errorMessage))
		if err != nil {
			log.Printf("failed to add comment: %v", err)
		}
	} else {
		log.Printf("Error commenting disabled, not adding error comment for ticket %s: %s", ticketKey, errorMessage)
	}

}

// generatePrompt generates a prompt for Claude CLI based on the ticket
func (p *TicketProcessorImpl) generatePrompt(ticket *models.JiraTicketResponse) string {
	prompt := fmt.Sprintf("Please help me fix the issue described in Jira ticket %s.\n\n", ticket.Key)
	prompt += fmt.Sprintf("Summary: %s\n\n", ticket.Fields.Summary)
	prompt += fmt.Sprintf("Description: %s\n\n", ticket.Fields.Description)

	// Add comments if available, filtering out bot comments
	if ticket.Fields.Comment.Comments != nil {
		prompt += "Comments:\n"
		for _, comment := range ticket.Fields.Comment.Comments {
			// Skip comments made by our Jira bot
			if comment.Author.Name == p.config.Jira.Username {
				continue
			}
			prompt += fmt.Sprintf("- %s: %s\n", comment.Author.DisplayName, comment.Body)
		}
		prompt += "\n"
	}

	prompt += "Please analyze the codebase and implement the necessary changes to fix this issue. " +
		"Make sure to follow the existing code style and patterns in the codebase."

	return prompt
}

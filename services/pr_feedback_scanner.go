package services

import (
	"fmt"
	"log"
	"time"

	"jira-ai-issue-solver/models"
)

// PRFeedbackScannerService defines the interface for scanning tickets in "In Review" status
type PRFeedbackScannerService interface {
	// Start starts the periodic scanning for PR feedback
	Start()
	// Stop stops the periodic scanning
	Stop()
}

// PRFeedbackScannerServiceImpl implements the PRFeedbackScannerService interface
type PRFeedbackScannerServiceImpl struct {
	jiraService       JiraService
	githubService     GitHubService
	aiService         AIService
	prReviewProcessor PRReviewProcessor
	config            *models.Config
	stopChan          chan struct{}
	isRunning         bool
}

// NewPRFeedbackScannerService creates a new PRFeedbackScannerService
func NewPRFeedbackScannerService(
	jiraService JiraService,
	githubService GitHubService,
	aiService AIService,
	config *models.Config,
) PRFeedbackScannerService {
	prReviewProcessor := NewPRReviewProcessor(jiraService, githubService, aiService, config)

	return &PRFeedbackScannerServiceImpl{
		jiraService:       jiraService,
		githubService:     githubService,
		aiService:         aiService,
		prReviewProcessor: prReviewProcessor,
		config:            config,
		stopChan:          make(chan struct{}),
		isRunning:         false,
	}
}

// Start starts the periodic scanning for PR feedback
func (s *PRFeedbackScannerServiceImpl) Start() {
	if s.isRunning {
		log.Println("PR feedback scanner is already running")
		return
	}

	s.isRunning = true
	log.Println("Starting PR feedback scanner...")

	go func() {
		ticker := time.NewTicker(time.Duration(s.config.Jira.IntervalSeconds) * time.Second)
		defer ticker.Stop()

		// Run initial scan immediately
		s.scanForPRFeedback()

		for {
			select {
			case <-ticker.C:
				s.scanForPRFeedback()
			case <-s.stopChan:
				log.Println("Stopping PR feedback scanner...")
				return
			}
		}
	}()
}

// Stop stops the periodic scanning
func (s *PRFeedbackScannerServiceImpl) Stop() {
	if !s.isRunning {
		return
	}

	s.isRunning = false
	close(s.stopChan)
}

// scanForPRFeedback searches for tickets in "In Review" status that need PR feedback processing
func (s *PRFeedbackScannerServiceImpl) scanForPRFeedback() {
	log.Println("Scanning for tickets in 'In Review' status that need PR feedback processing...")

	inReviewStatus := s.config.Jira.StatusTransitions.InReview

	// Build JQL query to find tickets assigned to current user in "In Review" status
	// and that have a PR URL set
	jql := fmt.Sprintf(`Contributors = currentUser() AND status = "%s" AND "%s" IS NOT EMPTY ORDER BY updated DESC`,
		inReviewStatus, s.config.Jira.GitPullRequestFieldName)

	searchResponse, err := s.jiraService.SearchTickets(jql)
	if err != nil {
		log.Printf("Failed to search for tickets in 'In Review' status: %v", err)
		return
	}

	if searchResponse.Total == 0 {
		log.Println("No tickets found in 'In Review' status that need PR feedback processing")
		return
	}

	log.Printf("Found %d tickets in 'In Review' status that need PR feedback processing", searchResponse.Total)

	// Process each ticket
	for _, issue := range searchResponse.Issues {
		log.Printf("Found ticket %s in 'In Review' status", issue.Key)

		// Process the ticket asynchronously
		go func(ticketKey string) {
			if err := s.prReviewProcessor.ProcessPRReviewFeedback(ticketKey); err != nil {
				log.Printf("Failed to process PR feedback for ticket %s: %v", ticketKey, err)
			}
		}(issue.Key)
	}
}

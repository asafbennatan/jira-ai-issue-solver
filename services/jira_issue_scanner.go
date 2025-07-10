package services

import (
	"fmt"
	"log"
	"time"

	"jira-ai-issue-solver/models"
)

// JiraIssueScannerService defines the interface for the Jira issue scanner
type JiraIssueScannerService interface {
	// Start starts the periodic scanning
	Start()
	// Stop stops the periodic scanning
	Stop()
}

// JiraIssueScannerServiceImpl implements the JiraIssueScannerService interface
type JiraIssueScannerServiceImpl struct {
	jiraService     JiraService
	githubService   GitHubService
	aiService       AIService
	ticketProcessor TicketProcessor
	config          *models.Config
	stopChan        chan struct{}
	isRunning       bool
}

// NewJiraIssueScannerService creates a new JiraIssueScannerService
func NewJiraIssueScannerService(
	jiraService JiraService,
	githubService GitHubService,
	aiService AIService,
	config *models.Config,
) JiraIssueScannerService {
	ticketProcessor := NewTicketProcessor(jiraService, githubService, aiService, config)

	return &JiraIssueScannerServiceImpl{
		jiraService:     jiraService,
		githubService:   githubService,
		aiService:       aiService,
		ticketProcessor: ticketProcessor,
		config:          config,
		stopChan:        make(chan struct{}),
		isRunning:       false,
	}
}

// Start starts the periodic scanning
func (s *JiraIssueScannerServiceImpl) Start() {
	if s.isRunning {
		log.Println("Jira issue scanner is already running")
		return
	}

	s.isRunning = true
	log.Println("Starting Jira issue scanner...")

	go func() {
		ticker := time.NewTicker(time.Duration(s.config.Jira.IntervalSeconds) * time.Second)
		defer ticker.Stop()

		// Run initial scan immediately
		s.scanForTickets()

		for {
			select {
			case <-ticker.C:
				s.scanForTickets()
			case <-s.stopChan:
				log.Println("Stopping Jira issue scanner...")
				return
			}
		}
	}()
}

// Stop stops the periodic scanning
func (s *JiraIssueScannerServiceImpl) Stop() {
	if !s.isRunning {
		return
	}

	s.isRunning = false
	close(s.stopChan)
}

// scanForTickets searches for tickets that need AI processing
func (s *JiraIssueScannerServiceImpl) scanForTickets() {
	log.Println("Scanning for tickets that need AI processing...")

	todoStatus := s.config.Jira.StatusTransitions.Todo

	// Build JQL query to find tickets assigned to current user in TODO status
	jql := fmt.Sprintf(`Contributors = currentUser() AND status = "%s" ORDER BY updated DESC`, todoStatus)

	searchResponse, err := s.jiraService.SearchTickets(jql)
	if err != nil {
		log.Printf("Failed to search for tickets: %v", err)
		return
	}

	if searchResponse.Total == 0 {
		log.Println("No tickets found that need AI processing")
		return
	}

	log.Printf("Found %d tickets that need AI processing", searchResponse.Total)

	// Process each ticket
	for _, issue := range searchResponse.Issues {
		log.Printf("Found ticket %s", issue.Key)

		// Process all tickets returned by the search

		// Process the ticket asynchronously
		go s.ticketProcessor.ProcessTicket(issue.Key)
	}
}

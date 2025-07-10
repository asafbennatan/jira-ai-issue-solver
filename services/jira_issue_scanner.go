package services

import (
	"fmt"
	"time"

	"jira-ai-issue-solver/models"

	"go.uber.org/zap"
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
	logger          *zap.Logger
	stopChan        chan struct{}
	isRunning       bool
}

// NewJiraIssueScannerService creates a new JiraIssueScannerService
func NewJiraIssueScannerService(
	jiraService JiraService,
	githubService GitHubService,
	aiService AIService,
	config *models.Config,
	logger *zap.Logger,
) JiraIssueScannerService {
	ticketProcessor := NewTicketProcessor(jiraService, githubService, aiService, config, logger)

	return &JiraIssueScannerServiceImpl{
		jiraService:     jiraService,
		githubService:   githubService,
		aiService:       aiService,
		ticketProcessor: ticketProcessor,
		config:          config,
		logger:          logger,
		stopChan:        make(chan struct{}),
		isRunning:       false,
	}
}

// Start starts the periodic scanning
func (s *JiraIssueScannerServiceImpl) Start() {
	if s.isRunning {
		s.logger.Info("Jira issue scanner is already running")
		return
	}

	s.isRunning = true
	s.logger.Info("Starting Jira issue scanner...")

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
				s.logger.Info("Stopping Jira issue scanner...")
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
	s.logger.Info("Scanning for tickets that need AI processing...")

	todoStatus := s.config.Jira.StatusTransitions.Todo

	// Build JQL query to find tickets assigned to current user in TODO status
	jql := fmt.Sprintf(`Contributors = currentUser() AND status = "%s" ORDER BY updated DESC`, todoStatus)

	searchResponse, err := s.jiraService.SearchTickets(jql)
	if err != nil {
		s.logger.Error("Failed to search for tickets", zap.Error(err))
		return
	}

	if searchResponse.Total == 0 {
		s.logger.Info("No tickets found that need AI processing")
		return
	}

	s.logger.Info("Found tickets that need AI processing", zap.Int("count", searchResponse.Total))

	// Process each ticket
	for _, issue := range searchResponse.Issues {
		s.logger.Info("Found ticket", zap.String("ticket", issue.Key))

		// Process all tickets returned by the search

		// Process the ticket asynchronously
		go s.ticketProcessor.ProcessTicket(issue.Key)
	}
}

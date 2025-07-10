package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jira-ai-issue-solver/models"
	"jira-ai-issue-solver/services"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := models.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate required configuration
	if config.Jira.BaseURL == "" {
		log.Fatal("JIRA_BASE_URL is required")
	}
	if config.Jira.Username == "" {
		log.Fatal("JIRA_USERNAME is required")
	}
	if config.Jira.APIToken == "" {
		log.Fatal("JIRA_API_TOKEN is required")
	}
	if config.GitHub.PersonalAccessToken == "" {
		log.Fatal("GITHUB_PERSONAL_ACCESS_TOKEN is required")
	}
	if config.GitHub.BotUsername == "" {
		log.Fatal("GITHUB_BOT_USERNAME is required")
	}
	if config.GitHub.BotEmail == "" {
		log.Fatal("GITHUB_BOT_EMAIL is required")
	}
	if len(config.ComponentToRepo) == 0 {
		log.Fatal("At least one component_to_repo mapping is required")
	}

	// Create services
	jiraService := services.NewJiraService(config)
	githubService := services.NewGitHubService(config)

	// Create AI service based on provider selection
	var aiService services.AIService
	switch config.AIProvider {
	case "claude":
		aiService = services.NewClaudeService(config)
		log.Printf("Using Claude AI service")
	case "gemini":
		aiService = services.NewGeminiService(config)
		log.Printf("Using Gemini AI service")
	default:
		log.Fatalf("Unsupported AI provider: %s", config.AIProvider)
	}

	jiraIssueScannerService := services.NewJiraIssueScannerService(jiraService, githubService, aiService, config)
	prFeedbackScannerService := services.NewPRFeedbackScannerService(jiraService, githubService, aiService, config)

	// Start the Jira issue scanner service for periodic ticket scanning
	log.Println("Starting Jira issue scanner service...")
	jiraIssueScannerService.Start()

	// Start the PR feedback scanner service for processing PR review feedback
	log.Println("Starting PR feedback scanner service...")
	prFeedbackScannerService.Start()

	// Create HTTP server (simplified for health checks only)
	mux := http.NewServeMux()

	// Add a health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, "OK")
		if err != nil {
			return
		}
	})

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Server.Port),
		Handler: mux,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting server on port %d", config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Gracefully shutdown the scanner services
	log.Println("Shutting down scanner services...")
	jiraIssueScannerService.Stop()
	prFeedbackScannerService.Stop()

	// Gracefully shutdown the server
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}

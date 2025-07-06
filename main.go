package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jira-ai-issue-solver/handlers"
	"jira-ai-issue-solver/models"
	"jira-ai-issue-solver/services"
	"jira-ai-issue-solver/utils"

	"github.com/kelseyhightower/envconfig"
)

// loadConfig loads the configuration from environment variables
func loadConfig() *models.Config {
	config := &models.Config{}

	// Process environment variables using envconfig
	err := envconfig.Process("", config)
	if err != nil {
		log.Fatalf("Failed to process config: %v", err)
	}

	return config
}

func main() {
	// Parse command-line flags
	configFile := flag.String("config", "", "Path to config file")
	testMode := flag.Bool("test", false, "Run in test mode")
	flag.Parse()

	// Load configuration
	var config *models.Config

	if *configFile != "" {
		// Load from file if provided
		// For simplicity, we're not implementing file loading
		log.Printf("Config file loading not implemented, using environment variables")
		config = loadConfig()
	} else {
		// Load from environment variables
		config = loadConfig()
	}

	// Run tests if in test mode
	if *testMode {
		log.Println("Running in test mode")
		utils.RunTests(config)
		return
	}

	// Create services
	jiraService := services.NewJiraService(config)
	githubService := services.NewGitHubService(config)
	claudeService := services.NewClaudeService(config)

	// Register or refresh Jira webhook
	if config.Server.URL == "" {
		log.Println("SERVER_URL environment variable not set, skipping Jira webhook registration")
	} else {
		log.Println("Registering Jira webhook...")
		err := jiraService.RegisterOrRefreshWebhook(config.Server.URL)
		if err != nil {
			log.Printf("Failed to register Jira webhook: %v", err)
		} else {
			log.Println("Successfully registered Jira webhook")
		}
	}

	// Create handlers
	jiraHandler := handlers.NewJiraWebhookHandler(jiraService, githubService, claudeService, config)

	// Start janitor process
	jiraHandler.StartJanitor()

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/jira", jiraHandler.HandleWebhook)

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

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %d", config.Server.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Gracefully shutdown the server
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}

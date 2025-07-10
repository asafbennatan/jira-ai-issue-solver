package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jira-ai-issue-solver/models"
	"jira-ai-issue-solver/services"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// InitLogger initializes the global logger with appropriate configuration
func InitLogger(config *models.Config) {
	// Get log level from config
	level := getLogLevel(config.Logging.Level)

	// Create encoder config based on format
	var encoderConfig zapcore.EncoderConfig
	if config.Logging.Format == models.LogFormatJSON {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		// Console format (default)
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Create core based on format
	var core zapcore.Core
	if config.Logging.Format == models.LogFormatJSON {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
	} else {
		// Console format (default)
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
	}

	// Create logger
	Logger = zap.New(core)
}

// getLogLevel returns the log level based on config
func getLogLevel(level models.LogLevel) zapcore.Level {
	switch level {
	case models.LogLevelDebug:
		return zapcore.DebugLevel
	case models.LogLevelInfo:
		return zapcore.InfoLevel
	case models.LogLevelWarn:
		return zapcore.WarnLevel
	case models.LogLevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := models.LoadConfig(*configPath)
	if err != nil {
		// Use fmt for this error since logger isn't initialized yet
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	InitLogger(config)
	defer Logger.Sync()

	// Validate required configuration
	if config.Jira.BaseURL == "" {
		Logger.Fatal("JIRA_BASE_URL is required")
	}
	if config.Jira.Username == "" {
		Logger.Fatal("JIRA_USERNAME is required")
	}
	if config.Jira.APIToken == "" {
		Logger.Fatal("JIRA_API_TOKEN is required")
	}
	if config.GitHub.PersonalAccessToken == "" {
		Logger.Fatal("GITHUB_PERSONAL_ACCESS_TOKEN is required")
	}
	if config.GitHub.BotUsername == "" {
		Logger.Fatal("GITHUB_BOT_USERNAME is required")
	}
	if config.GitHub.BotEmail == "" {
		Logger.Fatal("GITHUB_BOT_EMAIL is required")
	}
	if len(config.ComponentToRepo) == 0 {
		Logger.Fatal("At least one component_to_repo mapping is required")
	}

	// Create services
	jiraService := services.NewJiraService(config)
	githubService := services.NewGitHubService(config, Logger)

	// Create AI service based on provider selection
	var aiService services.AIService
	switch config.AIProvider {
	case "claude":
		aiService = services.NewClaudeService(config, Logger)
		Logger.Info("Using Claude AI service")
	case "gemini":
		aiService = services.NewGeminiService(config, Logger)
		Logger.Info("Using Gemini AI service")
	default:
		Logger.Fatal("Unsupported AI provider", zap.String("provider", config.AIProvider))
	}

	jiraIssueScannerService := services.NewJiraIssueScannerService(jiraService, githubService, aiService, config, Logger)
	prFeedbackScannerService := services.NewPRFeedbackScannerService(jiraService, githubService, aiService, config, Logger)

	// Start the Jira issue scanner service for periodic ticket scanning
	Logger.Info("Starting Jira issue scanner service...")
	jiraIssueScannerService.Start()

	// Start the PR feedback scanner service for processing PR review feedback
	Logger.Info("Starting PR feedback scanner service...")
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
		Logger.Info("Starting server", zap.Int("port", config.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Gracefully shutdown the scanner services
	Logger.Info("Shutting down scanner services...")
	jiraIssueScannerService.Stop()
	prFeedbackScannerService.Stop()

	// Gracefully shutdown the server
	Logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		Logger.Fatal("Server shutdown failed", zap.Error(err))
	}

	Logger.Info("Server stopped")
}

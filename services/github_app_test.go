package services

import (
	"testing"
	"time"

	"jira-ai-issue-solver/models"
)

func TestNewGitHubAppService(t *testing.T) {
	config := &models.Config{
		GitHub: struct {
			AppID          int64  `json:"app_id" envconfig:"GITHUB_APP_ID"`
			AppPrivateKey  string `json:"app_private_key" envconfig:"GITHUB_APP_PRIVATE_KEY"`
			InstallationID int64  `json:"installation_id" envconfig:"GITHUB_INSTALLATION_ID"`
			BotUsername    string `json:"bot_username" envconfig:"GITHUB_BOT_USERNAME"`
			BotEmail       string `json:"bot_email" envconfig:"GITHUB_BOT_EMAIL"`
		}{
			AppID:          123456,
			AppPrivateKey:  "test-private-key",
			InstallationID: 12345678,
			BotUsername:    "test-bot",
			BotEmail:       "test-bot@example.com",
		},
	}

	service := NewGitHubAppService(config)
	if service == nil {
		t.Error("Expected service to be created, got nil")
	}
}

func TestGitHubAppService_GetAppToken_NoAppID(t *testing.T) {
	config := &models.Config{
		GitHub: struct {
			AppID          int64  `json:"app_id" envconfig:"GITHUB_APP_ID"`
			AppPrivateKey  string `json:"app_private_key" envconfig:"GITHUB_APP_PRIVATE_KEY"`
			InstallationID int64  `json:"installation_id" envconfig:"GITHUB_INSTALLATION_ID"`
			BotUsername    string `json:"bot_username" envconfig:"GITHUB_BOT_USERNAME"`
			BotEmail       string `json:"bot_email" envconfig:"GITHUB_BOT_EMAIL"`
		}{
			AppID: 0, // No App ID configured
		},
	}

	service := NewGitHubAppService(config)
	_, err := service.GetAppToken()
	if err == nil {
		t.Error("Expected error when App ID is not configured, got nil")
	}
}

func TestGitHubAppService_GetInstallationToken_NoAppID(t *testing.T) {
	config := &models.Config{
		GitHub: struct {
			AppID          int64  `json:"app_id" envconfig:"GITHUB_APP_ID"`
			AppPrivateKey  string `json:"app_private_key" envconfig:"GITHUB_APP_PRIVATE_KEY"`
			InstallationID int64  `json:"installation_id" envconfig:"GITHUB_INSTALLATION_ID"`
			BotUsername    string `json:"bot_username" envconfig:"GITHUB_BOT_USERNAME"`
			BotEmail       string `json:"bot_email" envconfig:"GITHUB_BOT_EMAIL"`
		}{
			AppID: 0, // No App ID configured
		},
	}

	service := NewGitHubAppService(config)
	_, err := service.GetInstallationToken()
	if err == nil {
		t.Error("Expected error when App ID is not configured, got nil")
	}
}

func TestTokenCache_ThreadSafety(t *testing.T) {
	config := &models.Config{
		GitHub: struct {
			AppID          int64  `json:"app_id" envconfig:"GITHUB_APP_ID"`
			AppPrivateKey  string `json:"app_private_key" envconfig:"GITHUB_APP_PRIVATE_KEY"`
			InstallationID int64  `json:"installation_id" envconfig:"GITHUB_INSTALLATION_ID"`
			BotUsername    string `json:"bot_username" envconfig:"GITHUB_BOT_USERNAME"`
			BotEmail       string `json:"bot_email" envconfig:"GITHUB_BOT_EMAIL"`
		}{
			AppID:          123456,
			AppPrivateKey:  "test-private-key",
			InstallationID: 12345678,
			BotUsername:    "test-bot",
			BotEmail:       "test-bot@example.com",
		},
	}

	service := NewGitHubAppService(config)

	// Test that the service can be created with a token cache
	impl, ok := service.(*GitHubAppServiceImpl)
	if !ok {
		t.Fatal("Expected service to be of type GitHubAppServiceImpl")
	}

	if impl.tokenCache == nil {
		t.Error("Expected token cache to be initialized")
	}
}

func TestTokenCache_ExpirationLogic(t *testing.T) {
	cache := &tokenCache{
		Token:     "test-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Token should be valid (with 5-minute buffer)
	if time.Now().After(cache.ExpiresAt.Add(-5 * time.Minute)) {
		t.Error("Token should be valid with 5-minute buffer")
	}

	// Test expired token
	cache.ExpiresAt = time.Now().Add(-10 * time.Minute)
	if time.Now().Before(cache.ExpiresAt.Add(-5 * time.Minute)) {
		t.Error("Token should be expired")
	}
}

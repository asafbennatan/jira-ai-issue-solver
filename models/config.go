package models

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server struct {
		Port int `json:"port" envconfig:"SERVER_PORT" default:"8080"`
		URL  string `json:"url" envconfig:"SERVER_URL"`
	} `json:"server"`

	// Jira configuration
	Jira struct {
		BaseURL   string `json:"base_url" envconfig:"JIRA_BASE_URL" default:"https://your-domain.atlassian.net"`
		Username  string `json:"username" envconfig:"JIRA_USERNAME"`
		APIToken  string `json:"api_token" envconfig:"JIRA_API_TOKEN"`
		WebhookSecret string `json:"webhook_secret" envconfig:"JIRA_WEBHOOK_SECRET"`
	} `json:"jira"`

	// GitHub configuration
	GitHub struct {
		APIToken     string `json:"api_token" envconfig:"GITHUB_API_TOKEN"`
		WebhookSecret string `json:"webhook_secret" envconfig:"GITHUB_WEBHOOK_SECRET"`
		Username     string `json:"username" envconfig:"GITHUB_USERNAME"`
		Email        string `json:"email" envconfig:"GITHUB_EMAIL"`
		BotUsername  string `json:"bot_username" envconfig:"GITHUB_BOT_USERNAME"`
		BotToken     string `json:"bot_token" envconfig:"GITHUB_BOT_TOKEN"`
		BotEmail     string `json:"bot_email" envconfig:"GITHUB_BOT_EMAIL"`
	} `json:"github"`

	// Claude CLI configuration
	Claude struct {
		Path                      string `json:"path" envconfig:"CLAUDE_CLI_PATH" default:"claude-cli"`
		Timeout                   int    `json:"timeout" envconfig:"CLAUDE_TIMEOUT" default:"300"`
		DangerouslySkipPermissions bool   `json:"dangerously_skip_permissions" envconfig:"CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS" default:"false"`
		AllowedTools              string `json:"allowed_tools" envconfig:"CLAUDE_ALLOWED_TOOLS"`
		DisallowedTools           string `json:"disallowed_tools" envconfig:"CLAUDE_DISALLOWED_TOOLS"`
	} `json:"claude"`

	// Temporary directory for cloning repositories
	TempDir string `json:"temp_dir" envconfig:"TEMP_DIR" default:"/tmp/jira-ai-issue-solver"`
}

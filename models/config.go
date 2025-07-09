package models

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server struct {
		Port int `yaml:"port" default:"8080"`
	} `yaml:"server"`

	// Jira configuration
	Jira struct {
		BaseURL              string `yaml:"base_url"`
		Username             string `yaml:"username"`
		APIToken             string `yaml:"api_token"`
		IntervalSeconds      int    `yaml:"interval_seconds" default:"300"`
		DisableErrorComments bool   `yaml:"disable_error_comments" default:"false"`
		StatusTransitions    struct {
			Todo       string `yaml:"todo" default:"To Do"`
			InProgress string `yaml:"in_progress" default:"In Progress"`
			InReview   string `yaml:"in_review" default:"In Review"`
		} `yaml:"status_transitions"`
	} `yaml:"jira"`

	// GitHub configuration
	GitHub struct {
		PersonalAccessToken string `yaml:"personal_access_token"`
		BotUsername         string `yaml:"bot_username"`
		BotEmail            string `yaml:"bot_email"`
	} `yaml:"github"`

	// AI Provider selection
	AIProvider string `yaml:"ai_provider" default:"claude"` // "claude" or "gemini"

	// Claude CLI configuration
	Claude struct {
		CLIPath                    string `yaml:"cli_path" default:"claude-cli"`
		Timeout                    int    `yaml:"timeout" default:"300"`
		DangerouslySkipPermissions bool   `yaml:"dangerously_skip_permissions" default:"false"`
		AllowedTools               string `yaml:"allowed_tools" default:"Bash Edit"`
		DisallowedTools            string `yaml:"disallowed_tools" default:"Python"`
	} `yaml:"claude"`

	// Gemini CLI configuration
	Gemini struct {
		CLIPath  string `yaml:"cli_path" default:"gemini"`
		Timeout  int    `yaml:"timeout" default:"300"`
		Model    string `yaml:"model" default:"gemini-2.5-pro"`
		AllFiles bool   `yaml:"all_files" default:"false"`
		Sandbox  bool   `yaml:"sandbox" default:"false"`
		APIKey   string `yaml:"api_key"`
	} `yaml:"gemini"`

	// Component to Repository mapping
	ComponentToRepo map[string]string `yaml:"component_to_repo"`

	// Temporary directory for cloning repositories
	TempDir string `yaml:"temp_dir" default:"/tmp/jira-ai-issue-solver"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Validate AI provider configuration
	if err := config.validateAIProvider(); err != nil {
		return nil, err
	}

	// Validate status transitions configuration
	if err := config.validateStatusTransitions(); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateAIProvider ensures only one AI provider is configured
func (c *Config) validateAIProvider() error {
	if c.AIProvider != "claude" && c.AIProvider != "gemini" {
		return errors.New("ai_provider must be either 'claude' or 'gemini'")
	}
	return nil
}

// validateStatusTransitions ensures status transitions are properly configured
func (c *Config) validateStatusTransitions() error {
	if c.Jira.StatusTransitions.Todo == "" {
		return errors.New("jira.status_transitions.todo cannot be empty")
	}
	if c.Jira.StatusTransitions.InProgress == "" {
		return errors.New("jira.status_transitions.in_progress cannot be empty")
	}
	if c.Jira.StatusTransitions.InReview == "" {
		return errors.New("jira.status_transitions.in_review cannot be empty")
	}
	return nil
}

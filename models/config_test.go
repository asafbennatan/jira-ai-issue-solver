package models

import (
	"os"
	"testing"
)

func TestConfig_validateStatusTransitions(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid status transitions",
			config: Config{
				Jira: struct {
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
				}{
					StatusTransitions: struct {
						Todo       string `yaml:"todo" default:"To Do"`
						InProgress string `yaml:"in_progress" default:"In Progress"`
						InReview   string `yaml:"in_review" default:"In Review"`
					}{
						Todo:       "To Do",
						InProgress: "In Progress",
						InReview:   "In Review",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty todo status",
			config: Config{
				Jira: struct {
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
				}{
					StatusTransitions: struct {
						Todo       string `yaml:"todo" default:"To Do"`
						InProgress string `yaml:"in_progress" default:"In Progress"`
						InReview   string `yaml:"in_review" default:"In Review"`
					}{
						Todo:       "",
						InProgress: "In Progress",
						InReview:   "In Review",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty in_progress status",
			config: Config{
				Jira: struct {
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
				}{
					StatusTransitions: struct {
						Todo       string `yaml:"todo" default:"To Do"`
						InProgress string `yaml:"in_progress" default:"In Progress"`
						InReview   string `yaml:"in_review" default:"In Review"`
					}{
						Todo:       "To Do",
						InProgress: "",
						InReview:   "In Review",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty in_review status",
			config: Config{
				Jira: struct {
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
				}{
					StatusTransitions: struct {
						Todo       string `yaml:"todo" default:"To Do"`
						InProgress string `yaml:"in_progress" default:"In Progress"`
						InReview   string `yaml:"in_review" default:"In Review"`
					}{
						Todo:       "To Do",
						InProgress: "In Progress",
						InReview:   "",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateStatusTransitions()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.validateStatusTransitions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_WithStatusTransitions(t *testing.T) {
	// Create a temporary config file
	configContent := `
jira:
  status_transitions:
    todo: "To Do"
    in_progress: "In Progress"
    in_review: "In Review"
github:
  target_branch: "develop"
`
	tmpfile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load the config
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify status transitions
	if config.Jira.StatusTransitions.Todo != "To Do" {
		t.Errorf("Expected todo status 'To Do', got '%s'", config.Jira.StatusTransitions.Todo)
	}
	if config.Jira.StatusTransitions.InProgress != "In Progress" {
		t.Errorf("Expected in_progress status 'In Progress', got '%s'", config.Jira.StatusTransitions.InProgress)
	}
	if config.Jira.StatusTransitions.InReview != "In Review" {
		t.Errorf("Expected in_review status 'In Review', got '%s'", config.Jira.StatusTransitions.InReview)
	}

	// Verify target branch
	if config.GitHub.TargetBranch != "develop" {
		t.Errorf("Expected target branch 'develop', got '%s'", config.GitHub.TargetBranch)
	}
}

func TestLoadConfig_WithDefaultTargetBranch(t *testing.T) {
	// Create a temporary config file without target_branch (should default to "main")
	configContent := `
jira:
  status_transitions:
    todo: "To Do"
    in_progress: "In Progress"
    in_review: "In Review"
`
	tmpfile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load the config
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify target branch defaults to "main"
	if config.GitHub.TargetBranch != "main" {
		t.Errorf("Expected default target branch 'main', got '%s'", config.GitHub.TargetBranch)
	}
}

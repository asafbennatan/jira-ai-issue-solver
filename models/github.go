package models

import "time"

// GitHubWebhook represents the webhook payload from GitHub
type GitHubWebhook struct {
	Action      string            `json:"action"`
	PullRequest GitHubPullRequest `json:"pull_request"`
	Repository  GitHubRepository  `json:"repository"`
	Sender      GitHubUser        `json:"sender"`
	Review      GitHubReview      `json:"review,omitempty"`
}

// GitHubPullRequest represents a GitHub pull request
type GitHubPullRequest struct {
	ID        int64      `json:"id"`
	Number    int        `json:"number"`
	State     string     `json:"state"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	HTMLURL   string     `json:"html_url"`
	User      GitHubUser `json:"user"`
	Head      GitHubRef  `json:"head"`
	Base      GitHubRef  `json:"base"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// GitHubRepository represents a GitHub repository
type GitHubRepository struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	FullName string     `json:"full_name"`
	Owner    GitHubUser `json:"owner"`
	HTMLURL  string     `json:"html_url"`
	CloneURL string     `json:"clone_url"`
	SSHURL   string     `json:"ssh_url"`
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// GitHubRef represents a Git reference in a GitHub pull request
type GitHubRef struct {
	Label string           `json:"label"`
	Ref   string           `json:"ref"`
	SHA   string           `json:"sha"`
	Repo  GitHubRepository `json:"repo"`
}

// GitHubReview represents a GitHub pull request review
type GitHubReview struct {
	ID          int64      `json:"id"`
	User        GitHubUser `json:"user"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	HTMLURL     string     `json:"html_url"`
	SubmittedAt time.Time  `json:"submitted_at"`
}

// GitHubCreatePRRequest represents the request to create a pull request
type GitHubCreatePRRequest struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Head   string   `json:"head"`
	Base   string   `json:"base"`
	Labels []string `json:"labels,omitempty"`
}

// GitHubCreatePRResponse represents the response from creating a pull request
type GitHubCreatePRResponse struct {
	ID        int64     `json:"id"`
	Number    int       `json:"number"`
	State     string    `json:"state"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

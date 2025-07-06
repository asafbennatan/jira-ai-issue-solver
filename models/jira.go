package models

import "time"

// JiraWebhookRegistration represents the request to register a webhook with Jira
type JiraWebhookRegistration struct {
	Name      string                 `json:"name"`
	URL       string                 `json:"url"`
	Events    []string               `json:"events"`
	Filters   map[string]interface{} `json:"jqlFilter,omitempty"`
	ExpirationDate int64             `json:"expirationDate,omitempty"`
	Enabled   bool                   `json:"enabled"`
}

// JiraWebhookResponse represents the response from Jira when registering a webhook
type JiraWebhookResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Events    []string `json:"events"`
	Filters   map[string]interface{} `json:"jqlFilter,omitempty"`
	ExpirationDate int64 `json:"expirationDate,omitempty"`
	Enabled   bool   `json:"enabled"`
	Self      string `json:"self"`
}

// JiraWebhook represents the webhook payload from Jira
type JiraWebhook struct {
	Timestamp    int64       `json:"timestamp"`
	WebhookEvent string      `json:"webhookEvent"`
	Issue        JiraIssue   `json:"issue"`
	User         JiraUser    `json:"user"`
	Changelog    JiraChangelog `json:"changelog,omitempty"`
}

// JiraIssue represents a Jira issue
type JiraIssue struct {
	ID     string      `json:"id"`
	Self   string      `json:"self"`
	Key    string      `json:"key"`
	Fields JiraFields  `json:"fields"`
}

// JiraFields represents the fields of a Jira issue
type JiraFields struct {
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	Status      JiraStatus   `json:"status"`
	Project     JiraProject  `json:"project"`
	Labels      []string     `json:"labels"`
	Created     time.Time    `json:"created"`
	Updated     time.Time    `json:"updated"`
	Creator     JiraUser     `json:"creator"`
	Reporter    JiraUser     `json:"reporter"`
	Comment     JiraComments `json:"comment,omitempty"`
}

// JiraStatus represents the status of a Jira issue
type JiraStatus struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// JiraProject represents a Jira project
type JiraProject struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Properties map[string]string `json:"properties"`
}

// JiraUser represents a Jira user
type JiraUser struct {
	Self         string `json:"self"`
	Name         string `json:"name"`
	Key          string `json:"key"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
}

// JiraComments represents the comments of a Jira issue
type JiraComments struct {
	Comments []JiraComment `json:"comments"`
}

// JiraComment represents a comment on a Jira issue
type JiraComment struct {
	ID      string    `json:"id"`
	Body    string    `json:"body"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Author  JiraUser  `json:"author"`
}

// JiraChangelog represents the changelog of a Jira issue
type JiraChangelog struct {
	ID    string        `json:"id"`
	Items []JiraChangeItem `json:"items"`
}

// JiraChangeItem represents an item in the changelog of a Jira issue
type JiraChangeItem struct {
	Field      string `json:"field"`
	FromString string `json:"fromString"`
	ToString   string `json:"toString"`
}

// JiraTicketResponse represents the response from the Jira API when fetching a ticket
type JiraTicketResponse struct {
	ID     string     `json:"id"`
	Key    string     `json:"key"`
	Self   string     `json:"self"`
	Fields JiraFields `json:"fields"`
}

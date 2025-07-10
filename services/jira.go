package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"jira-ai-issue-solver/models"
)

// JiraService defines the interface for interacting with Jira
type JiraService interface {
	// GetTicket fetches a ticket from Jira
	GetTicket(key string) (*models.JiraTicketResponse, error)

	// UpdateTicketLabels updates the labels of a ticket
	UpdateTicketLabels(key string, addLabels, removeLabels []string) error

	// UpdateTicketStatus updates the status of a ticket
	UpdateTicketStatus(key string, status string) error

	// UpdateTicketField updates a specific field of a ticket
	UpdateTicketField(key string, fieldID string, value interface{}) error

	// AddComment adds a comment to a ticket
	AddComment(key string, comment string) error

	// SearchTickets searches for tickets using JQL
	SearchTickets(jql string) (*models.JiraSearchResponse, error)
}

// JiraServiceImpl implements the JiraService interface
type JiraServiceImpl struct {
	config   *models.Config
	client   *http.Client
	executor models.CommandExecutor
}

// NewJiraService creates a new JiraService
func NewJiraService(config *models.Config, executor ...models.CommandExecutor) JiraService {
	commandExecutor := exec.Command
	if len(executor) > 0 {
		commandExecutor = executor[0]
	}
	return &JiraServiceImpl{
		config:   config,
		client:   &http.Client{},
		executor: commandExecutor,
	}
}

// GetTicket fetches a ticket from Jira
func (s *JiraServiceImpl) GetTicket(key string) (*models.JiraTicketResponse, error) {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s", s.config.Jira.BaseURL, key)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Jira.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get ticket: %s, status code: %d", string(body), resp.StatusCode)
	}

	var ticket models.JiraTicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ticket, nil
}

// UpdateTicketLabels updates the labels of a ticket
func (s *JiraServiceImpl) UpdateTicketLabels(key string, addLabels, removeLabels []string) error {
	// First, get the current labels
	ticket, err := s.GetTicket(key)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// Create a map of current labels for easy lookup
	currentLabels := make(map[string]bool)
	for _, label := range ticket.Fields.Labels {
		currentLabels[label] = true
	}

	// Remove labels
	for _, label := range removeLabels {
		delete(currentLabels, label)
	}

	// Add labels
	for _, label := range addLabels {
		currentLabels[label] = true
	}

	// Convert map back to slice
	labels := make([]string, 0, len(currentLabels))
	for label := range currentLabels {
		labels = append(labels, label)
	}

	// Update the ticket
	url := fmt.Sprintf("%s/rest/api/2/issue/%s", s.config.Jira.BaseURL, key)

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"labels": labels,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Jira.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update ticket labels: %s, status code: %d", string(body), resp.StatusCode)
	}

	return nil
}

// UpdateTicketStatus updates the status of a ticket
func (s *JiraServiceImpl) UpdateTicketStatus(key string, status string) error {
	// Get available transitions
	url := fmt.Sprintf("%s/rest/api/2/issue/%s/transitions", s.config.Jira.BaseURL, key)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Jira.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get transitions: %s, status code: %d", string(body), resp.StatusCode)
	}

	var transitions struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&transitions); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Find the transition ID for the target status
	var transitionID string
	for _, transition := range transitions.Transitions {
		if strings.EqualFold(transition.To.Name, status) {
			transitionID = transition.ID
			break
		}
	}

	if transitionID == "" {
		return fmt.Errorf("no transition found for status: %s", status)
	}

	// Perform the transition
	payload := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Jira.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err = s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update ticket status: %s, status code: %d", string(body), resp.StatusCode)
	}

	return nil
}

// AddComment adds a comment to a ticket
func (s *JiraServiceImpl) AddComment(key string, comment string) error {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s/comment", s.config.Jira.BaseURL, key)

	payload := map[string]string{
		"body": comment,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Jira.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add comment: %s, status code: %d", string(body), resp.StatusCode)
	}

	return nil
}

// UpdateTicketField updates a specific field of a ticket
func (s *JiraServiceImpl) UpdateTicketField(key string, fieldID string, value interface{}) error {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s", s.config.Jira.BaseURL, key)

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			fieldID: value,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Jira.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update ticket field %s: %s, status code: %d", fieldID, string(body), resp.StatusCode)
	}

	return nil
}

// SearchTickets searches for tickets using JQL
func (s *JiraServiceImpl) SearchTickets(jql string) (*models.JiraSearchResponse, error) {
	url := fmt.Sprintf("%s/rest/api/2/search", s.config.Jira.BaseURL)

	payload := map[string]interface{}{
		"jql":        jql,
		"startAt":    0,
		"maxResults": 100,
		"fields":     []string{"summary", "description", "status", "project", "components", "labels", "created", "updated", "creator", "reporter"},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Jira.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to search tickets: %s, status code: %d", string(body), resp.StatusCode)
	}

	var searchResponse models.JiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &searchResponse, nil
}

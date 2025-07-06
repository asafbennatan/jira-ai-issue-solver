package mocks

import (
	"jira-ai-issue-solver/models"
)

// MockJiraService is a mock implementation of the JiraService interface
type MockJiraService struct {
	GetTicketFunc                func(key string) (*models.JiraTicketResponse, error)
	UpdateTicketLabelsFunc       func(key string, addLabels, removeLabels []string) error
	UpdateTicketStatusFunc       func(key string, status string) error
	AddCommentFunc               func(key string, comment string) error
	ValidateWebhookSignatureFunc func(body []byte, signature string) bool
	RegisterWebhookFunc          func(webhook *models.JiraWebhookRegistration) (*models.JiraWebhookResponse, error)
	GetWebhooksFunc              func() ([]models.JiraWebhookResponse, error)
	DeleteWebhookFunc            func(webhookID int) error
	RegisterOrRefreshWebhookFunc func(serverURL string) error
}

// GetTicket is the mock implementation of JiraService's GetTicket method
func (m *MockJiraService) GetTicket(key string) (*models.JiraTicketResponse, error) {
	if m.GetTicketFunc != nil {
		return m.GetTicketFunc(key)
	}
	return nil, nil
}

// UpdateTicketLabels is the mock implementation of JiraService's UpdateTicketLabels method
func (m *MockJiraService) UpdateTicketLabels(key string, addLabels, removeLabels []string) error {
	if m.UpdateTicketLabelsFunc != nil {
		return m.UpdateTicketLabelsFunc(key, addLabels, removeLabels)
	}
	return nil
}

// UpdateTicketStatus is the mock implementation of JiraService's UpdateTicketStatus method
func (m *MockJiraService) UpdateTicketStatus(key string, status string) error {
	if m.UpdateTicketStatusFunc != nil {
		return m.UpdateTicketStatusFunc(key, status)
	}
	return nil
}

// AddComment is the mock implementation of JiraService's AddComment method
func (m *MockJiraService) AddComment(key string, comment string) error {
	if m.AddCommentFunc != nil {
		return m.AddCommentFunc(key, comment)
	}
	return nil
}

// ValidateWebhookSignature is the mock implementation of JiraService's ValidateWebhookSignature method
func (m *MockJiraService) ValidateWebhookSignature(body []byte, signature string) bool {
	if m.ValidateWebhookSignatureFunc != nil {
		return m.ValidateWebhookSignatureFunc(body, signature)
	}
	return true
}

// RegisterWebhook is the mock implementation of JiraService's RegisterWebhook method
func (m *MockJiraService) RegisterWebhook(webhook *models.JiraWebhookRegistration) (*models.JiraWebhookResponse, error) {
	if m.RegisterWebhookFunc != nil {
		return m.RegisterWebhookFunc(webhook)
	}
	return nil, nil
}

// GetWebhooks is the mock implementation of JiraService's GetWebhooks method
func (m *MockJiraService) GetWebhooks() ([]models.JiraWebhookResponse, error) {
	if m.GetWebhooksFunc != nil {
		return m.GetWebhooksFunc()
	}
	return nil, nil
}

// DeleteWebhook is the mock implementation of JiraService's DeleteWebhook method
func (m *MockJiraService) DeleteWebhook(webhookID int) error {
	if m.DeleteWebhookFunc != nil {
		return m.DeleteWebhookFunc(webhookID)
	}
	return nil
}

// RegisterOrRefreshWebhook is the mock implementation of JiraService's RegisterOrRefreshWebhook method
func (m *MockJiraService) RegisterOrRefreshWebhook(serverURL string) error {
	if m.RegisterOrRefreshWebhookFunc != nil {
		return m.RegisterOrRefreshWebhookFunc(serverURL)
	}
	return nil
}

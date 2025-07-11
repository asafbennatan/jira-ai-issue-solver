package mocks

import (
	"jira-ai-issue-solver/models"
)

// MockJiraService is a mock implementation of the JiraService interface
type MockJiraService struct {
	GetTicketFunc                   func(key string) (*models.JiraTicketResponse, error)
	GetTicketWithExpandedFieldsFunc func(key string) (map[string]interface{}, map[string]string, error)
	UpdateTicketLabelsFunc          func(key string, addLabels, removeLabels []string) error
	UpdateTicketStatusFunc          func(key string, status string) error
	UpdateTicketFieldFunc           func(key string, fieldID string, value interface{}) error
	UpdateTicketFieldByNameFunc     func(key string, fieldName string, value interface{}) error
	GetFieldIDByNameFunc            func(fieldName string) (string, error)
	AddCommentFunc                  func(key string, comment string) error
	SearchTicketsFunc               func(jql string) (*models.JiraSearchResponse, error)
}

// GetTicket is the mock implementation of JiraService's GetTicket method
func (m *MockJiraService) GetTicket(key string) (*models.JiraTicketResponse, error) {
	if m.GetTicketFunc != nil {
		return m.GetTicketFunc(key)
	}
	return nil, nil
}

// GetTicketWithExpandedFields is the mock implementation of JiraService's GetTicketWithExpandedFields method
func (m *MockJiraService) GetTicketWithExpandedFields(key string) (map[string]interface{}, map[string]string, error) {
	if m.GetTicketWithExpandedFieldsFunc != nil {
		return m.GetTicketWithExpandedFieldsFunc(key)
	}
	return nil, nil, nil
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

// UpdateTicketField is the mock implementation of JiraService's UpdateTicketField method
func (m *MockJiraService) UpdateTicketField(key string, fieldID string, value interface{}) error {
	if m.UpdateTicketFieldFunc != nil {
		return m.UpdateTicketFieldFunc(key, fieldID, value)
	}
	return nil
}

// UpdateTicketFieldByName is the mock implementation of JiraService's UpdateTicketFieldByName method
func (m *MockJiraService) UpdateTicketFieldByName(key string, fieldName string, value interface{}) error {
	if m.UpdateTicketFieldByNameFunc != nil {
		return m.UpdateTicketFieldByNameFunc(key, fieldName, value)
	}
	return nil
}

// GetFieldIDByName is the mock implementation of JiraService's GetFieldIDByName method
func (m *MockJiraService) GetFieldIDByName(fieldName string) (string, error) {
	if m.GetFieldIDByNameFunc != nil {
		return m.GetFieldIDByNameFunc(fieldName)
	}
	return "", nil
}

// AddComment is the mock implementation of JiraService's AddComment method
func (m *MockJiraService) AddComment(key string, comment string) error {
	if m.AddCommentFunc != nil {
		return m.AddCommentFunc(key, comment)
	}
	return nil
}

// SearchTickets is the mock implementation of JiraService's SearchTickets method
func (m *MockJiraService) SearchTickets(jql string) (*models.JiraSearchResponse, error) {
	if m.SearchTicketsFunc != nil {
		return m.SearchTicketsFunc(jql)
	}
	return nil, nil
}

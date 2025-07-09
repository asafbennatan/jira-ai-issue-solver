package services

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"jira-ai-issue-solver/models"
)

// RoundTripFunc is a function type that implements http.RoundTripper
type RoundTripFunc func(req *http.Request) (*http.Response, error)

// RoundTrip executes the mock round trip
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// NewTestClient returns a mock http.Client that will execute the provided function instead of making a real HTTP request
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

// TestGetTicket tests the GetTicket method
func TestGetTicket(t *testing.T) {
	// Test cases
	testCases := []struct {
		name           string
		key            string
		mockResponse   *http.Response
		mockError      error
		expectedTicket *models.JiraTicketResponse
		expectedError  bool
	}{
		{
			name: "successful request",
			key:  "TEST-123",
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"id": "12345",
					"key": "TEST-123",
					"self": "https://jira.example.com/rest/api/2/issue/12345",
					"fields": {
						"summary": "Test ticket",
						"description": "This is a test ticket",
						"status": {
							"id": "1",
							"name": "Open"
						},
						"project": {
							"id": "10000",
							"key": "TEST",
							"name": "Test Project",
							"properties": {
								"ai.bot.github.repo": "https://github.com/example/repo.git"
							}
						},
						"labels": ["good-for-ai"],
						"created": "2023-01-01T00:00:00.000Z",
						"updated": "2023-01-02T00:00:00.000Z"
					}
				}`))),
			},
			mockError: nil,
			expectedTicket: &models.JiraTicketResponse{
				ID:   "12345",
				Key:  "TEST-123",
				Self: "https://jira.example.com/rest/api/2/issue/12345",
				Fields: models.JiraFields{
					Summary:     "Test ticket",
					Description: "This is a test ticket",
					Status: models.JiraStatus{
						ID:   "1",
						Name: "Open",
					},
					Project: models.JiraProject{
						ID:   "10000",
						Key:  "TEST",
						Name: "Test Project",
						Properties: map[string]string{
							"ai.bot.github.repo": "https://github.com/example/repo.git",
						},
					},
					Labels: []string{"good-for-ai"},
				},
			},
			expectedError: false,
		},
		{
			name: "error response",
			key:  "TEST-456",
			mockResponse: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"errorMessages":["Issue does not exist or you do not have permission to see it."],"errors":{}}`))),
			},
			mockError:      nil,
			expectedTicket: nil,
			expectedError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock HTTP client
			mockClient := NewTestClient(func(req *http.Request) (*http.Response, error) {
				return tc.mockResponse, tc.mockError
			})

			// Create a JiraService with the mock client
			config := &models.Config{}
			config.Jira.BaseURL = "https://jira.example.com"
			config.Jira.Username = "test-user"
			config.Jira.APIToken = "test-token"

			service := &JiraServiceImpl{
				config:   config,
				client:   mockClient,
				executor: execCommand,
			}

			// Call the method being tested
			ticket, err := service.GetTicket(tc.key)

			// Check the results
			if tc.expectedError && err == nil {
				t.Errorf("Expected an error but got nil")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tc.expectedTicket != nil {
				if ticket == nil {
					t.Errorf("Expected a ticket but got nil")
				} else {
					if ticket.ID != tc.expectedTicket.ID {
						t.Errorf("Expected ticket ID %s but got %s", tc.expectedTicket.ID, ticket.ID)
					}
					if ticket.Key != tc.expectedTicket.Key {
						t.Errorf("Expected ticket Key %s but got %s", tc.expectedTicket.Key, ticket.Key)
					}
					// Add more assertions for other fields as needed
				}
			}
		})
	}
}

// TestUpdateTicketLabels tests the UpdateTicketLabels method
func TestUpdateTicketLabels(t *testing.T) {
	// Test cases
	testCases := []struct {
		name          string
		key           string
		addLabels     []string
		removeLabels  []string
		mockResponses []*http.Response
		mockErrors    []error
		expectedError bool
	}{
		{
			name:         "successful update",
			key:          "TEST-123",
			addLabels:    []string{"ai-in-progress"},
			removeLabels: []string{"good-for-ai"},
			mockResponses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`{
						"id": "12345",
						"key": "TEST-123",
						"self": "https://jira.example.com/rest/api/2/issue/12345",
						"fields": {
							"labels": ["good-for-ai"]
						}
					}`))),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(bytes.NewReader([]byte(``))),
				},
			},
			mockErrors:    []error{nil, nil},
			expectedError: false,
		},
		{
			name:         "error getting ticket",
			key:          "TEST-456",
			addLabels:    []string{"ai-in-progress"},
			removeLabels: []string{"good-for-ai"},
			mockResponses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"errorMessages":["Issue does not exist or you do not have permission to see it."],"errors":{}}`))),
				},
			},
			mockErrors:    []error{nil},
			expectedError: true,
		},
		{
			name:         "error updating ticket",
			key:          "TEST-789",
			addLabels:    []string{"ai-in-progress"},
			removeLabels: []string{"good-for-ai"},
			mockResponses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`{
						"id": "12345",
						"key": "TEST-789",
						"self": "https://jira.example.com/rest/api/2/issue/12345",
						"fields": {
							"labels": ["good-for-ai"]
						}
					}`))),
				},
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"errorMessages":["Error updating labels"],"errors":{}}`))),
				},
			},
			mockErrors:    []error{nil, nil},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock HTTP client with a counter for multiple requests
			callCount := 0
			mockClient := NewTestClient(func(req *http.Request) (*http.Response, error) {
				if callCount >= len(tc.mockResponses) {
					t.Fatalf("Unexpected request: %s %s", req.Method, req.URL.String())
				}
				response := tc.mockResponses[callCount]
				err := tc.mockErrors[callCount]
				callCount++
				return response, err
			})

			// Create a JiraService with the mock client
			config := &models.Config{}
			config.Jira.BaseURL = "https://jira.example.com"
			config.Jira.Username = "test-user"
			config.Jira.APIToken = "test-token"

			service := &JiraServiceImpl{
				config:   config,
				client:   mockClient,
				executor: execCommand,
			}

			// Call the method being tested
			err := service.UpdateTicketLabels(tc.key, tc.addLabels, tc.removeLabels)

			// Check the results
			if tc.expectedError && err == nil {
				t.Errorf("Expected an error but got nil")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

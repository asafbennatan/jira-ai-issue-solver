package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"jira-ai-issue-solver/mocks"
	"jira-ai-issue-solver/models"
)

// TestHandleWebhook tests the HandleWebhook method of JiraWebhookHandler
func TestHandleWebhook(t *testing.T) {
	// Test cases
	testCases := []struct {
		name               string
		requestBody        string
		signature          string
		validateSignature  bool
		expectedStatusCode int
		updateLabelsError  error
	}{
		{
			name: "valid webhook with good-for-ai label",
			requestBody: `{
				"webhookEvent": "jira:issue_updated",
				"issue": {
					"id": "12345",
					"key": "TEST-123",
					"fields": {
						"summary": "Test ticket",
						"description": "This is a test ticket",
						"labels": ["good-for-ai"],
						"project": {
							"id": "10000",
							"key": "TEST",
							"name": "Test Project",
							"properties": {
								"ai.bot.github.repo": "https://github.com/example/repo.git"
							}
						}
					}
				}
			}`,
			signature:          "test-signature",
			validateSignature:  true,
			expectedStatusCode: http.StatusOK,
			updateLabelsError:  nil,
		},
		{
			name: "invalid signature",
			requestBody: `{
				"webhookEvent": "jira:issue_updated",
				"issue": {
					"id": "12345",
					"key": "TEST-123",
					"fields": {
						"summary": "Test ticket",
						"description": "This is a test ticket",
						"labels": ["good-for-ai"],
						"project": {
							"id": "10000",
							"key": "TEST",
							"name": "Test Project",
							"properties": {
								"ai.bot.github.repo": "https://github.com/example/repo.git"
							}
						}
					}
				}
			}`,
			signature:          "invalid-signature",
			validateSignature:  false,
			expectedStatusCode: http.StatusUnauthorized,
			updateLabelsError:  nil,
		},
		{
			name: "not an issue update event",
			requestBody: `{
				"webhookEvent": "jira:issue_created",
				"issue": {
					"id": "12345",
					"key": "TEST-123",
					"fields": {
						"summary": "Test ticket",
						"description": "This is a test ticket",
						"labels": ["good-for-ai"],
						"project": {
							"id": "10000",
							"key": "TEST",
							"name": "Test Project",
							"properties": {
								"ai.bot.github.repo": "https://github.com/example/repo.git"
							}
						}
					}
				}
			}`,
			signature:          "test-signature",
			validateSignature:  true,
			expectedStatusCode: http.StatusOK,
			updateLabelsError:  nil,
		},
		{
			name: "issue without good-for-ai label",
			requestBody: `{
				"webhookEvent": "jira:issue_updated",
				"issue": {
					"id": "12345",
					"key": "TEST-123",
					"fields": {
						"summary": "Test ticket",
						"description": "This is a test ticket",
						"labels": ["other-label"],
						"project": {
							"id": "10000",
							"key": "TEST",
							"name": "Test Project",
							"properties": {
								"ai.bot.github.repo": "https://github.com/example/repo.git"
							}
						}
					}
				}
			}`,
			signature:          "test-signature",
			validateSignature:  true,
			expectedStatusCode: http.StatusOK,
			updateLabelsError:  nil,
		},
		{
			name: "issue with ai-in-progress label",
			requestBody: `{
				"webhookEvent": "jira:issue_updated",
				"issue": {
					"id": "12345",
					"key": "TEST-123",
					"fields": {
						"summary": "Test ticket",
						"description": "This is a test ticket",
						"labels": ["good-for-ai", "ai-in-progress"],
						"project": {
							"id": "10000",
							"key": "TEST",
							"name": "Test Project",
							"properties": {
								"ai.bot.github.repo": "https://github.com/example/repo.git"
							}
						}
					}
				}
			}`,
			signature:          "test-signature",
			validateSignature:  true,
			expectedStatusCode: http.StatusOK,
			updateLabelsError:  nil,
		},
		{
			name: "error updating labels",
			requestBody: `{
				"webhookEvent": "jira:issue_updated",
				"issue": {
					"id": "12345",
					"key": "TEST-123",
					"fields": {
						"summary": "Test ticket",
						"description": "This is a test ticket",
						"labels": ["good-for-ai"],
						"project": {
							"id": "10000",
							"key": "TEST",
							"name": "Test Project",
							"properties": {
								"ai.bot.github.repo": "https://github.com/example/repo.git"
							}
						}
					}
				}
			}`,
			signature:          "test-signature",
			validateSignature:  true,
			expectedStatusCode: http.StatusInternalServerError,
			updateLabelsError:  errors.New("failed to update labels"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock services
			mockJiraService := &mocks.MockJiraService{
				ValidateWebhookSignatureFunc: func(body []byte, signature string) bool {
					return tc.validateSignature
				},
				UpdateTicketLabelsFunc: func(key string, addLabels, removeLabels []string) error {
					return tc.updateLabelsError
				},
				GetTicketFunc: func(key string) (*models.JiraTicketResponse, error) {
					return &models.JiraTicketResponse{
						ID:   "12345",
						Key:  key,
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
					}, nil
				},
				AddCommentFunc: func(key string, comment string) error {
					return nil
				},
			}

			mockGitHubService := &mocks.MockGitHubService{
				CheckForkExistsFunc: func(owner, repo string) (exists bool, cloneURL string, err error) {
					return false, "https://github.com/bot/repo.git", nil
				},
				ForkRepositoryFunc: func(owner, repo string) (string, error) {
					return "https://github.com/bot/repo.git", nil
				},
				CloneRepositoryFunc: func(repoURL, directory string) error {
					return nil
				},
				CreateBranchFunc: func(directory, branchName string) error {
					return nil
				},
				CommitChangesFunc: func(directory, message string) error {
					return nil
				},
				PushChangesFunc: func(directory, branchName string) error {
					return nil
				},
				CreatePullRequestFunc: func(owner, repo, title, body, head, base string) (*models.GitHubCreatePRResponse, error) {
					return &models.GitHubCreatePRResponse{
						HTMLURL: "https://github.com/example/repo/pull/1",
					}, nil
				},
			}

			mockClaudeService := &mocks.MockClaudeService{}

			// Create the handler with a temp directory
			config := &models.Config{
				TempDir: t.TempDir(),
			}
			config.GitHub.BotUsername = "bot"
			config.GitHub.BotEmail = "bot@example.com"
			handler := NewJiraWebhookHandler(mockJiraService, mockGitHubService, mockClaudeService, config)

			// Create a test request
			req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(tc.requestBody))
			req.Header.Set("X-Jira-Signature", tc.signature)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Call the handler
			handler.HandleWebhook(w, req)

			// Check the response
			if w.Code != tc.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatusCode, w.Code)
			}
		})
	}
}

package services

// AIService defines the unified interface for AI services
type AIService interface {
	// GenerateCode generates code using the AI service
	GenerateCode(prompt string, repoDir string) (interface{}, error)
	// GenerateDocumentation generates documentation file (CLAUDE.md or GEMINI.md) if it doesn't exist
	GenerateDocumentation(repoDir string) error
}

// AIResponse represents a generic AI response that can be used by consumers
type AIResponse struct {
	Type         string      `json:"type"`
	IsError      bool        `json:"is_error"`
	Result       string      `json:"result"`
	SessionID    string      `json:"session_id"`
	TotalCostUsd float64     `json:"total_cost_usd"`
	Usage        interface{} `json:"usage"`
	Message      interface{} `json:"message"`
}

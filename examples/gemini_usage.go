package main

import (
	"fmt"
	"jira-ai-issue-solver/models"
	"jira-ai-issue-solver/services"
	"log"
)

func main() {
	// Load configuration
	config, err := models.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create AI service with Gemini
	aiService := services.NewAIService(config, services.AIServiceGemini)

	fmt.Printf("Using AI service: %s\n", aiService.GetServiceType())

	// Example prompt
	prompt := "# Task\n\n" +
		"## Add logging to user authentication\n\n" +
		"Please add comprehensive logging to the user authentication system to help with debugging and monitoring.\n\n" +
		"# Instructions\n\n" +
		"1. Analyze the current authentication code\n" +
		"2. Add appropriate logging statements\n" +
		"3. Ensure logs include relevant context (user ID, IP address, success/failure)\n" +
		"4. Use structured logging format\n" +
		"5. Provide a summary of the changes made\n\n" +
		"# Output Format\n\n" +
		"Please provide your response in the following format:\n\n" +
		"```\n" +
		"## Summary\n" +
		"<A brief summary of the changes made>\n\n" +
		"## Changes Made\n" +
		"<List of files modified and a description of the changes>\n\n" +
		"## Testing\n" +
		"<Description of how the changes were tested>\n" +
		"```\n"

	// Generate code using Gemini
	result, err := aiService.GenerateCode(prompt, "/path/to/your/repo")
	if err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	// Handle the result based on service type
	switch aiService.GetServiceType() {
	case services.AIServiceGemini:
		if geminiResult, ok := result.(*services.GeminiResponse); ok {
			fmt.Printf("Gemini Response:\n%s\n", geminiResult.Result)
		}
	case services.AIServiceClaude:
		if claudeResult, ok := result.(*services.ClaudeResponse); ok {
			fmt.Printf("Claude Response:\n%s\n", claudeResult.Result)
		}
	}
}

// Example of switching between AI services
func exampleSwitchAIServices() {
	config, _ := models.LoadConfig("config.yaml")

	// Use Claude
	claudeService := services.NewAIService(config, services.AIServiceClaude)
	fmt.Printf("Service type: %s\n", claudeService.GetServiceType())

	// Use Gemini
	geminiService := services.NewAIService(config, services.AIServiceGemini)
	fmt.Printf("Service type: %s\n", geminiService.GetServiceType())

	// Use default (Claude)
	defaultService := services.NewAIService(config, services.AIServiceClaude)
	fmt.Printf("Service type: %s\n", defaultService.GetServiceType())
}

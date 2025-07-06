package models

// JiraTicketLabel represents the possible labels for a Jira ticket
type JiraTicketLabel string

// Jira ticket labels
const (
	// LabelGoodForAI indicates that the ticket should be processed by the AI
	LabelGoodForAI JiraTicketLabel = "good-for-ai"
	
	// LabelAIInProgress indicates that the AI is currently processing the ticket
	LabelAIInProgress JiraTicketLabel = "ai-in-progress"
	
	// LabelAIPRCreated indicates that the AI has created a PR for the ticket
	LabelAIPRCreated JiraTicketLabel = "ai-pr-created"
	
	// LabelAIFixingPR indicates that the AI is fixing a PR based on feedback
	LabelAIFixingPR JiraTicketLabel = "ai-fixing-pr"
	
	// LabelAIFailed indicates that the AI failed to process the ticket
	LabelAIFailed JiraTicketLabel = "ai-failed"
)

// String returns the string representation of a JiraTicketLabel
func (l JiraTicketLabel) String() string {
	return string(l)
}
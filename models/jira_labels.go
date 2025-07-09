package models

// JiraTicketLabel represents the possible labels for a Jira ticket
type JiraTicketLabel string

// Jira ticket labels
const (
	// LabelGoodForAI indicates that the ticket should be processed by the AI
	LabelGoodForAI JiraTicketLabel = "good-for-ai"
)

// String returns the string representation of a JiraTicketLabel
func (l JiraTicketLabel) String() string {
	return string(l)
}

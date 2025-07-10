package mocks

type MockTicketProcessor struct {
	ProcessTicketFunc func(key string) error
}

func (m *MockTicketProcessor) ProcessTicket(key string) error {
	if m.ProcessTicketFunc != nil {
		return m.ProcessTicketFunc(key)
	}
	return nil
}

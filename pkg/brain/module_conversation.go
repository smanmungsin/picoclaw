package brain

import "sync"

// ConversationModule stores and retrieves conversation history
type ConversationModule struct {
	messages   []string
	mu         sync.RWMutex
	OnAdd      func(msg string)
}

func NewConversationModule() *ConversationModule {
	return &ConversationModule{
		messages: make([]string, 0, 1024),
	}
}

func (m *ConversationModule) AddMessage(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	if m.OnAdd != nil {
		 go m.OnAdd(msg)
	}
}

func (m *ConversationModule) GetHistory(n int) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if n > len(m.messages) {
		n = len(m.messages)
	}
	return m.messages[len(m.messages)-n:]
}

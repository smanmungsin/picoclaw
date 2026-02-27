package brain

import "sync"

// KnowledgeModule stores facts, Q&A, and documentation
// This can be extended to use embeddings for semantic search

type KnowledgeEntry struct {
	Question string
	Answer   string
}

type KnowledgeModule struct {
	entries []KnowledgeEntry
	mu      sync.RWMutex
}

func NewKnowledgeModule() *KnowledgeModule {
	return &KnowledgeModule{
		entries: make([]KnowledgeEntry, 0, 256),
	}
}

func (m *KnowledgeModule) Add(q, a string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, KnowledgeEntry{Question: q, Answer: a})
}

func (m *KnowledgeModule) Find(q string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.entries {
		if e.Question == q {
			return e.Answer, true
		}
	}
	return "", false
}

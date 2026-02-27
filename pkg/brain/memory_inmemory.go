package brain

import (
	"sync"
)

type InMemoryMemory struct {
	store map[string]any
	mu    sync.RWMutex
}

func NewInMemoryMemory() *InMemoryMemory {
	return &InMemoryMemory{
		store: make(map[string]any),
	}
}

func (m *InMemoryMemory) Remember(key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
	return nil
}

func (m *InMemoryMemory) Recall(key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.store[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (m *InMemoryMemory) Search(query string) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	results := []any{}
	for k, v := range m.store {
		if k == query {
			results = append(results, v)
		}
	}
	return results, nil
}

func (m *InMemoryMemory) Forget(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
	return nil
}

func (m *InMemoryMemory) ListKeys() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.store))
	for k := range m.store {
		keys = append(keys, k)
	}
	return keys, nil
}

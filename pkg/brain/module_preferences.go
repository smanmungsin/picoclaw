package brain

import "sync"

// PreferencesModule stores user preferences and settings
type PreferencesModule struct {
	prefs map[string]any
	mu    sync.RWMutex
}

func NewPreferencesModule() *PreferencesModule {
	return &PreferencesModule{
		prefs: make(map[string]any),
	}
}

func (m *PreferencesModule) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prefs[key] = value
}

func (m *PreferencesModule) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.prefs[key]
	return val, ok
}

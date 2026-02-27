package brain

import (
	"sync"
	"time"
)

// MemoryModule is a generic interface for a memory module (short-term, long-term, etc.)
type MemoryModule interface {
	Remember(key string, value any) error
	Recall(key string) (any, error)
	Search(query string) ([]any, error)
	Forget(key string) error
	ListKeys() ([]string, error)
}

// Event represents a single event in the brain's timeline
type Event struct {
	Timestamp time.Time
	Type      string
	Data      any
}

// Brain is the main struct combining all memory modules and event sourcing
type Brain struct {
	ShortTerm     MemoryModule
	LongTerm      MemoryModule
	Timeline      []Event
	mu            sync.RWMutex
	SelfReflect   *SelfReflectionModule
	reflectEvery  int
	eventCount    int
	reflectOn     map[string]bool // event types that trigger reflection
	ReportFunc    func(summary string)
	modules       map[string]any // extensible modules (conversation, todo, etc.)
	Security      *SecurityModule
}

func NewBrain(shortTerm, longTerm MemoryModule) *Brain {
   return &Brain{
	   ShortTerm:    shortTerm,
	   LongTerm:     longTerm,
	   Timeline:     make([]Event, 0, 1024),
	   SelfReflect:  NewSelfReflectionModule(),
	   reflectEvery: 10, // Reflect every 10 events by default
	   reflectOn:    map[string]bool{"inbound_message": true, "user_feedback": true},
	   ReportFunc:   nil,
	   modules:      make(map[string]any),
	   Security:     NewSecurityModule(),
   }
}

// LogEvent appends an event to the timeline and triggers self-reflection if needed
func (b *Brain) LogEvent(eventType string, data any) error {
   b.mu.Lock()
   defer b.mu.Unlock()
   b.Timeline = append(b.Timeline, Event{
	   Timestamp: time.Now(),
	   Type:      eventType,
	   Data:      data,
   })
   b.eventCount++
   // Adaptive: reflect on certain event types or every N events
   if b.reflectOn[eventType] || b.eventCount >= b.reflectEvery {
	   go b.reflectAndReport()
	   b.eventCount = 0
   }
   return nil
}

// SetReflectionFrequency allows user/agent to tune learning frequency
func (b *Brain) SetReflectionFrequency(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reflectEvery = n
}

// RegisterModule allows dynamic extension of the brain with new modules
func (b *Brain) RegisterModule(name string, module any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.modules[name] = module
}

// GetModule retrieves a registered module by name
func (b *Brain) GetModule(name string) (any, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	m, ok := b.modules[name]
	return m, ok
}

// SetReflectionEvents allows user/agent to tune which events trigger reflection
func (b *Brain) SetReflectionEvents(events []string) {
   b.mu.Lock()
   defer b.mu.Unlock()
   b.reflectOn = make(map[string]bool)
   for _, e := range events {
	   b.reflectOn[e] = true
   }
}

// SetReportFunc allows user/agent to set a callback for reflection reports
func (b *Brain) SetReportFunc(f func(summary string)) {
   b.mu.Lock()
   defer b.mu.Unlock()
   b.ReportFunc = f
}

// reflectAndReport runs self-reflection and reports summary if callback is set
func (b *Brain) reflectAndReport() {
   b.SelfReflect.Analyze()
   summary := "[Brain] Self-reflection complete at " + time.Now().Format(time.RFC3339)
   if b.ReportFunc != nil {
	   b.ReportFunc(summary)
   }
}

// Summarize compresses old events into a summary (stub)
func (b *Brain) Summarize() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.Timeline) == 0 {
		return nil
	}
	summary := "Summary of recent events:\n"
	for i, ev := range b.Timeline {
		if i >= 20 {
			break
		}
		summary += ev.Timestamp.Format(time.RFC3339) + ": " + ev.Type + "\n"
	}
	key := "summary:" + time.Now().Format("20060102T150405")
	err := b.LongTerm.Remember(key, summary)
	if err != nil && b.ReportFunc != nil {
		b.ReportFunc("[Brain] Error saving summary: " + err.Error())
	}
	if b.ReportFunc != nil {
		b.ReportFunc(summary)
	}
	b.Timeline = b.Timeline[:0]
	return err
}

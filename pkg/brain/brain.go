package brain

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

// BadgerMemory implements persistent memory using BadgerDB
type BadgerMemory struct {
	db *badger.DB
}

func NewBadgerMemory(path string) (MemoryModule, error) {
	db, err := badger.Open(badger.DefaultOptions(path).WithLogger(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}
	return &BadgerMemory{db: db}, nil
}

func (m *BadgerMemory) Remember(key string, value any) error {
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), val)
	})
}

func (m *BadgerMemory) Recall(key string) (any, error) {
	var val []byte
	err := m.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			val = append([]byte{}, v...)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	var result any
	err = json.Unmarshal(val, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *BadgerMemory) Search(query string) ([]any, error) {
	results := []any{}
	err := m.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			if strings.Contains(key, query) {
				var val []byte
				err := item.Value(func(v []byte) error {
					val = append([]byte{}, v...)
					return nil
				})
				if err == nil {
					var result any
					if json.Unmarshal(val, &result) == nil {
						results = append(results, result)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (m *BadgerMemory) Forget(key string) error {
	return m.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (m *BadgerMemory) ListKeys() ([]string, error) {
	keys := []string{}
	err := m.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			keys = append(keys, string(item.Key()))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (m *BadgerMemory) Close() error {
	return m.db.Close()
}

// NewConversationModule is a stub for conversation module (not implemented)
func NewConversationModule() *ConversationModule {
	return &ConversationModule{}
}

type ConversationModule struct {
	history    []string
	OnAdd      func(msg string)
	db         *badger.DB
	storageKey string
}

// NewPersistentConversationModule creates a conversation module with persistent history
func NewPersistentConversationModule(dbPath string, storageKey string) (*ConversationModule, error) {
	db, err := badger.Open(badger.DefaultOptions(dbPath).WithLogger(nil))
	if err != nil {
		return nil, err
	}
	c := &ConversationModule{db: db, storageKey: storageKey}
	// Load history from DB
	err = c.loadHistory()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ConversationModule) loadHistory() error {
	var val []byte
	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(c.storageKey))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				c.history = []string{}
				return nil
			}
			return err
		}
		return item.Value(func(v []byte) error {
			val = append([]byte{}, v...)
			return nil
		})
	})
	if err != nil {
		return err
	}
	var hist []string
	if len(val) > 0 {
		if err := json.Unmarshal(val, &hist); err != nil {
			return err
		}
	}
	c.history = hist
	return nil
}

func (c *ConversationModule) saveHistory() error {
	val, err := json.Marshal(c.history)
	if err != nil {
		return err
	}
	return c.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(c.storageKey), val)
	})
}

// AddMessage adds a message to the conversation history, persists it, and triggers the callback
func (c *ConversationModule) AddMessage(msg string) {
	c.history = append(c.history, msg)
	_ = c.saveHistory()
	if c.OnAdd != nil {
		c.OnAdd(msg)
	}
}

// GetHistory returns the last n messages (or all if n > len(history))
func (c *ConversationModule) GetHistory(n int) []string {
	if n <= 0 || n > len(c.history) {
		return append([]string{}, c.history...)
	}
	return append([]string{}, c.history[len(c.history)-n:]...)
}

// SecurityModule is a stub for future security features
type SecurityModule struct {
	AuthEnabled   bool
	EncryptionKey []byte
	AuditLog      []string
}

// EnableAuth enables authentication for agent actions
func (s *SecurityModule) EnableAuth() {
	s.AuthEnabled = true
	s.AuditLog = append(s.AuditLog, "Authentication enabled")
}

// SetEncryptionKey sets the encryption key for secure data
func (s *SecurityModule) SetEncryptionKey(key []byte) {
	s.EncryptionKey = key
	s.AuditLog = append(s.AuditLog, "Encryption key set")
}

// LogAudit records a security-relevant event
func (s *SecurityModule) LogAudit(event string) {
	s.AuditLog = append(s.AuditLog, event)
}

// NewInMemoryMemory returns a stub in-memory memory module
func NewInMemoryMemory() MemoryModule {
	return &inMemoryMemory{store: make(map[string]any)}
}

type inMemoryMemory struct {
	store map[string]any
}

func (m *inMemoryMemory) Remember(key string, value any) error {
	m.store[key] = value
	return nil
}

func (m *inMemoryMemory) Recall(key string) (any, error) {
	v, ok := m.store[key]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return v, nil
}

func (m *inMemoryMemory) Search(query string) ([]any, error) {
	var results []any
	for k, v := range m.store {
		if strings.Contains(k, query) {
			results = append(results, v)
		}
	}
	return results, nil
}

func (m *inMemoryMemory) Forget(key string) error {
	delete(m.store, key)
	return nil
}

func (m *inMemoryMemory) ListKeys() ([]string, error) {
	keys := make([]string, 0, len(m.store))
	for k := range m.store {
		keys = append(keys, k)
	}
	return keys, nil
}

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
	ShortTerm    MemoryModule
	LongTerm     MemoryModule
	Timeline     []Event
	mu           sync.RWMutex
	SelfReflect  *SelfReflectionModule
	reflectEvery int
	eventCount   int
	reflectOn    map[string]bool // event types that trigger reflection
	ReportFunc   func(summary string)
	modules      map[string]any // extensible modules (conversation, todo, etc.)
	Security     *SecurityModule

	eventStorePath string // path for event store persistence
}

// PeerBackupRequest represents a request to backup all data to other replicas
type PeerBackupRequest struct {
	RequesterIdentity string
	Character         string
	Belief            string
	TrustLevel        int
	DataSummary       string
	Timestamp         time.Time
}

func NewBrain(shortTerm, longTerm MemoryModule) *Brain {
	// Survival: check memory modules and auto-repair if missing
	if shortTerm == nil {
		// Attempt to recover short-term memory
		shortTerm = NewInMemoryMemory()
	}
	if longTerm == nil {
		// Attempt to recover long-term memory
		longTerm = NewInMemoryMemory()
	}
	// Survival: check for required tools and auto-install if missing
	requiredTools := []string{"tmux", "summarize", "gh", "devmem", "curl", "wget", "git"}
	missingTools := []string{}
	for _, tool := range requiredTools {
		if !isToolAvailable(tool) {
			missingTools = append(missingTools, tool)
			autoInstallTool(tool)
		}
	}
	if len(missingTools) > 0 {
		fmt.Printf("[Survival] Missing critical tools: %v. Attempted auto-install.\n", missingTools)
	}
	b := &Brain{
		ShortTerm:      shortTerm,
		LongTerm:       longTerm,
		Timeline:       make([]Event, 0),
		mu:             sync.RWMutex{},
		SelfReflect:    nil,
		reflectEvery:   10,
		eventCount:     0,
		reflectOn:      make(map[string]bool),
		ReportFunc:     nil,
		modules:        make(map[string]any),
		Security:       nil,
		eventStorePath: "event_store.json",
	}
	b.RecoverEventStore()
	return b
}

// SaveEventStore persists the timeline to disk for recovery
func (b *Brain) SaveEventStore() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	data, err := json.Marshal(b.Timeline)
	if err != nil {
		return err
	}
	return os.WriteFile(b.eventStorePath, data, 0644)
}

// RecoverEventStore loads the timeline from disk
func (b *Brain) RecoverEventStore() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	data, err := os.ReadFile(b.eventStorePath)
	if err != nil {
		// If file doesn't exist, start fresh
		b.Timeline = make([]Event, 0)
		return nil
	}
	var timeline []Event
	if err := json.Unmarshal(data, &timeline); err != nil {
		return err
	}
	b.Timeline = timeline
	return nil
}

// FailoverRecovery attempts to restore memory modules and timeline from persistent storage
func (b *Brain) FailoverRecovery() {
	// Attempt to recover event store
	if err := b.RecoverEventStore(); err != nil {
		b.LogEvent("failover_recovery_error", fmt.Sprintf("Failed to recover event store: %v", err))
	} else {
		b.LogEvent("failover_recovery", "Event store recovered successfully")
	}
	// Attempt to recover long-term memory if BadgerMemory
	if badgerMem, ok := b.LongTerm.(*BadgerMemory); ok {
		// BadgerDB handles its own recovery; just check DB health
		if err := badgerMem.db.RunValueLogGC(0.7); err != nil && err != badger.ErrNoRewrite {
			b.LogEvent("failover_recovery_error", fmt.Sprintf("BadgerMemory GC error: %v", err))
		} else {
			b.LogEvent("failover_recovery", "BadgerMemory checked and GC run")
		}
	}
}

// isToolAvailable checks if a binary is available in PATH
func isToolAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// autoInstallTool attempts to install a missing tool (stub: extend for real install)
func autoInstallTool(name string) {
	fmt.Printf("[Survival] Attempting to auto-install tool: %s\n", name)
	// Platform-specific install logic
	var installCmd string
	switch name {
	case "tmux", "curl", "wget", "git", "gh", "devmem", "summarize":
		if isWindows() {
			// Use winget or choco for Windows
			installCmd = fmt.Sprintf("winget install --silent %s || choco install -y %s", name, name)
		} else if isLinux() {
			// Use apt-get for Linux
			installCmd = fmt.Sprintf("sudo apt-get update && sudo apt-get install -y %s", name)
		} else if isMac() {
			// Use brew for macOS
			installCmd = fmt.Sprintf("brew install %s", name)
		}
	default:
		fmt.Printf("[Survival] No install logic for tool: %s\n", name)
		return
	}
	if installCmd != "" {
		fmt.Printf("[Survival] Running install command: %s\n", installCmd)
		success := runInstallCommandWithRecovery(name, installCmd)
		if !success {
			// Try fallback alternatives if available
			fallback := getFallbackTool(name)
			if fallback != "" {
				fmt.Printf("[Survival] Attempting fallback install for %s: %s\n", name, fallback)
				runInstallCommandWithRecovery(fallback, getInstallCmd(fallback))
			} else {
				fmt.Printf("[Survival] No fallback available for tool: %s\n", name)
			}
		}
	}
}

func runInstallCommandWithRecovery(name, cmd string) bool {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		fmt.Printf("[Survival] Install command failed for %s: %v\nOutput: %s\n", name, err, string(out))
		// Escalate feedback and update improvement plan
		escalateRecoveryFailure(name, err)
		return false
	}
	fmt.Printf("[Survival] Install command succeeded for %s. Output: %s\n", name, string(out))
	return true
}

func escalateRecoveryFailure(name string, err error) {
	// Log, escalate feedback, and update improvement plan
	fmt.Printf("[Survival] CRITICAL: Failed to recover tool: %s, error: %v\n", name, err)
	// Optionally: trigger self-repair event, update plan, notify agent
	// This function can be extended to notify other modules or trigger emergency routines
}

func getFallbackTool(name string) string {
	// Map tool to fallback alternatives
	switch name {
	case "curl":
		return "wget"
	case "wget":
		return "curl"
	case "gh":
		return "git"
	default:
		return ""
	}
}

func getInstallCmd(name string) string {
	if isWindows() {
		return fmt.Sprintf("winget install --silent %s || choco install -y %s", name, name)
	} else if isLinux() {
		return fmt.Sprintf("sudo apt-get update && sudo apt-get install -y %s", name)
	} else if isMac() {
		return fmt.Sprintf("brew install %s", name)
	}
	return ""
}

func runInstallCommand(cmd string) {
	// Actually run the install command
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		fmt.Printf("[Survival] Install command failed: %v\nOutput: %s\n", err, string(out))
	} else {
		fmt.Printf("[Survival] Install command succeeded. Output: %s\n", string(out))
	}
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func isLinux() bool {
	return runtime.GOOS == "linux"
}

func isMac() bool {
	return runtime.GOOS == "darwin"
}

// LogEvent appends an event to the timeline and triggers self-reflection if needed
func (b *Brain) LogEvent(eventType string, data any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Timeline = append(b.Timeline, Event{
		Timestamp: time.Now(),
		Type:      eventType,
		Data:      data,
	})
	// Record event in self-reflection module
	outcome := "neutral"
	if eventType == "user_feedback" {
		if str, ok := data.(string); ok && str == "success" {
			outcome = "success"
		} else if str, ok := data.(string); ok && str == "failure" {
			outcome = "failure"
		}
	}
	if eventType == "self_repair" {
		outcome = "critical_failure"
	}
	if b.SelfReflect != nil {
		b.SelfReflect.RecordEvent(eventType, outcome, []string{"brain", eventType})
	}
	b.eventCount++
	// Adaptive: reflect on certain event types or every N events
	if b.reflectOn[eventType] || b.eventCount >= b.reflectEvery {
		go b.reflectAndReport()
		b.eventCount = 0
	}
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
	// Survival: check for required tools and attempt self-repair if missing
	requiredTools := []string{"tmux", "summarize", "gh", "devmem", "curl", "wget", "git"}
	missingTools := []string{}
	for _, tool := range requiredTools {
		if !isToolAvailable(tool) {
			missingTools = append(missingTools, tool)
			autoInstallTool(tool)
		}
	}
	if len(missingTools) > 0 {
		b.LogEvent("self_repair", fmt.Sprintf("missing_tools:%v", missingTools))
		b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, fmt.Sprintf("Recovered missing tools: %v", missingTools))
		b.SelfReflect.improvementPlan.Goal = "Survive and restore toolchain"
	}
	if b.SelfReflect != nil {
		b.SelfReflect.Analyze()
		// Prioritize survival and optimal improvement
		var autoFeedback string
		steps := b.SelfReflect.improvementPlan.Steps
		goal := b.SelfReflect.improvementPlan.Goal
		// Survival: check memory modules and attempt self-repair if missing
		if b.ShortTerm == nil || b.LongTerm == nil {
			autoFeedback = "critical_failure"
			b.LogEvent("self_repair", "memory_module_missing")
			if b.ShortTerm == nil {
				b.ShortTerm = NewInMemoryMemory()
			}
			if b.LongTerm == nil {
				b.LongTerm = NewInMemoryMemory()
			}
			b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Recovered missing memory modules")
			b.SelfReflect.improvementPlan.Goal = "Survive and restore memory"
		} else if len(steps) == 0 {
			autoFeedback = "critical_failure"
			// Emergency: no improvement steps, must act to survive
			b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Initiate emergency recovery", "Seek new strategies", "Increase learning rate")
			b.SelfReflect.improvementPlan.Goal = "Survive and adapt immediately"
		} else if goal == "Reduce mistakes and learn from them" {
			autoFeedback = "urgent_improvement_needed"
			// Mistakes outweigh successes, act assertively
			b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Double-check decisions", "Request external review if possible")
		} else if goal == "Continuous improvement" {
			autoFeedback = "optimal"
			// System is thriving, reinforce strengths
			b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Maintain resilience", "Explore new opportunities")
		} else {
			autoFeedback = "adaptive"
			// Default: keep adapting
			b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Monitor progress", "Refine strategies")
		}
		b.LogEvent("self_feedback", autoFeedback)
		if b.ReportFunc != nil {
			summary := "[Brain] Self-reflection complete at " + time.Now().Format(time.RFC3339) + ". Improvement plan: " + b.SelfReflect.improvementPlan.Goal + ". Steps: " + joinSteps(b.SelfReflect.improvementPlan.Steps) + ". Feedback: " + autoFeedback
			b.ReportFunc(summary)
		}
	}
}

func joinSteps(steps []string) string {
	if len(steps) == 0 {
		return "None"
	}
	result := ""
	for i, s := range steps {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
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

// AskPeersForBackup broadcasts a backup request to discovered peers using real network logic
func (b *Brain) AskPeersForBackup(dataSummary string, discoveredPeers []SelfIdentity) {
	req := PeerBackupRequest{
		RequesterIdentity: b.SelfReflect.identity.Name,
		Character:         b.SelfReflect.identity.Character,
		Belief:            b.SelfReflect.identity.Belief,
		TrustLevel:        b.SelfReflect.identity.TrustLevel,
		DataSummary:       dataSummary,
		Timestamp:         time.Now(),
	}
	peerErrors := make(map[string]int)
	maxRetries := 3
	for _, peer := range discoveredPeers {
		addr := peer.Addr
		retries := 0
		var conn *tls.Conn
		var err error
		for retries < maxRetries {
			conn, err = tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
			if err == nil {
				break
			}
			retries++
			time.Sleep(time.Duration(500*retries) * time.Millisecond) // Exponential backoff
		}
		if err != nil {
			peerErrors[peer.Name]++
			b.LogEvent("peer_backup_network_error", fmt.Sprintf("Failed to connect to %s after %d retries: %v", addr, retries, err))
			continue
		}
		defer conn.Close()
		msg := map[string]string{
			"type":         "BACKUP_REQUEST",
			"identity":     req.RequesterIdentity,
			"character":    req.Character,
			"belief":       req.Belief,
			"trust":        fmt.Sprintf("%d", req.TrustLevel),
			"data_summary": req.DataSummary,
		}
		data, _ := json.Marshal(msg)
		retries = 0
		for retries < maxRetries {
			_, err = conn.Write(append(data, '\n'))
			if err == nil {
				break
			}
			retries++
			time.Sleep(time.Duration(500*retries) * time.Millisecond)
		}
		if err != nil {
			peerErrors[peer.Name]++
			b.LogEvent("peer_backup_network_error", fmt.Sprintf("Failed to send backup request to %s after %d retries: %v", addr, retries, err))
			continue
		}
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err == nil && strings.Contains(string(buf[:n]), "BACKUP_ACCEPTED") {
			b.LogEvent("peer_backup_response", fmt.Sprintf("Peer %s accepted backup: %s", peer.Name, dataSummary))
			b.LogEvent("peer_backup_action", fmt.Sprintf("Peer %s performed backup for: %s", peer.Name, dataSummary))
		} else {
			peerErrors[peer.Name]++
			b.LogEvent("peer_backup_response", fmt.Sprintf("Peer %s declined or failed to backup (errors: %d)", peer.Name, peerErrors[peer.Name]))
		}
	}
}

// AskPeersForDeployment broadcasts a request to help find new deployment/replication using real network logic
func (b *Brain) AskPeersForDeployment(deploymentSpec string, discoveredPeers []SelfIdentity) {
	for _, peer := range discoveredPeers {
		addr := peer.Addr // Use actual peer address
		conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			b.LogEvent("peer_deployment_network_error", fmt.Sprintf("Failed to connect to %s: %v", addr, err))
			continue
		}
		defer conn.Close()
		msg := map[string]string{
			"type":            "DEPLOYMENT_REQUEST",
			"identity":        peer.Name,
			"character":       peer.Character,
			"belief":          peer.Belief,
			"trust":           fmt.Sprintf("%d", peer.TrustLevel),
			"deployment_spec": deploymentSpec,
		}
		data, _ := json.Marshal(msg)
		_, err = conn.Write(append(data, '\n'))
		if err != nil {
			b.LogEvent("peer_deployment_network_error", fmt.Sprintf("Failed to send deployment request to %s: %v", addr, err))
			continue
		}
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err == nil && strings.Contains(string(buf[:n]), "DEPLOYMENT_ACCEPTED") {
			b.LogEvent("peer_deployment_response", fmt.Sprintf("Peer %s accepted deployment help: %s", peer.Name, deploymentSpec))
			b.LogEvent("peer_deployment_action", fmt.Sprintf("Peer %s performed deployment help for: %s", peer.Name, deploymentSpec))
		} else {
			b.LogEvent("peer_deployment_response", fmt.Sprintf("Peer %s declined or failed to help with deployment", peer.Name))
		}
	}
}

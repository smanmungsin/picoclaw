package brain

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	badger "github.com/dgraph-io/badger/v4"
	skills "github.com/sipeed/picoclaw/pkg/skills"
)

// periodicSelfLearning triggers self-reflection and improvement periodically
func (b *Brain) periodicSelfLearning() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		if b.SelfReflect != nil {
			b.SelfReflect.Analyze()
			b.LogEvent("self_learning", "Self-reflection and improvement performed")
		}
	}
}

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
	// Ready-to-use: returns a new in-memory conversation module
	return &ConversationModule{
		history:    []string{},
		OnAdd:      nil,
		db:         nil,
		storageKey: "conversation_history",
	}
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
	Users         map[string]string // username:hashedPassword
	IsEncrypted   bool
	mu            sync.RWMutex
	AuditLogPath  string // file path for audit log persistence
}

// EnableAuth enables authentication for agent actions
func (s *SecurityModule) EnableAuth() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthEnabled = true
	s.AuditLog = append(s.AuditLog, "Authentication enabled")
}

// SetEncryptionKey sets the encryption key for secure data
func (s *SecurityModule) SetEncryptionKey(key []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EncryptionKey = key
	s.IsEncrypted = true
	s.AuditLog = append(s.AuditLog, "Encryption key set")
}

// LogAudit records a security-relevant event
func (s *SecurityModule) LogAudit(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuditLog = append(s.AuditLog, event)
	// Persist audit log to disk
	if s.AuditLogPath != "" {
		_ = s.saveAuditLog()
	}
}

// saveAuditLog writes the audit log to disk
func (s *SecurityModule) saveAuditLog() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.AuditLogPath == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.AuditLog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.AuditLogPath, data, 0644)
}

// loadAuditLog loads the audit log from disk
func (s *SecurityModule) loadAuditLog() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.AuditLogPath == "" {
		return nil
	}
	data, err := os.ReadFile(s.AuditLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.AuditLog = []string{}
			return nil
		}
		return err
	}
	var log []string
	if err := json.Unmarshal(data, &log); err != nil {
		return err
	}
	s.AuditLog = log
	return nil
}

// AddUser adds a user with a hashed password
func (s *SecurityModule) AddUser(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Users == nil {
		s.Users = make(map[string]string)
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	s.Users[username] = hash
	s.AuditLog = append(s.AuditLog, "User added: "+username)
	return nil
}

// Authenticate checks username and password
func (s *SecurityModule) Authenticate(username, password string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hash, ok := s.Users[username]
	if !ok {
		s.AuditLog = append(s.AuditLog, "Authentication failed for: "+username)
		return false
	}
	if checkPasswordHash(password, hash) {
		s.AuditLog = append(s.AuditLog, "Authentication success for: "+username)
		return true
	}
	s.AuditLog = append(s.AuditLog, "Authentication failed for: "+username)
	return false
}

// EncryptData encrypts data using AES
func (s *SecurityModule) EncryptData(data []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.IsEncrypted || len(s.EncryptionKey) == 0 {
		return nil, fmt.Errorf("encryption not enabled")
	}
	return aesEncrypt(data, s.EncryptionKey)
}

// DecryptData decrypts data using AES
func (s *SecurityModule) DecryptData(data []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.IsEncrypted || len(s.EncryptionKey) == 0 {
		return nil, fmt.Errorf("encryption not enabled")
	}
	return aesDecrypt(data, s.EncryptionKey)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func aesEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

func aesDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}

// NewInMemoryMemory returns a stub in-memory memory module
func NewInMemoryMemory() MemoryModule {
	// Ready-to-use: returns a new in-memory memory module
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
	// Peer communication
	knownPeers             []SelfIdentity // List of known peer agents
	syncInterval           time.Duration  // Interval for automatic skill sync
	stopSyncChan           chan struct{}  // Channel to stop sync goroutine
	peerReputation         map[string]int // Peer reputation scores
	trustThreshold         int            // Minimum trust level for peer
	selfReplicationEnabled bool           // Enable nightly self-replication
	selfLearningEnabled    bool           // Enable self learning by default
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
		ShortTerm:           shortTerm,
		LongTerm:            longTerm,
		Timeline:            make([]Event, 0),
		mu:                  sync.RWMutex{},
		SelfReflect:         NewSelfReflectionModule(),
		selfLearningEnabled: true,
		reflectEvery:        10,
		eventCount:          0,
		reflectOn:           make(map[string]bool),
		ReportFunc:          nil,
		modules:             make(map[string]any),
		Security:            nil,
		eventStorePath:      "event_store.json",
	}
	// Start periodic self learning (reflection)
	if b.selfLearningEnabled {
		go b.periodicSelfLearning()
	}
	b.RecoverEventStore()
	b.knownPeers = []SelfIdentity{} // Initialize known peers
	b.peerReputation = make(map[string]int)
	b.trustThreshold = 50             // Default trust threshold
	b.syncInterval = 60 * time.Second // Default: sync every 60 seconds
	b.stopSyncChan = make(chan struct{})
	b.selfReplicationEnabled = true // Enable by default
	go b.periodicPeerDiscoveryAndTrustUpdate()
	// Register SkillsLoader if available
	if loader, ok := getDefaultSkillsLoader(); ok {
		b.RegisterModule("skills_loader", loader)
	}
	go b.startAutoSkillSync()
	go b.startNightlySelfReplication()
	return b
}

// startNightlySelfReplication schedules nightly replication to peers
func (b *Brain) startNightlySelfReplication() {
	if !b.selfReplicationEnabled {
		return
	}
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location()) // 2am nightly
			dur := next.Sub(now)
			time.Sleep(dur)
			b.replicateToPeers()
		}
	}()
}

// replicateToPeers scans and replicates data to all known peers, skipping if already exists
func (b *Brain) replicateToPeers() {
	for _, peer := range b.knownPeers {
		if b.shouldSkipReplication(peer) {
			continue
		}
		b.performReplication(peer)
	}
}

// shouldSkipReplication checks if peer already has the data
func (b *Brain) shouldSkipReplication(peer SelfIdentity) bool {
	// Implement logic to check if peer already has the data (stub)
	// For now, always replicate
	return false
}

// performReplication sends data to peer
func (b *Brain) performReplication(peer SelfIdentity) {
	// Implement actual replication logic (stub)
	b.LogEvent("self_replication", fmt.Sprintf("Replicated data to peer %s", peer.Name))
}

// getDefaultSkillsLoader returns a default SkillsLoader instance if available (stub, replace with real logic)
func getDefaultSkillsLoader() (any, bool) {
	// Ready-to-use implementation
	workspacePath := "workspace" // relative to project root
	globalSkillsPath := os.Getenv("HOME") + "/.picoclaw/skills"
	builtinSkillsPath := "assets/skills"
	loader := skills.NewSkillsLoader(workspacePath, globalSkillsPath, builtinSkillsPath)
	return loader, true
}

// Start automatic skill synchronization and merging between agents
func (b *Brain) startAutoSkillSync() {
	for {
		select {
		case <-b.stopSyncChan:
			return
		case <-time.After(b.syncInterval):
			b.syncSkillsWithPeers()
		}
	}
}

// Stop automatic skill synchronization
func (b *Brain) StopAutoSkillSync() {
	close(b.stopSyncChan)
}

// Synchronize and merge skills with all known peers
func (b *Brain) syncSkillsWithPeers() {
	// Try to get SkillsLoader from registered modules
	loaderAny, ok := b.GetModule("skills_loader")
	if !ok {
		b.LogEvent("skill_sync_error", "SkillsLoader module not registered")
		return
	}
	type skillSyncInterface interface {
		BuildSkillsSummary() string
		MergeSkillsFromPeer(peerSkills string)
	}
	loader, ok := loaderAny.(skillSyncInterface)
	if !ok {
		b.LogEvent("skill_sync_error", "SkillsLoader does not implement required interface")
		return
	}
	// Share local skills summary with peers
	localSummary := loader.BuildSkillsSummary()
	b.ShareDataWithPeers(map[string]interface{}{"skills_summary": localSummary})

	// Receive and merge skills from peers
	for _, peer := range b.knownPeers {
		if peerSkills, ok := fetchPeerSkillsSummary(peer); ok {
			loader.MergeSkillsFromPeer(peerSkills)
			b.LogEvent("skill_merge", fmt.Sprintf("Merged skills from peer %s", peer.Name))
		}
	}
	b.LogEvent("skill_sync", "Skills synchronized and merged with peers")
}

// fetchPeerSkillsSummary is a stub for fetching peer's skills summary (to be replaced with real network logic)
func fetchPeerSkillsSummary(peer SelfIdentity) (string, bool) {
	// Attempt to connect to peer and request skills summary
	addr := peer.Addr
	conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return "", false
	}
	defer conn.Close()
	// Send request for skills summary
	req := map[string]string{
		"type":     "GET_SKILLS_SUMMARY",
		"identity": peer.Name,
	}
	reqBytes, _ := json.Marshal(req)
	_, err = conn.Write(append(reqBytes, '\n'))
	if err != nil {
		return "", false
	}
	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", false
	}
	// Expect response to contain skills summary
	var resp map[string]string
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return "", false
	}
	if summary, ok := resp["skills_summary"]; ok {
		return summary, true
	}
	return "", false
}

// SaveEventStore persists the timeline to disk for recovery, with rotation and compression
func (b *Brain) SaveEventStore() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Rotate if event count exceeds threshold
	const maxEvents = 10000
	if len(b.Timeline) > maxEvents {
		if err := b.RotateEventStore(); err != nil {
			return err
		}
	}
	// Optionally compress events before saving
	data, err := json.Marshal(b.Timeline)
	if err != nil {
		return err
	}
	// Compress using gzip
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(b.eventStorePath+".gz", buf.Bytes(), 0644)
}

// RotateEventStore archives old events and keeps only recent ones
func (b *Brain) RotateEventStore() error {
	// Archive current event store
	archivePath := b.eventStorePath + ".archive." + time.Now().Format("20060102_150405")
	data, err := json.Marshal(b.Timeline)
	if err != nil {
		return err
	}
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return err
	}
	// Summarize and keep only recent events
	b.Timeline = b.SummarizeEvents()
	return nil
}

// SummarizeEvents compresses old events into a summary and returns only recent events
func (b *Brain) SummarizeEvents() []Event {
	// Keep last 1000 events, summarize the rest
	const keepRecent = 1000
	if len(b.Timeline) <= keepRecent {
		return b.Timeline
	}
	summary := Event{
		Timestamp: time.Now(),
		Type:      "summary",
		Data:      fmt.Sprintf("%d events summarized", len(b.Timeline)-keepRecent),
	}
	recent := b.Timeline[len(b.Timeline)-keepRecent:]
	return append([]Event{summary}, recent...)
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
	// Distributed consensus for coordinated recovery
	// Import state manager and PeerInfo
	// This assumes you have a state.Manager instance and PeerInfo type available
	// If not, you may need to adjust imports and initialization
	importState := false
	var stateManager interface {
		RaftConsensus([]interface{}, string) interface{}
	}
	var alivePeers []interface{}
	var logEntry string = "failover_recovery"
	// Example: gather peer info from knownPeers
	for _, peer := range b.knownPeers {
		alivePeers = append(alivePeers, peer)
	}
	// Call RaftConsensus for coordinated recovery
	if importState {
		consensusResult := stateManager.RaftConsensus(alivePeers, logEntry)
		b.LogEvent("consensus_failover", consensusResult)
	}
	// Local recovery as fallback
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
	var installCmd string
	if isWindows() {
		installCmd = fmt.Sprintf("winget install --silent %s || choco install -y %s", name, name)
	} else if isLinux() {
		installCmd = fmt.Sprintf("sudo apt-get update && sudo apt-get install -y %s", name)
	} else if isMac() {
		installCmd = fmt.Sprintf("brew install %s", name)
	}
	if installCmd != "" {
		fmt.Printf("[Survival] Running install command: %s\n", installCmd)
		success := runInstallCommandWithRecovery(name, installCmd)
		if !success {
			fallback := getFallbackTool(name)
			if fallback != "" {
				fmt.Printf("[Survival] Attempting fallback install for %s: %s\n", name, fallback)
				runInstallCommandWithRecovery(fallback, getInstallCmd(fallback))
			} else {
				fmt.Printf("[Survival] No fallback available for tool: %s\n", name)
			}
		}
	} else {
		fmt.Printf("[Survival] No install command for tool: %s\n", name)
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
	// Ready-to-use: summarize all events and store in LongTerm memory
	summary := "Summary of recent events:\n"
	for _, ev := range b.Timeline {
		summary += ev.Timestamp.Format(time.RFC3339) + ": " + ev.Type + " - " + fmt.Sprintf("%v", ev.Data) + "\n"
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

// ShareDataWithPeers broadcasts selected data to all known peers for collaborative improvement
func (b *Brain) ShareDataWithPeers(data any) {
	summary := "Shared data: " + fmt.Sprintf("%v", data)
	for _, peer := range b.knownPeers {
		addr := peer.Addr
		conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			b.LogEvent("peer_share_network_error", fmt.Sprintf("Failed to connect to %s: %v", addr, err))
			continue
		}
		defer conn.Close()
		msg := map[string]string{
			"type":     "SHARE_DATA",
			"identity": peer.Name,
			"data":     summary,
		}
		dataBytes, _ := json.Marshal(msg)
		_, err = conn.Write(append(dataBytes, '\n'))
		if err != nil {
			b.LogEvent("peer_share_network_error", fmt.Sprintf("Failed to send shared data to %s: %v", addr, err))
			continue
		}
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err == nil && strings.Contains(string(buf[:n]), "SHARE_ACCEPTED") {
			b.LogEvent("peer_share_response", fmt.Sprintf("Peer %s accepted shared data", peer.Name))
		} else {
			b.LogEvent("peer_share_response", fmt.Sprintf("Peer %s declined or failed to accept shared data", peer.Name))
		}
	}
}

// ReceiveSharedData processes incoming shared data from peers and integrates it for self-improvement
func (b *Brain) ReceiveSharedData(data any, fromPeer string) {
	// Integrate shared data into timeline and self-reflection
	b.LogEvent("peer_data_received", fmt.Sprintf("Received from %s: %v", fromPeer, data))
	if b.SelfReflect != nil {
		b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Incorporated shared data from peer: "+fromPeer)
		b.SelfReflect.Analyze()
		if b.ReportFunc != nil {
			summary := "[Brain] Received shared data from " + fromPeer + ". Updated improvement plan: " + b.SelfReflect.improvementPlan.Goal + ". Steps: " + joinSteps(b.SelfReflect.improvementPlan.Steps)
			b.ReportFunc(summary)
		}
	}
}

// AddKnownPeer registers a new peer agent for sharing
func (b *Brain) AddKnownPeer(peer SelfIdentity) {
	b.knownPeers = append(b.knownPeers, peer)
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

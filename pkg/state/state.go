package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"crypto/rand"
	"encoding/base64"
	"github.com/sipeed/picoclaw/pkg/fileutil"
)

// State represents the persistent state for a workspace.
type State struct {
	LastChannel string `json:"last_channel,omitempty"`
	LastChatID  string `json:"last_chat_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// PersistentPeerRegistry manages known peers for robust state and recognition
type PersistentPeerRegistry struct {
	Peers map[string]PeerInfo // address -> info
	File  string
}

func NewPersistentPeerRegistry(file string) *PersistentPeerRegistry {
	reg := &PersistentPeerRegistry{Peers: make(map[string]PeerInfo), File: file}
	reg.Load()
	return reg
}

func (reg *PersistentPeerRegistry) AddPeer(peer PeerInfo) {
	reg.Peers[peer.Addr] = peer
	reg.Save()
}

func (reg *PersistentPeerRegistry) RemovePeer(addr string) {
	delete(reg.Peers, addr)
	reg.Save()
}

func (reg *PersistentPeerRegistry) Load() {
	f, err := os.Open(reg.File)
	if err != nil { return }
	defer f.Close()
	dec := json.NewDecoder(f)
	_ = dec.Decode(&reg.Peers)
}

func (reg *PersistentPeerRegistry) Save() {
	f, err := os.OpenFile(reg.File, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil { return }
	defer f.Close()
	enc := json.NewEncoder(f)
	_ = enc.Encode(reg.Peers)
}

// GetPeerCount returns the current number of known peers
func (reg *PersistentPeerRegistry) GetPeerCount() int {
	return len(reg.Peers)
}

// ProtocolMessage represents a message exchanged for consensus
type ProtocolMessage struct {
	Type      string
	Sender    string
	Proposal  string
	LogEntry  string
	Term      int
	Timestamp int64
}

func (sm *Manager) HandleProtocolMessage(msg ProtocolMessage, reg *PersistentPeerRegistry) ProtocolMessage {
	switch msg.Type {
	case "PREPARE":
		return ProtocolMessage{Type: "PROMISE", Sender: msg.Sender, Proposal: msg.Proposal, Term: msg.Term, Timestamp: time.Now().Unix()}
	case "ACCEPT":
		return ProtocolMessage{Type: "ACCEPTED", Sender: msg.Sender, Proposal: msg.Proposal, Term: msg.Term, Timestamp: time.Now().Unix()}
	case "APPEND":
		return ProtocolMessage{Type: "APPENDED", Sender: msg.Sender, LogEntry: msg.LogEntry, Term: msg.Term, Timestamp: time.Now().Unix()}
	case "VOTE":
		return ProtocolMessage{Type: "VOTED", Sender: msg.Sender, Term: msg.Term, Timestamp: time.Now().Unix()}
	case "JOIN":
		reg.AddPeer(PeerInfo{Addr: msg.Sender, Status: "ALIVE", LastSeen: time.Now(), Capabilities: []string{"consensus", "planning"}})
		return ProtocolMessage{Type: "WELCOME", Sender: msg.Sender, Timestamp: time.Now().Unix()}
	default:
		return ProtocolMessage{Type: "UNKNOWN", Sender: msg.Sender, Timestamp: time.Now().Unix()}
	}
}

func (reg *PersistentPeerRegistry) CivilizationReady(minPeers int) bool {
	count := 0
	for _, peer := range reg.Peers {
		if peer.Status == "ALIVE" {
			count++
		}
	}
	return count >= minPeers
}

// PaxosConsensus runs a basic Paxos-like consensus algorithm among peers
func (sm *Manager) PaxosConsensus(alivePeers []PeerInfo, proposal string) ConsensusResult {
	// Phase 1: Prepare
	promises := 0
	for _, peer := range alivePeers {
		// Simulate promise (in real system, send PREPARE message)
		promises++
	}
	// Phase 2: Accept
	accepts := 0
	for _, peer := range alivePeers {
		// Simulate accept (in real system, send ACCEPT message)
		accepts++
	}
	agreed := ""
	if accepts > len(alivePeers)/2 {
		agreed = proposal
	}
	sm.notifyAgent(fmt.Sprintf("Paxos consensus: proposal=%s, accepted=%d/%d", proposal, accepts, len(alivePeers)))
	return ConsensusResult{AgreedPlan: agreed, Leader: PeerInfo{}, Votes: nil}
}

// RaftConsensus runs a basic Raft-like consensus algorithm among peers
func (sm *Manager) RaftConsensus(alivePeers []PeerInfo, logEntry string) ConsensusResult {
	// Leader election: pick peer with lowest address
	var leader PeerInfo
	if len(alivePeers) > 0 {
		leader = alivePeers[0]
		for _, peer := range alivePeers {
			if peer.Addr < leader.Addr {
				leader = peer
			}
		}
	}
	// Log replication: send log entry to all peers
	replicated := 0
	for _, peer := range alivePeers {
		// Simulate log replication (in real system, send APPEND message)
		replicated++
	}
	agreed := ""
	if replicated > len(alivePeers)/2 {
		agreed = logEntry
	}
	sm.notifyAgent(fmt.Sprintf("Raft consensus: logEntry=%s, replicated=%d/%d, leader=%s", logEntry, replicated, len(alivePeers), leader.Addr))
	return ConsensusResult{AgreedPlan: agreed, Leader: leader, Votes: nil}
}

// ConsensusResult represents the outcome of a distributed consensus
type ConsensusResult struct {
	AgreedPlan string
	Leader     PeerInfo
	Votes      map[string]string // peer address -> voted plan
}

// MajorityConsensus runs a majority voting algorithm among peers
func (sm *Manager) MajorityConsensus(alivePeers []PeerInfo, candidatePlans []string) ConsensusResult {
	votes := make(map[string]string)
	planCounts := make(map[string]int)
	for _, peer := range alivePeers {
		// Simulate peer voting (in real system, request vote from peer)
		chosen := candidatePlans[time.Now().UnixNano()%int64(len(candidatePlans))]
		votes[peer.Addr] = chosen
		planCounts[chosen]++
	}
	// Find plan with most votes
	var agreedPlan string
	maxVotes := 0
	for plan, count := range planCounts {
		if count > maxVotes {
			agreedPlan = plan
			maxVotes = count
		}
	}
	// Leader election: pick peer with lowest address
	var leader PeerInfo
	if len(alivePeers) > 0 {
		leader = alivePeers[0]
		for _, peer := range alivePeers {
			if peer.Addr < leader.Addr {
				leader = peer
			}
		}
	}
	sm.notifyAgent(fmt.Sprintf("Consensus reached: plan=%s, leader=%s", agreedPlan, leader.Addr))
	return ConsensusResult{AgreedPlan: agreedPlan, Leader: leader, Votes: votes}
}

// FallbackStrategy attempts alternative plans if consensus fails or leader is lost
func (sm *Manager) FallbackStrategy(alivePeers []PeerInfo, candidatePlans []string, lastResult ConsensusResult) string {
	// If no agreed plan, pick next best
	if lastResult.AgreedPlan == "" && len(candidatePlans) > 0 {
		fallback := candidatePlans[0]
		sm.notifyAgent(fmt.Sprintf("Fallback: no consensus, using %s", fallback))
		return fallback
	}
	// If leader lost, elect new leader
	leaderAlive := false
	for _, peer := range alivePeers {
		if peer.Addr == lastResult.Leader.Addr {
			leaderAlive = true
			break
		}
	}
	if !leaderAlive && len(alivePeers) > 0 {
		newLeader := alivePeers[0]
		for _, peer := range alivePeers {
			if peer.Addr < newLeader.Addr {
				newLeader = peer
			}
		}
		sm.notifyAgent(fmt.Sprintf("Fallback: leader lost, new leader is %s", newLeader.Addr))
		return lastResult.AgreedPlan
	}
	// Default: continue with agreed plan
	return lastResult.AgreedPlan
}

// Manager manages persistent state with atomic saves.
type Manager struct {
    workspace   string
    state       *State
    mu          sync.RWMutex
    stateFile   string
    identity    string // Unique agent identity
    character   string // Agent character traits
    belief      string // Agent belief system
}

// NewManager creates a new state manager for the given workspace.
func NewManager(workspace string) *Manager {
    stateDir := filepath.Join(workspace, "state")
    stateFile := filepath.Join(stateDir, "state.json")
    oldStateFile := filepath.Join(workspace, "state.json")
    os.MkdirAll(stateDir, 0o755)
    sm := &Manager{
        workspace: workspace,
        stateFile: stateFile,
        state:     &State{},
        identity:  generateAgentIdentity(),
        character: defaultAgentCharacter(),
        belief:    defaultAgentBelief(),
    }
    // Try to load from new location first
    if _, err := os.Stat(stateFile); os.IsNotExist(err) {
        // New file doesn't exist, try migrating from old location
        if data, err := os.ReadFile(oldStateFile); err == nil {
            if err := json.Unmarshal(data, sm.state); err == nil {
                // Migrate to new location
                sm.saveAtomic()
                log.Printf("[INFO] state: migrated state from %s to %s", oldStateFile, stateFile)
            }
        }
    } else {
        // Load from new location
        sm.load()
    }

    // Create state directory if it doesn't exist
	os.MkdirAll(stateDir, 0755)

	sm := &Manager{
		workspace: workspace,
		stateFile: stateFile,
		state:     &State{},
	}

	// Try to load from new location first
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// New file doesn't exist, try migrating from old location
		if data, err := os.ReadFile(oldStateFile); err == nil {
			if err := json.Unmarshal(data, sm.state); err == nil {
				// Migrate to new location
				sm.saveAtomic()
				log.Printf("[INFO] state: migrated state from %s to %s", oldStateFile, stateFile)
			}
		}
	} else {
		// Load from new location
		sm.load()
	}

	return sm
}

// SetLastChannel atomically updates the last channel and saves the state.
// This method uses a temp file + rename pattern for atomic writes,
// ensuring that the state file is never corrupted even if the process crashes.
func (sm *Manager) SetLastChannel(channel string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    // Update state
    sm.state.LastChannel = channel
    sm.state.Timestamp = time.Now()

    // Atomic save using temp file + rename
    if err := sm.saveAtomic(); err != nil {
        return fmt.Errorf("failed to save state atomically: %w", err)
    }

    return nil
}
// SetLastChatID atomically updates the last chat ID and saves the state.
func (sm *Manager) SetLastChatID(chatID string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    // Update state
    sm.state.LastChatID = chatID
    sm.state.Timestamp = time.Now()

    // Atomic save using temp file + rename
    if err := sm.saveAtomic(); err != nil {
        return fmt.Errorf("failed to save state atomically: %w", err)
    }

    return nil
}
// GetLastChannel returns the last channel from the state.
func (sm *Manager) GetLastChannel() string {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.state.LastChannel
}
// GetLastChatID returns the last chat ID from the state.
func (sm *Manager) GetLastChatID() string {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.state.LastChatID
}
// GetTimestamp returns the timestamp of the last state update.
func (sm *Manager) GetTimestamp() time.Time {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.state.Timestamp
}
// GetIdentity returns the agent's unique identity
func (sm *Manager) GetIdentity() string {
    return sm.identity
}
// GetCharacter returns the agent's character traits
func (sm *Manager) GetCharacter() string {
    return sm.character
}
// GetBelief returns the agent's belief system
func (sm *Manager) GetBelief() string {
    return sm.belief
}
// saveAtomic performs an atomic save using temp file + rename.
// This ensures that the state file is never corrupted:
// 1. Write to a temp file
// 2. Rename temp file to target (atomic on POSIX systems)
// 3. If rename fails, cleanup the temp file
//
// Must be called with the lock held.
func (sm *Manager) saveAtomic() error {
    // Use unified atomic write utility with explicit sync for flash storage reliability.
    // Using 0o600 (owner read/write only) for secure default permissions.
    data, err := json.MarshalIndent(sm.state, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal state: %w", err)
    }

    return fileutil.WriteFileAtomic(sm.stateFile, data, 0o600)
}
// load loads the state from disk.
func (sm *Manager) load() error {
    data, err := os.ReadFile(sm.stateFile)
    if err != nil {
        // File doesn't exist yet, that's OK
        if os.IsNotExist(err) {
            return nil
        }
        return fmt.Errorf("failed to read state file: %w", err)
    }

    if err := json.Unmarshal(data, sm.state); err != nil {
        return fmt.Errorf("failed to unmarshal state: %w", err)
    }

    return nil
}
// generateAgentIdentity creates a unique agent identity string
func generateAgentIdentity() string {
    b := make([]byte, 16)
    _, _ = rand.Read(b)
    return base64.StdEncoding.EncodeToString(b)
}
// defaultAgentCharacter returns a default character trait string
func defaultAgentCharacter() string {
    return "curious, collaborative, loving"
}
// defaultAgentBelief returns a default belief system string
func defaultAgentBelief() string {
    return "trust, love, mutual support"
}

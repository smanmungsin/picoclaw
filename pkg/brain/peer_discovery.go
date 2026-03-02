package brain

import (
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/state"
)

func (b *Brain) periodicPeerDiscoveryAndTrustUpdate() {
	ticker := time.NewTicker(b.syncInterval)
	defer ticker.Stop()
	for range ticker.C {
		b.discoverPeers()
		b.recalculatePeerTrust()
	}
}

func (b *Brain) discoverPeers() {
	// Real peer discovery: scan persistent registry and update knownPeers
	// Import state package and use PersistentPeerRegistry
	// Assume registry file path is "workspace/peers.json"
	// You may want to make this configurable
	registryFile := "workspace/peers.json"
	reg := state.NewPersistentPeerRegistry(registryFile)
	var discovered []SelfIdentity
	for _, peer := range reg.Peers {
		discovered = append(discovered, SelfIdentity{
			Name:       peer.Addr,
			Addr:       peer.Addr,
			Character:  "", // Could be extended
			Belief:     "",
			TrustLevel: 0, // Will be recalculated
		})
	}
	b.knownPeers = discovered
	b.LogEvent("peer_discovery", fmt.Sprintf("Discovered %d peers", len(discovered)))
}

func (b *Brain) recalculatePeerTrust() {
	for i, peer := range b.knownPeers {
		rep := b.peerReputation[peer.Name]
		if rep >= b.trustThreshold {
			b.knownPeers[i].TrustLevel = rep
		} else {
			b.knownPeers[i].TrustLevel = 0 // Not trusted
		}
	}
	b.LogEvent("peer_trust_update", "Peer trust recalculated")
}

func (b *Brain) updatePeerReputation(peerName string, delta int) {
	b.peerReputation[peerName] += delta
	b.LogEvent("peer_reputation_update", fmt.Sprintf("Peer %s reputation changed by %d", peerName, delta))
}

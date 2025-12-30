package network

import (
	"sync"
	"time"
)

// P2PManager handles peer discovery and connection management.
type P2PManager struct {
	node       *ZmqNode
	knownPeers map[string]*PeerInfo
	seedNodes  []string
	mu         sync.RWMutex

	// Configuration
	pruneInterval time.Duration
	staleTimeout  time.Duration

	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
}

// NewP2PManager creates a new P2P manager.
func NewP2PManager(node *ZmqNode) *P2PManager {
	return &P2PManager{
		node:          node,
		knownPeers:    make(map[string]*PeerInfo),
		pruneInterval: 30 * time.Second,
		staleTimeout:  5 * time.Minute,
		stopChan:      make(chan struct{}),
	}
}

// Start begins P2P management operations.
func (p *P2PManager) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	// Start stale peer pruner
	p.wg.Add(1)
	go p.pruneStalePeers()

	// Set message handler for peer exchange
	p.node.SetHandler(p.handleMessage)
}

// Stop stops P2P management.
func (p *P2PManager) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	close(p.stopChan)
	p.wg.Wait()
}

// DiscoverPeers initiates peer discovery from seed nodes.
func (p *P2PManager) DiscoverPeers(seeds []string) error {
	p.mu.Lock()
	p.seedNodes = seeds
	p.mu.Unlock()

	// Register seed nodes as peers
	for i, addr := range seeds {
		peerID := addr // Use address as ID for seeds
		p.node.RegisterPeer(peerID, addr, nil)
		p.knownPeers[peerID] = &PeerInfo{
			ID:       peerID,
			Address:  addr,
			LastSeen: time.Now(),
		}

		// Request peer list from seeds
		_ = p.node.SendDirect(peerID, map[string]interface{}{
			"action": "peer_exchange_request",
			"index":  i,
		})
	}

	return nil
}

// handleMessage processes P2P-related messages.
func (p *P2PManager) handleMessage(msg *Message) error {
	payload := msg.Payload
	action, ok := payload["action"].(string)
	if !ok {
		return nil // Not a P2P message
	}

	switch action {
	case "peer_exchange_request":
		return p.handlePeerExchangeRequest(msg)
	case "peer_exchange_response":
		return p.handlePeerExchangeResponse(msg)
	case "peer_announce":
		return p.handlePeerAnnounce(msg)
	}

	return nil
}

// handlePeerExchangeRequest responds with known peers.
func (p *P2PManager) handlePeerExchangeRequest(msg *Message) error {
	p.mu.RLock()
	peers := make([]map[string]interface{}, 0, len(p.knownPeers))
	for _, peer := range p.knownPeers {
		peers = append(peers, map[string]interface{}{
			"id":        peer.ID,
			"address":   peer.Address,
			"last_seen": peer.LastSeen.Unix(),
		})
	}
	p.mu.RUnlock()

	return p.node.SendDirect(msg.From, map[string]interface{}{
		"action": "peer_exchange_response",
		"peers":  peers,
	})
}

// handlePeerExchangeResponse processes received peer list.
func (p *P2PManager) handlePeerExchangeResponse(msg *Message) error {
	peersData, ok := msg.Payload["peers"].([]interface{})
	if !ok {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pData := range peersData {
		peerMap, ok := pData.(map[string]interface{})
		if !ok {
			continue
		}

		peerID, _ := peerMap["id"].(string)
		address, _ := peerMap["address"].(string)

		if peerID == "" || address == "" {
			continue
		}

		// Don't add ourselves
		if peerID == p.node.nodeID {
			continue
		}

		// Add or update peer
		if _, exists := p.knownPeers[peerID]; !exists {
			p.knownPeers[peerID] = &PeerInfo{
				ID:       peerID,
				Address:  address,
				LastSeen: time.Now(),
			}
			p.node.RegisterPeer(peerID, address, nil)
		}
	}

	return nil
}

// handlePeerAnnounce processes peer announcements.
func (p *P2PManager) handlePeerAnnounce(msg *Message) error {
	peerID, _ := msg.Payload["peer_id"].(string)
	address, _ := msg.Payload["address"].(string)

	if peerID == "" || address == "" {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.knownPeers[peerID]; !exists {
		p.knownPeers[peerID] = &PeerInfo{
			ID:       peerID,
			Address:  address,
			LastSeen: time.Now(),
		}
		p.node.RegisterPeer(peerID, address, nil)
	} else {
		p.knownPeers[peerID].LastSeen = time.Now()
	}

	return nil
}

// AnnounceSelf broadcasts this node's presence to the network.
func (p *P2PManager) AnnounceSelf() error {
	stats := p.node.GetStats()
	return p.node.Broadcast(map[string]interface{}{
		"action":  "peer_announce",
		"peer_id": stats.NodeID,
		"address": stats.Address,
	}, nil)
}

// pruneStalePeers periodically removes stale peers.
func (p *P2PManager) pruneStalePeers() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.pruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.prune()
		}
	}
}

// prune removes peers that haven't been seen recently.
func (p *P2PManager) prune() {
	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := time.Now().Add(-p.staleTimeout)
	for peerID, peer := range p.knownPeers {
		if peer.LastSeen.Before(cutoff) {
			delete(p.knownPeers, peerID)
			p.node.UnregisterPeer(peerID)
		}
	}
}

// GetHealthyPeers returns peers that are considered healthy.
func (p *P2PManager) GetHealthyPeers() []*PeerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cutoff := time.Now().Add(-p.staleTimeout)
	healthy := make([]*PeerInfo, 0)

	for _, peer := range p.knownPeers {
		if peer.LastSeen.After(cutoff) {
			healthy = append(healthy, &PeerInfo{
				ID:       peer.ID,
				Address:  peer.Address,
				LastSeen: peer.LastSeen,
			})
		}
	}

	return healthy
}

// PeerCount returns the number of known peers.
func (p *P2PManager) PeerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.knownPeers)
}

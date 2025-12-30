// Package network provides ZeroMQ-based P2P networking for HieraChain.
package network

import (
	"fmt"
	"log"
	"sync"
)

// NetworkConfig defines configuration for the network service.
type NetworkConfig struct {
	NodeID    string   `json:"node_id"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	SeedNodes []string `json:"seed_nodes"`
}

// DefaultNetworkConfig returns a configuration with sensible defaults.
func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		NodeID:    "node-1",
		Host:      "127.0.0.1",
		Port:      5555,
		SeedNodes: []string{},
	}
}

// NetworkStatus represents the current status of the network service.
type NetworkStatus struct {
	NodeID       string    `json:"node_id"`
	Address      string    `json:"address"`
	IsRunning    bool      `json:"is_running"`
	PeerCount    int       `json:"peer_count"`
	HealthyPeers int       `json:"healthy_peers"`
	NodeStats    NodeStats `json:"node_stats"`
}

// NetworkService orchestrates all network components: ZmqNode, P2PManager, and Propagator.
type NetworkService struct {
	config     NetworkConfig
	node       *ZmqNode
	p2p        *P2PManager
	propagator *Propagator

	mu      sync.RWMutex
	running bool
}

// NewNetworkService creates a new network service with the given configuration.
func NewNetworkService(config NetworkConfig) *NetworkService {
	node := NewZmqNode(config.NodeID, config.Host, config.Port)
	p2p := NewP2PManager(node)
	propagator := NewPropagator(node)

	return &NetworkService{
		config:     config,
		node:       node,
		p2p:        p2p,
		propagator: propagator,
	}
}

// Start initializes and starts the network service.
func (ns *NetworkService) Start() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if ns.running {
		return nil
	}

	// Start the ZMQ node
	if err := ns.node.Start(); err != nil {
		return fmt.Errorf("failed to start ZMQ node: %w", err)
	}

	// Start P2P manager
	ns.p2p.Start()

	// Start propagator
	ns.propagator.Start()

	// Discover peers from seed nodes
	if len(ns.config.SeedNodes) > 0 {
		if err := ns.p2p.DiscoverPeers(ns.config.SeedNodes); err != nil {
			log.Printf("Warning: peer discovery failed: %v", err)
		}
	}

	// Announce ourselves to the network
	if err := ns.p2p.AnnounceSelf(); err != nil {
		log.Printf("Warning: self-announce failed: %v", err)
	}

	ns.running = true
	log.Printf("NetworkService started: %s at %s:%d", ns.config.NodeID, ns.config.Host, ns.config.Port)
	return nil
}

// Stop gracefully shuts down the network service.
func (ns *NetworkService) Stop() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.running {
		return
	}

	// Stop in reverse order
	ns.propagator.Stop()
	ns.p2p.Stop()
	ns.node.Stop()

	ns.running = false
	log.Printf("NetworkService stopped: %s", ns.config.NodeID)
}

// GetStatus returns the current status of the network service.
func (ns *NetworkService) GetStatus() NetworkStatus {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	healthyPeers := ns.p2p.GetHealthyPeers()
	nodeStats := ns.node.GetStats()

	return NetworkStatus{
		NodeID:       ns.config.NodeID,
		Address:      fmt.Sprintf("tcp://%s:%d", ns.config.Host, ns.config.Port),
		IsRunning:    ns.running,
		PeerCount:    ns.p2p.PeerCount(),
		HealthyPeers: len(healthyPeers),
		NodeStats:    nodeStats,
	}
}

// BroadcastBlock propagates a block to all peers in the network.
func (ns *NetworkService) BroadcastBlock(blockData []byte) error {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if !ns.running {
		return ErrNodeNotRunning
	}

	return ns.propagator.PropagateBlock(blockData)
}

// BroadcastTransaction propagates a transaction to all peers in the network.
func (ns *NetworkService) BroadcastTransaction(txData []byte) error {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if !ns.running {
		return ErrNodeNotRunning
	}

	return ns.propagator.PropagateTransaction(txData)
}

// SendDirect sends a message directly to a specific peer.
func (ns *NetworkService) SendDirect(peerID string, payload map[string]interface{}) error {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if !ns.running {
		return ErrNodeNotRunning
	}

	return ns.node.SendDirect(peerID, payload)
}

// RegisterPeer adds a peer to the network.
func (ns *NetworkService) RegisterPeer(peerID, address string, publicKey []byte) {
	ns.node.RegisterPeer(peerID, address, publicKey)
}

// UnregisterPeer removes a peer from the network.
func (ns *NetworkService) UnregisterPeer(peerID string) {
	ns.node.UnregisterPeer(peerID)
}

// GetPeers returns all known peers.
func (ns *NetworkService) GetPeers() map[string]*PeerInfo {
	return ns.node.GetPeers()
}

// GetHealthyPeers returns all healthy (active) peers.
func (ns *NetworkService) GetHealthyPeers() []*PeerInfo {
	return ns.p2p.GetHealthyPeers()
}

// SetMessageHandler sets a custom handler for received messages.
func (ns *NetworkService) SetMessageHandler(handler MessageHandler) {
	ns.node.SetHandler(handler)
}

// GetPropagatorStats returns propagation statistics.
func (ns *NetworkService) GetPropagatorStats() PropagatorStats {
	return ns.propagator.GetStats()
}

// IsRunning returns whether the service is currently running.
func (ns *NetworkService) IsRunning() bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.running
}

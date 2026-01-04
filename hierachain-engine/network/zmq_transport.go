// Package network provides ZeroMQ-based P2P networking for HieraChain.
//
// This package implements:
//   - ZmqNode: ZeroMQ transport with ROUTER/DEALER pattern
//   - P2PManager: Peer discovery and management
//   - Propagator: Message propagation with gossip protocol
package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-zeromq/zmq4"
)

// Common errors for network operations
var (
	ErrNodeNotRunning = errors.New("node is not running")
	ErrPeerNotFound   = errors.New("peer not found")
	ErrSendFailed     = errors.New("failed to send message")
)

// PeerInfo contains information about a network peer.
type PeerInfo struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	PublicKey []byte    `json:"public_key,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
}

// Message represents a network message.
type Message struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Nonce     string                 `json:"nonce,omitempty"`
	Hops      int                    `json:"hops,omitempty"`
}

// MessageHandler is a callback for processing received messages.
type MessageHandler func(msg *Message) error

// ZmqNode is a ZeroMQ-based network node.
type ZmqNode struct {
	nodeID  string
	host    string
	port    int
	address string

	ctx    context.Context
	cancel context.CancelFunc

	router  zmq4.Socket            // ROUTER socket for receiving
	dealers map[string]zmq4.Socket // DEALER sockets for sending (per peer)

	peers map[string]*PeerInfo
	mu    sync.RWMutex

	// Message handling
	handler MessageHandler
	msgChan chan *Message

	// Replay protection
	replayCache     map[string]time.Time
	replayCacheMu   sync.RWMutex
	replayTolerance time.Duration

	running bool
	wg      sync.WaitGroup
}

// NewZmqNode creates a new ZeroMQ node.
func NewZmqNode(nodeID string, host string, port int) *ZmqNode {
	ctx, cancel := context.WithCancel(context.Background())

	return &ZmqNode{
		nodeID:          nodeID,
		host:            host,
		port:            port,
		address:         fmt.Sprintf("tcp://%s:%d", host, port),
		ctx:             ctx,
		cancel:          cancel,
		dealers:         make(map[string]zmq4.Socket),
		peers:           make(map[string]*PeerInfo),
		msgChan:         make(chan *Message, 1000),
		replayCache:     make(map[string]time.Time),
		replayTolerance: 60 * time.Second,
	}
}

// Start begins the node's network operations.
func (n *ZmqNode) Start() error {
	n.mu.Lock()
	if n.running {
		n.mu.Unlock()
		return errors.New("node already running")
	}

	// Create ROUTER socket for receiving messages
	n.router = zmq4.NewRouter(n.ctx, zmq4.WithID(zmq4.SocketIdentity(n.nodeID)))

	// Bind to address
	if err := n.router.Listen(n.address); err != nil {
		n.mu.Unlock()
		return fmt.Errorf("failed to bind router: %w", err)
	}

	n.running = true
	n.mu.Unlock()

	// Start receiver goroutine
	n.wg.Add(1)
	go n.receiverLoop()

	// Start message processor
	n.wg.Add(1)
	go n.messageProcessor()

	// Start replay cache cleaner
	n.wg.Add(1)
	go n.replayCacheCleaner()

	return nil
}

// Stop gracefully shuts down the node.
func (n *ZmqNode) Stop() {
	n.mu.Lock()
	if !n.running {
		n.mu.Unlock()
		return
	}
	n.running = false
	n.mu.Unlock()

	// Cancel context to stop goroutines
	n.cancel()

	// Close router socket (best effort - ignore errors during shutdown)
	if n.router != nil {
		if err := n.router.Close(); err != nil {
			// Log in production; during shutdown, errors are expected
			_ = err // G104: explicitly acknowledge
		}
	}

	// Close all dealer sockets (best effort)
	for _, dealer := range n.dealers {
		if err := dealer.Close(); err != nil {
			_ = err // G104: explicitly acknowledge during cleanup
		}
	}

	// Wait for goroutines to finish
	n.wg.Wait()

	close(n.msgChan)
}

// RegisterPeer adds a peer to the known peers list.
func (n *ZmqNode) RegisterPeer(peerID, address string, publicKey []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.peers[peerID] = &PeerInfo{
		ID:        peerID,
		Address:   address,
		PublicKey: publicKey,
		LastSeen:  time.Now(),
	}
}

// UnregisterPeer removes a peer from the known peers list.
func (n *ZmqNode) UnregisterPeer(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(n.peers, peerID)

	// Close dealer socket if exists (best effort)
	if dealer, ok := n.dealers[peerID]; ok {
		if err := dealer.Close(); err != nil {
			_ = err // G104: explicitly acknowledge during cleanup
		}
		delete(n.dealers, peerID)
	}
}

// SetHandler sets the message handler callback.
func (n *ZmqNode) SetHandler(handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handler = handler
}

// SendDirect sends a message directly to a specific peer.
func (n *ZmqNode) SendDirect(peerID string, payload map[string]interface{}) error {
	n.mu.RLock()
	if !n.running {
		n.mu.RUnlock()
		return ErrNodeNotRunning
	}

	peer, ok := n.peers[peerID]
	if !ok {
		n.mu.RUnlock()
		return ErrPeerNotFound
	}
	n.mu.RUnlock()

	// Get or create dealer socket
	dealer, err := n.getOrCreateDealer(peerID, peer.Address)
	if err != nil {
		return err
	}

	// Create message
	msg := &Message{
		Type:      "direct",
		From:      n.nodeID,
		To:        peerID,
		Payload:   payload,
		Timestamp: time.Now(),
		Nonce:     fmt.Sprintf("%d-%s", time.Now().UnixNano(), n.nodeID),
	}

	// Serialize and send
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msgFrame := zmq4.NewMsg(data)
	if err := dealer.Send(msgFrame); err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	return nil
}

// Broadcast sends a message to all registered peers.
func (n *ZmqNode) Broadcast(payload map[string]interface{}, exclude []string) error {
	n.mu.RLock()
	if !n.running {
		n.mu.RUnlock()
		return ErrNodeNotRunning
	}

	peers := make(map[string]*PeerInfo)
	for id, peer := range n.peers {
		peers[id] = peer
	}
	n.mu.RUnlock()

	// Create exclude set
	excludeSet := make(map[string]bool)
	for _, id := range exclude {
		excludeSet[id] = true
	}

	var lastErr error
	for peerID := range peers {
		if excludeSet[peerID] {
			continue
		}
		if err := n.SendDirect(peerID, payload); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetPeers returns a copy of all registered peers.
func (n *ZmqNode) GetPeers() map[string]*PeerInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make(map[string]*PeerInfo)
	for id, peer := range n.peers {
		peers[id] = &PeerInfo{
			ID:        peer.ID,
			Address:   peer.Address,
			PublicKey: peer.PublicKey,
			LastSeen:  peer.LastSeen,
		}
	}
	return peers
}

// Messages returns the channel for received messages.
func (n *ZmqNode) Messages() <-chan *Message {
	return n.msgChan
}

// getOrCreateDealer gets or creates a DEALER socket for a peer.
func (n *ZmqNode) getOrCreateDealer(peerID, address string) (zmq4.Socket, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if dealer, ok := n.dealers[peerID]; ok {
		return dealer, nil
	}

	// Create new DEALER socket
	dealer := zmq4.NewDealer(n.ctx, zmq4.WithID(zmq4.SocketIdentity(n.nodeID)))

	if err := dealer.Dial(address); err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	n.dealers[peerID] = dealer
	return dealer, nil
}

// receiverLoop continuously receives messages from the ROUTER socket.
func (n *ZmqNode) receiverLoop() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ctx.Done():
			return
		default:
			msg, err := n.router.Recv()
			if err != nil {
				// Check if context cancelled
				select {
				case <-n.ctx.Done():
					return
				default:
					continue
				}
			}

			// Parse message
			var netMsg Message
			if err := json.Unmarshal(msg.Bytes(), &netMsg); err != nil {
				continue
			}

			// Check replay
			if !n.isValidReplay(&netMsg) {
				continue
			}

			// Update peer last seen
			n.mu.Lock()
			if peer, ok := n.peers[netMsg.From]; ok {
				peer.LastSeen = time.Now()
			}
			n.mu.Unlock()

			// Send to channel (non-blocking)
			select {
			case n.msgChan <- &netMsg:
			default:
				// Channel full, drop message
			}
		}
	}
}

// messageProcessor processes messages from the channel.
func (n *ZmqNode) messageProcessor() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ctx.Done():
			return
		case msg, ok := <-n.msgChan:
			if !ok {
				return
			}

			n.mu.RLock()
			handler := n.handler
			n.mu.RUnlock()

			if handler != nil {
				_ = handler(msg)
			}
		}
	}
}

// isValidReplay checks if a message is not a replay attack.
func (n *ZmqNode) isValidReplay(msg *Message) bool {
	if msg.Nonce == "" {
		return true // No nonce, skip replay check
	}

	n.replayCacheMu.Lock()
	defer n.replayCacheMu.Unlock()

	// Check if already seen
	if _, seen := n.replayCache[msg.Nonce]; seen {
		return false
	}

	// Check timestamp tolerance
	if time.Since(msg.Timestamp) > n.replayTolerance {
		return false
	}

	// Add to cache
	n.replayCache[msg.Nonce] = time.Now()
	return true
}

// replayCacheCleaner periodically cleans old entries from replay cache.
func (n *ZmqNode) replayCacheCleaner() {
	defer n.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.cleanReplayCache()
		}
	}
}

// cleanReplayCache removes old entries from the replay cache.
func (n *ZmqNode) cleanReplayCache() {
	n.replayCacheMu.Lock()
	defer n.replayCacheMu.Unlock()

	cutoff := time.Now().Add(-n.replayTolerance)
	for nonce, ts := range n.replayCache {
		if ts.Before(cutoff) {
			delete(n.replayCache, nonce)
		}
	}
}

// NodeStats contains node statistics.
type NodeStats struct {
	NodeID    string `json:"node_id"`
	Address   string `json:"address"`
	PeerCount int    `json:"peer_count"`
	IsRunning bool   `json:"is_running"`
	QueueSize int    `json:"queue_size"`
}

// GetStats returns current node statistics.
func (n *ZmqNode) GetStats() NodeStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return NodeStats{
		NodeID:    n.nodeID,
		Address:   n.address,
		PeerCount: len(n.peers),
		IsRunning: n.running,
		QueueSize: len(n.msgChan),
	}
}

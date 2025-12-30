package network

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// Propagator handles message propagation across the network using gossip protocol.
type Propagator struct {
	node *ZmqNode

	// Seen messages cache (hash -> timestamp)
	seenMessages sync.Map

	// Configuration
	maxHops       int
	cacheExpiry   time.Duration
	cleanInterval time.Duration

	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewPropagator creates a new message propagator.
func NewPropagator(node *ZmqNode) *Propagator {
	return &Propagator{
		node:          node,
		maxHops:       5,
		cacheExpiry:   5 * time.Minute,
		cleanInterval: time.Minute,
		stopChan:      make(chan struct{}),
	}
}

// Start begins propagation operations.
func (p *Propagator) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	// Start cache cleaner
	p.wg.Add(1)
	go p.cacheCleaner()
}

// Stop stops propagation operations.
func (p *Propagator) Stop() {
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

// Propagate sends a message to all peers using gossip protocol.
func (p *Propagator) Propagate(msgType string, payload map[string]interface{}) error {
	msg := &Message{
		Type:      msgType,
		From:      p.node.nodeID,
		Payload:   payload,
		Timestamp: time.Now(),
		Hops:      0,
	}

	// Mark as seen
	hash := p.hashMessage(msg)
	p.seenMessages.Store(hash, time.Now())

	// Broadcast to all peers
	return p.node.Broadcast(payload, nil)
}

// PropagateBlock broadcasts a block to all peers.
func (p *Propagator) PropagateBlock(blockData []byte) error {
	return p.Propagate("block", map[string]interface{}{
		"action": "new_block",
		"data":   string(blockData),
	})
}

// PropagateTransaction broadcasts a transaction to all peers.
func (p *Propagator) PropagateTransaction(txData []byte) error {
	return p.Propagate("transaction", map[string]interface{}{
		"action": "new_transaction",
		"data":   string(txData),
	})
}

// HandleIncoming processes an incoming message for propagation.
// Returns true if the message should be processed, false if it's a duplicate.
func (p *Propagator) HandleIncoming(msg *Message) bool {
	hash := p.hashMessage(msg)

	// Check if already seen
	if p.IsDuplicate(hash) {
		return false
	}

	// Mark as seen
	p.seenMessages.Store(hash, time.Now())

	// Check hop count
	if msg.Hops >= p.maxHops {
		return true // Process but don't propagate further
	}

	// Increment hops and propagate
	msg.Hops++

	// Propagate to all peers except sender
	_ = p.node.Broadcast(msg.Payload, []string{msg.From})

	return true
}

// IsDuplicate checks if a message hash has been seen before.
func (p *Propagator) IsDuplicate(hash string) bool {
	_, seen := p.seenMessages.Load(hash)
	return seen
}

// hashMessage creates a hash of the message for deduplication.
func (p *Propagator) hashMessage(msg *Message) string {
	// Hash based on type, from, payload, and timestamp
	data := struct {
		Type      string
		From      string
		Payload   map[string]interface{}
		Timestamp int64
	}{
		Type:      msg.Type,
		From:      msg.From,
		Payload:   msg.Payload,
		Timestamp: msg.Timestamp.UnixNano(),
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// cacheCleaner periodically cleans old entries from the seen messages cache.
func (p *Propagator) cacheCleaner() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cleanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.cleanCache()
		}
	}
}

// cleanCache removes expired entries from the seen messages cache.
func (p *Propagator) cleanCache() {
	cutoff := time.Now().Add(-p.cacheExpiry)

	p.seenMessages.Range(func(key, value interface{}) bool {
		if ts, ok := value.(time.Time); ok {
			if ts.Before(cutoff) {
				p.seenMessages.Delete(key)
			}
		}
		return true
	})
}

// SetMaxHops sets the maximum number of hops for message propagation.
func (p *Propagator) SetMaxHops(hops int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxHops = hops
}

// PropagatorStats contains propagator statistics.
type PropagatorStats struct {
	MaxHops   int  `json:"max_hops"`
	CacheSize int  `json:"cache_size"`
	IsRunning bool `json:"is_running"`
}

// GetStats returns propagator statistics.
func (p *Propagator) GetStats() PropagatorStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	cacheSize := 0
	p.seenMessages.Range(func(key, value interface{}) bool {
		cacheSize++
		return true
	})

	return PropagatorStats{
		MaxHops:   p.maxHops,
		CacheSize: cacheSize,
		IsRunning: p.running,
	}
}

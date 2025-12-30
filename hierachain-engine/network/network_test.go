package network

import (
	"testing"
	"time"
)

func TestNewZmqNode(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)
	if node == nil {
		t.Fatal("NewZmqNode returned nil")
	}

	if node.nodeID != "test-node" {
		t.Errorf("Expected nodeID 'test-node', got %s", node.nodeID)
	}

	if node.address != "tcp://127.0.0.1:5555" {
		t.Errorf("Expected address 'tcp://127.0.0.1:5555', got %s", node.address)
	}
}

func TestZmqNodeRegisterPeer(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)

	node.RegisterPeer("peer1", "tcp://127.0.0.1:5556", nil)

	peers := node.GetPeers()
	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	if peers["peer1"] == nil {
		t.Error("peer1 not found")
	}

	// Unregister
	node.UnregisterPeer("peer1")
	peers = node.GetPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 peers after unregister, got %d", len(peers))
	}
}

func TestMessage(t *testing.T) {
	msg := &Message{
		Type:      "test",
		From:      "node1",
		To:        "node2",
		Payload:   map[string]interface{}{"data": "hello"},
		Timestamp: time.Now(),
	}

	if msg.Type != "test" {
		t.Errorf("Expected type 'test', got %s", msg.Type)
	}

	if msg.Payload["data"] != "hello" {
		t.Error("Payload data mismatch")
	}
}

func TestNodeStats(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)
	node.RegisterPeer("peer1", "tcp://127.0.0.1:5556", nil)

	stats := node.GetStats()

	if stats.NodeID != "test-node" {
		t.Errorf("Expected NodeID 'test-node', got %s", stats.NodeID)
	}

	if stats.PeerCount != 1 {
		t.Errorf("Expected PeerCount 1, got %d", stats.PeerCount)
	}

	if stats.IsRunning {
		t.Error("Node should not be running")
	}
}

func TestNewP2PManager(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)
	p2p := NewP2PManager(node)

	if p2p == nil {
		t.Fatal("NewP2PManager returned nil")
	}

	if p2p.PeerCount() != 0 {
		t.Errorf("Expected 0 peers, got %d", p2p.PeerCount())
	}
}

func TestNewPropagator(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)
	prop := NewPropagator(node)

	if prop == nil {
		t.Fatal("NewPropagator returned nil")
	}

	stats := prop.GetStats()
	if stats.MaxHops != 5 {
		t.Errorf("Expected MaxHops 5, got %d", stats.MaxHops)
	}
}

func TestPropagatorIsDuplicate(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)
	prop := NewPropagator(node)

	hash := "test-hash-123"

	// First check - not a duplicate
	if prop.IsDuplicate(hash) {
		t.Error("Should not be duplicate initially")
	}

	// Store the hash
	prop.seenMessages.Store(hash, time.Now())

	// Second check - should be duplicate
	if !prop.IsDuplicate(hash) {
		t.Error("Should be duplicate after storing")
	}
}

func TestPropagatorSetMaxHops(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)
	prop := NewPropagator(node)

	prop.SetMaxHops(10)

	stats := prop.GetStats()
	if stats.MaxHops != 10 {
		t.Errorf("Expected MaxHops 10, got %d", stats.MaxHops)
	}
}

func TestP2PManagerGetHealthyPeers(t *testing.T) {
	node := NewZmqNode("test-node", "127.0.0.1", 5555)
	p2p := NewP2PManager(node)

	// Add a peer
	node.RegisterPeer("peer1", "tcp://127.0.0.1:5556", nil)
	p2p.knownPeers["peer1"] = &PeerInfo{
		ID:       "peer1",
		Address:  "tcp://127.0.0.1:5556",
		LastSeen: time.Now(),
	}

	healthy := p2p.GetHealthyPeers()
	if len(healthy) != 1 {
		t.Errorf("Expected 1 healthy peer, got %d", len(healthy))
	}
}

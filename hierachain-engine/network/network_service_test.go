package network

import (
	"testing"
)

func TestNewNetworkService(t *testing.T) {
	config := DefaultNetworkConfig()
	ns := NewNetworkService(config)

	if ns == nil {
		t.Fatal("NewNetworkService returned nil")
	}

	if ns.config.NodeID != "node-1" {
		t.Errorf("Expected NodeID 'node-1', got %s", ns.config.NodeID)
	}

	if ns.node == nil {
		t.Error("ZmqNode should be initialized")
	}

	if ns.p2p == nil {
		t.Error("P2PManager should be initialized")
	}

	if ns.propagator == nil {
		t.Error("Propagator should be initialized")
	}
}

func TestDefaultNetworkConfig(t *testing.T) {
	config := DefaultNetworkConfig()

	if config.NodeID != "node-1" {
		t.Errorf("Expected NodeID 'node-1', got %s", config.NodeID)
	}

	if config.Host != "127.0.0.1" {
		t.Errorf("Expected Host '127.0.0.1', got %s", config.Host)
	}

	if config.Port != 5555 {
		t.Errorf("Expected Port 5555, got %d", config.Port)
	}

	if len(config.SeedNodes) != 0 {
		t.Errorf("Expected empty SeedNodes, got %v", config.SeedNodes)
	}
}

func TestNetworkServiceGetStatusNotRunning(t *testing.T) {
	config := DefaultNetworkConfig()
	ns := NewNetworkService(config)

	status := ns.GetStatus()

	if status.NodeID != "node-1" {
		t.Errorf("Expected NodeID 'node-1', got %s", status.NodeID)
	}

	if status.IsRunning {
		t.Error("Service should not be running yet")
	}

	if status.PeerCount != 0 {
		t.Errorf("Expected 0 peers, got %d", status.PeerCount)
	}
}

func TestNetworkServiceIsRunning(t *testing.T) {
	config := DefaultNetworkConfig()
	ns := NewNetworkService(config)

	if ns.IsRunning() {
		t.Error("Service should not be running initially")
	}
}

func TestNetworkServiceRegisterPeer(t *testing.T) {
	config := DefaultNetworkConfig()
	ns := NewNetworkService(config)

	ns.RegisterPeer("peer1", "tcp://127.0.0.1:5556", nil)

	peers := ns.GetPeers()
	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	if peers["peer1"] == nil {
		t.Error("peer1 not found")
	}

	// Unregister
	ns.UnregisterPeer("peer1")
	peers = ns.GetPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 peers after unregister, got %d", len(peers))
	}
}

func TestNetworkServiceBroadcastBeforeStart(t *testing.T) {
	config := DefaultNetworkConfig()
	ns := NewNetworkService(config)

	// Should fail because service is not running
	err := ns.BroadcastBlock([]byte("test-block"))
	if err != ErrNodeNotRunning {
		t.Errorf("Expected ErrNodeNotRunning, got %v", err)
	}

	err = ns.BroadcastTransaction([]byte("test-tx"))
	if err != ErrNodeNotRunning {
		t.Errorf("Expected ErrNodeNotRunning, got %v", err)
	}
}

func TestNetworkServiceSendDirectBeforeStart(t *testing.T) {
	config := DefaultNetworkConfig()
	ns := NewNetworkService(config)

	err := ns.SendDirect("peer1", map[string]interface{}{"data": "test"})
	if err != ErrNodeNotRunning {
		t.Errorf("Expected ErrNodeNotRunning, got %v", err)
	}
}

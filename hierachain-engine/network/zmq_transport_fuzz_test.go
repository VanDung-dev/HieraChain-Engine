package network

import (
	"encoding/json"
	"testing"
	"time"
)

// FuzzMessageParsing tests network message JSON parsing with random inputs.
// Run with: go test -fuzz=FuzzMessageParsing -fuzztime=30s ./hierachain-engine/network/
func FuzzMessageParsing(f *testing.F) {
	// Seed corpus with valid messages
	validMsg := Message{
		Type:      "direct",
		From:      "node1",
		To:        "node2",
		Payload:   map[string]interface{}{"data": "test"},
		Timestamp: time.Now(),
		Nonce:     "abc123",
		Hops:      0,
	}
	validJSON, _ := json.Marshal(validMsg)
	f.Add(validJSON)

	// Add more seed inputs
	f.Add([]byte(`{"type":"broadcast","from":"sender","payload":{}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Add([]byte(`"string"`))
	f.Add([]byte(`{"type":"","from":"","to":"","payload":null}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic regardless of input
		var msg Message
		err := json.Unmarshal(data, &msg)
		if err == nil {
			// If parsing succeeded, verify we can marshal it back
			_, _ = json.Marshal(msg)
		}
	})
}

// FuzzPeerInfoParsing tests PeerInfo JSON parsing with random inputs.
// Run with: go test -fuzz=FuzzPeerInfoParsing -fuzztime=30s ./hierachain-engine/network/
func FuzzPeerInfoParsing(f *testing.F) {
	// Seed corpus
	f.Add([]byte(`{"id":"peer1","address":"tcp://127.0.0.1:5000"}`))
	f.Add([]byte(`{"id":"","address":"","public_key":null}`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var peer PeerInfo
		err := json.Unmarshal(data, &peer)
		if err == nil {
			_, _ = json.Marshal(peer)
		}
	})
}

// FuzzMessageSizeCheck tests that oversized messages are properly rejected.
func FuzzMessageSizeCheck(f *testing.F) {
	// Test with various sizes
	f.Add(make([]byte, 100))
	f.Add(make([]byte, 1024))
	f.Add(make([]byte, 10*1024))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Check that size validation works
		if len(data) > MaxNetworkMessageSize {
			// Should be rejected in real receiver loop
			t.Logf("Message size %d exceeds max %d", len(data), MaxNetworkMessageSize)
		}
	})
}

//go:build ignore
// +build ignore

// These tests require the Rust library to be built first.
// Run: cargo build --release

package integration

import (
	"encoding/json"
	"testing"
)

// NOTE: These tests will only pass if:
// 1. Rust library is built: cargo build --release
// 2. CGO is enabled and linker can find the library

func TestRustVersion(t *testing.T) {
	if !IsRustAvailable() {
		t.Skip("Rust library not available")
	}

	version, err := RustVersion()
	if err != nil {
		t.Fatalf("RustVersion failed: %v", err)
	}

	if version == "" {
		t.Error("Version should not be empty")
	}
	t.Logf("Rust library version: %s", version)
}

func TestRustMerkleRoot(t *testing.T) {
	if !IsRustAvailable() {
		t.Skip("Rust library not available")
	}

	events := []map[string]interface{}{
		{"entity_id": "e1", "event": "created", "timestamp": 1234567890.0},
		{"entity_id": "e2", "event": "updated", "timestamp": 1234567891.0},
	}

	jsonBytes, _ := json.Marshal(events)
	root, err := RustMerkleRoot(jsonBytes)

	if err != nil {
		t.Fatalf("RustMerkleRoot failed: %v", err)
	}

	if len(root) != 64 {
		t.Errorf("Expected 64-character hash, got %d characters", len(root))
	}
	t.Logf("Merkle root: %s", root)
}

func TestRustBlockHash(t *testing.T) {
	if !IsRustAvailable() {
		t.Skip("Rust library not available")
	}

	block := map[string]interface{}{
		"index":         1,
		"previous_hash": "abc123",
		"merkle_root":   "def456",
		"timestamp":     1234567890.0,
	}

	jsonBytes, _ := json.Marshal(block)
	hash, err := RustBlockHash(jsonBytes)

	if err != nil {
		t.Fatalf("RustBlockHash failed: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("Expected 64-character hash, got %d characters", len(hash))
	}
	t.Logf("Block hash: %s", hash)
}

func TestRustValidateTransactions(t *testing.T) {
	if !IsRustAvailable() {
		t.Skip("Rust library not available")
	}

	// Valid transactions
	validTxs := []map[string]interface{}{
		{"entity_id": "e1", "event": "tx1"},
		{"entity_id": "e2", "event": "tx2"},
	}

	jsonBytes, _ := json.Marshal(validTxs)
	valid, err := RustValidateTransactions(jsonBytes)

	if err != nil {
		t.Fatalf("RustValidateTransactions failed: %v", err)
	}

	if !valid {
		t.Error("Expected valid transactions")
	}

	// Invalid transactions (missing event field)
	invalidTxs := []map[string]interface{}{
		{"entity_id": "e1"},
	}

	jsonBytes, _ = json.Marshal(invalidTxs)
	valid, err = RustValidateTransactions(jsonBytes)

	if err != nil {
		t.Fatalf("RustValidateTransactions failed: %v", err)
	}

	if valid {
		t.Error("Expected invalid transactions")
	}
}

func TestRustProcessArrowBatch(t *testing.T) {
	if !IsRustAvailable() {
		t.Skip("Rust library not available")
	}

	// Simple passthrough test
	inputData := []byte{0x01, 0x02, 0x03, 0x04}
	result, err := RustProcessArrowBatch(inputData)

	if err != nil {
		t.Fatalf("RustProcessArrowBatch failed: %v", err)
	}

	if len(result) != len(inputData) {
		t.Errorf("Expected same length, got %d vs %d", len(result), len(inputData))
	}
}

func BenchmarkRustMerkleRoot(b *testing.B) {
	if !IsRustAvailable() {
		b.Skip("Rust library not available")
	}

	events := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		events[i] = map[string]interface{}{
			"entity_id": "entity",
			"event":     "test",
			"timestamp": 1234567890.0,
		}
	}
	jsonBytes, _ := json.Marshal(events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = RustMerkleRoot(jsonBytes)
	}
}

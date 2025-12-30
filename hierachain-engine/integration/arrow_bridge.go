package integration

import (
	data "github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/data"
	arrowlib "github.com/apache/arrow-go/v18/arrow"
)

// BridgeSerializeForRust converts a Go Arrow Record to IPC bytes for Rust processing.
func BridgeSerializeForRust(record arrowlib.Record) ([]byte, error) {
	writer := data.NewIPCWriter()
	return writer.SerializeToIPC(record)
}

// BridgeDeserializeFromRust converts Rust Arrow IPC bytes to a Go Arrow Record.
func BridgeDeserializeFromRust(ipcBytes []byte) (arrowlib.Record, error) {
	writer := data.NewIPCWriter()
	return writer.DeserializeFromIPC(ipcBytes)
}

// ProcessEventsViaRust converts events to Arrow, processes through Rust, and returns Arrow.
// This provides a seamless Go <-> Rust data pipeline using Arrow IPC.
func ProcessEventsViaRust(events []data.EventJSON) (arrowlib.Record, error) {
	converter := data.NewConverter()

	// Convert events to Arrow
	record, err := converter.EventsToArrowBatch(events)
	if err != nil {
		return nil, err
	}
	defer record.Release()

	// Serialize to IPC
	ipcBytes, err := BridgeSerializeForRust(record)
	if err != nil {
		return nil, err
	}

	// Process through Rust
	processedBytes, err := RustProcessArrowBatch(ipcBytes)
	if err != nil {
		return nil, err
	}

	// Deserialize result
	return BridgeDeserializeFromRust(processedBytes)
}

// CalculateMerkleRootViaRust calculates Merkle root for events using Rust.
// This leverages Rust's optimized cryptographic implementation.
func CalculateMerkleRootViaRust(eventsJSON []byte) (string, error) {
	return RustMerkleRoot(eventsJSON)
}

// CalculateBlockHashViaRust calculates block hash using Rust.
func CalculateBlockHashViaRust(blockJSON []byte) (string, error) {
	return RustBlockHash(blockJSON)
}

// ValidateTransactionsViaRust validates transactions using Rust.
func ValidateTransactionsViaRust(transactionsJSON []byte) (bool, error) {
	return RustValidateTransactions(transactionsJSON)
}

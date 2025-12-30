// Package data provides Apache Arrow schema definitions matching HieraChain-Consensus (Rust).
// Schemas defined here MUST match exactly with src/core/schemas.rs to ensure
// Arrow IPC compatibility between Go and Rust components.
package data

import (
	"github.com/apache/arrow-go/v18/arrow"
)

// EventSchema returns the Arrow schema for an Event.
// Matches Rust: src/core/schemas.rs::get_event_schema()
//
// Fields:
//   - entity_id: string (nullable) - Entity identifier
//   - event: string (nullable) - Event type name
//   - timestamp: float64 (nullable) - Unix timestamp
//   - details: map<string, string> (nullable) - Key-value metadata
//   - data: binary (nullable) - Raw event data
func EventSchema() *arrow.Schema {
	return arrow.NewSchema(
		[]arrow.Field{
			{Name: "entity_id", Type: arrow.BinaryTypes.String, Nullable: true},
			{Name: "event", Type: arrow.BinaryTypes.String, Nullable: true},
			{Name: "timestamp", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
			{
				Name: "details",
				Type: arrow.MapOf(
					arrow.BinaryTypes.String, // key type
					arrow.BinaryTypes.String, // value type
				),
				Nullable: true,
			},
			{Name: "data", Type: arrow.BinaryTypes.Binary, Nullable: true},
		},
		nil,
	)
}

// BlockHeaderSchema returns the Arrow schema for a Block Header.
// Matches Rust: src/core/schemas.rs::get_block_header_schema()
//
// Fields:
//   - index: int64 (nullable) - Block index/height
//   - timestamp: float64 (nullable) - Block creation timestamp
//   - previous_hash: string (nullable) - Hash of previous block
//   - nonce: int64 (nullable) - Block nonce
//   - merkle_root: string (nullable) - Merkle root of events
//   - hash: string (nullable) - Block hash
func BlockHeaderSchema() *arrow.Schema {
	return arrow.NewSchema(
		[]arrow.Field{
			{Name: "index", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			{Name: "timestamp", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
			{Name: "previous_hash", Type: arrow.BinaryTypes.String, Nullable: true},
			{Name: "nonce", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			{Name: "merkle_root", Type: arrow.BinaryTypes.String, Nullable: true},
			{Name: "hash", Type: arrow.BinaryTypes.String, Nullable: true},
		},
		nil,
	)
}

// eventStructFields returns the struct fields for an event within a block.
// Used internally by BlockSchema.
func eventStructFields() []arrow.Field {
	return []arrow.Field{
		{Name: "entity_id", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "event", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "timestamp", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{
			Name: "details",
			Type: arrow.MapOf(
				arrow.BinaryTypes.String,
				arrow.BinaryTypes.String,
			),
			Nullable: true,
		},
		{Name: "data", Type: arrow.BinaryTypes.Binary, Nullable: true},
	}
}

// BlockSchema returns the Arrow schema for a full Block (header + events).
// Matches Rust: src/core/schemas.rs::get_block_schema()
//
// Fields:
//   - index: int64 - Block index/height
//   - timestamp: float64 - Block creation timestamp
//   - previous_hash: string - Hash of previous block
//   - nonce: int64 - Block nonce
//   - merkle_root: string - Merkle root of events
//   - hash: string - Block hash
//   - events: list<struct> - List of events in the block
func BlockSchema() *arrow.Schema {
	eventStruct := arrow.StructOf(eventStructFields()...)

	return arrow.NewSchema(
		[]arrow.Field{
			{Name: "index", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			{Name: "timestamp", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
			{Name: "previous_hash", Type: arrow.BinaryTypes.String, Nullable: true},
			{Name: "nonce", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			{Name: "merkle_root", Type: arrow.BinaryTypes.String, Nullable: true},
			{Name: "hash", Type: arrow.BinaryTypes.String, Nullable: true},
			{
				Name:     "events",
				Type:     arrow.ListOf(eventStruct),
				Nullable: true,
			},
		},
		nil,
	)
}

package data

import (
	"encoding/json"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
)

func TestEventSchema(t *testing.T) {
	schema := EventSchema()

	if schema.NumFields() != 5 {
		t.Errorf("Expected 5 fields, got %d", schema.NumFields())
	}

	expectedFields := []struct {
		name     string
		nullable bool
	}{
		{"entity_id", true},
		{"event", true},
		{"timestamp", true},
		{"details", true},
		{"data", true},
	}

	for i, expected := range expectedFields {
		field := schema.Field(i)
		if field.Name != expected.name {
			t.Errorf("Field %d: expected name %s, got %s", i, expected.name, field.Name)
		}
		if field.Nullable != expected.nullable {
			t.Errorf("Field %s: expected nullable=%v, got %v",
				expected.name, expected.nullable, field.Nullable)
		}
	}
}

func TestBlockHeaderSchema(t *testing.T) {
	schema := BlockHeaderSchema()

	if schema.NumFields() != 6 {
		t.Errorf("Expected 6 fields, got %d", schema.NumFields())
	}

	expectedNames := []string{
		"index", "timestamp", "previous_hash", "nonce", "merkle_root", "hash",
	}

	for i, name := range expectedNames {
		if schema.Field(i).Name != name {
			t.Errorf("Field %d: expected %s, got %s", i, name, schema.Field(i).Name)
		}
	}
}

func TestBlockSchema(t *testing.T) {
	schema := BlockSchema()

	if schema.NumFields() != 7 {
		t.Errorf("Expected 7 fields, got %d", schema.NumFields())
	}

	// Check that the last field is "events" with List type
	eventsField := schema.Field(6)
	if eventsField.Name != "events" {
		t.Errorf("Expected field 6 to be 'events', got %s", eventsField.Name)
	}

	if eventsField.Type.ID() != arrow.LIST {
		t.Errorf("Expected 'events' to be List type, got %s", eventsField.Type.ID())
	}
}

func TestConverterJSONToArrowRoundTrip(t *testing.T) {
	converter := NewConverter()

	events := []EventJSON{
		{
			EntityID:  "entity-1",
			Event:     "created",
			Timestamp: 1704067200.0,
			Details:   map[string]string{"key1": "value1"},
			Data:      []byte("test data"),
		},
		{
			EntityID:  "entity-2",
			Event:     "updated",
			Timestamp: 1704067300.0,
			Details:   map[string]string{"key2": "value2"},
			Data:      nil,
		},
	}

	// Convert to Arrow
	record, err := converter.EventsToArrowBatch(events)
	if err != nil {
		t.Fatalf("Failed to convert to Arrow: %v", err)
	}
	defer record.Release()

	if record.NumRows() != 2 {
		t.Errorf("Expected 2 rows, got %d", record.NumRows())
	}

	// Convert back to JSON
	jsonBytes, err := converter.ArrowBatchToJSON(record)
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	var result []EventJSON
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 events, got %d", len(result))
	}

	if result[0].EntityID != "entity-1" {
		t.Errorf("Expected entity_id 'entity-1', got %s", result[0].EntityID)
	}
}

func TestValidateSchema(t *testing.T) {
	converter := NewConverter()

	events := []EventJSON{
		{EntityID: "test", Event: "test", Timestamp: 0.0},
	}

	record, err := converter.EventsToArrowBatch(events)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}
	defer record.Release()

	// Should pass validation with correct schema
	if err := ValidateSchema(record, EventSchema()); err != nil {
		t.Errorf("Validation should pass: %v", err)
	}

	// Should fail with different schema
	if err := ValidateSchema(record, BlockHeaderSchema()); err == nil {
		t.Error("Validation should fail with wrong schema")
	}
}

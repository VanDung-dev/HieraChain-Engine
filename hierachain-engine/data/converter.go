package data

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// EventJSON represents an event in JSON format for conversion.
type EventJSON struct {
	EntityID  string            `json:"entity_id"`
	Event     string            `json:"event"`
	Timestamp float64           `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
	Data      []byte            `json:"data,omitempty"`
}

// TransactionJSON represents a transaction in JSON format.
type TransactionJSON struct {
	TxID      string            `json:"tx_id"`
	EntityID  string            `json:"entity_id"`
	EventType string            `json:"event_type"`
	Timestamp float64           `json:"timestamp"`
	Data      []byte            `json:"data,omitempty"`
	Signature string            `json:"signature,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

// Converter handles JSON to Arrow conversion.
type Converter struct {
	allocator memory.Allocator
	schema    *arrow.Schema
}

// NewConverter creates a new Converter with the default memory allocator.
func NewConverter() *Converter {
	return &Converter{
		allocator: memory.DefaultAllocator,
		schema:    EventSchema(),
	}
}

// NewConverterWithSchema creates a Converter with a custom schema.
func NewConverterWithSchema(schema *arrow.Schema) *Converter {
	return &Converter{
		allocator: memory.DefaultAllocator,
		schema:    schema,
	}
}

// EventsToArrowBatch converts a slice of EventJSON to Arrow RecordBatch.
func (c *Converter) EventsToArrowBatch(events []EventJSON) (arrow.Record, error) {
	if len(events) == 0 {
		return nil, errors.New("empty events slice")
	}

	builder := array.NewRecordBuilder(c.allocator, c.schema)
	defer builder.Release()

	entityIDBuilder := builder.Field(0).(*array.StringBuilder)
	eventBuilder := builder.Field(1).(*array.StringBuilder)
	timestampBuilder := builder.Field(2).(*array.Float64Builder)
	detailsBuilder := builder.Field(3).(*array.MapBuilder)
	dataBuilder := builder.Field(4).(*array.BinaryBuilder)

	keyBuilder := detailsBuilder.KeyBuilder().(*array.StringBuilder)
	valueBuilder := detailsBuilder.ItemBuilder().(*array.StringBuilder)

	for _, event := range events {
		entityIDBuilder.Append(event.EntityID)
		eventBuilder.Append(event.Event)
		timestampBuilder.Append(event.Timestamp)

		if event.Details != nil && len(event.Details) > 0 {
			detailsBuilder.Append(true)
			for k, v := range event.Details {
				keyBuilder.Append(k)
				valueBuilder.Append(v)
			}
		} else {
			detailsBuilder.AppendNull()
		}

		if event.Data != nil {
			dataBuilder.Append(event.Data)
		} else {
			dataBuilder.AppendNull()
		}
	}

	return builder.NewRecord(), nil
}

// JSONToArrowBatch converts JSON bytes to Arrow RecordBatch.
func (c *Converter) JSONToArrowBatch(jsonData []byte) (arrow.Record, error) {
	var events []EventJSON
	if err := json.Unmarshal(jsonData, &events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return c.EventsToArrowBatch(events)
}

// ArrowBatchToJSON converts an Arrow RecordBatch back to JSON bytes.
func (c *Converter) ArrowBatchToJSON(record arrow.Record) ([]byte, error) {
	if record == nil || record.NumRows() == 0 {
		return []byte("[]"), nil
	}

	events := make([]EventJSON, record.NumRows())

	entityIDCol := record.Column(0).(*array.String)
	eventCol := record.Column(1).(*array.String)
	timestampCol := record.Column(2).(*array.Float64)
	detailsCol := record.Column(3).(*array.Map)
	dataCol := record.Column(4).(*array.Binary)

	for i := int64(0); i < record.NumRows(); i++ {
		idx := int(i)
		events[idx] = EventJSON{
			EntityID:  entityIDCol.Value(idx),
			Event:     eventCol.Value(idx),
			Timestamp: timestampCol.Value(idx),
		}

		if !detailsCol.IsNull(idx) {
			events[idx].Details = extractMapValues(detailsCol, idx)
		}

		if !dataCol.IsNull(idx) {
			events[idx].Data = dataCol.Value(idx)
		}
	}

	return json.Marshal(events)
}

// extractMapValues extracts key-value pairs from a Map column at the given index.
func extractMapValues(mapCol *array.Map, idx int) map[string]string {
	result := make(map[string]string)

	offsets := mapCol.Offsets()
	start := offsets[idx]
	end := offsets[idx+1]

	keys := mapCol.Keys().(*array.String)
	values := mapCol.Items().(*array.String)

	for j := start; j < end; j++ {
		key := keys.Value(int(j))
		value := values.Value(int(j))
		result[key] = value
	}

	return result
}

// ValidateSchema checks if a record matches the expected schema.
func ValidateSchema(record arrow.Record, expected *arrow.Schema) error {
	if record == nil {
		return errors.New("record is nil")
	}

	actual := record.Schema()
	if actual.NumFields() != expected.NumFields() {
		return fmt.Errorf("field count mismatch: got %d, expected %d",
			actual.NumFields(), expected.NumFields())
	}

	for i := 0; i < actual.NumFields(); i++ {
		actualField := actual.Field(i)
		expectedField := expected.Field(i)

		if actualField.Name != expectedField.Name {
			return fmt.Errorf("field %d name mismatch: got %s, expected %s",
				i, actualField.Name, expectedField.Name)
		}

		if !arrow.TypeEqual(actualField.Type, expectedField.Type) {
			return fmt.Errorf("field %s type mismatch: got %s, expected %s",
				actualField.Name, actualField.Type, expectedField.Type)
		}
	}

	return nil
}

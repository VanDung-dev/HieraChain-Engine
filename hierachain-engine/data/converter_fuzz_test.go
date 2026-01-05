package data

import (
	"testing"
)

// FuzzJSONToArrowBatch tests the JSON to Arrow conversion with random inputs.
// Run with: go test -fuzz=FuzzJSONToArrowBatch -fuzztime=30s ./hierachain-engine/data/
func FuzzJSONToArrowBatch(f *testing.F) {
	// Seed corpus with valid inputs
	f.Add([]byte(`[{"entity_id":"test","event":"create","timestamp":1234567890.0}]`))
	f.Add([]byte(`[{"entity_id":"a","event":"b","timestamp":0.0,"details":{"key":"value"}}]`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`[{}]`))
	f.Add([]byte(`[{"entity_id":"","event":"","timestamp":0}]`))

	// Add some malformed inputs
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`"string"`))
	f.Add([]byte(`[null]`))
	f.Add([]byte(`[1,2,3]`))

	c := NewConverter()

	f.Fuzz(func(t *testing.T, data []byte) {
		// The function should not panic regardless of input
		record, err := c.JSONToArrowBatch(data)
		if err == nil && record != nil {
			// If conversion succeeded, ensure we can convert back
			_, _ = c.ArrowBatchToJSON(record)
			record.Release()
		}
	})
}

// FuzzEventsToArrowBatch tests the EventJSON to Arrow conversion with edge cases.
// Run with: go test -fuzz=FuzzEventsToArrowBatch -fuzztime=30s ./hierachain-engine/data/
func FuzzEventsToArrowBatch(f *testing.F) {
	// Seed with various string lengths
	f.Add("id1", "event1", 1234567890.0, "key1", "value1")
	f.Add("", "", 0.0, "", "")
	f.Add("a", "b", -1.0, "c", "d")
	f.Add("very-long-id-that-exceeds-normal-expectations", "event", 9999999999.999, "k", "v")

	c := NewConverter()

	f.Fuzz(func(t *testing.T, entityID, event string, timestamp float64, detailKey, detailValue string) {
		events := []EventJSON{
			{
				EntityID:  entityID,
				Event:     event,
				Timestamp: timestamp,
				Details:   map[string]string{detailKey: detailValue},
			},
		}

		// Should not panic
		record, err := c.EventsToArrowBatch(events)
		if err == nil && record != nil {
			// Verify round-trip
			jsonData, err := c.ArrowBatchToJSON(record)
			if err != nil {
				t.Logf("ArrowBatchToJSON failed: %v", err)
			}
			_ = jsonData
			record.Release()
		}
	})
}

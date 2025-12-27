// Package arrow provides Arrow IPC serialization for zero-copy data transfer.
package arrow

import (
	"bytes"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// IPCWriter writes Arrow RecordBatches to IPC format.
type IPCWriter struct {
	allocator memory.Allocator
}

// NewIPCWriter creates a new IPCWriter.
func NewIPCWriter() *IPCWriter {
	return &IPCWriter{
		allocator: memory.DefaultAllocator,
	}
}

// SerializeToIPC serializes an Arrow Record to IPC bytes.
func (w *IPCWriter) SerializeToIPC(record arrow.Record) ([]byte, error) {
	var buf bytes.Buffer

	writer := ipc.NewWriter(&buf, ipc.WithSchema(record.Schema()))
	defer writer.Close()

	if err := writer.Write(record); err != nil {
		return nil, fmt.Errorf("failed to write record: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeFromIPC deserializes IPC bytes to an Arrow Record.
func (w *IPCWriter) DeserializeFromIPC(data []byte) (arrow.Record, error) {
	reader, err := ipc.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Release()

	if !reader.Next() {
		if reader.Err() != nil {
			return nil, reader.Err()
		}
		return nil, fmt.Errorf("no records in IPC data")
	}

	record := reader.Record()
	record.Retain() // Retain the record to prevent it from being released

	return record, nil
}

// SerializeMultipleToIPC serializes multiple records to IPC bytes.
func (w *IPCWriter) SerializeMultipleToIPC(records []arrow.Record) ([]byte, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to serialize")
	}

	var buf bytes.Buffer
	writer := ipc.NewWriter(&buf, ipc.WithSchema(records[0].Schema()))
	defer writer.Close()

	for i, record := range records {
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write record %d: %w", i, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeAllFromIPC deserializes IPC bytes to all Arrow Records.
func (w *IPCWriter) DeserializeAllFromIPC(data []byte) ([]arrow.Record, error) {
	reader, err := ipc.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Release()

	var records []arrow.Record
	for reader.Next() {
		record := reader.Record()
		record.Retain()
		records = append(records, record)
	}

	if reader.Err() != nil {
		// Release any records we've already retained
		for _, r := range records {
			r.Release()
		}
		return nil, reader.Err()
	}

	return records, nil
}

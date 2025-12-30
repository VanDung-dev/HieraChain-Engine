package api

import (
	"bytes"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// ArrowHandler handles processing of Arrow IPC batches.
type ArrowHandler struct {
	mem memory.Allocator
}

// NewArrowHandler creates a new ArrowHandler.
func NewArrowHandler() *ArrowHandler {
	return &ArrowHandler{
		mem: memory.NewGoAllocator(),
	}
}

// ProcessBatch parses the input bytes as an Arrow IPC stream and returns a response.
// For now, it simply validates the IPC stream and allows it.
// In the future, this will extract transactions and forward them to the Core Engine.
func (h *ArrowHandler) ProcessBatch(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("received empty data")
	}

	reader, err := ipc.NewReader(bytes.NewReader(data), ipc.WithAllocator(h.mem))
	if err != nil {
		return nil, fmt.Errorf("failed to create IPC reader: %w", err)
	}
	defer reader.Release()

	// Read first record batch to ensure validity
	if reader.Next() {
		rec := reader.Record()
		rec.Retain()
		defer rec.Release()
	}

	if reader.Err() != nil {
		return nil, fmt.Errorf("error reading Arrow stream: %w", reader.Err())
	}

	return h.createSuccessResponse()
}

func (h *ArrowHandler) createSuccessResponse() ([]byte, error) {
	return []byte("OK"), nil // Temporary simplification for Phase 1 verification
}

package api

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// MaxMessageSize is the maximum allowed message size (50MB).
// This prevents DoS attacks via oversized messages.
const MaxMessageSize = 50 * 1024 * 1024 // 50MB

// ErrMessageTooLarge is returned when a message exceeds MaxMessageSize.
var ErrMessageTooLarge = errors.New("message size exceeds maximum allowed size")

// ReadMessage reads a length-prefixed message from the reader.
// Format: [4 bytes length (BigEndian)] [N bytes payload]
func ReadMessage(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// Prevent DoS by limiting message size
	if length > MaxMessageSize {
		return nil, fmt.Errorf("%w: %d bytes (max: %d)", ErrMessageTooLarge, length, MaxMessageSize)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("failed to read message body: %w", err)
	}

	return buf, nil
}

// WriteMessage writes a length-prefixed message to the writer.
// Format: [4 bytes length (BigEndian)] [N bytes payload]
func WriteMessage(w io.Writer, data []byte) error {
	// Check for integer overflow before conversion (G115 fix)
	if len(data) > math.MaxUint32 {
		return fmt.Errorf("%w: data length %d exceeds uint32 max", ErrMessageTooLarge, len(data))
	}

	// Check against our limit
	if len(data) > MaxMessageSize {
		return fmt.Errorf("%w: %d bytes (max: %d)", ErrMessageTooLarge, len(data), MaxMessageSize)
	}

	length := uint32(len(data)) // #nosec G115 - bounds checked above
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write message body: %w", err)
	}

	return nil
}

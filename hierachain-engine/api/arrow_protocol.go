package api

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadMessage reads a length-prefixed message from the reader.
// Format: [4 bytes length (BigEndian)] [N bytes payload]
func ReadMessage(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
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
	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write message body: %w", err)
	}

	return nil
}

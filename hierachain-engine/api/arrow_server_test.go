package api

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func TestArrowServer_BasicConnection(t *testing.T) {
	// 1. Start Server
	server := NewArrowServer()
	addr := "127.0.0.1:0" // random port
	err := server.StartAsync(addr)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	realAddr := server.listener.Addr().String()

	// 2. Connect Client
	conn, err := net.Dial("tcp", realAddr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// 3. Prepare Dummy Arrow Data
	mem := memory.NewGoAllocator()
	schema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "int32_col", Type: arrow.PrimitiveTypes.Int32},
		},
		nil,
	)

	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	b.Field(0).(*array.Int32Builder).AppendValues([]int32{1, 2, 3, 4, 5}, nil)
	rec := b.NewRecord()
	defer rec.Release()

	// Serialize to IPC Stream
	var buf bytes.Buffer
	writer := ipc.NewWriter(&buf, ipc.WithSchema(schema))
	if err := writer.Write(rec); err != nil {
		t.Fatalf("Failed to write record: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	arrowData := buf.Bytes()

	// 4. Send Request (Length + Data)
	if err := WriteMessage(conn, arrowData); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// 5. Read Response
	respData, err := ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// 6. Verify Response
	if string(respData) != "OK" {
		t.Errorf("Expected response 'OK', got '%s'", string(respData))
	}
}

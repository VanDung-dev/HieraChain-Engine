package api

import (
	"context"
	"testing"
	"time"

	pb "github.com/VanDung-dev/HieraChain-Engine/api/proto"
)

func TestNewServer(t *testing.T) {
	server, err := NewServer(nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.workerPool == nil {
		t.Fatal("WorkerPool not initialized")
	}
	if server.mempool == nil {
		t.Fatal("Mempool not initialized")
	}
}

func TestNewServerWithConfig(t *testing.T) {
	config := &ServerConfig{
		Address:        ":50052",
		WorkerPoolSize: 50,
		MempoolSize:    5000,
	}
	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("NewServer with config failed: %v", err)
	}
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestHealthCheck(t *testing.T) {
	server, _ := NewServer(nil)
	server.running = true
	server.startTime = time.Now()

	ctx := context.Background()
	resp, err := server.HealthCheck(ctx, &pb.Empty{})

	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if !resp.Healthy {
		t.Error("Expected healthy=true")
	}
	if resp.Version != Version {
		t.Errorf("Expected version %s, got %s", Version, resp.Version)
	}
	if resp.Stats == nil {
		t.Error("Expected stats not nil")
	}
}

func TestSubmitBatch_Empty(t *testing.T) {
	server, _ := NewServer(nil)
	server.running = true

	ctx := context.Background()

	// Empty batch should return error
	_, err := server.SubmitBatch(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil batch")
	}

	_, err = server.SubmitBatch(ctx, &pb.TransactionBatch{})
	if err == nil {
		t.Error("Expected error for empty batch")
	}
}

func TestSubmitBatch_Valid(t *testing.T) {
	server, _ := NewServer(nil)
	server.running = true

	ctx := context.Background()

	batch := &pb.TransactionBatch{
		Transactions: []*pb.Transaction{
			{
				TxId:      "tx-1",
				EntityId:  "entity-1",
				EventType: "created",
			},
			{
				TxId:      "tx-2",
				EntityId:  "entity-2",
				EventType: "updated",
			},
		},
	}

	result, err := server.SubmitBatch(ctx, batch)
	if err != nil {
		t.Fatalf("SubmitBatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}
	if !result.Success {
		t.Errorf("Expected success=true, errors: %v", result.Errors)
	}
	if len(result.ProcessedTxIds) != 2 {
		t.Errorf("Expected 2 processed IDs, got %d", len(result.ProcessedTxIds))
	}
	if result.ProcessingTimeMs < 0 {
		t.Error("Processing time should be >= 0")
	}
}

func TestSubmitBatch_InvalidTransaction(t *testing.T) {
	server, _ := NewServer(nil)
	server.running = true

	ctx := context.Background()

	batch := &pb.TransactionBatch{
		Transactions: []*pb.Transaction{
			{
				TxId:     "", // Missing ID
				EntityId: "entity-1",
			},
		},
	}

	result, err := server.SubmitBatch(ctx, batch)
	if err != nil {
		t.Fatalf("SubmitBatch failed unexpectedly: %v", err)
	}
	if result.Success {
		t.Error("Expected success=false for invalid transaction")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestGetStats(t *testing.T) {
	server, _ := NewServer(nil)
	server.running = true
	server.startTime = time.Now()

	stats := server.GetStats()
	if stats == nil {
		t.Fatal("Stats is nil")
	}
	if stats["version"] != Version {
		t.Errorf("Expected version %s, got %v", Version, stats["version"])
	}
	if stats["worker_pool"] == nil {
		t.Error("Worker pool stats missing")
	}
	if stats["mempool"] == nil {
		t.Error("Mempool stats missing")
	}
}

func TestStartAsyncAndStop(t *testing.T) {
	server, _ := NewServer(nil)

	// Start on a random available port
	err := server.StartAsync(":0")
	if err != nil {
		t.Fatalf("StartAsync failed: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check it's running
	server.mu.RLock()
	running := server.running
	server.mu.RUnlock()

	if !running {
		t.Error("Server should be running")
	}

	// Stop
	server.Stop()

	// Check it stopped
	server.mu.RLock()
	running = server.running
	server.mu.RUnlock()

	if running {
		t.Error("Server should not be running after Stop")
	}
}

func BenchmarkSubmitBatch(b *testing.B) {
	server, _ := NewServer(&ServerConfig{
		WorkerPoolSize: 100,
		MempoolSize:    b.N * 10,
	})
	server.running = true

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := &pb.TransactionBatch{
			Transactions: make([]*pb.Transaction, 10),
		}
		for j := 0; j < 10; j++ {
			batch.Transactions[j] = &pb.Transaction{
				TxId:      "tx-" + string(rune(i*10+j)),
				EntityId:  "entity-1",
				EventType: "test",
			}
		}
		_, _ = server.SubmitBatch(ctx, batch)
	}
}

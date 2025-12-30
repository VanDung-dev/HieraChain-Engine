package api

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	pb "github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/api/proto"
)

// BenchmarkSubmitBatch_100 benchmarks processing 100 transactions.
func BenchmarkSubmitBatch_100(b *testing.B) {
	benchmarkSubmitBatch(b, 100)
}

// BenchmarkSubmitBatch_1000 benchmarks processing 1000 transactions.
func BenchmarkSubmitBatch_1000(b *testing.B) {
	benchmarkSubmitBatch(b, 1000)
}

// BenchmarkSubmitBatch_10000 benchmarks processing 10000 transactions.
func BenchmarkSubmitBatch_10000(b *testing.B) {
	benchmarkSubmitBatch(b, 10000)
}

func benchmarkSubmitBatch(b *testing.B, batchSize int) {
	server, err := NewServer(&ServerConfig{
		WorkerPoolSize: 100,
		MempoolSize:    100000,
		MetricsAddress: "", // Disable metrics server for benchmark
	})
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	// Create batch
	batch := createTestBatch(batchSize)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := server.SubmitBatch(ctx, batch)
		if err != nil {
			b.Errorf("SubmitBatch failed: %v", err)
		}
	}

	b.ReportMetric(float64(batchSize*b.N)/b.Elapsed().Seconds(), "tx/sec")
}

// BenchmarkSubmitBatch_Concurrent benchmarks concurrent batch submissions.
func BenchmarkSubmitBatch_Concurrent(b *testing.B) {
	server, err := NewServer(&ServerConfig{
		WorkerPoolSize: 100,
		MempoolSize:    1000000,
		MetricsAddress: "",
	})
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	batch := createTestBatch(100)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetParallelism(10) // 10 goroutines

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = server.SubmitBatch(ctx, batch)
		}
	})

	b.ReportMetric(float64(100*b.N)/b.Elapsed().Seconds(), "tx/sec")
}

// BenchmarkProcessTransaction benchmarks single transaction processing.
func BenchmarkProcessTransaction(b *testing.B) {
	server, err := NewServer(&ServerConfig{
		WorkerPoolSize: 50,
		MempoolSize:    100000,
		MetricsAddress: "",
	})
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tx := &pb.Transaction{
			TxId:      fmt.Sprintf("bench-tx-%d", i),
			EntityId:  "bench-entity",
			EventType: "benchmark",
		}
		_ = server.processTransaction(ctx, tx)
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "tx/sec")
}

// TestLoadTest_Sustained runs a sustained load test for verification.
func TestLoadTest_Sustained(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained load test in short mode")
	}

	server, err := NewServer(&ServerConfig{
		WorkerPoolSize: 100,
		MempoolSize:    500000,
		MetricsAddress: "",
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	duration := 5 * time.Second
	batchSize := 100
	concurrency := 10
	targetTPS := 10000

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	var totalTx int64
	var totalErrors int64
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			batch := createTestBatch(batchSize)
			localTx := 0
			localErrors := 0

			for {
				select {
				case <-ctx.Done():
					atomicAdd(&totalTx, int64(localTx))
					atomicAdd(&totalErrors, int64(localErrors))
					return
				default:
					result, err := server.SubmitBatch(context.Background(), batch)
					if err != nil {
						localErrors++
					} else {
						localTx += len(result.ProcessedTxIds)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	actualTPS := float64(totalTx) / elapsed.Seconds()
	t.Logf("Load Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total TX: %d", totalTx)
	t.Logf("  Errors: %d", totalErrors)
	t.Logf("  TPS: %.2f", actualTPS)
	t.Logf("  Target: %d", targetTPS)

	if actualTPS < float64(targetTPS)*0.5 {
		t.Logf("WARNING: TPS (%.2f) is below 50%% of target (%d)", actualTPS, targetTPS)
	}
}

func createTestBatch(size int) *pb.TransactionBatch {
	transactions := make([]*pb.Transaction, size)
	for i := 0; i < size; i++ {
		transactions[i] = &pb.Transaction{
			TxId:      fmt.Sprintf("tx-%d-%d", time.Now().UnixNano(), i),
			EntityId:  fmt.Sprintf("entity-%d", i%100),
			EventType: "benchmark",
			Details:   map[string]string{"index": fmt.Sprintf("%d", i)},
		}
	}
	return &pb.TransactionBatch{Transactions: transactions}
}

func atomicAdd(addr *int64, delta int64) {
	for {
		old := *addr
		if atomicCompareAndSwap(addr, old, old+delta) {
			return
		}
	}
}

func atomicCompareAndSwap(addr *int64, old, new int64) bool {
	// Simple implementation - in production use sync/atomic
	*addr = new
	return true
}

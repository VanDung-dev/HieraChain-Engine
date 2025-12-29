package engine

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewMempool(t *testing.T) {
	m := NewMempool(100)
	if m == nil {
		t.Fatal("NewMempool returned nil")
	}
	if m.Size() != 0 {
		t.Errorf("Expected size 0, got %d", m.Size())
	}
	if m.maxSize != 100 {
		t.Errorf("Expected maxSize 100, got %d", m.maxSize)
	}
}

func TestMempoolAdd(t *testing.T) {
	m := NewMempool(10)

	tx := &Transaction{
		ID:        "tx-1",
		EntityID:  "entity-1",
		EventType: "created",
		Priority:  1,
	}

	err := m.Add(tx)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if m.Size() != 1 {
		t.Errorf("Expected size 1, got %d", m.Size())
	}
}

func TestMempoolAddDuplicate(t *testing.T) {
	m := NewMempool(10)

	tx := &Transaction{
		ID:        "tx-1",
		EntityID:  "entity-1",
		EventType: "created",
	}

	_ = m.Add(tx)
	err := m.Add(tx)
	if err != ErrTxAlreadyExists {
		t.Errorf("Expected ErrTxAlreadyExists, got %v", err)
	}
}

func TestMempoolFull(t *testing.T) {
	m := NewMempool(2)

	for i := 0; i < 2; i++ {
		tx := &Transaction{
			ID:        fmt.Sprintf("tx-%d", i),
			EntityID:  "entity",
			EventType: "test",
		}
		_ = m.Add(tx)
	}

	tx := &Transaction{
		ID:        "tx-overflow",
		EntityID:  "entity",
		EventType: "test",
	}
	err := m.Add(tx)
	if err != ErrMempoolFull {
		t.Errorf("Expected ErrMempoolFull, got %v", err)
	}
}

func TestMempoolGet(t *testing.T) {
	m := NewMempool(10)

	tx := &Transaction{
		ID:        "tx-1",
		EntityID:  "entity-1",
		EventType: "created",
	}
	_ = m.Add(tx)

	retrieved := m.Get("tx-1")
	if retrieved == nil {
		t.Fatal("Get returned nil")
	}
	if retrieved.ID != tx.ID {
		t.Errorf("Expected ID %s, got %s", tx.ID, retrieved.ID)
	}

	notFound := m.Get("non-existent")
	if notFound != nil {
		t.Error("Expected nil for non-existent tx")
	}
}

func TestMempoolRemove(t *testing.T) {
	m := NewMempool(10)

	tx := &Transaction{
		ID:        "tx-1",
		EntityID:  "entity-1",
		EventType: "created",
	}
	_ = m.Add(tx)

	removed := m.Remove("tx-1")
	if !removed {
		t.Error("Remove should return true")
	}
	if m.Size() != 0 {
		t.Errorf("Expected size 0 after remove, got %d", m.Size())
	}

	removed = m.Remove("non-existent")
	if removed {
		t.Error("Remove should return false for non-existent")
	}
}

func TestMempoolPopBatch(t *testing.T) {
	m := NewMempool(10)

	// Add transactions with different priorities
	for i := 0; i < 5; i++ {
		tx := &Transaction{
			ID:        fmt.Sprintf("tx-%d", i),
			EntityID:  "entity",
			EventType: "test",
			Priority:  i, // 0, 1, 2, 3, 4
		}
		_ = m.Add(tx)
	}

	// Pop 3 highest priority
	batch := m.PopBatch(3)
	if len(batch) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(batch))
	}

	// Should be in priority order (highest first)
	if batch[0].Priority != 4 {
		t.Errorf("Expected priority 4, got %d", batch[0].Priority)
	}
	if batch[1].Priority != 3 {
		t.Errorf("Expected priority 3, got %d", batch[1].Priority)
	}
	if batch[2].Priority != 2 {
		t.Errorf("Expected priority 2, got %d", batch[2].Priority)
	}

	// Size should be reduced
	if m.Size() != 2 {
		t.Errorf("Expected size 2, got %d", m.Size())
	}
}

func TestMempoolConcurrency(t *testing.T) {
	m := NewMempool(1000)
	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tx := &Transaction{
				ID:        fmt.Sprintf("tx-%d", id),
				EntityID:  "entity",
				EventType: "test",
			}
			_ = m.Add(tx)
		}(i)
	}

	wg.Wait()

	if m.Size() != 100 {
		t.Errorf("Expected 100 transactions, got %d", m.Size())
	}
}

func BenchmarkMempoolAdd(b *testing.B) {
	m := NewMempool(b.N + 1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tx := &Transaction{
			ID:        fmt.Sprintf("tx-%d", i),
			EntityID:  "entity",
			EventType: "test",
			Priority:  i % 10,
			Timestamp: time.Now(),
		}
		_ = m.Add(tx)
	}
}

func BenchmarkMempoolPopBatch(b *testing.B) {
	m := NewMempool(10000)

	// Pre-populate
	for i := 0; i < 10000; i++ {
		tx := &Transaction{
			ID:        fmt.Sprintf("tx-%d", i),
			EntityID:  "entity",
			EventType: "test",
			Priority:  i % 10,
		}
		_ = m.Add(tx)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.PopBatch(100)
		// Re-add for next iteration
		for j := 0; j < 100; j++ {
			tx := &Transaction{
				ID:        fmt.Sprintf("tx-new-%d-%d", i, j),
				EntityID:  "entity",
				EventType: "test",
			}
			_ = m.Add(tx)
		}
	}
}

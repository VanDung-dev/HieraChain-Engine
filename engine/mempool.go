package engine

import (
	"container/heap"
	"errors"
	"sync"
	"time"
)

// Common errors for mempool operations
var (
	ErrMempoolFull     = errors.New("mempool is full")
	ErrTxAlreadyExists = errors.New("transaction already exists")
	ErrTxNotFound      = errors.New("transaction not found")
	ErrInvalidTx       = errors.New("invalid transaction")
)

// Transaction represents a pending transaction in the mempool.
type Transaction struct {
	ID        string                 `json:"id"`
	EntityID  string                 `json:"entity_id"`
	EventType string                 `json:"event_type"`
	Data      []byte                 `json:"data,omitempty"`
	Priority  int                    `json:"priority"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Validate checks if the transaction has required fields.
func (tx *Transaction) Validate() error {
	if tx.ID == "" {
		return errors.New("transaction ID is required")
	}
	if tx.EntityID == "" {
		return errors.New("entity ID is required")
	}
	if tx.EventType == "" {
		return errors.New("event type is required")
	}
	return nil
}

// priorityQueue implements heap.Interface for Transaction priority ordering.
type priorityQueue []*Transaction

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Higher priority first, then earlier timestamp
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	return pq[i].Timestamp.Before(pq[j].Timestamp)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Transaction))
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	tx := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[0 : n-1]
	return tx
}

// Mempool manages pending transactions with thread-safe operations.
type Mempool struct {
	pending map[string]*Transaction
	queue   priorityQueue
	maxSize int
	mu      sync.RWMutex
}

// NewMempool creates a new Mempool with the specified maximum size.
func NewMempool(maxSize int) *Mempool {
	m := &Mempool{
		pending: make(map[string]*Transaction),
		queue:   make(priorityQueue, 0),
		maxSize: maxSize,
	}
	heap.Init(&m.queue)
	return m
}

// Add adds a transaction to the mempool.
// Returns error if mempool is full or transaction already exists.
func (m *Mempool) Add(tx *Transaction) error {
	if tx == nil {
		return ErrInvalidTx
	}

	if err := tx.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if _, exists := m.pending[tx.ID]; exists {
		return ErrTxAlreadyExists
	}

	// Check size limit
	if len(m.pending) >= m.maxSize {
		return ErrMempoolFull
	}

	// Set timestamp if not set
	if tx.Timestamp.IsZero() {
		tx.Timestamp = time.Now()
	}

	// Add to map and priority queue
	m.pending[tx.ID] = tx
	heap.Push(&m.queue, tx)

	return nil
}

// Get retrieves a transaction by ID without removing it.
func (m *Mempool) Get(txID string) *Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pending[txID]
}

// Remove removes a transaction by ID.
// Returns true if the transaction was found and removed.
func (m *Mempool) Remove(txID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pending[txID]; !exists {
		return false
	}

	delete(m.pending, txID)

	// Rebuild the queue without the removed transaction
	newQueue := make(priorityQueue, 0, len(m.queue)-1)
	for _, tx := range m.queue {
		if tx.ID != txID {
			newQueue = append(newQueue, tx)
		}
	}
	m.queue = newQueue
	heap.Init(&m.queue)

	return true
}

// PopBatch removes and returns up to n highest-priority transactions.
func (m *Mempool) PopBatch(n int) []*Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	if n <= 0 || len(m.queue) == 0 {
		return nil
	}

	// Limit to available transactions
	if n > len(m.queue) {
		n = len(m.queue)
	}

	batch := make([]*Transaction, 0, n)
	for i := 0; i < n; i++ {
		tx := heap.Pop(&m.queue).(*Transaction)
		delete(m.pending, tx.ID)
		batch = append(batch, tx)
	}

	return batch
}

// Peek returns up to n highest-priority transactions without removing them.
func (m *Mempool) Peek(n int) []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n <= 0 || len(m.queue) == 0 {
		return nil
	}

	if n > len(m.queue) {
		n = len(m.queue)
	}

	// Create a copy of queue for sorting
	sorted := make(priorityQueue, len(m.queue))
	copy(sorted, m.queue)
	heap.Init(&sorted)

	batch := make([]*Transaction, 0, n)
	for i := 0; i < n; i++ {
		tx := heap.Pop(&sorted).(*Transaction)
		batch = append(batch, tx)
	}

	return batch
}

// Size returns the current number of transactions in the mempool.
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pending)
}

// IsFull returns true if the mempool has reached its maximum size.
func (m *Mempool) IsFull() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pending) >= m.maxSize
}

// Clear removes all transactions from the mempool.
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pending = make(map[string]*Transaction)
	m.queue = make(priorityQueue, 0)
	heap.Init(&m.queue)
}

// Stats returns mempool statistics.
type MempoolStats struct {
	Size      int `json:"size"`
	MaxSize   int `json:"max_size"`
	Available int `json:"available"`
}

func (m *Mempool) Stats() MempoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MempoolStats{
		Size:      len(m.pending),
		MaxSize:   m.maxSize,
		Available: m.maxSize - len(m.pending),
	}
}

// Contains checks if a transaction exists in the mempool.
func (m *Mempool) Contains(txID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.pending[txID]
	return exists
}

package core

import (
	"errors"
	"sync"
	"time"
)

// OrderingStatus represents the status of the ordering service.
type OrderingStatus int

const (
	StatusActive OrderingStatus = iota
	StatusMaintenance
	StatusLockdown
	StatusShutdown
	StatusError
)

func (s OrderingStatus) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusMaintenance:
		return "maintenance"
	case StatusLockdown:
		return "lockdown"
	case StatusShutdown:
		return "shutdown"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// EventStatus represents the processing status of an event.
type EventStatus int

const (
	EventPending EventStatus = iota
	EventProcessing
	EventOrdered
	EventCertified
	EventRejected
)

func (s EventStatus) String() string {
	switch s {
	case EventPending:
		return "pending"
	case EventProcessing:
		return "processing"
	case EventOrdered:
		return "ordered"
	case EventCertified:
		return "certified"
	case EventRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// PendingEvent represents an event waiting to be ordered.
type PendingEvent struct {
	ID         string
	Data       map[string]interface{}
	ChannelID  string
	Submitter  string
	ReceivedAt time.Time
	Status     EventStatus
	Cert       *Certification
}

// Certification contains validation result for an event.
type Certification struct {
	EventID  string
	Valid    bool
	Errors   []string
	CertAt   time.Time
	Metadata map[string]interface{}
}

// ValidationRule is a function that validates event data.
type ValidationRule func(data map[string]interface{}) error

// EventCertifier validates events before ordering.
type EventCertifier struct {
	rules []ValidationRule
	certs map[string]*Certification
	mu    sync.RWMutex
}

// NewEventCertifier creates a new event certifier.
func NewEventCertifier() *EventCertifier {
	return &EventCertifier{
		rules: make([]ValidationRule, 0),
		certs: make(map[string]*Certification),
	}
}

// AddRule registers a validation rule.
func (c *EventCertifier) AddRule(rule ValidationRule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = append(c.rules, rule)
}

// Validate validates an event and returns certification result.
func (c *EventCertifier) Validate(event *PendingEvent) *Certification {
	c.mu.Lock()
	defer c.mu.Unlock()

	cert := &Certification{
		EventID:  event.ID,
		Valid:    true,
		Errors:   make([]string, 0),
		CertAt:   time.Now(),
		Metadata: make(map[string]interface{}),
	}

	// Check required fields
	requiredFields := []string{"entity_id", "event", "timestamp"}
	for _, field := range requiredFields {
		if _, ok := event.Data[field]; !ok {
			cert.Valid = false
			cert.Errors = append(cert.Errors, "missing required field: "+field)
		}
	}

	// Apply custom rules
	for _, rule := range c.rules {
		if err := rule(event.Data); err != nil {
			cert.Valid = false
			cert.Errors = append(cert.Errors, err.Error())
		}
	}

	// Store certification
	c.certs[event.ID] = cert
	event.Cert = cert

	return cert
}

// GetCertification retrieves a certification by event ID.
func (c *EventCertifier) GetCertification(eventID string) *Certification {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.certs[eventID]
}

// BlockBuilder batches certified events into blocks.
type BlockBuilder struct {
	blockSize    int
	batchTimeout time.Duration
	currentBatch []*PendingEvent
	batchIDs     map[string]bool
	batchStart   time.Time
	mu           sync.Mutex
}

// NewBlockBuilder creates a new block builder.
func NewBlockBuilder(blockSize int, timeout time.Duration) *BlockBuilder {
	return &BlockBuilder{
		blockSize:    blockSize,
		batchTimeout: timeout,
		currentBatch: make([]*PendingEvent, 0, blockSize),
		batchIDs:     make(map[string]bool),
		batchStart:   time.Now(),
	}
}

// AddEvent adds a certified event to the current batch.
// Returns the batch if ready for block creation, nil otherwise.
func (b *BlockBuilder) AddEvent(event *PendingEvent) []*PendingEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Skip duplicates
	if b.batchIDs[event.ID] {
		return nil
	}

	// Start timer on first event
	if len(b.currentBatch) == 0 {
		b.batchStart = time.Now()
	}

	b.currentBatch = append(b.currentBatch, event)
	b.batchIDs[event.ID] = true

	// Check if batch is ready
	if b.isReady() {
		return b.finalize()
	}

	return nil
}

// ForceFlush forces block creation from current batch.
func (b *BlockBuilder) ForceFlush() []*PendingEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.currentBatch) == 0 {
		return nil
	}
	return b.finalize()
}

// isReady checks if batch is ready (called with lock held).
func (b *BlockBuilder) isReady() bool {
	if len(b.currentBatch) >= b.blockSize {
		return true
	}
	if time.Since(b.batchStart) >= b.batchTimeout {
		return true
	}
	return false
}

// finalize returns current batch and resets (called with lock held).
func (b *BlockBuilder) finalize() []*PendingEvent {
	batch := b.currentBatch
	b.currentBatch = make([]*PendingEvent, 0, b.blockSize)
	b.batchIDs = make(map[string]bool)
	b.batchStart = time.Now()
	return batch
}

// BatchSize returns current batch size.
func (b *BlockBuilder) BatchSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.currentBatch)
}

// OrderingConfig contains configuration for the ordering service.
type OrderingConfig struct {
	BlockSize    int
	BatchTimeout time.Duration
	Workers      int
	MaxPending   int
}

// DefaultOrderingConfig returns default configuration.
func DefaultOrderingConfig() OrderingConfig {
	return OrderingConfig{
		BlockSize:    500,
		BatchTimeout: 2 * time.Second,
		Workers:      8,
		MaxPending:   10000,
	}
}

// OrderingService coordinates event ordering and block creation.
type OrderingService struct {
	config       OrderingConfig
	status       OrderingStatus
	certifier    *EventCertifier
	blockBuilder *BlockBuilder
	workerPool   *WorkerPool

	eventChan chan *PendingEvent
	blockChan chan []*PendingEvent

	pending map[string]*PendingEvent
	mu      sync.RWMutex

	// Stats
	eventsReceived  int64
	eventsCertified int64
	eventsRejected  int64
	blocksCreated   int64

	// Control
	stopCh  chan struct{}
	wg      sync.WaitGroup
	running bool
}

// NewOrderingService creates a new ordering service.
func NewOrderingService(config OrderingConfig) *OrderingService {
	s := &OrderingService{
		config:       config,
		status:       StatusMaintenance,
		certifier:    NewEventCertifier(),
		blockBuilder: NewBlockBuilder(config.BlockSize, config.BatchTimeout),
		workerPool:   NewWorkerPool("ordering", config.Workers),
		eventChan:    make(chan *PendingEvent, config.MaxPending),
		blockChan:    make(chan []*PendingEvent, 100),
		pending:      make(map[string]*PendingEvent),
		stopCh:       make(chan struct{}),
	}

	// Add default validation rules
	s.addDefaultRules()

	return s
}

// addDefaultRules adds standard validation rules.
func (s *OrderingService) addDefaultRules() {
	// Timestamp validation
	s.certifier.AddRule(func(data map[string]interface{}) error {
		ts, ok := data["timestamp"]
		if !ok {
			return nil // Will be caught by required field check
		}

		var timestamp float64
		switch v := ts.(type) {
		case float64:
			timestamp = v
		case int64:
			timestamp = float64(v)
		case int:
			timestamp = float64(v)
		default:
			return errors.New("invalid timestamp type")
		}

		// Check if within 24 hours
		now := float64(time.Now().Unix())
		if timestamp < now-86400 || timestamp > now+86400 {
			return errors.New("timestamp out of valid range")
		}

		return nil
	})
}

// Start begins the ordering service.
func (s *OrderingService) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("service already running")
	}
	s.running = true
	s.status = StatusActive
	s.mu.Unlock()

	// Start event processor
	s.wg.Add(1)
	go s.processEvents()

	// Start timeout checker
	s.wg.Add(1)
	go s.checkTimeouts()

	return nil
}

// Stop stops the ordering service.
func (s *OrderingService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.status = StatusShutdown
	s.mu.Unlock()

	close(s.stopCh)
	s.wg.Wait()
	s.workerPool.Shutdown()
}

// processEvents is the main event processing loop.
func (s *OrderingService) processEvents() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			// Flush remaining events
			if batch := s.blockBuilder.ForceFlush(); batch != nil {
				s.blockChan <- batch
			}
			return

		case event := <-s.eventChan:
			s.handleEvent(event)
		}
	}
}

// checkTimeouts periodically flushes batches on timeout.
func (s *OrderingService) checkTimeouts() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.BatchTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if batch := s.blockBuilder.ForceFlush(); batch != nil {
				s.mu.Lock()
				s.blocksCreated++
				s.mu.Unlock()
				s.blockChan <- batch
			}
		}
	}
}

// handleEvent processes a single event.
func (s *OrderingService) handleEvent(event *PendingEvent) {
	s.mu.Lock()
	s.eventsReceived++
	s.pending[event.ID] = event
	s.mu.Unlock()

	// Certify event
	event.Status = EventProcessing
	cert := s.certifier.Validate(event)

	if !cert.Valid {
		s.mu.Lock()
		s.eventsRejected++
		delete(s.pending, event.ID)
		s.mu.Unlock()
		event.Status = EventRejected
		return
	}

	s.mu.Lock()
	s.eventsCertified++
	s.mu.Unlock()
	event.Status = EventCertified

	// Add to block builder
	if batch := s.blockBuilder.AddEvent(event); batch != nil {
		s.mu.Lock()
		s.blocksCreated++
		for _, e := range batch {
			delete(s.pending, e.ID)
			e.Status = EventOrdered
		}
		s.mu.Unlock()
		s.blockChan <- batch
	}
}

// SubmitEvent submits an event for ordering.
func (s *OrderingService) SubmitEvent(event *PendingEvent) error {
	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		return errors.New("service not running")
	}
	s.mu.RUnlock()

	event.Status = EventPending
	event.ReceivedAt = time.Now()

	select {
	case s.eventChan <- event:
		return nil
	default:
		return errors.New("event queue full")
	}
}

// Blocks returns the channel for receiving completed blocks.
func (s *OrderingService) Blocks() <-chan []*PendingEvent {
	return s.blockChan
}

// GetStatus returns current service status.
func (s *OrderingService) GetStatus() OrderingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// OrderingStats contains service statistics.
type OrderingStats struct {
	Status          string `json:"status"`
	EventsReceived  int64  `json:"events_received"`
	EventsCertified int64  `json:"events_certified"`
	EventsRejected  int64  `json:"events_rejected"`
	BlocksCreated   int64  `json:"blocks_created"`
	PendingCount    int    `json:"pending_count"`
	BatchSize       int    `json:"current_batch_size"`
}

// GetStats returns service statistics.
func (s *OrderingService) GetStats() OrderingStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return OrderingStats{
		Status:          s.status.String(),
		EventsReceived:  s.eventsReceived,
		EventsCertified: s.eventsCertified,
		EventsRejected:  s.eventsRejected,
		BlocksCreated:   s.blocksCreated,
		PendingCount:    len(s.pending),
		BatchSize:       s.blockBuilder.BatchSize(),
	}
}

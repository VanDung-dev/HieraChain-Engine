package engine

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewEventCertifier(t *testing.T) {
	c := NewEventCertifier()
	if c == nil {
		t.Fatal("NewEventCertifier returned nil")
	}
}

func TestEventCertifierValidate(t *testing.T) {
	c := NewEventCertifier()

	// Valid event
	event := &PendingEvent{
		ID: "event-1",
		Data: map[string]interface{}{
			"entity_id": "entity-1",
			"event":     "created",
			"timestamp": float64(time.Now().Unix()),
		},
	}

	cert := c.Validate(event)
	if !cert.Valid {
		t.Errorf("Expected valid, got errors: %v", cert.Errors)
	}
}

func TestEventCertifierValidateMissingFields(t *testing.T) {
	c := NewEventCertifier()

	// Missing required fields
	event := &PendingEvent{
		ID:   "event-bad",
		Data: map[string]interface{}{},
	}

	cert := c.Validate(event)
	if cert.Valid {
		t.Error("Expected invalid for missing fields")
	}
	if len(cert.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(cert.Errors))
	}
}

func TestBlockBuilder(t *testing.T) {
	bb := NewBlockBuilder(3, time.Second)

	events := make([]*PendingEvent, 0)
	for i := 0; i < 3; i++ {
		event := &PendingEvent{
			ID: fmt.Sprintf("event-%d", i),
			Data: map[string]interface{}{
				"entity_id": "entity",
				"event":     "test",
				"timestamp": float64(time.Now().Unix()),
			},
		}
		result := bb.AddEvent(event)
		if result != nil {
			events = result
		}
	}

	if len(events) != 3 {
		t.Errorf("Expected batch of 3, got %d", len(events))
	}
}

func TestBlockBuilderTimeout(t *testing.T) {
	bb := NewBlockBuilder(100, 50*time.Millisecond)

	event := &PendingEvent{
		ID: "event-1",
		Data: map[string]interface{}{
			"entity_id": "entity",
			"event":     "test",
			"timestamp": float64(time.Now().Unix()),
		},
	}
	bb.AddEvent(event)

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Force flush should return the batch
	batch := bb.ForceFlush()
	if batch == nil {
		t.Error("Expected batch after timeout")
	}
}

func TestOrderingService(t *testing.T) {
	config := OrderingConfig{
		BlockSize:    5,
		BatchTimeout: 100 * time.Millisecond,
		Workers:      2,
		MaxPending:   100,
	}

	svc := NewOrderingService(config)
	if err := svc.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer svc.Stop()

	// Submit events
	for i := 0; i < 5; i++ {
		event := &PendingEvent{
			ID: fmt.Sprintf("event-%d", i),
			Data: map[string]interface{}{
				"entity_id": fmt.Sprintf("entity-%d", i),
				"event":     "created",
				"timestamp": float64(time.Now().Unix()),
			},
		}
		if err := svc.SubmitEvent(event); err != nil {
			t.Errorf("Submit failed: %v", err)
		}
	}

	// Wait for block
	select {
	case block := <-svc.Blocks():
		if len(block) != 5 {
			t.Errorf("Expected block of 5, got %d", len(block))
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for block")
	}
}

func TestOrderingServiceRejectsInvalid(t *testing.T) {
	config := DefaultOrderingConfig()
	config.BlockSize = 10

	svc := NewOrderingService(config)
	_ = svc.Start()
	defer svc.Stop()

	// Submit invalid event (missing fields)
	event := &PendingEvent{
		ID:   "bad-event",
		Data: map[string]interface{}{},
	}
	_ = svc.SubmitEvent(event)

	// Wait a bit for processing
	time.Sleep(50 * time.Millisecond)

	stats := svc.GetStats()
	if stats.EventsRejected != 1 {
		t.Errorf("Expected 1 rejected, got %d", stats.EventsRejected)
	}
}

func TestOrderingServiceConcurrent(t *testing.T) {
	config := OrderingConfig{
		BlockSize:    100,
		BatchTimeout: 500 * time.Millisecond,
		Workers:      8,
		MaxPending:   1000,
	}

	svc := NewOrderingService(config)
	_ = svc.Start()
	defer svc.Stop()

	var wg sync.WaitGroup
	numEvents := 200

	// Concurrent submissions
	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := &PendingEvent{
				ID: fmt.Sprintf("event-%d", id),
				Data: map[string]interface{}{
					"entity_id": fmt.Sprintf("entity-%d", id),
					"event":     "created",
					"timestamp": float64(time.Now().Unix()),
				},
			}
			_ = svc.SubmitEvent(event)
		}(i)
	}

	wg.Wait()

	// Collect blocks
	var totalEvents int
	timeout := time.After(2 * time.Second)

loop:
	for {
		select {
		case block := <-svc.Blocks():
			totalEvents += len(block)
			if totalEvents >= numEvents {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	if totalEvents != numEvents {
		t.Errorf("Expected %d events, got %d", numEvents, totalEvents)
	}
}

func BenchmarkOrderingServiceSubmit(b *testing.B) {
	config := OrderingConfig{
		BlockSize:    1000,
		BatchTimeout: time.Second,
		Workers:      16,
		MaxPending:   100000,
	}

	svc := NewOrderingService(config)
	_ = svc.Start()
	defer svc.Stop()

	// Consumer
	go func() {
		for range svc.Blocks() {
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event := &PendingEvent{
			ID: fmt.Sprintf("event-%d", i),
			Data: map[string]interface{}{
				"entity_id": fmt.Sprintf("entity-%d", i),
				"event":     "created",
				"timestamp": float64(time.Now().Unix()),
			},
		}
		_ = svc.SubmitEvent(event)
	}
}

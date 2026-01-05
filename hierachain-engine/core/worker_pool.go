package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Task represents a processing task for the worker pool.
type Task struct {
	ID          string
	Data        interface{}
	ProcessFunc func(interface{}) (interface{}, error)
	Priority    int
	CreatedAt   time.Time
	Ctx         context.Context
}

// NewTask creates a new task with default values.
func NewTask(id string, data interface{}, fn func(interface{}) (interface{}, error)) *Task {
	return &Task{
		ID:          id,
		Data:        data,
		ProcessFunc: fn,
		Priority:    0,
		CreatedAt:   time.Now(),
		Ctx:         context.Background(),
	}
}

// Result represents the result of task processing.
type Result struct {
	TaskID   string
	Success  bool
	Data     interface{}
	Error    error
	Duration time.Duration
	WorkerID int
}

// PoolStats contains worker pool statistics.
type PoolStats struct {
	Name        string  `json:"name"`
	Workers     int     `json:"workers"`
	Active      int64   `json:"active"`
	Completed   int64   `json:"completed"`
	Failed      int64   `json:"failed"`
	Pending     int     `json:"pending"`
	SuccessRate float64 `json:"success_rate"`
}

// WorkerPool manages a pool of goroutine workers for parallel processing.
type WorkerPool struct {
	name       string
	workers    int
	taskChan   chan *Task
	resultChan chan *Result
	wg         sync.WaitGroup

	// Atomic counters for thread-safe statistics
	active    int64
	completed int64
	failed    int64

	// Control
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      sync.RWMutex
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
func NewWorkerPool(name string, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		name:       name,
		workers:    workers,
		taskChan:   make(chan *Task, workers*100), // buffered channel
		resultChan: make(chan *Result, workers*100),
		ctx:        ctx,
		cancel:     cancel,
		running:    true,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	return pool
}

// worker is the goroutine that processes tasks.
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskChan:
			if !ok {
				return
			}
			p.processTask(id, task)
		}
	}
}

// processTask executes a single task and sends the result.
func (p *WorkerPool) processTask(workerID int, task *Task) {
	atomic.AddInt64(&p.active, 1)
	defer atomic.AddInt64(&p.active, -1)

	start := time.Now()

	result := &Result{
		TaskID:   task.ID,
		WorkerID: workerID,
	}

	// Panic recovery to prevent one task from crashing the entire pool
	defer func() {
		if r := recover(); r != nil {
			result.Success = false
			result.Error = errors.New("panic in task processing: " + panicToString(r))
			result.Duration = time.Since(start)
			atomic.AddInt64(&p.failed, 1)
			p.sendResult(result)
		}
	}()

	// Check context cancellation
	if task.Ctx != nil {
		select {
		case <-task.Ctx.Done():
			result.Success = false
			result.Error = task.Ctx.Err()
			result.Duration = time.Since(start)
			atomic.AddInt64(&p.failed, 1)
			p.sendResult(result)
			return
		default:
		}
	}

	// Execute the task
	if task.ProcessFunc != nil {
		data, err := task.ProcessFunc(task.Data)
		result.Data = data
		result.Error = err
		result.Success = err == nil
	} else {
		result.Error = errors.New("no process function defined")
		result.Success = false
	}

	result.Duration = time.Since(start)

	if result.Success {
		atomic.AddInt64(&p.completed, 1)
	} else {
		atomic.AddInt64(&p.failed, 1)
	}

	p.sendResult(result)
}

// panicToString converts a recovered panic value to a string.
func panicToString(r interface{}) string {
	switch v := r.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return "unknown panic"
	}
}

// sendResult sends a result to the result channel (non-blocking).
func (p *WorkerPool) sendResult(result *Result) {
	select {
	case p.resultChan <- result:
	default:
		// Channel full, result dropped (caller should consume results)
	}
}

// Submit adds a task to the worker pool for processing.
func (p *WorkerPool) Submit(task *Task) error {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	if !running {
		return errors.New("worker pool is shut down")
	}

	select {
	case p.taskChan <- task:
		return nil
	default:
		return errors.New("task queue is full")
	}
}

// SubmitAndWait submits a task and waits for its result.
func (p *WorkerPool) SubmitAndWait(task *Task, timeout time.Duration) (*Result, error) {
	if err := p.Submit(task); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-p.resultChan:
			if result.TaskID == task.ID {
				return result, nil
			}
			// Put back non-matching results (not ideal, but simple)
			p.sendResult(result)
		}
	}
}

// Results returns the result channel for consuming results.
func (p *WorkerPool) Results() <-chan *Result {
	return p.resultChan
}

// GetStats returns current worker pool statistics.
func (p *WorkerPool) GetStats() PoolStats {
	completed := atomic.LoadInt64(&p.completed)
	failed := atomic.LoadInt64(&p.failed)
	total := completed + failed

	var successRate float64
	if total > 0 {
		successRate = float64(completed) / float64(total) * 100
	}

	return PoolStats{
		Name:        p.name,
		Workers:     p.workers,
		Active:      atomic.LoadInt64(&p.active),
		Completed:   completed,
		Failed:      failed,
		Pending:     len(p.taskChan),
		SuccessRate: successRate,
	}
}

// Shutdown gracefully shuts down the worker pool.
func (p *WorkerPool) Shutdown() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	p.cancel()
	close(p.taskChan)
	p.wg.Wait()
	close(p.resultChan)
}

// ShutdownWithTimeout shuts down with a timeout.
func (p *WorkerPool) ShutdownWithTimeout(timeout time.Duration) error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	p.cancel()
	close(p.taskChan)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(p.resultChan)
		return nil
	case <-time.After(timeout):
		return errors.New("shutdown timeout")
	}
}

// IsRunning returns true if the pool is still accepting tasks.
func (p *WorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

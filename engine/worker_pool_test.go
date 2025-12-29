package engine

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool("test", 4)
	defer pool.Shutdown()

	if pool == nil {
		t.Fatal("NewWorkerPool returned nil")
	}

	stats := pool.GetStats()
	if stats.Workers != 4 {
		t.Errorf("Expected 4 workers, got %d", stats.Workers)
	}
	if stats.Name != "test" {
		t.Errorf("Expected name 'test', got %s", stats.Name)
	}
}

func TestWorkerPoolSubmit(t *testing.T) {
	pool := NewWorkerPool("test", 2)
	defer pool.Shutdown()

	var processed int64

	task := NewTask("task-1", "data", func(data interface{}) (interface{}, error) {
		atomic.AddInt64(&processed, 1)
		return data, nil
	})

	err := pool.Submit(task)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Wait for result
	select {
	case result := <-pool.Results():
		if !result.Success {
			t.Errorf("Task should succeed")
		}
		if result.TaskID != "task-1" {
			t.Errorf("Expected task ID 'task-1', got %s", result.TaskID)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for result")
	}

	if atomic.LoadInt64(&processed) != 1 {
		t.Error("Task was not processed")
	}
}

func TestWorkerPoolSubmitWithError(t *testing.T) {
	pool := NewWorkerPool("test", 2)
	defer pool.Shutdown()

	expectedErr := errors.New("task failed")
	task := NewTask("task-error", nil, func(data interface{}) (interface{}, error) {
		return nil, expectedErr
	})

	_ = pool.Submit(task)

	select {
	case result := <-pool.Results():
		if result.Success {
			t.Error("Task should have failed")
		}
		if result.Error == nil {
			t.Error("Expected error in result")
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for result")
	}

	stats := pool.GetStats()
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", stats.Failed)
	}
}

func TestWorkerPoolConcurrency(t *testing.T) {
	pool := NewWorkerPool("test", 8)
	defer pool.Shutdown()

	numTasks := 100
	var completed int64
	var wg sync.WaitGroup

	// Consumer goroutine
	go func() {
		for range pool.Results() {
			atomic.AddInt64(&completed, 1)
			wg.Done()
		}
	}()

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		task := NewTask(fmt.Sprintf("task-%d", i), i, func(data interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond) // Simulate work
			return data, nil
		})
		_ = pool.Submit(task)
	}

	// Wait for all tasks to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout: only %d/%d completed", atomic.LoadInt64(&completed), numTasks)
	}

	if atomic.LoadInt64(&completed) != int64(numTasks) {
		t.Errorf("Expected %d completed, got %d", numTasks, completed)
	}
}

func TestWorkerPoolShutdown(t *testing.T) {
	pool := NewWorkerPool("test", 4)

	// Submit a task
	task := NewTask("task-1", nil, func(data interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return nil, nil
	})
	_ = pool.Submit(task)

	// Shutdown
	pool.Shutdown()

	if pool.IsRunning() {
		t.Error("Pool should not be running after shutdown")
	}

	// Submit after shutdown should fail
	err := pool.Submit(task)
	if err == nil {
		t.Error("Submit after shutdown should fail")
	}
}

func TestWorkerPoolStats(t *testing.T) {
	pool := NewWorkerPool("stats-test", 2)
	defer pool.Shutdown()

	// Submit successful tasks
	for i := 0; i < 5; i++ {
		task := NewTask(fmt.Sprintf("ok-%d", i), nil, func(data interface{}) (interface{}, error) {
			return nil, nil
		})
		_ = pool.Submit(task)
	}

	// Submit failing tasks
	for i := 0; i < 3; i++ {
		task := NewTask(fmt.Sprintf("fail-%d", i), nil, func(data interface{}) (interface{}, error) {
			return nil, errors.New("fail")
		})
		_ = pool.Submit(task)
	}

	// Consume results
	for i := 0; i < 8; i++ {
		<-pool.Results()
	}

	stats := pool.GetStats()
	if stats.Completed != 5 {
		t.Errorf("Expected 5 completed, got %d", stats.Completed)
	}
	if stats.Failed != 3 {
		t.Errorf("Expected 3 failed, got %d", stats.Failed)
	}
}

func BenchmarkWorkerPoolSubmit(b *testing.B) {
	pool := NewWorkerPool("bench", 8)
	defer pool.Shutdown()

	// Consumer goroutine
	go func() {
		for range pool.Results() {
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		task := NewTask(fmt.Sprintf("task-%d", i), i, func(data interface{}) (interface{}, error) {
			return data, nil
		})
		_ = pool.Submit(task)
	}
}

func BenchmarkWorkerPoolThroughput(b *testing.B) {
	pool := NewWorkerPool("throughput", 16)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	var processed int64

	// Consumer
	go func() {
		for range pool.Results() {
			atomic.AddInt64(&processed, 1)
			wg.Done()
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		task := NewTask(fmt.Sprintf("task-%d", i), i, func(data interface{}) (interface{}, error) {
			return data, nil
		})
		_ = pool.Submit(task)
	}

	wg.Wait()
}

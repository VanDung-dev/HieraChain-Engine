// Package api provides the gRPC server implementation for HieraChain Engine.
package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/VanDung-dev/HieraChain-Engine/api/proto"
	"github.com/VanDung-dev/HieraChain-Engine/engine"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Version is the current version of the HieraChain Engine.
const Version = "0.1.0"

// Server implements the HieraChainEngineServer gRPC service.
type Server struct {
	pb.UnimplementedHieraChainEngineServer

	// Core components
	workerPool *engine.WorkerPool
	mempool    *engine.Mempool

	// Server state
	grpcServer *grpc.Server
	listener   net.Listener
	startTime  time.Time

	// Statistics (atomic for thread-safety)
	txProcessed   int64
	blocksCreated int64
	totalTime     int64 // nanoseconds

	// Control
	running bool
	mu      sync.RWMutex
}

// ServerConfig holds configuration for the gRPC server.
type ServerConfig struct {
	// Address to listen on (e.g., ":50051")
	Address string

	// WorkerPoolSize is the number of workers for processing transactions
	WorkerPoolSize int

	// MempoolSize is the maximum number of pending transactions
	MempoolSize int

	// MaxRecvMsgSize is the maximum message size in bytes
	MaxRecvMsgSize int

	// MaxSendMsgSize is the maximum message size in bytes
	MaxSendMsgSize int
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:        ":50051",
		WorkerPoolSize: 100,
		MempoolSize:    10000,
		MaxRecvMsgSize: 16 * 1024 * 1024, // 16MB
		MaxSendMsgSize: 16 * 1024 * 1024, // 16MB
	}
}

// NewServer creates a new gRPC server instance.
func NewServer(config *ServerConfig) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	return &Server{
		workerPool: engine.NewWorkerPool("grpc-workers", config.WorkerPoolSize),
		mempool:    engine.NewMempool(config.MempoolSize),
		startTime:  time.Now(),
		running:    false,
	}, nil
}

// Start starts the gRPC server on the configured address.
func (s *Server) Start(address string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	s.listener = lis

	s.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(16*1024*1024),
		grpc.MaxSendMsgSize(16*1024*1024),
	)
	pb.RegisterHieraChainEngineServer(s.grpcServer, s)

	s.running = true
	s.startTime = time.Now()
	s.mu.Unlock()

	// Start serving (blocking)
	return s.grpcServer.Serve(lis)
}

// StartAsync starts the gRPC server asynchronously and returns immediately.
func (s *Server) StartAsync(address string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	s.listener = lis

	s.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(16*1024*1024),
		grpc.MaxSendMsgSize(16*1024*1024),
	)
	pb.RegisterHieraChainEngineServer(s.grpcServer, s)

	s.running = true
	s.startTime = time.Now()
	s.mu.Unlock()

	// Start serving asynchronously
	go func() {
		_ = s.grpcServer.Serve(lis)
	}()

	return nil
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if s.workerPool != nil {
		s.workerPool.Shutdown()
	}
}

// SubmitBatch processes a batch of transactions.
func (s *Server) SubmitBatch(ctx context.Context, req *pb.TransactionBatch) (*pb.BatchResult, error) {
	if req == nil || len(req.Transactions) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty transaction batch")
	}

	start := time.Now()
	processedIDs := make([]string, 0, len(req.Transactions))
	errors := make([]*pb.TxError, 0)

	// Process each transaction
	for _, tx := range req.Transactions {
		if err := s.processTransaction(ctx, tx); err != nil {
			errors = append(errors, &pb.TxError{
				TxId:         tx.TxId,
				ErrorMessage: err.Error(),
				ErrorCode:    "PROCESSING_ERROR",
			})
		} else {
			processedIDs = append(processedIDs, tx.TxId)
			atomic.AddInt64(&s.txProcessed, 1)
		}
	}

	processingTime := time.Since(start)
	atomic.AddInt64(&s.totalTime, processingTime.Nanoseconds())

	return &pb.BatchResult{
		Success:          len(errors) == 0,
		Message:          fmt.Sprintf("Processed %d/%d transactions", len(processedIDs), len(req.Transactions)),
		ProcessedTxIds:   processedIDs,
		ProcessingTimeMs: processingTime.Milliseconds(),
		Errors:           errors,
	}, nil
}

// StreamTransactions handles bidirectional streaming of transactions.
func (s *Server) StreamTransactions(stream grpc.BidiStreamingServer[pb.Transaction, pb.TxStatus]) error {
	for {
		tx, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive transaction: %v", err)
		}

		// Process the transaction
		procErr := s.processTransaction(stream.Context(), tx)

		// Send status back
		txStatus := &pb.TxStatus{
			TxId:      tx.TxId,
			Timestamp: time.Now().UnixMilli(),
		}

		if procErr != nil {
			txStatus.Status = pb.TxStatus_FAILED
		} else {
			txStatus.Status = pb.TxStatus_CONFIRMED
			atomic.AddInt64(&s.txProcessed, 1)
		}

		if err := stream.Send(txStatus); err != nil {
			return status.Errorf(codes.Internal, "failed to send status: %v", err)
		}
	}
}

// HealthCheck returns the health status of the engine.
func (s *Server) HealthCheck(ctx context.Context, _ *pb.Empty) (*pb.HealthResponse, error) {
	s.mu.RLock()
	running := s.running
	startTime := s.startTime
	s.mu.RUnlock()

	txProcessed := atomic.LoadInt64(&s.txProcessed)
	blocksCreated := atomic.LoadInt64(&s.blocksCreated)
	totalTime := atomic.LoadInt64(&s.totalTime)

	var avgProcessingTime float64
	if txProcessed > 0 {
		avgProcessingTime = float64(totalTime) / float64(txProcessed) / 1e6 // convert to ms
	}

	mempoolStats := s.mempool.Stats()

	return &pb.HealthResponse{
		Healthy:       running,
		Version:       Version,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		Stats: &pb.EngineStats{
			TransactionsProcessed: txProcessed,
			BlocksCreated:         blocksCreated,
			PendingTransactions:   int64(mempoolStats.Size),
			AvgProcessingTimeMs:   avgProcessingTime,
		},
	}, nil
}

// processTransaction handles a single transaction through the worker pool.
func (s *Server) processTransaction(ctx context.Context, tx *pb.Transaction) error {
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}
	if tx.TxId == "" {
		return fmt.Errorf("transaction ID is required")
	}
	if tx.EntityId == "" {
		return fmt.Errorf("entity ID is required")
	}

	// Create internal transaction
	internalTx := &engine.Transaction{
		ID:        tx.TxId,
		EntityID:  tx.EntityId,
		EventType: tx.EventType,
		Data:      tx.ArrowPayload,
		Priority:  0, // Could be derived from tx metadata
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Copy details to metadata
	for k, v := range tx.Details {
		internalTx.Metadata[k] = v
	}

	// Add to mempool
	if err := s.mempool.Add(internalTx); err != nil {
		return fmt.Errorf("failed to add to mempool: %w", err)
	}

	// Submit to worker pool for processing
	task := engine.NewTask(tx.TxId, internalTx, func(data interface{}) (interface{}, error) {
		// Transaction processing logic
		txData := data.(*engine.Transaction)
		_ = txData // Placeholder for actual processing
		return txData, nil
	})
	task.Ctx = ctx

	if err := s.workerPool.Submit(task); err != nil {
		// Remove from mempool if we couldn't submit
		s.mempool.Remove(tx.TxId)
		return fmt.Errorf("failed to submit task: %w", err)
	}

	return nil
}

// GetMempool returns the mempool for external access.
func (s *Server) GetMempool() *engine.Mempool {
	return s.mempool
}

// GetWorkerPool returns the worker pool for external access.
func (s *Server) GetWorkerPool() *engine.WorkerPool {
	return s.workerPool
}

// GetStats returns current server statistics.
func (s *Server) GetStats() map[string]interface{} {
	s.mu.RLock()
	startTime := s.startTime
	s.mu.RUnlock()

	poolStats := s.workerPool.GetStats()
	mempoolStats := s.mempool.Stats()

	return map[string]interface{}{
		"version":                Version,
		"uptime_seconds":         time.Since(startTime).Seconds(),
		"transactions_processed": atomic.LoadInt64(&s.txProcessed),
		"blocks_created":         atomic.LoadInt64(&s.blocksCreated),
		"worker_pool":            poolStats,
		"mempool":                mempoolStats,
	}
}

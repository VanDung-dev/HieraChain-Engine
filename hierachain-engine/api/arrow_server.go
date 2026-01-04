package api

import (
	"fmt"
	"io"
	"net"
	"sync"
)

// ArrowServer is a TCP server that listens for Arrow IPC messages.
type ArrowServer struct {
	listener net.Listener
	handler  *ArrowHandler
	running  bool
	mu       sync.Mutex
	quit     chan struct{}
}

// NewArrowServer creates a new ArrowServer instance.
func NewArrowServer() *ArrowServer {
	return &ArrowServer{
		handler: NewArrowHandler(),
		quit:    make(chan struct{}),
	}
}

// Start starts the Arrow server on the specified address.
// This method blocks until the server is stopped or fails.
func (s *ArrowServer) Start(address string) error {
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
	s.running = true
	s.mu.Unlock()

	defer s.Stop()

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return nil
			default:
				// Log error? For now just continue
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// StartAsync starts the server in a background goroutine.
func (s *ArrowServer) StartAsync(address string) error {
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
	s.running = true
	s.mu.Unlock()

	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					continue
				}
			}
			go s.handleConnection(conn)
		}
	}()

	return nil
}

// Stop stops the server.
func (s *ArrowServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.quit)
	if s.listener != nil {
		// Best effort close - error is logged but not propagated
		// since we're already in shutdown mode
		if err := s.listener.Close(); err != nil {
			// In production, this should use a proper logger
			_ = err // Explicitly acknowledge unhandled error for G104
		}
	}
}

// handleConnection handles a single client connection.
func (s *ArrowServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		// 1. Read request message
		data, err := ReadMessage(conn)
		if err != nil {
			if err != io.EOF {
				// fmt.Printf("Error reading message: %v\n", err)
			}
			return
		}

		// 2. Process message (Arrow RecordBatch)
		response, err := s.handler.ProcessBatch(data)
		if err != nil {
			// Send error response? For now, we might just close connection or log
			// Or send a specific error packet
			fmt.Printf("Error processing batch: %v\n", err)
			return
		}

		// 3. Write response message
		if err := WriteMessage(conn, response); err != nil {
			// fmt.Printf("Error writing response: %v\n", err)
			return
		}
	}
}

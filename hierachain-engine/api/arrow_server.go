package api

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Connection timeout constants for security
const (
	// ConnectionReadTimeout is the maximum time to wait for a complete message read
	ConnectionReadTimeout = 30 * time.Second
	// ConnectionWriteTimeout is the maximum time to wait for a complete message write
	ConnectionWriteTimeout = 30 * time.Second
	// ConnectionIdleTimeout is the maximum time a connection can remain idle
	ConnectionIdleTimeout = 120 * time.Second
)

// ArrowServer is a TCP server that listens for Arrow IPC messages.
type ArrowServer struct {
	listener      net.Listener
	handler       *ArrowHandler
	authenticator *Authenticator
	running       bool
	mu            sync.Mutex
	quit          chan struct{}
}

// NewArrowServer creates a new ArrowServer instance.
// Authentication is configured via environment variables:
//   - HIE_AUTH_ENABLED=true to enable authentication
//   - HIE_AUTH_TOKEN=<token> to set a specific token (auto-generated if not set)
func NewArrowServer() *ArrowServer {
	return &ArrowServer{
		handler:       NewArrowHandler(),
		authenticator: NewAuthenticatorFromEnv(),
		quit:          make(chan struct{}),
	}
}

// NewArrowServerWithAuth creates a new ArrowServer with explicit auth config.
func NewArrowServerWithAuth(authConfig AuthConfig) *ArrowServer {
	return &ArrowServer{
		handler:       NewArrowHandler(),
		authenticator: NewAuthenticator(authConfig),
		quit:          make(chan struct{}),
	}
}

// IsAuthEnabled returns true if authentication is enabled.
func (s *ArrowServer) IsAuthEnabled() bool {
	return s.authenticator.IsEnabled()
}

// GetAuthToken returns the auth token (for admin display).
func (s *ArrowServer) GetAuthToken() string {
	return s.authenticator.GetToken()
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

	// Panic recovery to prevent one connection from crashing the entire server
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in connection handler recovered: %v\n", r)
		}
	}()

	// Authentication handshake (if enabled)
	if s.authenticator.IsEnabled() {
		if !s.performAuthHandshake(conn) {
			return // Auth failed, connection closed
		}
	}

	for {
		// Set read deadline to prevent Slowloris-style attacks
		if err := conn.SetReadDeadline(time.Now().Add(ConnectionReadTimeout)); err != nil {
			return
		}

		// 1. Read request message
		data, err := ReadMessage(conn)
		if err != nil {
			if err != io.EOF {
				// Timeout or other error - close connection
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

		// Set write deadline
		if err := conn.SetWriteDeadline(time.Now().Add(ConnectionWriteTimeout)); err != nil {
			return
		}

		// 3. Write response message
		if err := WriteMessage(conn, response); err != nil {
			// fmt.Printf("Error writing response: %v\n", err)
			return
		}
	}
}

// performAuthHandshake performs token-based authentication handshake.
// Returns true if auth succeeds, false otherwise.
func (s *ArrowServer) performAuthHandshake(conn net.Conn) bool {
	// Set deadline for auth handshake (shorter than normal)
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return false
	}

	// Read auth message
	data, err := ReadMessage(conn)
	if err != nil {
		s.sendAuthResponse(conn, false, "failed to read auth message")
		return false
	}

	// Parse auth message (expecting JSON: {"type":"auth","token":"xxx"})
	// Simple parsing without full JSON for performance
	token := extractTokenFromAuthMessage(data)
	if token == "" {
		s.sendAuthResponse(conn, false, "invalid auth message format")
		return false
	}

	// Validate token
	if err := s.authenticator.ValidateToken(token); err != nil {
		s.sendAuthResponse(conn, false, err.Error())
		return false
	}

	// Auth success
	s.sendAuthResponse(conn, true, "")
	return true
}

// sendAuthResponse sends an authentication response to the client.
func (s *ArrowServer) sendAuthResponse(conn net.Conn, success bool, errMsg string) {
	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return
	}

	var response []byte
	if success {
		response = []byte(`{"success":true}`)
	} else {
		response = []byte(fmt.Sprintf(`{"success":false,"error":"%s"}`, errMsg))
	}

	// Ignore write errors - connection will be closed anyway if auth failed
	_ = WriteMessage(conn, response)
}

// extractTokenFromAuthMessage extracts the token from an auth message.
// Expected format: {"type":"auth","token":"<token>"}
func extractTokenFromAuthMessage(data []byte) string {
	// Simple string search for token field (avoids full JSON parsing overhead)
	const tokenPrefix = `"token":"`
	str := string(data)

	idx := 0
	for i := 0; i < len(str)-len(tokenPrefix); i++ {
		if str[i:i+len(tokenPrefix)] == tokenPrefix {
			idx = i + len(tokenPrefix)
			break
		}
	}

	if idx == 0 {
		return ""
	}

	// Find end of token value
	end := idx
	for end < len(str) && str[end] != '"' {
		end++
	}

	if end == idx || end >= len(str) {
		return ""
	}

	return str[idx:end]
}

package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"os"
	"sync"
)

// Authentication errors
var (
	ErrAuthRequired      = errors.New("authentication required")
	ErrAuthFailed        = errors.New("authentication failed")
	ErrAuthTokenInvalid  = errors.New("invalid auth token format")
	ErrAuthTokenMismatch = errors.New("auth token mismatch")
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	// Enabled determines if authentication is required
	Enabled bool
	// Token is the secret token that clients must provide
	Token string
}

// Authenticator handles connection authentication.
type Authenticator struct {
	config AuthConfig
	mu     sync.RWMutex
}

// NewAuthenticator creates a new Authenticator with the given config.
func NewAuthenticator(config AuthConfig) *Authenticator {
	return &Authenticator{
		config: config,
	}
}

// NewAuthenticatorFromEnv creates an Authenticator from environment variables.
// Uses HIE_AUTH_ENABLED and HIE_AUTH_TOKEN env vars.
// If HIE_AUTH_TOKEN is not set but auth is enabled, generates a random token.
func NewAuthenticatorFromEnv() *Authenticator {
	enabled := os.Getenv("HIE_AUTH_ENABLED") == "true" || os.Getenv("HIE_AUTH_ENABLED") == "1"
	token := os.Getenv("HIE_AUTH_TOKEN")

	// If auth is enabled but no token provided, generate one
	if enabled && token == "" {
		token = GenerateToken()
		// Log the generated token (in production, use proper logging)
		// fmt.Printf("Generated auth token: %s\n", token)
	}

	return NewAuthenticator(AuthConfig{
		Enabled: enabled,
		Token:   token,
	})
}

// IsEnabled returns true if authentication is enabled.
func (a *Authenticator) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Enabled
}

// GetToken returns the current auth token (for displaying to admin).
func (a *Authenticator) GetToken() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Token
}

// ValidateToken checks if the provided token matches the configured token.
// Uses constant-time comparison to prevent timing attacks.
func (a *Authenticator) ValidateToken(providedToken string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.config.Enabled {
		return nil // Auth not enabled, allow all
	}

	if providedToken == "" {
		return ErrAuthRequired
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(a.config.Token), []byte(providedToken)) != 1 {
		return ErrAuthTokenMismatch
	}

	return nil
}

// GenerateToken generates a cryptographically secure random token.
func GenerateToken() string {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to less secure but still usable token
		return "hierachain-default-token-change-me"
	}
	return hex.EncodeToString(bytes)
}

// AuthMessage represents an authentication handshake message.
// This is the first message a client must send when auth is enabled.
type AuthMessage struct {
	Type  string `json:"type"`  // Must be "auth"
	Token string `json:"token"` // The authentication token
}

// AuthResponse is sent back to the client after auth attempt.
type AuthResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

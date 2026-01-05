package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/api"
)

// Version information
const (
	Version = "0.0.1.dev2"
	Name    = "HieraChain-Engine"
)

func main() {
	fmt.Printf("%s v%s\n", Name, Version)
	fmt.Println("High-performance Go engine for HieraChain blockchain")

	// Default to localhost only for security - prevents external access
	// Set HIE_GO_ENGINE_ADDRESS environment variable to override (e.g., "0.0.0.0:50051" for external access)
	address := "127.0.0.1:50051"
	if envAddr := os.Getenv("HIE_GO_ENGINE_ADDRESS"); envAddr != "" {
		address = envAddr
	}

	server := api.NewArrowServer()

	// Display auth status
	if server.IsAuthEnabled() {
		log.Printf("Authentication: ENABLED")
		log.Printf("Auth Token: %s", server.GetAuthToken())
		log.Printf("Clients must send auth message first: {\"type\":\"auth\",\"token\":\"<token>\"}")
	} else {
		log.Printf("Authentication: DISABLED (set HIE_AUTH_ENABLED=true to enable)")
	}

	log.Printf("Starting Arrow Server on %s...", address)

	// Start async
	if err := server.StartAsync(address); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	server.Stop()
	log.Println("Server stopped.")
}

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
	Version = "0.0.1.dev1"
	Name    = "HieraChain-Engine"
)

func main() {
	fmt.Printf("%s v%s\n", Name, Version)
	fmt.Println("High-performance Go engine for HieraChain blockchain")

	address := ":50051"
	if envAddr := os.Getenv("HIE_GO_ENGINE_ADDRESS"); envAddr != "" {
		address = envAddr
	}

	server := api.NewArrowServer()

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

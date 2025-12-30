package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/api"
)

func main() {
	// Simple entry point to run the Arrow Server
	address := ":50051"
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

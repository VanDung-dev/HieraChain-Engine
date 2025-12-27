package main

import (
	"fmt"
	"os"
)

// Version information
const (
	Version = "0.1.0"
	Name    = "HieraChain-Engine"
)

func main() {
	fmt.Printf("%s v%s\n", Name, Version)
	fmt.Println("High-performance Go engine for HieraChain blockchain")
	fmt.Println("Status: Development")
	os.Exit(0)
}

package main

import (
	"fmt"
	"os"
)

// Version information
const (
	Version = "0.0.0.dev1"
	Name    = "HieraChain-Engine"
)

func main() {
	fmt.Printf("%s v%s\n", Name, Version)
	fmt.Println("High-performance Go engine for HieraChain blockchain")
	fmt.Println("Status: Development")
	os.Exit(0)
}

# Developer Guide

This guide contains all the information developers need to work with the HieraChain-Engine framework.

---

## Prerequisites

- **Go** 1.24+ (recommended: latest stable version)
- **Protocol Buffers** compiler (`protoc`)
- **Make** (optional, for using Makefile commands)
- **Rust** toolchain (optional, for HieraChain-Consensus integration)

---

## Installation

### Clone the Repository

```bash
git clone https://github.com/VanDung-dev/HieraChain-Engine.git
cd HieraChain-Engine
```

### Install Dependencies

```bash
go mod download
go mod verify
```

### Build the Project

```bash
# Standard build
go build ./...

# Build with Rust FFI integration (requires HieraChain-Consensus)
CGO_ENABLED=1 go build ./...

# Build the main command
go build -o hierachain-engine ./cmd/hierachain-engine
```

### Install the Binary

```bash
go install ./cmd/hierachain-engine
```

---

## Running the Server

### Start gRPC Server

```bash
# Run with default configuration
go run ./cmd/hierachain-engine

# Or use the built binary
./hierachain-engine
```

### Configuration via Environment Variables

| Variable | Default | Description |
|:---------|:--------|:------------|
| `HIE_USE_GO_ENGINE` | `false` | Enable Go Engine integration |
| `HIE_GO_ENGINE_ADDRESS` | `localhost:50051` | gRPC server address |
| `HIE_WORKER_POOL_SIZE` | `runtime.NumCPU()` | Number of worker threads |
| `HIE_MEMPOOL_SIZE` | `100000` | Maximum pending transactions |
| `HIE_METRICS_ENABLED` | `true` | Enable Prometheus metrics |

---

## Using the Package

After installation, you can import components from the package:

```go
package main

import (
    "github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/api"
    "github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/core"
)

func main() {
    // Create server with default config
    config := api.DefaultServerConfig()
    config.Address = ":50051"
    config.MetricsAddress = ":2112"
    
    // Start gRPC server
    server, _ := api.NewServer(config)
    server.Start(config.Address)
}
```

---

## Running Tests

### Run All Tests

```bash
go test ./... -v
```

### Run Tests by Package

```bash
# Core package tests
go test ./hierachain-engine/core/... -v

# API package tests
go test ./hierachain-engine/api/... -v

# Network package tests
go test ./hierachain-engine/network/... -v

# Data package tests
go test ./hierachain-engine/data/... -v
```

### Run Tests with Coverage

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Benchmarks

```bash
# Run all benchmarks
go test ./... -bench=. -benchmem

# Run specific package benchmarks
go test ./hierachain-engine/core/... -bench=. -benchmem
```

### Run Race Condition Detection

```bash
go test ./... -race
```

### End-to-End (E2E) Tests

The project includes a comprehensive suite of E2E tests located in `tests/e2e/`. These tests verify the integration between different layers of the HieraChain ecosystem.

| Layer | Test File | Description |
|-------|-----------|-------------|
| Layer 1 | `tests/e2e/test_layer1_fastapi.py` | Tests the FastAPI endpoint and JSON validation. |
| Layer 2 | `tests/e2e/test_layer2_python_to_go_direct.py` | Tests direct communication from Python to the Go Engine via Arrow IPC. |
| Layer 3 | `tests/e2e/test_layer3_go_engine.py` | Tests internal Go Engine processing, including Mempool and Arrow parsing. |
| Layer 4 | `tests/e2e/test_layer4_go_rust_ffi.py` | Tests the FFI bridge between Go and Rust Core. |
| Layer 5 | `tests/e2e/test_layer5_rust_python.py` | Tests direct Rust integration with Python via PyO3. |
| Full Chain | `tests/e2e/test_full_chain.py` | Tests the complete data flow from JSON input to JSON response. |

To run these tests, ensure you have the necessary Python dependencies installed and the HieraChain-Engine server running if required by the specific test.

```bash
# Run all E2E tests
pytest tests/e2e/
```

---

## Developer Scripts

### Using Makefile

```bash
# Build the project
make build

# Run tests
make test

# Run linting
make lint

# Clean build artifacts
make clean

# Generate protobuf files
make proto
```

### Static Analysis

```bash
# Run go vet
go vet ./...

# Run staticcheck (if installed)
staticcheck ./...

# Run golangci-lint (if installed)
golangci-lint run
```

---

## Generating Protocol Buffers

If you modify `.proto` files, regenerate Go code:

```bash
protoc --go_out=. --go-grpc_out=. proto/*.proto
```

---

## Code Style Guidelines

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` or `goimports` to format code
- Write meaningful commit messages following [Conventional Commits](https://www.conventionalcommits.org/)
- Add tests for new functionality
- Document exported functions and types

---

## Debugging

### Enable Debug Logging

```go
import "log"

log.SetFlags(log.LstdFlags | log.Lshortfile)
```

### Using Delve Debugger

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/hierachain-engine

# Attach to running process
dlv attach <pid>
```

---

## Integration with HieraChain Ecosystem

### Python Framework Integration

HieraChain-Engine communicates with the Python HieraChain framework via gRPC:

```
Python (FastAPI)  ──gRPC──►  Go Engine  ──FFI──►  Rust Core
     │                           │                    │
  REST API                  Worker Pool          Consensus
  Business Logic            Mempool              Merkle Tree
  Domain Contracts          Arrow IPC            Crypto
```

### Rust FFI Integration

For cryptographic operations, build HieraChain-Consensus first:

```bash
# In HieraChain-Consensus directory
cargo build --release

# Then build HieraChain-Engine with CGO
CGO_ENABLED=1 go build ./...
```

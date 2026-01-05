# HieraChain-Engine

![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE-APACHE)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE-MIT)
![Version](https://img.shields.io/badge/version-0.0.1.dev2-orange)

**English** | [Tiếng Việt](README_vi.md)

## Overview

HieraChain-Engine is a high-performance, Go-based blockchain engine designed for enterprise and consortium blockchain applications. It provides high-concurrency transaction processing, efficient memory management, and seamless integration with the HieraChain ecosystem through gRPC.

**This is the official High-Performance Engine of the HieraChain ecosystem.** While HieraChain (Python) handles business logic and REST API, this Go-based engine is the recommended choice for production deployments requiring high throughput, low latency, and efficient resource utilization.

## Features

### Core Functionality

- **High-Concurrency Processing**:
  - **Worker Pool**: Configurable worker threads for parallel transaction processing
  - **Mempool Management**: Efficient pending transaction queue with priority sorting
  - **Batch Processing**: Optimized batch operations for high throughput

- **gRPC Service Layer**:
  - High-performance gRPC server with streaming support
  - Protocol Buffer serialization for efficient data transfer
  - Health checks and graceful shutdown

- **Data Management**:
  - Apache Arrow IPC integration for zero-copy data transfer
  - Efficient serialization/deserialization pipelines
  - Memory-optimized block storage

- **Network Layer**:
  - P2P peer discovery and management
  - Gossip protocol for transaction propagation
  - Connection pooling and multiplexing

- **Rust Integration**:
  - FFI bindings to HieraChain-Consensus for cryptographic operations
  - Zero-copy data exchange via Arrow
  - Merkle tree verification and signature validation

### Technical Highlights

- **Go Implementation**: High-performance, concurrent runtime with goroutines
- **gRPC Communication**: Low-latency communication with Python framework
- **Arrow Integration**: Zero-copy interoperability for efficient data handling
- **Prometheus Metrics**: Built-in observability and monitoring
- **Modular Architecture**: Clean separation across API, core, data, and network layers

## Quick Start

### Installation

```bash
# Prerequisites: Go 1.24+, Protocol Buffers
go version
protoc --version

# Clone the repository
git clone https://github.com/VanDung-dev/HieraChain-Engine.git
cd HieraChain-Engine

# Build
go build ./hierachain-engine/...

# Build with Rust integration (optional)
# First: cargo build --release (in HieraChain-Consensus)
CGO_ENABLED=1 go build ./...
```

### Basic Usage

```go
package main

import (
    "github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/api"
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

## Architecture Overview

HieraChain-Engine is built with a modular architecture that separates concerns across multiple layers:

- **API Layer**: gRPC server, request handlers, and Prometheus metrics
- **Core Layer**: Transaction processing, mempool, worker pool, and executor
- **Data Layer**: Arrow IPC adapter, serialization, and batch processing
- **Network Layer**: Peer management, protocol handlers, and discovery service

### System Integration

```
┌─────────────────────────────────────────────────────────────────────┐
│                        HieraChain Ecosystem                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Python (FastAPI)  ──gRPC──►  Go Engine  ──FFI──►  Rust Core       │
│        │                           │                    │           │
│     REST API                  Worker Pool          Consensus        │
│     Business Logic            Mempool              Merkle Tree      │
│     Domain Contracts          Arrow IPC            Crypto           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Processing Flow

1. **Request Reception** → gRPC server receives transaction requests from Python framework
2. **Validation** → Transactions are validated against schema and business rules
3. **Mempool** → Valid transactions are queued in the mempool
4. **Processing** → Worker pool processes transactions in parallel
5. **Consensus** → Rust FFI provides cryptographic verification (optional)
6. **Response** → Results are returned via gRPC streaming

## Configuration

### Environment Variables

| Variable | Default | Description |
|:---------|:--------|:------------|
| `HIE_USE_GO_ENGINE` | `false` | Enable Go Engine integration |
| `HIE_GO_ENGINE_ADDRESS` | `localhost:50051` | gRPC server address |
| `HIE_WORKER_POOL_SIZE` | `runtime.NumCPU()` | Number of worker threads |
| `HIE_MEMPOOL_SIZE` | `100000` | Maximum pending transactions |
| `HIE_METRICS_ENABLED` | `true` | Enable Prometheus metrics |

## Related Projects

**HieraChain-Engine** is the official High-Performance Engine of the HieraChain ecosystem:

| Project | Language | Description |
|---------|----------|-------------|
| [HieraChain](https://github.com/VanDung-dev/HieraChain) | Python | Main hierarchical blockchain framework (REST API, business logic, domain contracts) |
| **HieraChain-Engine** (this repo) | Go | **Official High-Performance Engine** - gRPC server, worker pool, mempool |
| [HieraChain-Consensus](https://github.com/VanDung-dev/HieraChain-Consensus) | Rust | Official Core Consensus - cryptography, Merkle tree, BFT/PoF/PoA |

> 💡 **Why Go?** While HieraChain's Python implementation handles business logic well, this Go implementation offers better performance for I/O-bound operations, efficient goroutine-based concurrency, and seamless gRPC integration for high-throughput scenarios.

## License

This project is dual licensed under either the [Apache-2.0 License](LICENSE-APACHE) or the [MIT License](LICENSE-MIT). You may choose either license.

---

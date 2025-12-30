# HieraChain-Engine

HieraChain engine is a high-concurrency infrastructure layer built on Go

## Architecture

```
Python (FastAPI)  ──gRPC──►  Go Engine  ──FFI──►  Rust Core
    │                            │                    │
 REST API                   Worker Pool          Consensus
 Business Logic             Mempool              Merkle Tree
                            Arrow IPC            Crypto
```

## Performance

| Metric | Result |
|:-------|:-------|
| Throughput | **1.4M tx/sec** |
| Target | 10k tx/sec |

---

## Go Development

### Prerequisites

```bash
# Go 1.24+
go version

# Protocol Buffers
protoc --version

# (Optional) Rust for integration module
rustc --version
cargo --version
```

### Build

```bash
# Build all packages (excluding CGO/Rust integration)
go build ./hierachain-engine/...

# Build with CGO (requires Rust library)
# First: cargo build --release
CGO_ENABLED=1 go build ./...

# Build main application
go build -o hierachain ./cmd/hierachain/
```

### Run Tests

```bash
# Run all tests
go test ./hierachain-engine/... -v

# Run short tests only
go test ./hierachain-engine/... -short

# Run benchmarks
go test ./hierachain-engine/api/... -bench=. -benchtime=3s

# Run specific package tests
go test ./hierachain-engine/core/... -v
go test ./hierachain-engine/data/... -v
go test ./hierachain-engine/network/... -v
```

### Run gRPC Server

```go
package main

import (
    "github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/api"
)

func main() {
    config := api.DefaultServerConfig()
    config.Address = ":50051"
    config.MetricsAddress = ":2112"
    
    server, _ := api.NewServer(config)
    server.Start(config.Address)
}
```

### Generate Protobuf

```bash
protoc --go_out=. --go-grpc_out=. hierachain-engine/api/proto/hierachain.proto
```

### Environment Variables

| Variable | Default | Description |
|:---------|:--------|:------------|
| `HIE_USE_GO_ENGINE` | `false` | Enable Go Engine |
| `HIE_GO_ENGINE_ADDRESS` | `localhost:50051` | gRPC address |

---

## Related Repositories

- [HieraChain](https://github.com/VanDung-dev/HieraChain) - Python framework
- [HieraChain-Consensus](https://github.com/VanDung-dev/HieraChain-Consensus) - Rust consensus library

---

## License

This project is dual licensed under either the [Apache-2.0 License](LICENSE-APACHE) or the [MIT License](LICENSE-MIT). You may choose either license.

---

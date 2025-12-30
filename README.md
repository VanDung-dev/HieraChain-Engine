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
```

### Build

```bash
# Build all
go build ./...

# Run tests
go test ./... -v

# Run benchmarks
go test ./api/... -bench=. -benchtime=3s
```

### Project Structure

```
├── api/              # gRPC server, metrics
│   ├── grpc_server.go
│   ├── metrics.go
│   └── proto/        # Protobuf definitions
├── arrow/            # Arrow schema, IPC
├── bridge/           # Rust FFI bindings
├── engine/           # Worker pool, mempool
└── network/          # ZeroMQ, P2P
```

### Run gRPC Server

```go
package main

import (
    "github.com/VanDung-dev/HieraChain-Engine/api"
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
protoc --go_out=. --go-grpc_out=. api/proto/hierachain.proto
```

### Environment Variables

| Variable | Default | Description |
|:---------|:--------|:------------|
| `HIE_USE_GO_ENGINE` | `false` | Enable Go Engine |
| `HIE_GO_ENGINE_ADDRESS` | `localhost:50051` | gRPC address |

---

## License

This project is dual licensed under either the [Apache-2.0 License](LICENSE-APACHE) or the [MIT License](LICENSE-MIT). You may choose either license.

---

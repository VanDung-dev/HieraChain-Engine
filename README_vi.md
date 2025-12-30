# HieraChain-Engine

![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE-APACHE)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE-MIT)
![Version](https://img.shields.io/badge/version-0.0.1.dev1-orange)

[English](README.md) | **Tiếng Việt**

## Tổng Quan

HieraChain-Engine là một blockchain engine hiệu năng cao, được xây dựng trên nền tảng Go, thiết kế dành cho các ứng dụng blockchain doanh nghiệp và consortium. Engine cung cấp khả năng xử lý giao dịch đồng thời cao, quản lý bộ nhớ hiệu quả, và tích hợp liền mạch với hệ sinh thái HieraChain thông qua gRPC.

**Đây là Engine Hiệu Năng Cao chính thức của hệ sinh thái HieraChain.** Trong khi HieraChain (Python) xử lý logic nghiệp vụ và REST API, engine Go này là lựa chọn được khuyến nghị cho các triển khai production yêu cầu thông lượng cao, độ trễ thấp, và sử dụng tài nguyên hiệu quả.

## Tính Năng

### Chức Năng Cốt Lõi

- **Xử Lý Đồng Thời Cao**:
  - **Worker Pool**: Các worker thread có thể cấu hình cho xử lý giao dịch song song
  - **Quản Lý Mempool**: Hàng đợi giao dịch chờ xử lý hiệu quả với sắp xếp ưu tiên
  - **Xử Lý Batch**: Các thao tác batch được tối ưu hóa cho thông lượng cao

- **Tầng Dịch Vụ gRPC**:
  - Server gRPC hiệu năng cao với hỗ trợ streaming
  - Serialization Protocol Buffer cho truyền dữ liệu hiệu quả
  - Health checks và graceful shutdown

- **Quản Lý Dữ Liệu**:
  - Tích hợp Apache Arrow IPC cho truyền dữ liệu zero-copy
  - Pipelines serialization/deserialization hiệu quả
  - Lưu trữ block tối ưu bộ nhớ

- **Tầng Mạng**:
  - Khám phá và quản lý peer P2P
  - Giao thức Gossip cho lan truyền giao dịch
  - Connection pooling và multiplexing

- **Tích Hợp Rust**:
  - FFI bindings tới HieraChain-Consensus cho các thao tác mật mã
  - Trao đổi dữ liệu zero-copy qua Arrow
  - Xác minh Merkle tree và xác thực chữ ký

### Điểm Nổi Bật Kỹ Thuật

- **Triển Khai Go**: Runtime hiệu năng cao, đồng thời với goroutines
- **Giao Tiếp gRPC**: Giao tiếp độ trễ thấp với Python framework
- **Tích Hợp Arrow**: Tương tác zero-copy cho xử lý dữ liệu hiệu quả
- **Prometheus Metrics**: Observability và monitoring tích hợp sẵn
- **Kiến Trúc Module**: Phân tách rõ ràng giữa các tầng API, core, data, và network

## Bắt Đầu Nhanh

### Cài Đặt

```bash
# Yêu cầu: Go 1.24+, Protocol Buffers
go version
protoc --version

# Clone repository
git clone https://github.com/VanDung-dev/HieraChain-Engine.git
cd HieraChain-Engine

# Build
go build ./hierachain-engine/...

# Build với tích hợp Rust (tùy chọn)
# Trước tiên: cargo build --release (trong HieraChain-Consensus)
CGO_ENABLED=1 go build ./...
```

### Sử Dụng Cơ Bản

```go
package main

import (
    "github.com/VanDung-dev/HieraChain-Engine/hierachain-engine/api"
)

func main() {
    // Tạo server với config mặc định
    config := api.DefaultServerConfig()
    config.Address = ":50051"
    config.MetricsAddress = ":2112"
    
    // Khởi động gRPC server
    server, _ := api.NewServer(config)
    server.Start(config.Address)
}
```

## Tổng Quan Kiến Trúc

HieraChain-Engine được xây dựng với kiến trúc module phân tách các mối quan tâm thành nhiều tầng:

- **Tầng API**: gRPC server, request handlers, và Prometheus metrics
- **Tầng Core**: Xử lý giao dịch, mempool, worker pool, và executor
- **Tầng Data**: Arrow IPC adapter, serialization, và xử lý batch
- **Tầng Network**: Quản lý peer, protocol handlers, và dịch vụ discovery

### Tích Hợp Hệ Thống

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Hệ Sinh Thái HieraChain                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Python (FastAPI)  ──gRPC──►  Go Engine  ──FFI──►  Rust Core       │
│        │                           │                    │           │
│     REST API                  Worker Pool          Consensus        │
│     Logic Nghiệp vụ           Mempool              Merkle Tree      │
│     Domain Contracts          Arrow IPC            Crypto           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Luồng Xử Lý

1. **Tiếp Nhận Request** → gRPC server nhận các yêu cầu giao dịch từ Python framework
2. **Xác Thực** → Các giao dịch được xác thực theo schema và quy tắc nghiệp vụ
3. **Mempool** → Các giao dịch hợp lệ được xếp hàng trong mempool
4. **Xử Lý** → Worker pool xử lý các giao dịch song song
5. **Consensus** → Rust FFI cung cấp xác minh mật mã (tùy chọn)
6. **Phản Hồi** → Kết quả được trả về qua gRPC streaming

## Cấu Hình

### Biến Môi Trường

| Biến | Mặc định | Mô tả |
|:-----|:---------|:------|
| `HIE_USE_GO_ENGINE` | `false` | Kích hoạt tích hợp Go Engine |
| `HIE_GO_ENGINE_ADDRESS` | `localhost:50051` | Địa chỉ gRPC server |
| `HIE_WORKER_POOL_SIZE` | `runtime.NumCPU()` | Số lượng worker threads |
| `HIE_MEMPOOL_SIZE` | `100000` | Số giao dịch chờ xử lý tối đa |
| `HIE_METRICS_ENABLED` | `true` | Kích hoạt Prometheus metrics |

## Các Dự Án Liên Quan

**HieraChain-Engine** là Engine Hiệu Năng Cao chính thức của hệ sinh thái HieraChain:

| Dự án | Ngôn ngữ | Mô tả |
|-------|----------|-------|
| [HieraChain](https://github.com/VanDung-dev/HieraChain) | Python | Framework blockchain phân cấp chính (REST API, logic nghiệp vụ, domain contracts) |
| **HieraChain-Engine** (repo này) | Go | **Engine Hiệu Năng Cao Chính Thức** - gRPC server, worker pool, mempool |
| [HieraChain-Consensus](https://github.com/VanDung-dev/HieraChain-Consensus) | Rust | Core Consensus Chính Thức - mật mã, Merkle tree, BFT/PoF/PoA |

> 💡 **Tại sao chọn Go?** Trong khi triển khai Python của HieraChain xử lý logic nghiệp vụ tốt, triển khai Go này cung cấp hiệu năng tốt hơn cho các thao tác I/O-bound, xử lý đồng thời hiệu quả dựa trên goroutine, và tích hợp gRPC liền mạch cho các kịch bản thông lượng cao.

## Giấy Phép

Dự án này được cấp phép kép theo [Giấy Phép Apache-2.0](LICENSE-APACHE) hoặc [Giấy Phép MIT](LICENSE-MIT). Bạn có thể chọn một trong hai giấy phép.

---

# `HieraChain Ecosystem` Data Flow Documentation

## Overview

HieraChain uses a multi-layer architecture for processing blockchain transactions. Data flows through 6 distinct layers, from JSON input to JSON response.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    HieraChain Ecosystem Data Flow                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   [Client]                                                              │
│      │                                                                  │
│      ▼                                                                  │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Layer 1: JSON → FastAPI                                         │   │
│   │ ─────────────────────────────────────────────────────────────── │   │
│   │ • HTTP POST /api/v1/chains/{chain_id}/events                    │   │
│   │ • Pydantic validation (EventRequest schema)                     │   │
│   │ • Latency: ~6ms                                                 │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│      │                                                                  │
│      ▼                                                                  │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Layer 2: FastAPI → Go Engine                                    │   │
│   │ ─────────────────────────────────────────────────────────────── │   │
│   │ • Arrow IPC over TCP (127.0.0.1:50051)                          │   │
│   │ • Length-prefixed protocol (4 bytes BE + payload)               │   │
│   │ • Throughput: ~23 ops/s                                         │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│      │                                                                  │
│      ▼                                                                  │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Layer 3: Go Engine Processing                                   │   │
│   │ ─────────────────────────────────────────────────────────────── │   │
│   │ • Arrow IPC parsing (arrow-go/v18)                              │   │
│   │ • Mempool: Thread-safe priority queue                           │   │ 
│   │ • Converter: JSON ↔ Arrow round-trip                            │   │
│   └─────────────────────────────────────────────────────────────────┘   │ 
│      │                                                                  │ 
│      ▼                                                                  │ 
│   ┌─────────────────────────────────────────────────────────────────┐   │ 
│   │ Layer 4: Go → Rust Core (FFI)                                   │   │ 
│   │ ─────────────────────────────────────────────────────────────── │   │ 
│   │ • CGO bindings (integration/rust_ffi.go)                        │   │
│   │ • Functions: MerkleRoot, BlockHash, ValidateTx                  │   │
│   │ • Max input: 100MB                                              │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│      │                                                                  │
│      ▼                                                                  │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Layer 5: Rust Core Processing                                   │   │ 
│   │ ─────────────────────────────────────────────────────────────── │   │ 
│   │ • hierachain-consensus library                                  │   │ 
│   │ • MerkleTree, hash generation (SHA256)                          │   │
│   │ • BFT consensus algorithm                                       │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│      │                                                                  │
│      ▼                                                                  │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Layer 6: Rust → Python (PyO3)                                   │   │
│   │ ─────────────────────────────────────────────────────────────── │   │
│   │ • hierachain_consensus module                                   │   │ 
│   │ • Type conversion: dict → serde_json → str                      │   │
│   │ • JSON serialization for response                               │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│      │                                                                  │
│      ▼                                                                  │
│   [JSON Response]                                                       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Layer Details

### Layer 1: JSON → FastAPI

**Entry Point:** `hierachain/api/server.py`

**Endpoint:** `POST /api/v1/chains/{chain_id}/events`

**Request Schema (Pydantic):**
```python
class EventRequest(BaseModel):
    entity_id: str          # Required
    event_type: str         # Required
    details: dict | None    # Optional
```

**Response Schema:**
```python
class EventResponse(BaseModel):
    success: bool
    message: str
    event_id: str
```

**Performance:**
- Latency: ~6ms per request
- Throughput: ~150-200 req/s

---

### Layer 2: FastAPI → Go Engine

**Connection:** TCP `127.0.0.1:50051`

**Protocol:** Length-prefixed binary
```
[4 bytes: length (Big Endian)] + [N bytes: Arrow IPC payload]
```

**Arrow Schema:**
```
tx_id: string
entity_id: string
event_type: string
timestamp: int64
```

**Response:** `OK` or error message

---

### Layer 3: Go Engine Processing

**Components:**

| Component | File | Description |
|-----------|------|-------------|
| ArrowServer | `api/arrow_server.go` | TCP server for Arrow IPC |
| ArrowHandler | `api/arrow_handler.go` | IPC stream processing |
| Converter | `data/converter.go` | JSON ↔ Arrow conversion |
| Mempool | `core/mempool.go` | Transaction pool |

**Mempool Features:**
- Thread-safe (sync.RWMutex)
- Priority queue (heap-based)
- Max size configurable

---

### Layer 4: Go → Rust FFI

**FFI Functions:**

| Function | Input | Output |
|----------|-------|--------|
| `ffi_calculate_merkle_root` | JSON events | 64-char hex hash |
| `ffi_calculate_block_hash` | JSON block | 64-char hex hash |
| `ffi_bulk_validate_transactions` | JSON txs | 1 (valid) / 0 (invalid) |
| `ffi_process_arrow_batch` | Arrow IPC bytes | Processed bytes |

**Error Codes:**
- 0: Success
- -1: Null pointer
- -2: Invalid UTF-8
- -3: JSON parse error
- -4: Buffer too small
- -5: Internal error

---

### Layer 5: Rust Core

**Library:** `hierachain-consensus`

**Key Modules:**
- `core/utils.rs` - MerkleTree, hash generation
- `ffi.rs` - FFI exports
- `hierarchical/consensus/` - BFT consensus

**Hash Algorithm:** SHA256 (64-char hex output)

---

### Layer 6: Rust → Python (PyO3)

**Module:** `hierachain_consensus`

**Exported Functions:**
- `calculate_merkle_root(events: list[dict]) -> str`
- `calculate_block_hash(block: dict) -> str`
- `bulk_validate_transactions(txs: list[dict]) -> bool`
- `batch_calculate_hashes(items: list[dict]) -> list[str]`

**Exported Classes:**
- `Block`, `Blockchain`, `MainChain`, `SubChain`
- `BFTConsensus`, `OrderingService`
- `KeyPair`

---

## Performance Summary

| Metric | Value |
|--------|-------|
| Single request latency | ~6ms |
| Batch throughput (HTTP) | 150-200 tx/s |
| Arrow IPC round-trip | ~0.3ms |
| Direct Rust (merkle) | ~4.7ms / 100 events |
| HTTP overhead factor | 11.6x |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HIE_AUTH_ENABLED` | `false` | Enable token auth |
| `HIE_AUTH_TOKEN` | auto-gen | Auth token |

### Ports

| Service | Port | Protocol |
|---------|------|----------|
| FastAPI | 8001 | HTTP |
| Arrow Server | 50051 | TCP |
| Metrics | 9090 | HTTP (Prometheus) |

## Observability

### Tracing

- Trace ID: 32-char hex UUID
- Span ID: 16-char hex UUID
- Header: `X-Trace-ID`

### Metrics (Prometheus)

```
hierachain_transactions_total
hierachain_transactions_processed_total
hierachain_transaction_latency_seconds
hierachain_batches_total
hierachain_mempool_size
hierachain_worker_pool_active
```

## Test Coverage

| Layer | Test File | Tests |
|-------|-----------|-------|
| Layer 1 | `test_layer1_fastapi.py` | 1 |
| Layer 2 | `test_layer2_python_to_go_direct.py` | 1 |
| Layer 3 | `test_layer3_go_engine.py` | 4 |
| Layer 4 | `test_layer4_go_rust_ffi.py` | 7 |
| Layer 5 | `test_layer5_rust_python.py` | 1 |
| E2E | `test_full_chain.py` | 2 |

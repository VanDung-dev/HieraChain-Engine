// Package bridge provides integration with the Rust Consensus library.
//
// This package contains:
//   - CGO bindings to Rust FFI functions (rust_ffi.go)
//   - Arrow IPC serialization/deserialization helpers (arrow_bridge.go)
//
// The Rust library must be built before using this package:
//
//	cd HieraChain-Engine
//	cargo build --release
//
// The static library will be created at:
//   - Windows: target/release/hierachain_consensus.lib
//   - Linux/macOS: target/release/libhierachain_consensus.a
package bridge

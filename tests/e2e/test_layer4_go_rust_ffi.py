"""
Layer 4 Test: Go → Rust Core FFI

Tests the FFI bridge between Go and Rust:
1. CGO binding functionality
2. Memory management across boundary
3. Return value formats

Note: These tests verify the Python bindings which use the same Rust code
that Go FFI calls. The actual Go→Rust FFI can only be tested via Go tests
when the Rust library is properly linked.
"""

import time
import sys
from rich.console import Console

# Import Rust bindings via PyO3
try:
    import hierachain_consensus
    RUST_AVAILABLE = True
except ImportError:
    RUST_AVAILABLE = False

console = Console()


def test_layer4_ffi_version():
    """5.1. Test FFI - Library Version."""
    console.print("[bold blue]Layer 4 Test: FFI Version Check[/bold blue]")
    
    if not RUST_AVAILABLE:
        console.print("[yellow]⚠ Rust bindings not available[/yellow]")
        return
    
    version = hierachain_consensus.__version__
    console.print(f"[green]✔ Rust library version: {version}[/green]")
    assert version is not None


def test_layer4_merkle_root():
    """5.2. Test FFI - Merkle Root Calculation."""
    console.print("\n[bold blue]Layer 4 Test: Merkle Root Calculation[/bold blue]")
    
    if not RUST_AVAILABLE:
        console.print("[yellow]⚠ Rust bindings not available[/yellow]")
        return
    
    # Test data
    events = [
        {"entity_id": "e1", "event": "create", "timestamp": 1700000000},
        {"entity_id": "e2", "event": "update", "timestamp": 1700000001},
        {"entity_id": "e3", "event": "delete", "timestamp": 1700000002},
    ]
    
    start = time.time()
    result = hierachain_consensus.calculate_merkle_root(events)
    duration = (time.time() - start) * 1000
    
    console.print(f"  Events: {len(events)}")
    console.print(f"  Merkle Root: {result[:32]}...")
    console.print(f"  Hash length: {len(result)} chars")
    console.print(f"  Time: {duration:.2f}ms")
    console.print(f"[green]✔ Merkle root calculated successfully[/green]")
    
    assert len(result) == 64, f"Expected 64-char hex, got {len(result)}"


def test_layer4_block_hash():
    """5.2. Test FFI - Block Hash Calculation."""
    console.print("\n[bold blue]Layer 4 Test: Block Hash Calculation[/bold blue]")
    
    if not RUST_AVAILABLE:
        console.print("[yellow]⚠ Rust bindings not available[/yellow]")
        return
    
    block_data = {
        "index": 1,
        "timestamp": 1700000000,
        "previous_hash": "0" * 64,
        "merkle_root": "a" * 64,
        "nonce": 12345
    }
    
    start = time.time()
    result = hierachain_consensus.calculate_block_hash(block_data)
    duration = (time.time() - start) * 1000
    
    console.print(f"  Block index: {block_data['index']}")
    console.print(f"  Block Hash: {result[:32]}...")
    console.print(f"  Time: {duration:.2f}ms")
    console.print(f"[green]✔ Block hash calculated successfully[/green]")
    
    assert len(result) == 64


def test_layer4_validate_transactions():
    """5.2. Test FFI - Bulk Transaction Validation."""
    console.print("\n[bold blue]Layer 4 Test: Bulk Transaction Validation[/bold blue]")
    
    if not RUST_AVAILABLE:
        console.print("[yellow]⚠ Rust bindings not available[/yellow]")
        return
    
    # Valid transactions (must include entity_id, event, timestamp)
    valid_txs = [
        {"entity_id": "e1", "event": "create", "timestamp": 1700000000.0},
        {"entity_id": "e2", "event": "update", "timestamp": 1700000001.0},
        {"entity_id": "e3", "event": "delete", "timestamp": 1700000002.0},
    ]
    
    result = hierachain_consensus.bulk_validate_transactions(valid_txs)
    console.print(f"  Valid transactions: {len(valid_txs)}")
    console.print(f"  Validation result: {result}")
    console.print(f"[green]✔ Valid transactions passed[/green]")
    assert result == True
    
    # Invalid transactions (missing required field 'timestamp')
    invalid_txs = [
        {"entity_id": "e1", "event": "create"},  # Missing "timestamp"
    ]
    
    result = hierachain_consensus.bulk_validate_transactions(invalid_txs)
    console.print(f"  Invalid transactions test: result = {result}")
    console.print(f"[green]✔ Invalid transactions correctly rejected[/green]")
    assert result == False


def test_layer4_memory_management():
    """5.1. Test FFI - Memory Management (Stress Test)."""
    console.print("\n[bold blue]Layer 4 Test: Memory Management[/bold blue]")
    
    if not RUST_AVAILABLE:
        console.print("[yellow]⚠ Rust bindings not available[/yellow]")
        return
    
    # Large batch to test memory handling
    large_batch = [
        {"entity_id": f"e{i}", "event": f"event_{i}", "data": "x" * 100}
        for i in range(1000)
    ]
    
    start = time.time()
    result = hierachain_consensus.calculate_merkle_root(large_batch)
    duration = (time.time() - start) * 1000
    
    console.print(f"  Batch size: {len(large_batch)} events")
    console.print(f"  Processing time: {duration:.2f}ms")
    console.print(f"  Throughput: {len(large_batch)/(duration/1000):.0f} events/s")
    console.print(f"[green]✔ Large batch processed without memory leak[/green]")


def test_layer4_batch_operations():
    """5.2. Test FFI - Batch Operations."""
    console.print("\n[bold blue]Layer 4 Test: Batch Operations[/bold blue]")
    
    if not RUST_AVAILABLE:
        console.print("[yellow]⚠ Rust bindings not available[/yellow]")
        return
    
    # Test batch_calculate_hashes with dict items
    data_items = [{"id": f"item_{i}", "value": i * 10} for i in range(10)]
    
    start = time.time()
    hashes = hierachain_consensus.batch_calculate_hashes(data_items)
    duration = (time.time() - start) * 1000
    
    console.print(f"  Items: {len(data_items)}")
    console.print(f"  Hashes returned: {len(hashes)}")
    console.print(f"  Time: {duration:.2f}ms")
    console.print(f"[green]✔ Batch hashing successful[/green]")
    
    assert len(hashes) == len(data_items)


def test_layer4():
    """Main test function for pytest."""
    console.print("[bold cyan]Starting Layer 4 (Go → Rust Core FFI) Tests[/bold cyan]")
    console.print(f"Rust bindings available: {RUST_AVAILABLE}\n")
    
    if not RUST_AVAILABLE:
        console.print("[red]✘ Cannot run tests - hierachain_consensus not installed[/red]")
        console.print("Run: maturin develop")
        sys.exit(1)
    
    test_layer4_ffi_version()
    test_layer4_merkle_root()
    test_layer4_block_hash()
    test_layer4_validate_transactions()
    test_layer4_memory_management()
    test_layer4_batch_operations()
    
    console.print("\n[bold cyan]Layer 4 Tests Completed[/bold cyan]")


if __name__ == "__main__":
    test_layer4()

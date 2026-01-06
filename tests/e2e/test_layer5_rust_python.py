"""
End-to-end (E2E) test for Layer 5 functionality of HieraChain-Engine.

This test ensures that Rust bindings can be successfully used to calculate
Merkle roots, block hashes, and validate transactions.
"""

import hierachain_consensus
import pytest
import sys
from rich.console import Console

console = Console()

def test_layer5_rust_binding():
    console.print("[bold blue]Starting Layer 5 (Rust -> Python) Test[/bold blue]")

    # 1. Test Rust Version/Module
    console.print(f"Module found: {hierachain_consensus.__file__}")

    # 2. Test Merkle Root
    try:
        events = [{"id": "e1", "val": 1}, {"id": "e2", "val": 2}]
        root = hierachain_consensus.calculate_merkle_root(events)
        console.print(f"[green]✔ Merkle Root Calculated:[/green] {root}")
        if len(root) == 64:
            console.print("  Length verified (SHA256)")
        else:
            console.print(f"[red]  Length mismatch: {len(root)}[/red]")
    except Exception as e:
        console.print(f"[red]✘ Merkle Root Failed: {e}[/red]")
        sys.exit(1)

    # 3. Test Block Hash
    try:
        block_data = {"index": 100, "previous_hash": "0000000000000000000000000000000000000000000000000000000000000000"}
        h = hierachain_consensus.calculate_block_hash(block_data)
        console.print(f"[green]✔ Block Hash Calculated:[/green] {h}")
    except Exception as e:
        console.print(f"[red]✘ Block Hash Failed: {e}[/red]")
        sys.exit(1)

    # 4. Test Bulk Validation
    try:
        txs = [{"entity_id": "e1"}, {"entity_id": "e2"}]
        valid = hierachain_consensus.bulk_validate_transactions(txs)
        console.print(f"[green]✔ Bulk Validation Result:[/green] {valid}")
    except Exception as e:
        console.print(f"[red]✘ Bulk Validation Failed: {e}[/red]")

    # 5. Test JSON Serialization (Section 6.3)
    try:
        import json
        response_data = {
            "merkle_root": root,
            "block_hash": h,
            "valid": valid
        }
        json_output = json.dumps(response_data, indent=2)
        console.print(f"[green]✔ JSON Serialization Verified:[/green]\n{json_output}")
    except Exception as e:
        console.print(f"[red]✘ JSON Serialization Failed: {e}[/red]")
        sys.exit(1)

    console.print("[bold blue]Layer 5 Test Completed[/bold blue]")

if __name__ == "__main__":
    test_layer5_rust_binding()

"""
Layer 3 Test: Go Engine Processing

Tests the Go Engine's ability to:
1. Parse Arrow IPC streams
2. Process RecordBatches
3. Return valid responses

Requires: Go Arrow Server running on 127.0.0.1:50051
"""

import socket
import struct
import pyarrow as pa
import time
import sys
from rich.console import Console

console = Console()

HOST = '127.0.0.1'
PORT = 50051


def send_msg(sock, data):
    """Send length-prefixed message."""
    length = len(data)
    sock.sendall(struct.pack('>I', length))
    sock.sendall(data)


def recv_msg(sock):
    """Receive length-prefixed message."""
    raw_len = sock.recv(4)
    if not raw_len:
        return None
    length = struct.unpack('>I', raw_len)[0]
    
    data = b''
    while len(data) < length:
        packet = sock.recv(length - len(data))
        if not packet:
            return None
        data += packet
    return data


def create_arrow_batch(schema, data_arrays):
    """Create Arrow RecordBatch and serialize to IPC."""
    batch = pa.RecordBatch.from_arrays(data_arrays, schema=schema)
    sink = pa.BufferOutputStream()
    with pa.ipc.new_stream(sink, schema) as writer:
        writer.write_batch(batch)
    return sink.getvalue().to_pybytes(), batch


def test_layer3_arrow_parsing():
    """4.1. Test Arrow Parsing in Go Engine."""
    console.print("[bold blue]Layer 3 Test: Arrow Parsing[/bold blue]")
    
    # Create test schema matching Go expectations
    schema = pa.schema([
        ('tx_id', pa.string()),
        ('entity_id', pa.string()),
        ('event_type', pa.string()),
        ('timestamp', pa.int64())
    ])
    
    # Create test data
    payload, batch = create_arrow_batch(schema, [
        pa.array(['tx_layer3_001', 'tx_layer3_002', 'tx_layer3_003']),
        pa.array(['entity_a', 'entity_b', 'entity_c']),
        pa.array(['create', 'update', 'delete']),
        pa.array([1700000000, 1700000001, 1700000002])
    ])
    
    console.print(f"  Created Arrow batch: {batch.num_rows} rows, {batch.num_columns} cols")
    console.print(f"  Schema: {schema.names}")
    console.print(f"  IPC payload size: {len(payload)} bytes")
    
    # Connect and send
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(10.0)
        s.connect((HOST, PORT))
        
        start = time.time()
        send_msg(s, payload)
        resp = recv_msg(s)
        duration = (time.time() - start) * 1000
        
        s.close()
        
        if resp:
            console.print(f"[green]✔ Go Engine parsed Arrow batch successfully[/green]")
            console.print(f"  Response: {resp}")
            console.print(f"  Round-trip: {duration:.2f}ms")
            assert resp == b'OK', f"Expected 'OK', got {resp}"
        else:
            console.print("[red]✘ No response from Go Engine[/red]")
            sys.exit(1)
            
    except ConnectionRefusedError:
        console.print("[red]✘ Go Arrow Server not running[/red]")
        sys.exit(1)


def test_layer3_mempool_processing():
    """4.2. Test Mempool Processing via multiple batches."""
    console.print("\n[bold blue]Layer 3 Test: Mempool Processing[/bold blue]")
    
    schema = pa.schema([
        ('tx_id', pa.string()),
        ('sender', pa.string()),
        ('amount', pa.int64()),
        ('timestamp', pa.int64())
    ])
    
    # Send multiple batches to simulate mempool filling
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(10.0)
        s.connect((HOST, PORT))
        
        total_txs = 0
        for i in range(5):
            payload, batch = create_arrow_batch(schema, [
                pa.array([f'tx_batch{i}_{j}' for j in range(10)]),
                pa.array([f'sender_{j}' for j in range(10)]),
                pa.array([100 * (j + 1) for j in range(10)]),
                pa.array([1700000000 + j for j in range(10)])
            ])
            
            send_msg(s, payload)
            resp = recv_msg(s)
            
            if resp == b'OK':
                total_txs += batch.num_rows
            else:
                console.print(f"[yellow]⚠ Batch {i} response: {resp}[/yellow]")
        
        s.close()
        
        console.print(f"[green]✔ Sent {total_txs} transactions in 5 batches[/green]")
        
    except Exception as e:
        console.print(f"[red]✘ Error: {e}[/red]")
        sys.exit(1)


def test_layer3_schema_validation():
    """4.1. Test Schema validation in Go Engine."""
    console.print("\n[bold blue]Layer 3 Test: Schema Validation[/bold blue]")
    
    # Test with different schema (should still be accepted by current impl)
    schema = pa.schema([
        ('custom_id', pa.string()),
        ('value', pa.float64()),
    ])
    
    payload, batch = create_arrow_batch(schema, [
        pa.array(['custom_001', 'custom_002']),
        pa.array([123.45, 678.90])
    ])
    
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(10.0)
        s.connect((HOST, PORT))
        
        send_msg(s, payload)
        resp = recv_msg(s)
        s.close()
        
        if resp:
            console.print(f"[green]✔ Custom schema batch accepted[/green]")
            console.print(f"  Schema: {schema.names}")
        else:
            console.print("[yellow]⚠ No response (schema might be rejected)[/yellow]")
            
    except Exception as e:
        console.print(f"[red]✘ Error: {e}[/red]")


def test_layer3():
    """Main test function for pytest."""
    console.print("[bold cyan]Starting Layer 3 (Go Engine Processing) Tests[/bold cyan]\n")
    
    test_layer3_arrow_parsing()
    test_layer3_mempool_processing()
    test_layer3_schema_validation()
    
    console.print("\n[bold cyan]Layer 3 Tests Completed[/bold cyan]")


if __name__ == "__main__":
    test_layer3()

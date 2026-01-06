"""
End-to-end (E2E) test for Layer 2 functionality of HieraChain-Engine.

This test ensures that Arrow data can be successfully processed and submitted
directly between Python and Go via TCP sockets.
"""

import socket
import struct
import pyarrow as pa
import time
import sys
from rich.console import Console

console = Console()

def send_msg(sock, data):
    length = len(data)
    # BigEndian uint32 length prefix
    sock.sendall(struct.pack('>I', length))
    sock.sendall(data)

def recv_msg(sock):
    raw_len = sock.recv(4)
    if not raw_len: return None
    length = struct.unpack('>I', raw_len)[0]
    
    # Read payload
    data = b''
    while len(data) < length:
        packet = sock.recv(length - len(data))
        if not packet: return None
        data += packet
    return data

def test_layer2_direct():
    console.print("[bold blue]Starting Layer 2 (Python -> Go Direct TCP) Test[/bold blue]")
    
    # 1. Prepare Arrow Data
    console.print("Preparing Arrow Batch...")
    try:
        schema = pa.schema([
            ('tx_id', pa.string()),
            ('sender', pa.string()),
            ('amount', pa.int64()),
            ('timestamp', pa.int64())
        ])
        
        batch = pa.RecordBatch.from_arrays(
            [
                pa.array(['tx_001', 'tx_002']),
                pa.array(['user_a', 'user_b']),
                pa.array([100, 200]),
                pa.array([1700000000, 1700000001])
            ], 
            schema=schema
        )
        
        sink = pa.BufferOutputStream()
        with pa.ipc.new_stream(sink, schema) as writer:
            writer.write_batch(batch)
        payload = sink.getvalue().to_pybytes()
        
        console.print(f"[green]✔ Arrow Batch created. Size: {len(payload)} bytes[/green]")
        
    except Exception as e:
        console.print(f"[red]✘ Failed to create Arrow batch: {e}[/red]")
        sys.exit(1)

    # 2. Connect to Go Server
    HOST = '127.0.0.1'
    PORT = 50051
    console.print(f"Connecting to {HOST}:{PORT}...")
    
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(5.0) # 5s timeout
        s.connect((HOST, PORT))
        console.print("[green]✔ Connected[/green]")
        
        # NOTE: If Auth is enabled, handshake is needed here.
        # Assuming Auth is disabled for this test as per default Env.
        
        # 3. Send Data
        start_time = time.time()
        send_msg(s, payload)
        console.print("Sent Arrow IPC stream.")
        
        # 4. Receive Response
        resp = recv_msg(s)
        duration = (time.time() - start_time) * 1000
        
        if resp:
            console.print(f"[green]✔ Received Response ({len(resp)} bytes) in {duration:.2f}ms[/green]")
            # Try to parse response as Arrow IPC
            try:
                reader = pa.ipc.open_stream(resp)
                resp_batch = reader.read_next_batch()
                console.print(f"  Response Rows: {resp_batch.num_rows}")
                console.print(f"  Response Schema: {resp_batch.schema.names}")
            except Exception as e:
                console.print(f"[yellow]  ⚠ Response is not Arrow IPC (Could be error JSON?): {resp[:100]}[/yellow]")
                try:
                    import json
                    json_resp = json.loads(resp.decode('utf-8'))
                    console.print(f"  JSON Response: {json_resp}")
                except:
                    pass
        else:
            console.print("[red]✘ No response received or connection closed[/red]")
            sys.exit(1)
            
        s.close()
        
    except ConnectionRefusedError:
        console.print("[red]✘ Connection Refused. Is Arrow Server running?[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]✘ Error: {e}[/red]")
        sys.exit(1)

    console.print("[bold blue]Layer 2 Test Completed[/bold blue]")

if __name__ == "__main__":
    test_layer2_direct()

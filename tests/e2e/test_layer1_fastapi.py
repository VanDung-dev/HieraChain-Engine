"""
End-to-end (E2E) test for Layer 1 functionality of HieraChain-Engine.

This test ensures that JSON data can be successfully processed and submitted
to FastAPI for chain creation and event handling.
"""

import httpx
import time
import sys
from rich.console import Console

console = Console()

BASE_URL = "http://localhost:2661/api/v1"

def test_layer1():
    console.print("[bold blue]Starting Layer 1 (JSON -> FastAPI) Test[/bold blue]")

    # 1. Health Check
    try:
        r = httpx.get(f"{BASE_URL}/health")
        if r.status_code == 200:
            console.print("[green]✔ Health Check Passed[/green]")
        else:
            console.print(f"[red]✘ Health Check Failed: {r.status_code}[/red]")
            sys.exit(1)
    except Exception as e:
        console.print(f"[red]✘ Connection Failed: {e}[/red]")
        sys.exit(1)

    # 2. Create Chain
    chain_name = "test-e2e-layer1"
    console.print(f"Creating chain: [cyan]{chain_name}[/cyan]")
    r = httpx.post(f"{BASE_URL}/chains/{chain_name}/create", json={"chain_type": "test"})
    
    # 201 Created or 400 if already exists (which is fine for re-run)
    if r.status_code in [201, 200] or (r.status_code == 500 and "already exists" in r.text):
        console.print("[green]✔ Chain Creation Passed[/green]")
    else:
        console.print(f"[red]✘ Chain Creation Failed: {r.status_code} {r.text}[/red]")
        sys.exit(1)

    # 3. Test Valid Event (JSON Parsing & Validation)
    console.print("Testing [bold]Valid[/bold] Event Submission...")
    payload = {
        "entity_id": "item-001",
        "event_type": "CREATED",
        "details": {"price": 100, "currency": "USD"}
    }
    
    start_time = time.time()
    r = httpx.post(f"{BASE_URL}/chains/{chain_name}/events", json=payload)
    duration = (time.time() - start_time) * 1000

    if r.status_code == 200:
        data = r.json()
        console.print(f"[green]✔ Valid Event Accepted[/green] in {duration:.2f}ms")
        console.print(f"  Response: {data}")
        # Verify response structure matches Pydantic schema
        if "success" in data and "event_id" in data:
            console.print("[green]  ✔ Response Schema Matches[/green]")
        else:
            console.print("[red]  ✘ Response Schema Mismatch[/red]")
    else:
        console.print(f"[red]✘ Valid Event Failed: {r.status_code} {r.text}[/red]")

    # 4. Test Invalid Event (Missing Required Field)
    console.print("Testing [bold]Invalid[/bold] Event Submission (Missing entity_id)...")
    invalid_payload = {
        "event_type": "Updated",
        "details": {"note": "Missing ID"}
    }
    
    r = httpx.post(f"{BASE_URL}/chains/{chain_name}/events", json=invalid_payload)
    
    if r.status_code == 422:
        console.print("[green]✔ Validation Error correctly caught (422 Unprocessable Entity)[/green]")
        console.print(f"  Error Detail: {r.json()}")
    else:
        console.print(f"[red]✘ Expected 422, got {r.status_code}[/red]")

    console.print("[bold blue]Layer 1 Test Completed[/bold blue]")

if __name__ == "__main__":
    # Wait for server to start if needed
    time.sleep(2) 
    test_layer1()

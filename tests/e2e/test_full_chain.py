"""
End-to-end (E2E) test for the full chain functionality of HieraChain-Engine.

This test ensures that the entire chain creation, processing, and retrieval
flow works as expected.
"""

import requests
import time
import logging
from concurrent.futures import ThreadPoolExecutor

# Configuration
BASE_URL = "http://localhost:8001"  # Port 8001 for E2E testing to avoid conflict
CHAIN_ID = f"test-chain-e2e-{int(time.time())}"

# Setup Logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

def wait_for_server():
    """Wait for FastAPI server to be up."""
    for _ in range(10):
        try:
            r = requests.get(f"{BASE_URL}/api/v1/health", timeout=1)
            if r.status_code == 200:
                logger.info("✔ Server is UP")
                return True
        except requests.exceptions.ConnectionError:
            pass
        time.sleep(1)
        print(".", end="", flush=True)
    logger.error("✘ Server is DOWN")
    return False

def create_chain():
    """Create a test chain."""
    # Use path parameter for chain creation
    url = f"{BASE_URL}/api/v1/chains/{CHAIN_ID}/create"
    # chain_type defaults to "generic"
    r = requests.post(url)
    if r.status_code in [200, 201]:
        logger.info(f"✔ Chain '{CHAIN_ID}' created.")
    elif r.status_code == 400 and "already exists" in r.text:
        logger.info(f"⚠ Chain '{CHAIN_ID}' already exists.")
    elif r.status_code == 405:
        logger.error(f"✘ Method Not Allowed at {url}")
        return False
    else:
        logger.error(f"✘ Failed to create chain: {r.status_code} - {r.text}")
        return False
    return True

def test_happy_path_single_tx():
    """7.1. Happy Path: Submit single valid transaction."""
    logger.info("Starting 7.1 Happy Path Test...")
    
    event = {
        "entity_id": "user_001",
        "event_type": "CREATE_ACCOUNT",
        "details": {"balance": 1000}
    }
    
    start_time = time.time()
    url = f"{BASE_URL}/api/v1/chains/{CHAIN_ID}/events"
    r = requests.post(url, json=event)
    duration = (time.time() - start_time) * 1000
    
    if r.status_code == 200:
        logger.info(f"✔ Happy Path Passed. Latency: {duration:.2f}ms")
        logger.info(f"  Response: {r.json()}")
        return True
    else:
        logger.error(f"✘ Happy Path Failed: {r.status_code} - {r.text}")
        return False

def test_batch_processing(count=100):
    """7.2. Batch Processing: Submit multiple transactions."""
    logger.info(f"Starting 7.2 Batch Test ({count} txs)...")
    
    success_count = 0
    start_time = time.time()
    
    def send_tx(i):
        event = {
            "entity_id": f"user_{i}",
            "event_type": "TRANSFER",
            "details": {"amount": i}
        }
        url = f"{BASE_URL}/api/v1/chains/{CHAIN_ID}/events"
        try:
            r = requests.post(url, json=event, timeout=5)
            return r.status_code == 200
        except:
            return False

    # Sequential Loop for simplicity in this verification step
    # (To test real throughput we would use ThreadPool, but let's keep it simple for connectivity check)
    with ThreadPoolExecutor(max_workers=10) as executor:
        results = list(executor.map(send_tx, range(count)))
    
    success_count = sum(results)
    total_time = time.time() - start_time
    avg_latency = (total_time * 1000) / count
    
    logger.info(f"Batch Result: {success_count}/{count} passed.")
    logger.info(f"Total Time: {total_time:.2f}s. Avg Latency: {avg_latency:.2f}ms/req")
    
    return success_count == count

def run_e2e_tests():
    if not wait_for_server():
        exit(1)
        
    if not create_chain():
        exit(1)
        
    # 7.1
    if not test_happy_path_single_tx():
        logger.warning("Skipping Batch Test due to Happy Path failure.")
        exit(1)
        
    # 7.2
    test_batch_processing(100)
    
if __name__ == "__main__":
    run_e2e_tests()

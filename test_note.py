#!/usr/bin/env python3
import asyncio
import websockets
import json
import time
import hashlib
import hmac
import base64
import secrets

# Generate a test keypair (this is just for testing)
def generate_keypair():
    private_key = secrets.token_bytes(32)
    public_key = hashlib.sha256(private_key).digest()
    return private_key.hex(), public_key.hex()

def sign_event(event, private_key_hex):
    # Simple signature for testing (not cryptographically secure)
    private_key = bytes.fromhex(private_key_hex)
    event_json = json.dumps(event, separators=(',', ':'), sort_keys=True)
    signature = hmac.new(private_key, event_json.encode(), hashlib.sha256).hexdigest()
    return signature

# Generate test keypair
private_key, public_key = generate_keypair()
print(f"Test public key: {public_key}")

# Create a test note
event = {
    "id": "",
    "pubkey": public_key,
    "created_at": int(time.time()),
    "kind": 1,
    "tags": [],
    "content": "Hello from Mercury Relay! This is a test note.",
    "sig": ""
}

# Calculate event ID (simplified)
event_json = json.dumps(event, separators=(',', ':'), sort_keys=True)
event_id = hashlib.sha256(event_json.encode()).hexdigest()
event["id"] = event_id

# Sign the event
signature = sign_event(event, private_key)
event["sig"] = signature

print(f"Event ID: {event_id}")
print(f"Event: {json.dumps(event, indent=2)}")

async def send_note():
    uri = "ws://192.168.222.109:8080"
    
    try:
        async with websockets.connect(uri) as websocket:
            print(f"Connected to {uri}")
            
            # Send the event
            message = ["EVENT", event]
            await websocket.send(json.dumps(message))
            print("Sent EVENT message")
            
            # Wait for response
            response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
            print(f"Received response: {response}")
            
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    asyncio.run(send_note())

#!/usr/bin/env python3
import socket
import base64
import hashlib
import hmac
import json
import time
import struct

def create_websocket_frame(data, opcode=1):
    """Create a WebSocket frame"""
    payload = data.encode('utf-8')
    length = len(payload)
    
    if length < 126:
        header = struct.pack('!BB', 0x80 | opcode, 0x80 | length)
    else:
        header = struct.pack('!BBH', 0x80 | opcode, 0x80 | 126, length)
    
    # Simple masking key (not cryptographically secure)
    mask_key = b'\x12\x34\x56\x78'
    masked_payload = bytearray()
    for i, byte in enumerate(payload):
        masked_payload.append(byte ^ mask_key[i % 4])
    
    return header + mask_key + masked_payload

def create_test_event_with_your_npub():
    """Create a test event with your actual npub"""
    # Your npub from the config
    your_npub = "npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5y2sd8v6cnsvq465zjx"
    content = "Hello Mercury Relay! This is a test note from your whitelisted account."
    created_at = int(time.time())
    
    # Create event object
    event = {
        "id": "",
        "pubkey": your_npub,
        "created_at": created_at,
        "kind": 1,
        "tags": [],
        "content": content,
        "sig": "test_signature_123456789abcdef"
    }
    
    # Calculate event ID (simplified)
    event_json = json.dumps(event, separators=(',', ':'), sort_keys=True)
    event_id = hashlib.sha256(event_json.encode()).hexdigest()
    event["id"] = event_id
    
    return event

def test_mercury_relay_auth():
    """Test connection to Mercury Relay with authentication"""
    host = "192.168.222.109"
    port = 8080
    
    print(f"Connecting to Mercury Relay at {host}:{port}")
    
    try:
        # Create socket
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(10)
        sock.connect((host, port))
        
        print("✓ Connected to Mercury Relay!")
        
        # Send WebSocket upgrade request
        upgrade_request = (
            "GET / HTTP/1.1\r\n"
            "Host: 192.168.222.109:8080\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n"
            "Sec-WebSocket-Version: 13\r\n"
            "\r\n"
        )
        
        print("Sending WebSocket upgrade request...")
        sock.send(upgrade_request.encode())
        
        # Read response
        response = sock.recv(1024).decode()
        print("Response received:")
        print(response)
        
        if "101 Switching Protocols" in response:
            print("✓ WebSocket upgrade successful!")
            
            # Create test event with your npub
            event = create_test_event_with_your_npub()
            print(f"Created test event with your npub: {event['pubkey']}")
            
            # Send REQ message
            req_message = ["REQ", "test-sub", {"kinds": [1]}]
            req_frame = create_websocket_frame(json.dumps(req_message))
            sock.send(req_frame)
            print("✓ Sent REQ message")
            
            # Send EVENT message with your npub
            event_message = ["EVENT", event]
            event_frame = create_websocket_frame(json.dumps(event_message))
            sock.send(event_frame)
            print("✓ Sent EVENT message with your npub")
            
            # Try to read response
            try:
                response = sock.recv(1024)
                if response:
                    print(f"✓ Received response: {response}")
                else:
                    print("No response received")
            except socket.timeout:
                print("No immediate response")
            
        else:
            print("✗ WebSocket upgrade failed")
            print("Response:", response)
            
    except Exception as e:
        print(f"✗ Error: {e}")
    finally:
        try:
            sock.close()
        except:
            pass

if __name__ == "__main__":
    test_mercury_relay_auth()

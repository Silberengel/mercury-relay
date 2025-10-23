#!/bin/bash

# Test WebSocket connection to Mercury Relay
echo "Testing WebSocket connection to Mercury Relay..."

# Create a simple test event
cat > test_event.json << 'INNER_EOF'
{
  "id": "test123456789",
  "pubkey": "test_pubkey",
  "created_at": 1699123456,
  "kind": 1,
  "tags": [],
  "content": "Hello Mercury Relay!",
  "sig": "test_signature"
}
INNER_EOF

# Test WebSocket upgrade
echo "Testing WebSocket upgrade..."
curl -v \
  -H "Upgrade: websocket" \
  -H "Connection: Upgrade" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Sec-WebSocket-Version: 13" \
  --max-time 5 \
  http://192.168.222.109:8080

echo -e "\n\nTesting with wscat if available..."
if command -v wscat &> /dev/null; then
    echo '["REQ", "test-sub", {"kinds": [1]}]' | wscat -c ws://192.168.222.109:8080
else
    echo "wscat not available, trying netcat..."
    echo -e "GET / HTTP/1.1\r\nHost: 192.168.222.109:8080\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n" | nc 192.168.222.109 8080
fi

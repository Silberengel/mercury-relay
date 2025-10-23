#!/bin/bash

echo "Testing WebSocket connection to Mercury Relay..."

# Test basic connectivity
echo "1. Testing basic connectivity..."
timeout 3 telnet 192.168.222.109 8080 << 'TELNET_EOF'
GET / HTTP/1.1
Host: 192.168.222.109:8080
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13

TELNET_EOF

echo -e "\n2. Testing with curl WebSocket headers..."
curl -v \
  -H "Upgrade: websocket" \
  -H "Connection: Upgrade" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Sec-WebSocket-Version: 13" \
  --max-time 3 \
  http://192.168.222.109:8080 2>&1 | head -20

echo -e "\n3. Testing regular HTTP request..."
curl -s http://192.168.222.109:8080

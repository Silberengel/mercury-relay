#!/bin/bash

echo "=== Mercury Relay External Connection Test ==="
echo "Testing connection to: 192.168.222.109:8080"
echo

# Test 1: Basic connectivity
echo "1. Testing basic TCP connectivity..."
timeout 5 bash -c "echo > /dev/tcp/192.168.222.109/8080" && echo "✅ TCP connection successful" || echo "❌ TCP connection failed"

# Test 2: HTTP request
echo "2. Testing HTTP request..."
response=$(curl -s --connect-timeout 10 http://192.168.222.109:8080)
if [ $? -eq 0 ]; then
    echo "✅ HTTP request successful"
    echo "Response: $response"
else
    echo "❌ HTTP request failed"
fi

# Test 3: WebSocket upgrade headers
echo "3. Testing WebSocket upgrade headers..."
ws_response=$(curl -s --connect-timeout 10 -H "Upgrade: websocket" -H "Connection: Upgrade" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" -H "Sec-WebSocket-Version: 13" -I http://192.168.222.109:8080 | head -5)
if echo "$ws_response" | grep -q "101 Switching Protocols"; then
    echo "✅ WebSocket upgrade successful"
    echo "Response headers:"
    echo "$ws_response"
else
    echo "❌ WebSocket upgrade failed"
    echo "Response: $ws_response"
fi

echo
echo "=== Network Information ==="
echo "Your IP: $(hostname -I | awk '{print $1}')"
echo "Target IP: 192.168.222.109"
echo "Target Port: 8080"

echo
echo "=== Troubleshooting Tips ==="
echo "If tests fail:"
echo "1. Check if both machines are on the same network"
echo "2. Verify firewall settings on both machines"
echo "3. Try pinging the relay machine: ping 192.168.222.109"
echo "4. Check if the relay is actually running: netstat -tlnp | grep 8080"

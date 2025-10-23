#!/bin/bash

echo "=== Mercury Relay Network Diagnostics ==="
echo "Date: $(date)"
echo

# Get network information
echo "=== Network Configuration ==="
echo "Hostname: $(hostname)"
echo "Primary IP: $(hostname -I | awk '{print $1}')"
echo "All IPs:"
ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print "  " $2}'
echo

# Check if relay is running
echo "=== Relay Process Status ==="
if pgrep -f mercury-relay > /dev/null; then
    echo "✅ Mercury Relay is running"
    echo "Process info:"
    ps aux | grep mercury-relay | grep -v grep
else
    echo "❌ Mercury Relay is NOT running"
fi
echo

# Check port listening
echo "=== Port Status ==="
if netstat -tlnp | grep ":8080" > /dev/null; then
    echo "✅ Port 8080 is listening"
    echo "Listening details:"
    netstat -tlnp | grep ":8080"
else
    echo "❌ Port 8080 is NOT listening"
fi
echo

# Test local connectivity
echo "=== Local Connectivity Tests ==="
echo "1. Testing localhost connection..."
if curl -s --connect-timeout 5 http://localhost:8080 > /dev/null; then
    echo "✅ Localhost connection successful"
else
    echo "❌ Localhost connection failed"
fi

echo "2. Testing primary IP connection..."
primary_ip=$(hostname -I | awk '{print $1}')
if curl -s --connect-timeout 5 http://$primary_ip:8080 > /dev/null; then
    echo "✅ Primary IP ($primary_ip) connection successful"
else
    echo "❌ Primary IP ($primary_ip) connection failed"
fi

echo "3. Testing WebSocket upgrade..."
ws_test=$(curl -s --connect-timeout 5 -H "Upgrade: websocket" -H "Connection: Upgrade" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" -H "Sec-WebSocket-Version: 13" -I http://$primary_ip:8080 | head -1)
if echo "$ws_test" | grep -q "101"; then
    echo "✅ WebSocket upgrade successful"
else
    echo "❌ WebSocket upgrade failed"
fi
echo

# Network routing
echo "=== Network Routing ==="
echo "Default route:"
ip route | grep default | head -1
echo "Local network routes:"
ip route | grep "192.168\|10\." | head -5
echo

# Firewall check (basic)
echo "=== Firewall Status ==="
if command -v ufw > /dev/null; then
    echo "UFW status:"
    ufw status 2>/dev/null | head -3
else
    echo "UFW not available"
fi

if command -v iptables > /dev/null; then
    echo "iptables rules count: $(iptables -L | wc -l)"
else
    echo "iptables not available"
fi
echo

echo "=== Troubleshooting for External Machines ==="
echo "To test from another machine on the same network:"
echo "1. Run: curl http://$primary_ip:8080"
echo "2. Run: telnet $primary_ip 8080"
echo "3. Check if both machines are on the same subnet"
echo "4. Verify no firewall is blocking the connection"
echo "5. Try: ping $primary_ip"
echo
echo "Common issues:"
echo "- Windows Firewall blocking connections"
echo "- Network isolation (guest network, VLAN separation)"
echo "- Router blocking inter-device communication"
echo "- Different subnets or network segments"

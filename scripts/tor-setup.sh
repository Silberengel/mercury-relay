#!/bin/bash

# Tor Hidden Service Setup for Mercury Relay
# This script helps you configure Tor hidden services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧅 Tor Hidden Service Setup for Mercury Relay${NC}"
echo "=============================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Please run as root (use sudo)${NC}"
    exit 1
fi

# Check if Tor is installed
if ! command -v tor &> /dev/null; then
    echo -e "${YELLOW}📦 Installing Tor...${NC}"
    
    # Detect OS and install Tor
    if command -v apt &> /dev/null; then
        # Ubuntu/Debian
        apt update
        apt install -y tor
    elif command -v yum &> /dev/null; then
        # CentOS/RHEL
        yum install -y tor
    elif command -v dnf &> /dev/null; then
        # Fedora
        dnf install -y tor
    else
        echo -e "${RED}❌ Cannot detect package manager. Please install Tor manually.${NC}"
        exit 1
    fi
fi

# Create Tor configuration directory
TOR_CONFIG_DIR="/etc/tor"
TOR_SERVICE_DIR="/var/lib/tor"

echo -e "${YELLOW}🔧 Configuring Tor hidden service...${NC}"

# Backup existing configuration
if [ -f "$TOR_CONFIG_DIR/torrc" ]; then
    cp "$TOR_CONFIG_DIR/torrc" "$TOR_CONFIG_DIR/torrc.backup.$(date +%Y%m%d_%H%M%S)"
fi

# Create Mercury Relay Tor configuration
cat > "$TOR_CONFIG_DIR/torrc.mercury" << 'EOF'
# Mercury Relay Tor Configuration

# Basic Tor settings
SocksPort 9050
ControlPort 9051
CookieAuthentication 1

# Hidden service for Mercury Relay
HiddenServiceDir /var/lib/tor/mercury_relay
HiddenServicePort 80 127.0.0.1:8080
HiddenServicePort 8081 127.0.0.1:8081
HiddenServicePort 8082 127.0.0.1:8082

# Security settings
HiddenServiceMaxStreams 0
HiddenServiceNumIntroductionPoints 3

# Logging
Log notice file /var/log/tor/notices.log
Log info file /var/log/tor/info.log

# Disable client features (relay-only mode)
ClientOnly 1
EOF

# Create Tor service directory
mkdir -p "$TOR_SERVICE_DIR/mercury_relay"
chown -R debian-tor:debian-tor "$TOR_SERVICE_DIR/mercury_relay"
chmod 700 "$TOR_SERVICE_DIR/mercury_relay"

# Create log directory
mkdir -p /var/log/tor
chown -R debian-tor:debian-tor /var/log/tor

# Start Tor service
echo -e "${YELLOW}🚀 Starting Tor service...${NC}"
systemctl enable tor
systemctl start tor

# Wait for Tor to start
sleep 5

# Get the hidden service address
if [ -f "$TOR_SERVICE_DIR/mercury_relay/hostname" ]; then
    HIDDEN_SERVICE_ADDRESS=$(cat "$TOR_SERVICE_DIR/mercury_relay/hostname")
    echo -e "${GREEN}🎉 Tor hidden service configured!${NC}"
    echo ""
    echo -e "${GREEN}🔗 Your Mercury Relay Tor address:${NC}"
    echo "  • Nostr WebSocket: ws://$HIDDEN_SERVICE_ADDRESS/"
    echo "  • REST API: http://$HIDDEN_SERVICE_ADDRESS:8082/"
    echo "  • Admin API: http://$HIDDEN_SERVICE_ADDRESS:8081/"
    echo ""
    echo -e "${YELLOW}📝 Save this address: $HIDDEN_SERVICE_ADDRESS${NC}"
    echo -e "${YELLOW}📁 Configuration: $TOR_CONFIG_DIR/torrc.mercury${NC}"
    echo -e "${YELLOW}📁 Service directory: $TOR_SERVICE_DIR/mercury_relay${NC}"
else
    echo -e "${RED}❌ Failed to create hidden service${NC}"
    echo -e "${YELLOW}📝 Check Tor logs: journalctl -u tor${NC}"
    exit 1
fi

# Create systemd service for Mercury Relay with Tor
cat > /etc/systemd/system/mercury-relay-tor.service << EOF
[Unit]
Description=Mercury Relay with Tor
After=network.target tor.service
Requires=tor.service

[Service]
Type=simple
User=mercury
Group=mercury
WorkingDirectory=/app
ExecStart=/app/mercury-relay
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}✅ Tor hidden service setup complete!${NC}"
echo ""
echo -e "${GREEN}📋 Next steps:${NC}"
echo "1. Start Mercury Relay: docker-compose up -d"
echo "2. Test Tor connection: curl http://$HIDDEN_SERVICE_ADDRESS/health"
echo "3. Share your Tor address with Nostr clients"
echo ""
echo -e "${GREEN}🔧 Tor management:${NC}"
echo "  • Start: sudo systemctl start tor"
echo "  • Stop: sudo systemctl stop tor"
echo "  • Status: sudo systemctl status tor"
echo "  • Logs: journalctl -u tor -f"

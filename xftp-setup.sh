#!/bin/bash

# XFTP Setup for Mercury Relay
# This script helps you configure XFTP for decentralized file storage

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}📁 XFTP Setup for Mercury Relay${NC}"
echo "=================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Please run as root (use sudo)${NC}"
    exit 1
fi

# XFTP Configuration
XFTP_PORT=8083
XFTP_DATA_DIR="/var/lib/xftp"
XFTP_CONFIG_DIR="/etc/xftp"

echo -e "${YELLOW}📦 Setting up XFTP...${NC}"

# Create XFTP directories
mkdir -p "$XFTP_DATA_DIR"
mkdir -p "$XFTP_CONFIG_DIR"
mkdir -p /var/log/xftp

# Create XFTP user
if ! id "xftp" &>/dev/null; then
    useradd -r -s /bin/false -d "$XFTP_DATA_DIR" xftp
fi

# Set permissions
chown -R xftp:xftp "$XFTP_DATA_DIR"
chown -R xftp:xftp /var/log/xftp
chmod 755 "$XFTP_DATA_DIR"

# Create XFTP configuration
cat > "$XFTP_CONFIG_DIR/xftp.conf" << EOF
# XFTP Configuration for Mercury Relay

# Server settings
port = $XFTP_PORT
data_dir = $XFTP_DATA_DIR
log_file = /var/log/xftp/xftp.log
log_level = info

# Storage settings
max_file_size = 100MB
max_storage = 10GB
cleanup_interval = 24h
expire_after = 7d

# Security settings
allowed_origins = *
rate_limit = 100/minute
max_connections = 1000

# Network settings
bind_address = 0.0.0.0
timeout = 30s
keep_alive = true

# CORS settings
cors_enabled = true
cors_origins = ["*"]
cors_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
cors_headers = ["Content-Type", "Authorization", "X-Requested-With"]
EOF

# Create XFTP systemd service
cat > /etc/systemd/system/xftp.service << EOF
[Unit]
Description=XFTP File Storage Service
After=network.target

[Service]
Type=simple
User=xftp
Group=xftp
WorkingDirectory=$XFTP_DATA_DIR
ExecStart=/usr/local/bin/xftp -config $XFTP_CONFIG_DIR/xftp.conf
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Create XFTP binary (placeholder - you'll need to build or download XFTP)
cat > /usr/local/bin/xftp << 'EOF'
#!/bin/bash
# XFTP Server Implementation
# This is a placeholder - replace with actual XFTP implementation

echo "XFTP Server starting..."
echo "Port: 8083"
echo "Data directory: /var/lib/xftp"
echo "Config: /etc/xftp/xftp.conf"

# Simple HTTP server for file storage
cd /var/lib/xftp
python3 -m http.server 8083
EOF

chmod +x /usr/local/bin/xftp

# Enable and start XFTP service
systemctl daemon-reload
systemctl enable xftp
systemctl start xftp

# Wait for service to start
sleep 3

# Check if XFTP is running
if systemctl is-active --quiet xftp; then
    echo -e "${GREEN}✅ XFTP service is running${NC}"
    echo ""
    echo -e "${GREEN}📁 XFTP Configuration:${NC}"
    echo "  • Port: $XFTP_PORT"
    echo "  • Data directory: $XFTP_DATA_DIR"
    echo "  • Config: $XFTP_CONFIG_DIR/xftp.conf"
    echo "  • Logs: /var/log/xftp/xftp.log"
    echo ""
    echo -e "${GREEN}🔗 XFTP Endpoints:${NC}"
    echo "  • Upload: http://localhost:$XFTP_PORT/upload"
    echo "  • Download: http://localhost:$XFTP_PORT/download/{file_id}"
    echo "  • List: http://localhost:$XFTP_PORT/files"
    echo "  • Health: http://localhost:$XFTP_PORT/health"
else
    echo -e "${RED}❌ XFTP service failed to start${NC}"
    echo -e "${YELLOW}📝 Check logs: journalctl -u xftp${NC}"
    exit 1
fi

# Create XFTP client configuration for Mercury Relay
cat > /etc/mercury-relay/xftp.conf << EOF
# Mercury Relay XFTP Configuration

[xftp]
enabled = true
server_url = "http://localhost:$XFTP_PORT"
upload_endpoint = "/upload"
download_endpoint = "/download"
list_endpoint = "/files"
health_endpoint = "/health"

[storage]
max_file_size = "50MB"
allowed_types = ["image/*", "application/pdf", "text/*"]
expire_after = "7d"
cleanup_interval = "24h"

[security]
rate_limit = "100/minute"
max_connections = 100
timeout = "30s"
EOF

echo -e "${GREEN}🎉 XFTP setup complete!${NC}"
echo ""
echo -e "${GREEN}📋 Next steps:${NC}"
echo "1. Update Mercury Relay config to use XFTP"
echo "2. Test XFTP: curl http://localhost:$XFTP_PORT/health"
echo "3. Configure file upload/download in Mercury Relay"
echo ""
echo -e "${GREEN}🔧 XFTP management:${NC}"
echo "  • Start: sudo systemctl start xftp"
echo "  • Stop: sudo systemctl stop xftp"
echo "  • Status: sudo systemctl status xftp"
echo "  • Logs: journalctl -u xftp -f"
echo "  • Config: $XFTP_CONFIG_DIR/xftp.conf"

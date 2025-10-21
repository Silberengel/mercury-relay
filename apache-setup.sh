#!/bin/bash

# Apache Setup Script for Mercury Relay
# This script helps you configure Apache to work with Mercury Relay Docker containers

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🌐 Apache Setup for Mercury Relay${NC}"
echo "=================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Please run as root (use sudo)${NC}"
    exit 1
fi

# Check if Apache is installed
if ! command -v apache2 &> /dev/null && ! command -v httpd &> /dev/null; then
    echo -e "${RED}❌ Apache is not installed. Please install Apache first.${NC}"
    echo "Ubuntu/Debian: sudo apt install apache2"
    echo "CentOS/RHEL: sudo yum install httpd"
    exit 1
fi

# Detect Apache installation
if command -v apache2 &> /dev/null; then
    APACHE_SERVICE="apache2"
    APACHE_CONF_DIR="/etc/apache2"
    APACHE_SITES_DIR="/etc/apache2/sites-available"
    APACHE_ENABLE_DIR="/etc/apache2/sites-enabled"
elif command -v httpd &> /dev/null; then
    APACHE_SERVICE="httpd"
    APACHE_CONF_DIR="/etc/httpd"
    APACHE_SITES_DIR="/etc/httpd/conf.d"
    APACHE_ENABLE_DIR="/etc/httpd/conf.d"
fi

echo -e "${YELLOW}📁 Apache configuration directory: $APACHE_CONF_DIR${NC}"

# Enable required modules
echo -e "${YELLOW}🔧 Enabling required Apache modules...${NC}"

if [ "$APACHE_SERVICE" = "apache2" ]; then
    a2enmod rewrite
    a2enmod proxy
    a2enmod proxy_wstunnel
    a2enmod headers
    a2enmod ssl
    a2enmod proxy_http
else
    # For CentOS/RHEL, modules are usually enabled by default
    echo "Modules should be enabled by default on CentOS/RHEL"
fi

# Copy Apache configuration
echo -e "${YELLOW}📋 Installing Apache configuration...${NC}"

# Create the virtual host configuration
cat > $APACHE_SITES_DIR/mercury-relay.conf << 'EOF'
<VirtualHost *:80>
    ServerName mercury-relay.local
    DocumentRoot /var/www/html
    
    # Security headers
    Header always set X-Frame-Options DENY
    Header always set X-Content-Type-Options nosniff
    Header always set X-XSS-Protection "1; mode=block"
    Header always set Referrer-Policy "strict-origin-when-cross-origin"
    
    # Enable mod_rewrite
    RewriteEngine On
    
    # Proxy settings for Mercury Relay
    ProxyPreserveHost On
    ProxyRequests Off
    
    # Nostr WebSocket endpoint
    ProxyPass / ws://localhost:8080/
    ProxyPassReverse / ws://localhost:8080/
    
    # WebSocket upgrade headers
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/?(.*) "ws://localhost:8080/$1" [P,L]
    
    # Admin API
    ProxyPass /admin/ http://localhost:8081/
    ProxyPassReverse /admin/ http://localhost:8081/
    
    # REST API
    ProxyPass /api/ http://localhost:8082/
    ProxyPassReverse /api/ http://localhost:8082/
    
    # Health check
    ProxyPass /health http://localhost:8080/health
    ProxyPassReverse /health http://localhost:8080/health
    
    # Logging
    ErrorLog ${APACHE_LOG_DIR}/mercury-relay_error.log
    CustomLog ${APACHE_LOG_DIR}/mercury-relay_access.log combined
    
    # Timeout settings for WebSocket
    ProxyTimeout 86400
    ProxyPassTimeout 86400
</VirtualHost>
EOF

# Enable the site (Ubuntu/Debian)
if [ "$APACHE_SERVICE" = "apache2" ]; then
    a2ensite mercury-relay.conf
fi

# Test Apache configuration
echo -e "${YELLOW}🔍 Testing Apache configuration...${NC}"
if $APACHE_SERVICE -t; then
    echo -e "${GREEN}✅ Apache configuration is valid${NC}"
else
    echo -e "${RED}❌ Apache configuration has errors${NC}"
    exit 1
fi

# Restart Apache
echo -e "${YELLOW}🔄 Restarting Apache...${NC}"
systemctl restart $APACHE_SERVICE

# Check if Apache is running
if systemctl is-active --quiet $APACHE_SERVICE; then
    echo -e "${GREEN}✅ Apache is running${NC}"
else
    echo -e "${RED}❌ Apache failed to start${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 Apache setup complete!${NC}"
echo ""
echo -e "${GREEN}📋 Next steps:${NC}"
echo "1. Update ServerName in $APACHE_SITES_DIR/mercury-relay.conf"
echo "2. Start Mercury Relay: docker-compose up -d"
echo "3. Test the setup: curl http://localhost/health"
echo ""
echo -e "${GREEN}🔧 Configuration file: $APACHE_SITES_DIR/mercury-relay.conf${NC}"
echo -e "${GREEN}📝 Logs: /var/log/apache2/mercury-relay_*.log${NC}"

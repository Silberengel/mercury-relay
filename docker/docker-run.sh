#!/bin/bash

# Mercury Relay Docker Run Script
# This script shows how to run Mercury Relay with environment variables

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🐳 Mercury Relay Docker Run Examples${NC}"
echo "====================================="

# Default configuration
DEFAULT_RELAYS="wss://theforest.nostr1.com,wss://orly-relay.imwald.eu,wss://nostr.land,wss://nostr21.com"

# Function to run Mercury Relay with custom environment
run_mercury_relay() {
    local description="$1"
    local env_vars="$2"
    
    echo -e "${YELLOW}📋 $description${NC}"
    echo "Command:"
    echo "docker run -d --name mercury-relay \\"
    echo "  -p 8080:8080 -p 8081:8081 -p 8082:8082 \\"
    echo "  -v \$(pwd)/data:/app/data \\"
    echo "  -v \$(pwd)/logs:/app/logs \\"
    echo "  $env_vars \\"
    echo "  mercury-relay"
    echo ""
}

# Example 1: Default configuration
run_mercury_relay "Default Configuration" \
"  -e STREAMING_ENABLED=true \\
  -e UPSTREAM_RELAYS=\"$DEFAULT_RELAYS\""

# Example 2: Custom relays
run_mercury_relay "Custom Relays" \
"  -e STREAMING_ENABLED=true \\
  -e UPSTREAM_RELAYS=\"wss://relay.damus.io,wss://nos.lol,wss://relay.snort.social\""

# Example 3: Disable streaming
run_mercury_relay "No Streaming" \
"  -e STREAMING_ENABLED=false"

# Example 4: With Tor and XFTP
run_mercury_relay "With Tor and XFTP" \
"  -e STREAMING_ENABLED=true \\
  -e UPSTREAM_RELAYS=\"$DEFAULT_RELAYS\" \\
  -e TOR_ENABLED=true \\
  -e XFTP_ENABLED=true"

# Example 5: Production configuration
run_mercury_relay "Production Configuration" \
"  -e STREAMING_ENABLED=true \\
  -e UPSTREAM_RELAYS=\"$DEFAULT_RELAYS\" \\
  -e LOG_LEVEL=warn \\
  -e RATE_LIMIT_PER_MINUTE=200 \\
  -e API_KEY=\"your-secret-api-key\""

echo -e "${GREEN}🚀 Quick Start Commands:${NC}"
echo ""
echo -e "${YELLOW}1. Basic setup:${NC}"
echo "docker run -d --name mercury-relay \\"
echo "  -p 8080:8080 -p 8081:8081 -p 8082:8082 \\"
echo "  -e STREAMING_ENABLED=true \\"
echo "  -e UPSTREAM_RELAYS=\"$DEFAULT_RELAYS\" \\"
echo "  mercury-relay"
echo ""

echo -e "${YELLOW}2. Custom relays:${NC}"
echo "docker run -d --name mercury-relay \\"
echo "  -p 8080:8080 -p 8081:8081 -p 8082:8082 \\"
echo "  -e UPSTREAM_RELAYS=\"wss://your-relay1.com,wss://your-relay2.com\" \\"
echo "  mercury-relay"
echo ""

echo -e "${YELLOW}3. With environment file:${NC}"
echo "cp env.example .env"
echo "# Edit .env with your settings"
echo "docker run -d --name mercury-relay \\"
echo "  -p 8080:8080 -p 8081:8081 -p 8082:8082 \\"
echo "  --env-file .env \\"
echo "  mercury-relay"
echo ""

echo -e "${GREEN}🔧 Management Commands:${NC}"
echo ""
echo -e "${YELLOW}View logs:${NC}"
echo "docker logs -f mercury-relay"
echo ""
echo -e "${YELLOW}Stop container:${NC}"
echo "docker stop mercury-relay"
echo ""
echo -e "${YELLOW}Remove container:${NC}"
echo "docker rm mercury-relay"
echo ""
echo -e "${YELLOW}Update relays at runtime:${NC}"
echo "docker exec mercury-relay sh -c 'echo \"wss://new-relay.com\" > /tmp/new_relays'"
echo ""

echo -e "${GREEN}📊 Environment Variables Reference:${NC}"
echo ""
echo "STREAMING_ENABLED=true|false"
echo "UPSTREAM_RELAYS=relay1,relay2,relay3"
echo "TOR_ENABLED=true|false"
echo "XFTP_ENABLED=true|false"
echo "LOG_LEVEL=debug|info|warn|error"
echo "RATE_LIMIT_PER_MINUTE=100"
echo "API_KEY=your-secret-key"
echo ""
echo -e "${GREEN}📝 See env.example for all available variables${NC}"

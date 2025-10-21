#!/bin/bash

# Mercury Relay Streaming Setup
# This script configures streaming with specific upstream relays

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🌊 Mercury Relay Streaming Setup${NC}"
echo "=================================="

# Default upstream relays
UPSTREAM_RELAYS=(
    "wss://theforest.nostr1.com"
    "wss://orlay-relay.imwald.eu"
    "wss://nostr.land"
    "wss://nostr21.com"
)

# Backup relays
BACKUP_RELAYS=(
    "wss://relay.damus.io"
    "wss://nostr.wine"
    "wss://nos.lol"
    "wss://relay.snort.social"
)

echo -e "${YELLOW}🔧 Configuring upstream relays...${NC}"

# Test relay connectivity
test_relay() {
    local relay_url="$1"
    local relay_name=$(echo "$relay_url" | sed 's|wss://||' | sed 's|/||')
    
    echo -n "Testing $relay_name... "
    
    # Simple connectivity test (you might want to implement a proper WebSocket test)
    if timeout 5 bash -c "</dev/tcp/$(echo "$relay_url" | sed 's|wss://||' | cut -d: -f1)/443" 2>/dev/null; then
        echo -e "${GREEN}✅${NC}"
        return 0
    else
        echo -e "${RED}❌${NC}"
        return 1
    fi
}

# Test all upstream relays
echo -e "${YELLOW}🔍 Testing relay connectivity...${NC}"
WORKING_RELAYS=()
FAILED_RELAYS=()

for relay in "${UPSTREAM_RELAYS[@]}"; do
    if test_relay "$relay"; then
        WORKING_RELAYS+=("$relay")
    else
        FAILED_RELAYS+=("$relay")
    fi
done

# Test backup relays
echo -e "${YELLOW}🔍 Testing backup relays...${NC}"
for relay in "${BACKUP_RELAYS[@]}"; do
    if test_relay "$relay"; then
        WORKING_RELAYS+=("$relay")
    else
        FAILED_RELAYS+=("$relay")
    fi
done

# Update configuration with working relays
echo -e "${YELLOW}📝 Updating configuration...${NC}"

# Create streaming configuration
cat > streaming-config.yaml << EOF
# Mercury Relay Streaming Configuration
# Generated on $(date)

streaming:
  enabled: true
  
  # Working upstream relays
  upstream_relays:
EOF

# Add working relays to configuration
priority=1
for relay in "${WORKING_RELAYS[@]}"; do
    cat >> streaming-config.yaml << EOF
    - url: "$relay"
      enabled: true
      priority: $priority
EOF
    ((priority++))
done

# Add failed relays as disabled
cat >> streaming-config.yaml << EOF
  
  # Failed relays (disabled)
  backup_relays:
EOF

for relay in "${FAILED_RELAYS[@]}"; do
    cat >> streaming-config.yaml << EOF
    - url: "$relay"
      enabled: false
      priority: $priority
EOF
    ((priority++))
done

# Add remaining configuration
cat >> streaming-config.yaml << EOF

  # Transport methods
  transport_methods:
    websocket: true
    http_streaming: true
    sse: true
    tor: true
    i2p: true

  # Connection settings
  connection_pool_size: 10
  reconnect_interval: "30s"
  timeout: "60s"
  
  # Retry settings
  max_retries: 3
  retry_delay: "10s"
  
  # Load balancing
  load_balancing:
    strategy: "round_robin"
    health_check_interval: "60s"
    failover_enabled: true
    
  # Rate limiting
  rate_limiting:
    enabled: true
    requests_per_minute: 100
    burst_size: 20
    
  # Monitoring
  monitoring:
    enabled: true
    metrics_endpoint: "/metrics"
    health_check_endpoint: "/health"
    log_level: "info"
EOF

# Update main config.yaml
echo -e "${YELLOW}📝 Updating main configuration...${NC}"

# Backup original config
cp config.yaml config.yaml.backup.$(date +%Y%m%d_%H%M%S)

# Update streaming section in main config
python3 << EOF
import yaml
import sys

# Load current config
with open('config.yaml', 'r') as f:
    config = yaml.safe_load(f)

# Load streaming config
with open('streaming-config.yaml', 'r') as f:
    streaming_config = yaml.safe_load(f)

# Update streaming section
config['streaming'] = streaming_config['streaming']

# Save updated config
with open('config.yaml', 'w') as f:
    yaml.dump(config, f, default_flow_style=False, sort_keys=False)

print("Configuration updated successfully")
EOF

# Create Docker Compose override for streaming
cat > docker-compose.override.yml << EOF
version: '3.8'

services:
  mercury-relay:
    environment:
      - STREAMING_ENABLED=true
      - UPSTREAM_RELAYS=${WORKING_RELAYS[*]}
      - STREAMING_CONFIG_FILE=/app/streaming-config.yaml
    volumes:
      - ./streaming-config.yaml:/app/streaming-config.yaml:ro
EOF

echo -e "${GREEN}✅ Streaming configuration complete!${NC}"
echo ""
echo -e "${GREEN}📊 Summary:${NC}"
echo "  • Working relays: ${#WORKING_RELAYS[@]}"
echo "  • Failed relays: ${#FAILED_RELAYS[@]}"
echo ""
echo -e "${GREEN}🌐 Working relays:${NC}"
for relay in "${WORKING_RELAYS[@]}"; do
    echo "  • $relay"
done

if [ ${#FAILED_RELAYS[@]} -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}⚠️  Failed relays:${NC}"
    for relay in "${FAILED_RELAYS[@]}"; do
        echo "  • $relay"
    done
fi

echo ""
echo -e "${GREEN}📋 Next steps:${NC}"
echo "1. Start Mercury Relay: docker-compose up -d"
echo "2. Check streaming status: docker-compose logs mercury-relay"
echo "3. Test connectivity: curl http://localhost:8082/api/v1/health"
echo ""
echo -e "${GREEN}🔧 Management:${NC}"
echo "  • View logs: docker-compose logs -f mercury-relay"
echo "  • Restart: docker-compose restart mercury-relay"
echo "  • Update config: ./streaming-setup.sh"

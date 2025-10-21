#!/bin/bash

# Mercury Relay with Tor and XFTP Deployment Script
# This script deploys Mercury Relay with Tor hidden services and XFTP storage

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧅 Mercury Relay with Tor and XFTP Deployment${NC}"
echo "=============================================="

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed. Please install Docker Compose first.${NC}"
    exit 1
fi

# Create necessary directories
echo -e "${YELLOW}📁 Creating directories...${NC}"
mkdir -p data logs ssl tor-config xftp-config

# Set up SSL certificates (optional)
echo -e "${YELLOW}🔒 Setting up SSL certificates...${NC}"
if [ ! -f ssl/cert.pem ]; then
    echo "Creating self-signed certificate for development..."
    openssl req -x509 -newkey rsa:4096 -keyout ssl/key.pem -out ssl/cert.pem -days 365 -nodes -subj "/C=US/ST=State/L=City/O=Organization/CN=mercury-relay.local"
    echo -e "${GREEN}✅ Self-signed certificate created${NC}"
    echo -e "${YELLOW}⚠️  For production, use Let's Encrypt or a proper SSL certificate${NC}"
fi

# Build the Docker image
echo -e "${YELLOW}🔨 Building Docker image...${NC}"
docker build -t mercury-relay .

# Stop existing containers if running
echo -e "${YELLOW}🛑 Stopping existing containers...${NC}"
docker-compose -f docker-compose-tor.yml down 2>/dev/null || true

# Start the services with Tor and XFTP
echo -e "${YELLOW}🚀 Starting Mercury Relay with Tor and XFTP...${NC}"
docker-compose -f docker-compose-tor.yml up -d

# Wait for services to be ready
echo -e "${YELLOW}⏳ Waiting for services to start...${NC}"
sleep 15

# Get Tor hidden service address
echo -e "${YELLOW}🔍 Getting Tor hidden service address...${NC}"
sleep 10

# Check if Tor container is running and get the address
if docker-compose -f docker-compose-tor.yml ps | grep -q "mercury-tor.*Up"; then
    echo -e "${GREEN}✅ Tor service is running${NC}"
    
    # Try to get the hidden service address
    TOR_ADDRESS=""
    for i in {1..30}; do
        if docker-compose -f docker-compose-tor.yml exec mercury-tor cat /var/lib/tor/mercury_relay/hostname 2>/dev/null; then
            TOR_ADDRESS=$(docker-compose -f docker-compose-tor.yml exec mercury-tor cat /var/lib/tor/mercury_relay/hostname 2>/dev/null | tr -d '\r\n')
            break
        fi
        echo "Waiting for Tor hidden service to initialize... ($i/30)"
        sleep 2
    done
    
    if [ -n "$TOR_ADDRESS" ]; then
        echo -e "${GREEN}🎉 Tor hidden service address: $TOR_ADDRESS${NC}"
    else
        echo -e "${YELLOW}⚠️  Could not retrieve Tor address. Check Tor logs.${NC}"
    fi
else
    echo -e "${RED}❌ Tor service failed to start${NC}"
fi

# Check if services are running
echo -e "${YELLOW}🔍 Checking service status...${NC}"
if docker-compose -f docker-compose-tor.yml ps | grep -q "Up"; then
    echo -e "${GREEN}✅ Mercury Relay with Tor and XFTP is running!${NC}"
    echo ""
    echo -e "${GREEN}🌐 Services available at:${NC}"
    echo "  • Nostr WebSocket: ws://localhost:8080/"
    echo "  • REST API: http://localhost:8082/"
    echo "  • Admin API: http://localhost:8081/"
    echo "  • XFTP Storage: http://localhost:8083/"
    echo "  • Health Check: http://localhost:8080/health"
    echo ""
    if [ -n "$TOR_ADDRESS" ]; then
        echo -e "${GREEN}🧅 Tor Hidden Services:${NC}"
        echo "  • Nostr WebSocket: ws://$TOR_ADDRESS/"
        echo "  • REST API: http://$TOR_ADDRESS:8082/"
        echo "  • Admin API: http://$TOR_ADDRESS:8081/"
        echo ""
        echo -e "${YELLOW}📝 Save your Tor address: $TOR_ADDRESS${NC}"
    fi
    echo ""
    echo -e "${GREEN}📊 Container status:${NC}"
    docker-compose -f docker-compose-tor.yml ps
    echo ""
    echo -e "${GREEN}📝 View logs with: docker-compose -f docker-compose-tor.yml logs -f${NC}"
    echo -e "${GREEN}🛑 Stop with: docker-compose -f docker-compose-tor.yml down${NC}"
    echo ""
    echo -e "${GREEN}🔧 Management commands:${NC}"
    echo "  • View Tor logs: docker-compose -f docker-compose-tor.yml logs mercury-tor"
    echo "  • View XFTP logs: docker-compose -f docker-compose-tor.yml logs mercury-xftp"
    echo "  • Restart services: docker-compose -f docker-compose-tor.yml restart"
    echo "  • Update services: docker-compose -f docker-compose-tor.yml pull && docker-compose -f docker-compose-tor.yml up -d"
else
    echo -e "${RED}❌ Failed to start Mercury Relay with Tor and XFTP${NC}"
    echo -e "${YELLOW}📝 Check logs with: docker-compose -f docker-compose-tor.yml logs${NC}"
    exit 1
fi

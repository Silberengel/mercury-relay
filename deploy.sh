#!/bin/bash

# Mercury Relay Deployment Script
# This script helps you deploy Mercury Relay to your remote server

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DOCKER_IMAGE="mercury-relay"
CONTAINER_NAME="mercury-relay"
DOMAIN="your-domain.com"  # Change this to your domain
EMAIL="your-email@example.com"  # For Let's Encrypt

echo -e "${GREEN}🚀 Mercury Relay Deployment Script${NC}"
echo "=================================="

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
mkdir -p data logs ssl

# Set up SSL certificates (optional)
echo -e "${YELLOW}🔒 Setting up SSL certificates...${NC}"
if [ ! -f ssl/cert.pem ]; then
    echo "Creating self-signed certificate for development..."
    openssl req -x509 -newkey rsa:4096 -keyout ssl/key.pem -out ssl/cert.pem -days 365 -nodes -subj "/C=US/ST=State/L=City/O=Organization/CN=$DOMAIN"
    echo -e "${GREEN}✅ Self-signed certificate created${NC}"
    echo -e "${YELLOW}⚠️  For production, use Let's Encrypt or a proper SSL certificate${NC}"
fi

# Build the Docker image
echo -e "${YELLOW}🔨 Building Docker image...${NC}"
docker build -t $DOCKER_IMAGE .

# Stop existing container if running
echo -e "${YELLOW}🛑 Stopping existing container...${NC}"
docker-compose down 2>/dev/null || true

# Start the services
echo -e "${YELLOW}🚀 Starting Mercury Relay...${NC}"
docker-compose up -d

# Wait for services to be ready
echo -e "${YELLOW}⏳ Waiting for services to start...${NC}"
sleep 10

# Check if services are running
echo -e "${YELLOW}🔍 Checking service status...${NC}"
if docker-compose ps | grep -q "Up"; then
    echo -e "${GREEN}✅ Mercury Relay is running!${NC}"
    echo ""
    echo -e "${GREEN}🌐 Services available at:${NC}"
    echo "  • Nostr WebSocket: ws://$DOMAIN/"
    echo "  • REST API: http://$DOMAIN/api/"
    echo "  • Admin API: http://$DOMAIN/admin/"
    echo "  • Health Check: http://$DOMAIN/health"
    echo ""
    echo -e "${GREEN}📊 Container status:${NC}"
    docker-compose ps
    echo ""
    echo -e "${GREEN}📝 View logs with: docker-compose logs -f${NC}"
    echo -e "${GREEN}🛑 Stop with: docker-compose down${NC}"
else
    echo -e "${RED}❌ Failed to start Mercury Relay${NC}"
    echo -e "${YELLOW}📝 Check logs with: docker-compose logs${NC}"
    exit 1
fi

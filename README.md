# Mercury Relay

A high-performance, censorship-resistant Nostr relay with advanced quality control, REST API, streaming capabilities, e-paper reader support, and decentralized storage.

## Features

- **Censorship Resistance**: Operates over Tor (.onion) and I2P hidden services
- **Ephemeral Storage**: Uses XFTP for 48-hour file storage with metadata privacy
- **Access Control**: Owner-based write permissions using follow lists (Kind 3 events)
- **Quality Control**: Built-in spam detection and content moderation with NIP-based event kind validation
- **REST API**: Full REST API support for programmatic access
- **Streaming**: Connect to and stream events from other relays via WebSocket, Tor, I2P, and HTTP streaming
- **E-Paper Support**: Optimized for e-readers with EPUB generation and offline reading
- **NKBIP-01 Support**: Full support for Nostr publications (kind 30040/30041)
- **Environment Variables**: Runtime configuration via Docker environment variables
- **Apache Integration**: Native Apache reverse proxy support
- **Admin Interface**: Terminal UI for real-time monitoring and moderation
- **Test Data Generation**: Generate realistic Nostr events for testing
- **Multi-Transport**: Supports both Tor and I2P simultaneously
- **Message Queuing**: RabbitMQ for reliable event processing
- **Caching**: Redis for 28-hour event caching
- **Event Kind Configuration**: YAML-based configuration for Nostr event kind quality control

## Architecture

```
Nostr Client → [Tor .onion | I2P] → Go Relay → RabbitMQ → XFTP Server
                                        ↓
                                   Redis Cache (28hr TTL)
                                        ↓
                                   PostgreSQL (Analytics)
                                        ↓
                              Upstream Relays (Streaming)
```

## Quick Start

### Docker Deployment

```bash
# Clone the repository
git clone <your-repo-url>
cd mercury-relay

# Basic deployment
docker-compose up -d

# With custom relays
UPSTREAM_RELAYS="wss://your-relay1.com,wss://your-relay2.com" docker-compose up -d

# With Tor and XFTP
docker-compose -f docker-compose-tor.yml up -d
```

### Environment Variables

```bash
# Set upstream relays
export UPSTREAM_RELAYS="wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com"

# Enable features
export STREAMING_ENABLED=true
export TOR_ENABLED=true
export XFTP_ENABLED=true

# Run with environment
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -e STREAMING_ENABLED=true \
  -e UPSTREAM_RELAYS="$UPSTREAM_RELAYS" \
  mercury-relay
```

## Features

### REST API

Mercury Relay includes a comprehensive REST API for programmatic access:

- **GET /api/v1/events** - Query events with filters
- **POST /api/v1/query** - Advanced event queries
- **POST /api/v1/publish** - Publish events
- **GET /api/v1/stream** - HTTP streaming for server-side rendering
- **GET /api/v1/sse** - Server-Sent Events for monitoring (stats, health, admin)
- **GET /api/v1/ebooks** - Publication discovery (kind 30040, NKBIP-01)
- **GET /api/v1/ebooks/{id}/content** - Publication content with nested structure (kind 30041)
- **GET /api/v1/ebooks/{id}/epub** - Generate EPUB from Nostr publication
- **GET /api/v1/health** - Health check
- **GET /api/v1/stats** - Relay statistics

Example usage:
```bash
# Query events
curl "http://localhost:8082/api/v1/events?authors=pubkey1&kinds=1&limit=10"

# Publish an event
curl -X POST http://localhost:8082/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"event": {"content": "Hello world!", "kind": 1, ...}}'

# HTTP streaming for server-side rendering
curl "http://localhost:8082/api/v1/stream?authors=pubkey1&kinds=1"

# Server-Sent Events for monitoring
curl "http://localhost:8082/api/v1/sse?type=stats"    # Real-time statistics
curl "http://localhost:8082/api/v1/sse?type=health"  # Health monitoring
curl "http://localhost:8082/api/v1/sse?type=admin"   # Admin dashboard

# Publication discovery (optimized for e-paper readers)
curl "http://localhost:8082/api/v1/ebooks?format=epub&limit=20"
curl "http://localhost:8082/api/v1/ebooks?author=pubkey1&format=pdf"

# Publication content with nested structure (kind 30040 + 30041)
curl "http://localhost:8082/api/v1/ebooks/book_id_here/content?format=asciidoc&depth=3"
curl "http://localhost:8082/api/v1/ebooks/book_id_here/content?format=html&images=true&depth=2"

# Generate EPUB from Nostr publication
curl "http://localhost:8082/api/v1/ebooks/book_id_here/epub?images=true" -o "book.epub"
curl "http://localhost:8082/api/v1/ebooks/book_id_here/epub?format=epub&images=false" -o "book.epub"
```

## Deployment Options

### 1. Basic Docker Deployment

```bash
# Quick start with default configuration
docker-compose up -d

# Check status
docker-compose ps
docker-compose logs -f
```

### 2. Custom Configuration

```bash
# Set custom upstream relays
export UPSTREAM_RELAYS="wss://your-relay1.com,wss://your-relay2.com"

# Deploy with custom settings
docker-compose up -d
```

### 3. Tor Hidden Service

```bash
# Deploy with Tor support
docker-compose -f docker-compose-tor.yml up -d

# Get Tor address
docker-compose -f docker-compose-tor.yml exec mercury-tor cat /var/lib/tor/mercury_relay/hostname
```

### 4. Apache Integration

```bash
# Configure Apache on host system
sudo ./apache-setup.sh

# Start Mercury Relay
docker-compose up -d mercury-relay redis postgres
```

### 5. Production Deployment

```bash
# Production configuration
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -v /opt/mercury/data:/app/data \
  -v /opt/mercury/logs:/app/logs \
  -e STREAMING_ENABLED=true \
  -e UPSTREAM_RELAYS="wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com" \
  -e TOR_ENABLED=true \
  -e XFTP_ENABLED=true \
  -e LOG_LEVEL=warn \
  -e RATE_LIMIT_PER_MINUTE=200 \
  -e API_KEY="your-production-api-key" \
  --restart unless-stopped \
  mercury-relay
```

## Configuration

### Environment Variables

Mercury Relay supports extensive configuration via environment variables:

```bash
# Core Settings
NOSTR_RELAY_PORT=8080
ADMIN_PORT=8081
REST_API_PORT=8082
LOG_LEVEL=info

# Streaming
STREAMING_ENABLED=true
UPSTREAM_RELAYS=wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com

# Tor Support
TOR_ENABLED=false
TOR_SOCKS_PORT=9050
TOR_CONTROL_PORT=9051

# XFTP Storage
XFTP_ENABLED=false
XFTP_PORT=8083
XFTP_MAX_FILE_SIZE=50MB

# Security
API_KEY=change_this_secret_key
CORS_ENABLED=true
RATE_LIMIT_PER_MINUTE=100
```

### Configuration Files

- **`config.yaml`** - Main configuration file
- **`nostr-event-kinds.yaml`** - Event kind validation rules
- **`streaming-config.yaml`** - Streaming configuration
- **`env.example`** - Environment variables reference

### Default Upstream Relays

Mercury Relay is configured to stream from these relays by default:

1. **`wss://theforest.nostr1.com`** - High performance relay
2. **`wss://orlay-relay.imwald.eu`** - European relay  
3. **`wss://nostr.land`** - Community relay
4. **`wss://nostr21.com`** - Alternative relay

### Streaming from Other Relays

Connect to and stream events from other Nostr relays:

- **WebSocket connections** to upstream relays
- **HTTP streaming** for server-side rendering support
- **Server-Sent Events (SSE)** for monitoring and admin dashboards
- **Tor routing** for anonymous connections
- **I2P routing** for additional privacy
- **Automatic reconnection** and health monitoring

Configure upstream relays in `config.yaml`:
```yaml
streaming:
  enabled: true
  upstream_relays:
    - url: "wss://relay.damus.io"
      enabled: true
      priority: 1
    - url: "wss://nostr.wine"
      enabled: true
      priority: 2
    - url: "https://relay.example.com/api/v1/stream"
      enabled: true
      priority: 3
    - url: "sse://relay.example.com/api/v1/sse"
      enabled: true
      priority: 4
  transport_methods:
    websocket: true
    http_streaming: true
    sse: true
    tor: true
    i2p: true
```

### E-Paper Reader Support

Optimized for e-paper readers with Nostr publications (kind 30040, NKBIP-01):

- **HTTP REST API** - Simple polling-based discovery
- **Cached responses** - 1-hour cache for offline reading
- **Format filtering** - EPUB, PDF, MOBI, TXT support
- **Metadata extraction** - Title, author, cover, download URLs
- **Nested content structure** - Hierarchical book content (kind 30040 + 30041)
- **AsciiDoc support** - Native AsciiDoc content rendering
- **EPUB generation** - Convert any Nostr book to standard EPUB format
- **E-paper optimized** - Minimal data transfer, simple JSON

Perfect for:
- **Kindle** and other e-readers
- **E-ink tablets** with limited connectivity
- **Offline reading** with periodic sync
- **Low-power devices** with simple HTTP clients

Example e-paper reader integration:
```bash
# Discover new ebooks
curl "http://localhost:8082/api/v1/ebooks?format=epub&limit=10"

# Get specific author's books
curl "http://localhost:8082/api/v1/ebooks?author=pubkey1&format=pdf"

# Response optimized for e-paper readers
{
  "success": true,
  "count": 5,
  "ebooks": [
    {
      "id": "event_id",
      "title": "Book Title",
      "author_name": "Author Name",
      "format": "epub",
      "size": "2.5MB",
      "download_url": "https://...",
      "cover": "https://...",
      "created_at": 1234567890
    }
  ]
}

# Publication content with nested structure
curl "http://localhost:8082/api/v1/ebooks/book_id/content?format=asciidoc&depth=3"

# Response with hierarchical content structure
{
  "success": true,
  "book": {
    "id": "book_event_id",
    "title": "Advanced Programming",
    "author": "Author Name",
    "structure": {
      "title": "Book Structure",
      "type": "root",
      "children": [
        {
          "id": "chapter_1_id",
          "title": "Chapter 1: Introduction",
          "type": "chapter",
          "content": "= Chapter 1: Introduction\n\nThis is the introduction...",
          "format": "asciidoc",
          "children": [
            {
              "id": "section_1_1_id",
              "title": "Section 1.1: Getting Started",
              "type": "section",
              "content": "== Getting Started\n\nLet's begin...",
              "format": "asciidoc"
            }
          ]
        }
      ]
    }
  },
  "content_format": "asciidoc",
  "include_images": false,
  "max_depth": 3
}

# Generate EPUB from any Nostr publication
curl "http://localhost:8082/api/v1/ebooks/book_id/epub?images=true" -o "book.epub"

# Features of generated EPUB:
# - Standard EPUB 2.0 format
# - Proper metadata (title, author, language, etc.)
# - Table of contents (NCX)
# - XHTML chapters with proper structure
# - CSS styling for e-readers
# - Image support (optional)
# - AsciiDoc/Markdown to HTML conversion
# - Compatible with all major e-readers
```

### Event Kind Quality Control

Comprehensive YAML configuration for Nostr event kind validation based on NIPs:

- **Kind-specific validation rules** for each event type
- **Content validation** (JSON, text, encrypted, base64)
- **Tag validation** with regex patterns
- **Quality scoring** with configurable weights
- **Spam detection** with multiple detection methods
- **Rate limiting** per event kind

Example configuration in `nostr-event-kinds.yaml`:
```yaml
event_kinds:
  0: # User metadata
    name: "User Metadata"
    content_validation:
      type: "json"
      required_fields: ["name"]
    quality_rules:
      - name: "valid_json"
        weight: 1.0
  1: # Text note
    name: "Text Note"
    content_validation:
      type: "text"
      max_length: 10000
    quality_rules:
      - name: "not_spam"
        weight: 1.0
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- Tor (for .onion addresses)
- I2P (for I2P addresses)

### Using Docker Compose

1. **Clone and start services:**
   ```bash
   git clone <repository>
   cd mercury-relay
   docker-compose up -d
   ```

2. **Configure access control (optional):**
   ```bash
   # Set your own npub as owner
   export OWNER_NPUB="npub1your_npub_here"
   
   # Allow public write (default: false)
   export ACCESS_PUBLIC_WRITE="true"
   
   # Set custom relay for follow list updates
   export ACCESS_RELAY_URL="https://your-relay.com"
   
   # Restart with new settings
   docker-compose up -d
   ```

3. **Check service status:**
   ```bash
   docker-compose ps
   ```

4. **View logs:**
   ```bash
   docker-compose logs -f mercury-relay
   ```

5. **Get relay addresses:**
   ```bash
   # Tor .onion address
   docker-compose exec tor cat /var/lib/tor/mercury_relay/hostname
   
   # I2P address (check logs)
   docker-compose logs i2p | grep "Tunnel"
   ```

### Local Development

1. **Install dependencies:**
   ```bash
   go mod download
   ```

2. **Start services:**
   ```bash
   # Start RabbitMQ, Redis, PostgreSQL, XFTP, Tor, I2P
   docker-compose up -d rabbitmq redis postgres xftp-server tor i2p
   ```

3. **Run the relay:**
   ```bash
   go run cmd/mercury-relay/main.go
   ```

4. **Run admin interface:**
   ```bash
   go run cmd/mercury-admin/main.go --tui
   ```

## Configuration

Edit `config.yaml` to customize:

- **Server settings**: Port, timeouts
- **Tor settings**: Hidden service configuration
- **I2P settings**: SAM bridge configuration
- **Access control**: Owner npub, follow list updates, public access
- **Quality control**: Spam thresholds, rate limits
- **Storage**: XFTP server settings
- **Caching**: Redis configuration

### Access Control Configuration

The relay supports owner-based access control using your follow list:

```yaml
access:
  owner_npub: "npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl"
  update_interval: "1h"
  relay_url: "https://relay.damus.io"
  allow_public_read: true
  allow_public_write: false
```

**Environment Variables:**
- `OWNER_NPUB`: Your npub (default: provided npub)
- `ACCESS_UPDATE_INTERVAL`: How often to update follow list (default: 1h)
- `ACCESS_RELAY_URL`: Relay to fetch your follow list from (default: relay.damus.io)
- `ACCESS_PUBLIC_READ`: Allow anyone to read (default: true)
- `ACCESS_PUBLIC_WRITE`: Allow anyone to write (default: false)

**How it works:**
1. Only the owner and people in their follow list (Kind 3 event) can write
2. The relay periodically fetches your latest follow list
3. Public read is allowed by default for discovery
4. Write access is restricted to prevent spam

## Usage

### Connecting to the Relay

Use any Nostr client with the relay addresses:

```bash
# Tor .onion address
ws://your-onion-address.onion:8080

# I2P address  
ws://your-i2p-address.i2p:8080

# Direct IP (if accessible)
ws://your-server-ip:8080
```

### Admin Interface

#### Terminal UI
```bash
# Start interactive admin interface
./mercury-admin --tui

# Block a user
./mercury-admin --block npub1abc123...

# Unblock a user  
./mercury-admin --unblock npub1abc123...

# List blocked users
./mercury-admin --list-blocked
```

#### API
```bash
# Get statistics
curl -H "X-API-Key: your-api-key" http://localhost:8081/api/stats

# Block a user
curl -X POST -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"npub":"npub1abc123..."}' \
  http://localhost:8081/api/block

# List blocked users
curl -H "X-API-Key: your-api-key" http://localhost:8081/api/blocked
```

### Test Data Generation

Generate realistic test events:

```bash
# Generate 100 events with random personas
./test-data-gen --count 100 --persona random

# Generate spam events
./test-data-gen --count 50 --persona spammer

# Export to file
./test-data-gen --count 200 --output test-events.json --format json
```

## Services

### Core Services

- **mercury-relay**: Main Nostr relay server
- **mercury-admin**: Admin CLI and TUI interface
- **test-data-gen**: Test data generator

### Infrastructure Services

- **rabbitmq**: Message queuing (port 5672, management UI: 15672)
- **redis**: Caching layer (port 6379)
- **postgres**: Analytics database (port 5432)
- **xftp-server**: SimpleX file transfer server (port 443)
- **tor**: Tor hidden service (SOCKS: 9050, control: 9051)
- **i2p**: I2P router (SAM: 7656, HTTP: 4444/4445)

## Quality Control

The relay includes several quality control mechanisms:

### Spam Detection
- Content length analysis
- Tag count monitoring  
- Rate limiting per npub
- Quality scoring algorithm

### Moderation Tools
- Real-time event monitoring
- Npub blocking/unblocking
- Event quarantine system
- Quality score dashboard

### Analytics
- Event statistics
- Author behavior analysis
- Quality metrics over time
- Performance monitoring

## Security Features

### Privacy Protection
- No IP address correlation (XFTP)
- Anonymous transport (Tor/I2P)
- Ephemeral storage (48-hour TTL)
- Metadata padding

### Censorship Resistance
- Multiple transport protocols
- Hidden service addresses
- No DNS dependencies
- Distributed architecture

## Development

### Project Structure

```
mercury-relay/
├── cmd/                    # Application entry points
│   ├── mercury-relay/      # Main relay server
│   ├── mercury-admin/      # Admin interface
│   └── test-data-gen/     # Test data generator
├── internal/              # Internal packages
│   ├── config/           # Configuration management
│   ├── relay/            # Nostr relay core
│   ├── transport/        # Tor/I2P transport
│   ├── queue/            # RabbitMQ integration
│   ├── cache/            # Redis caching
│   ├── storage/          # XFTP storage
│   ├── quality/          # Quality control
│   ├── models/           # Data models
│   ├── admin/            # Admin interface
│   ├── api/              # REST API
│   └── testgen/          # Test data generation
├── docker-compose.yml     # Service orchestration
├── Dockerfile           # Container build
├── config.yaml          # Configuration
└── README.md           # This file
```

### Building

```bash
# Build all binaries
make build

# Build specific binary
go build -o mercury-relay ./cmd/mercury-relay
go build -o mercury-admin ./cmd/mercury-admin
go build -o test-data-gen ./cmd/test-data-gen
```

### Testing

```bash
# Run tests
go test ./...

# Generate test data
./test-data-gen --count 100 --persona random

# Test relay connection
curl -H "Upgrade: websocket" -H "Connection: Upgrade" \
  http://localhost:8080/
```

## Monitoring

### Health Checks

All services include health checks:

```bash
# Check service health
docker-compose ps

# View health check logs
docker-compose logs mercury-relay | grep health
```

### Metrics

Access metrics via admin API:

```bash
# Relay statistics
curl -H "X-API-Key: your-key" http://localhost:8081/api/stats

# Queue depth
curl -H "X-API-Key: your-key" http://localhost:8081/api/queue

# Cache statistics  
curl -H "X-API-Key: your-key" http://localhost:8081/api/cache
```

## Troubleshooting

### Common Issues

1. **Tor hidden service not starting:**
   ```bash
   # Check Tor logs
   docker-compose logs tor
   
   # Verify Tor configuration
   docker-compose exec tor cat /etc/tor/torrc
   ```

2. **I2P tunnel not created:**
   ```bash
   # Check I2P logs
   docker-compose logs i2p
   
   # Verify SAM bridge
   docker-compose exec i2p netstat -ln | grep 7656
   ```

3. **XFTP server not responding:**
   ```bash
   # Check XFTP logs
   docker-compose logs xftp-server
   
   # Test XFTP connection
   curl http://localhost:443/health
   ```

### Logs

View logs for specific services:

```bash
# All services
docker-compose logs

# Specific service
docker-compose logs mercury-relay
docker-compose logs rabbitmq
docker-compose logs redis
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Documentation

### Detailed Guides

- **[Docker Deployment](README-Docker.md)** - Complete Docker setup guide
- **[Apache Integration](README-Apache.md)** - Apache reverse proxy configuration
- **[Environment Variables](README-Environment.md)** - Runtime configuration guide
- **[Streaming Setup](README-Streaming.md)** - Upstream relay configuration
- **[Tor Setup](tor-setup.sh)** - Tor hidden service configuration
- **[XFTP Setup](xftp-setup.sh)** - Decentralized storage setup

### Scripts

- **`deploy.sh`** - Basic deployment script
- **`deploy-tor.sh`** - Deployment with Tor and XFTP
- **`streaming-setup.sh`** - Configure upstream relays
- **`apache-setup.sh`** - Apache configuration
- **`tor-setup.sh`** - Tor hidden service setup
- **`xftp-setup.sh`** - XFTP storage setup
- **`docker-run.sh`** - Docker run examples

### Configuration Files

- **`config.yaml`** - Main configuration
- **`nostr-event-kinds.yaml`** - Event kind validation
- **`streaming-config.yaml`** - Streaming configuration
- **`env.example`** - Environment variables
- **`docker-compose.yml`** - Basic Docker setup
- **`docker-compose-tor.yml`** - Tor and XFTP setup
- **`apache.conf`** - Apache virtual host
- **`nginx.conf`** - Nginx configuration (alternative)

## API Endpoints

### Nostr Protocol
- **WebSocket**: `ws://localhost:8080/` - Nostr protocol endpoint

### REST API
- **Events**: `GET /api/v1/events` - Query events
- **Query**: `POST /api/v1/query` - Advanced queries
- **Publish**: `POST /api/v1/publish` - Publish events
- **Stream**: `GET /api/v1/stream` - HTTP streaming
- **SSE**: `GET /api/v1/sse` - Server-Sent Events
- **E-books**: `GET /api/v1/ebooks` - Publication discovery
- **E-book Content**: `GET /api/v1/ebooks/{id}/content` - Nested content
- **EPUB Generation**: `GET /api/v1/ebooks/{id}/epub` - Generate EPUB
- **Health**: `GET /api/v1/health` - Health check
- **Stats**: `GET /api/v1/stats` - Statistics

### Admin API
- **Admin**: `http://localhost:8081/` - Admin interface
- **Metrics**: `http://localhost:8081/metrics` - Performance metrics

## Monitoring

### Health Checks
```bash
# Check service health
curl http://localhost:8080/health
curl http://localhost:8082/api/v1/health

# View logs
docker-compose logs -f mercury-relay

# Check container status
docker-compose ps
```

### Metrics
```bash
# View statistics
curl http://localhost:8082/api/v1/stats

# View metrics
curl http://localhost:8081/metrics
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Check if ports 8080, 8081, 8082 are available
2. **Relay connectivity**: Test upstream relay connections
3. **Permission issues**: Check Docker volume permissions
4. **Configuration errors**: Validate YAML configuration files

### Debug Commands

```bash
# Check configuration
docker-compose config

# View detailed logs
docker-compose logs -f mercury-relay

# Test connectivity
./streaming-setup.sh

# Check environment variables
docker exec mercury-relay env | grep -E "(STREAMING|UPSTREAM|TOR|XFTP)"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:

- Create an issue on GitHub
- Join the discussion in the repository
- Check the documentation for common solutions
# Mercury Relay Quick Start Guide

Get Mercury Relay up and running with the new dynamic kind-based filtering system in minutes.

## Prerequisites

- Docker and Docker Compose
- Git
- Basic understanding of Nostr

## Quick Setup

### 1. Clone and Start

```bash
# Clone the repository
git clone https://github.com/your-org/mercury-relay.git
cd mercury-relay

# Start all services
cd docker && docker-compose up -d

# Check status
docker-compose ps
```

### 2. Verify Installation

```bash
# Check health (public endpoint - no auth required)
curl http://localhost:8082/api/v1/health

# Expected response:
# {"status":"healthy","timestamp":"2024-01-15T10:30:00Z","version":"1.0.0","uptime":"2h30m15s"}
```

### 3. Access Services

- **WebSocket**: `ws://localhost:8080`
- **REST API**: `http://localhost:8082`
- **RabbitMQ Management**: `http://localhost:15672` (guest/guest)
- **Admin API**: `http://localhost:8081`

## SSH Tunnel Setup

### 1. Upload SSH Key (Requires Nostr Authentication)

```bash
# First, get a Nostr challenge
curl http://localhost:8082/api/v1/nostr/challenge

# Then authenticate with your Nostr key
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{
       "event": {
         "id": "your_event_id",
         "pubkey": "your_pubkey",
         "created_at": 1700000000,
         "kind": 22242,
         "tags": [["challenge", "challenge_from_previous_step"]],
         "content": "",
         "sig": "your_signature"
       }
     }' \
     http://localhost:8082/api/v1/nostr/auth

# Upload your SSH key
curl -X POST \
     -H "Authorization: Nostr <your_auth_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "my-tunnel-key",
       "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
       "public_key": "ssh-rsa AAAAB3NzaC1yc2E...",
       "key_type": "rsa",
       "description": "Tunnel key for remote access"
     }' \
     http://localhost:8082/api/v1/ssh-keys
```

### 2. Use SSH Tunnel (Standard SSH Authentication)

```bash
# Once SSH key is uploaded, use standard SSH authentication
ssh -i /path/to/your/private/key user@localhost -p 2222

# Or use the tunnel for port forwarding
ssh -L 8080:localhost:8080 -i /path/to/your/private/key user@localhost -p 2222
```

## Kind-Based Filtering

### View Kind Statistics

```bash
# Get all kind statistics (requires authentication)
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/stats
```

### Monitor Specific Kinds

```bash
# Get text notes (kind 1)
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/1/events?limit=10

# Get reactions (kind 7)
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/7/events?limit=10

# Check moderation queue
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/-2/stats
```

## Adding New Event Kinds

### 1. Create Kind Configuration

```bash
# Create a new kind file (example: kind 1337 for code snippets)
cat > configs/kinds/1337.yml << 'EOF'
# Kind 1337: Code Snippet
name: "Code Snippet"
description: "Code snippets with syntax highlighting"
required_tags: ["language"]
optional_tags: ["title", "description"]
content_validation:
  type: "text"
  max_length: 50000
  min_length: 10
quality_rules:
  - name: "valid_syntax"
    weight: 0.8
    description: "Code should have valid syntax"
  - name: "not_spam"
    weight: 1.0
    description: "Content should not be spam"
replaceable: false
ephemeral: false
EOF
```

### 2. Restart Relay

```bash
# Restart to pick up new kind
docker-compose restart mercury-relay

# Verify new kind is loaded
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/stats | grep "kind_1337"
```

### 3. Test New Kind

```bash
# Publish a code snippet event
curl -X POST \
     -H "Authorization: Nostr <your_auth_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "id": "code_event_id",
       "pubkey": "your_pubkey",
       "created_at": 1700000000,
       "kind": 1337,
       "tags": [["language", "javascript"]],
       "content": "console.log(\"Hello, World!\");",
       "sig": "your_signature"
     }' \
     http://localhost:8082/api/v1/publish

# Check if it appears in kind 1337 topic
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/1337/events
```

## Configuration

### Basic Configuration

Edit `config.local.yaml` for local development:

```yaml
# Server Configuration
server:
  host: "0.0.0.0"
  port: 8080
  websocket_port: 8080
  grpc_port: 8081
  rest_port: 8082

# Authentication (replace with your keys)
auth:
  admin_npubs: ["npub1your_admin_key..."]
  private_key: "nsec1your_private_key..."

# RabbitMQ (required for kind filtering)
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  exchange_name: "nostr_events"
  queue_name: "event_queue"
  dlx_name: "nostr_dlx"
  ttl: "28h"

# Quality Control
quality:
  enabled: true
  spam_threshold: 0.3
  rate_limit_per_minute: 10
```

### Environment Variables

```bash
# Set your admin keys
export MERCURY_ADMIN_NPUBS="npub1admin1...,npub1admin2..."
export MERCURY_PRIVATE_KEY="nsec1your_private_key..."

# Optional: Override other settings
export MERCURY_HOST="0.0.0.0"
export MERCURY_PORT="8080"
export MERCURY_QUALITY_ENABLED="true"
```

## Testing the System

### 1. Publish Test Events

```bash
# Publish a text note (kind 1)
curl -X POST \
     -H "Authorization: Nostr <your_auth_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "id": "test_note_id",
       "pubkey": "your_pubkey",
       "created_at": 1700000000,
       "kind": 1,
       "tags": [],
       "content": "Hello, Mercury Relay!",
       "sig": "your_signature"
     }' \
     http://localhost:8082/api/v1/publish

# Publish a reaction (kind 7)
curl -X POST \
     -H "Authorization: Nostr <your_auth_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "id": "test_reaction_id",
       "pubkey": "your_pubkey",
       "created_at": 1700000000,
       "kind": 7,
       "tags": [["e", "test_note_id"]],
       "content": "+",
       "sig": "your_signature"
     }' \
     http://localhost:8082/api/v1/publish
```

### 2. Verify Event Routing

```bash
# Check text notes topic
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/1/stats

# Check reactions topic
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/7/stats

# Check all topics
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/stats
```

### 3. Test Invalid Event Routing

```bash
# Publish an invalid event (missing required fields)
curl -X POST \
     -H "Authorization: Nostr <your_auth_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "id": "",
       "pubkey": "your_pubkey",
       "created_at": 1700000000,
       "kind": 1,
       "tags": [],
       "content": "Invalid event",
       "sig": "your_signature"
     }' \
     http://localhost:8082/api/v1/publish

# Check moderation queue
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/-2/stats
```

## Monitoring

### RabbitMQ Management UI

1. Open `http://localhost:15672`
2. Login with `guest/guest`
3. Navigate to "Queues" tab
4. Look for queues starting with `nostr_kind_`

### Logs

```bash
# View relay logs
docker-compose logs -f mercury-relay

# Filter for kind-related logs
docker-compose logs mercury-relay | grep "kind"

# Monitor API requests
docker-compose logs mercury-relay | grep "api/v1"
```

### Health Checks

```bash
# Basic health check
curl http://localhost:8082/api/v1/health

# Detailed statistics
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/stats
```

## Common Tasks

### Add a New Event Kind

1. **Create configuration file**:
```bash
# Example: Adding support for kind 30023 (long-form content)
cat > configs/kinds/30023.yml << 'EOF'
# Kind 30023: Long-form Content
name: "Long-form Content"
description: "Long-form articles and posts"
required_tags: ["d"]
optional_tags: ["title", "summary", "image"]
content_validation:
  type: "text"
  max_length: 100000
  min_length: 100
quality_rules:
  - name: "substantial_content"
    weight: 0.8
    description: "Content should be substantial for long-form"
replaceable: true
ephemeral: false
EOF
```

2. **Restart relay**:
```bash
docker-compose restart mercury-relay
```

3. **Verify new kind**:
```bash
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/stats | grep "kind_30023"
```

### Monitor Moderation Queue

```bash
# Check moderation queue size
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/-2/stats

# Get events in moderation queue
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/-2/events
```

### Clean Up Old Events

```bash
# Check queue depths
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/stats

# If queues are too deep, restart to clear
docker-compose restart mercury-relay
```

## Troubleshooting

### Common Issues

**1. Kind not detected after adding file**:
```bash
# Check file naming and syntax
ls -la configs/kinds/
yamllint configs/kinds/*.yml

# Restart relay
docker-compose restart mercury-relay
```

**2. Authentication errors**:
```bash
# Check your auth token format
echo $MERCURY_ADMIN_NPUBS
echo $MERCURY_PRIVATE_KEY

# Verify token is base64 encoded
echo "your_token" | base64 -d
```

**3. RabbitMQ connection issues**:
```bash
# Check RabbitMQ status
docker-compose logs rabbitmq

# Test connection
docker-compose exec mercury-relay curl http://rabbitmq:15672
```

**4. High memory usage**:
```bash
# Check queue depths
curl -H "Authorization: Nostr <your_auth_token>" \
     http://localhost:8082/api/v1/kind/stats

# Restart to clear queues
docker-compose restart mercury-relay
```

### Debug Mode

```bash
# Enable debug logging
export MERCURY_LOG_LEVEL=debug
docker-compose up -d

# View debug logs
docker-compose logs -f mercury-relay
```

### Reset Everything

```bash
# Stop and remove all containers and volumes
docker-compose down -v --remove-orphans

# Remove all data
docker system prune -a --volumes

# Start fresh
docker-compose up -d
```

## Next Steps

1. **Read the full documentation**: [Configuration Guide](configuration.md)
2. **Explore the API**: [API Documentation](api.md)
3. **Learn about kind filtering**: [Kind-Based Filtering](kind-based-filtering.md)
4. **Set up monitoring**: Configure alerts and metrics
5. **Scale for production**: Use load balancers and multiple instances

## Support

- **Documentation**: Check the `docs/` directory
- **Issues**: Report issues on GitHub
- **Community**: Join our Discord/Matrix channels
- **Email**: support@mercury-relay.com

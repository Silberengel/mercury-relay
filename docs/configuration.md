# Mercury Relay Configuration Guide

This guide covers all aspects of configuring Mercury Relay, including the new dynamic kind-based filtering system.

## Overview

Mercury Relay uses a hierarchical configuration system:

1. **Default values** (hardcoded)
2. **YAML configuration files** (main config)
3. **Environment variables** (overrides)
4. **Command-line flags** (runtime overrides)

## Main Configuration

### Configuration File Location

The main configuration file is `config.local.yaml` for local development or `config.yaml` for production.

### Basic Configuration Structure

```yaml
# Server Configuration
server:
  host: "0.0.0.0"
  port: 8080
  websocket_port: 8080
  grpc_port: 8081
  rest_port: 8082

# Authentication
auth:
  admin_npubs: ["npub1admin1...", "npub1admin2..."]
  private_key: "nsec1private..."

# Database
database:
  url: "postgres://user:pass@localhost:5432/mercury"
  max_connections: 10

# Cache
cache:
  url: "redis://localhost:6379"
  password: ""
  db: 0

# RabbitMQ (Required for kind-based filtering)
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

# Upstream Relays
upstream:
  relays: ["wss://relay1.example.com", "wss://relay2.example.com"]
  timeout: "30s"
```

## Kind-Based Filtering Configuration

### Individual Kind Files

Each event kind is configured in its own YAML file in the `configs/kinds/` directory.

#### File Naming Convention

- **Format**: `{kind_number}.yml`
- **Examples**: `0.yml`, `1.yml`, `7.yml`, `10002.yml`

#### Configuration Schema

```yaml
# Basic Information
name: "Human-readable name"
description: "Description of the event kind"

# Tag Requirements
required_tags: ["tag1", "tag2"]  # Required tags
optional_tags: ["tag3", "tag4"]  # Optional tags

# Content Validation
content_validation:
  type: "text|json|empty"        # Content type
  max_length: 10000              # Maximum length
  min_length: 1                  # Minimum length
  required_fields: ["field1"]    # Required JSON fields
  optional_fields: ["field2"]   # Optional JSON fields

# Quality Rules
quality_rules:
  - name: "rule_name"
    weight: 1.0                  # Rule weight (0.0-1.0)
    description: "Rule description"

# Event Properties
replaceable: true|false          # Replaceable events
ephemeral: true|false            # Ephemeral events
```

### Example Kind Configurations

#### Kind 0: User Metadata
**File**: `configs/kinds/0.yml`
```yaml
# Kind 0: User Metadata
name: "User Metadata"
description: "User profile information"
required_tags: []
optional_tags: ["d"]
content_validation:
  type: "json"
  required_fields: ["name"]
  optional_fields: ["about", "picture", "banner", "website", "lud16", "nip05"]
  max_length: 1000
quality_rules:
  - name: "valid_json"
    weight: 1.0
    description: "Content must be valid JSON"
  - name: "has_name"
    weight: 0.8
    description: "Must have a name field"
  - name: "reasonable_length"
    weight: 0.6
    description: "Content length should be reasonable"
replaceable: true
ephemeral: false
```

#### Kind 1: Text Notes
**File**: `configs/kinds/1.yml`
```yaml
# Kind 1: Text Note
name: "Text Note"
description: "Regular social media post"
required_tags: []
optional_tags: ["e", "p", "a", "t", "d", "r", "g", "alt"]
content_validation:
  type: "text"
  max_length: 10000
  min_length: 1
quality_rules:
  - name: "not_spam"
    weight: 1.0
    description: "Content should not be spam"
  - name: "meaningful_content"
    weight: 0.8
    description: "Content should have meaningful text"
  - name: "proper_encoding"
    weight: 0.9
    description: "Content should be properly encoded"
replaceable: false
ephemeral: false
```

#### Kind 7: Reactions
**File**: `configs/kinds/7.yml`
```yaml
# Kind 7: Reaction
name: "Reaction"
description: "User reaction to events"
required_tags: ["e"]
optional_tags: ["p", "a", "k", "emoji"]
content_validation:
  type: "text"
  max_length: 100
  min_length: 0
quality_rules:
  - name: "valid_e_tag"
    weight: 1.0
    description: "Must reference a valid event with e tag"
  - name: "reasonable_reaction"
    weight: 0.8
    description: "Reaction content should be reasonable"
replaceable: false
ephemeral: false
```

## Environment Variables

### Core Configuration

```bash
# Server
export MERCURY_HOST="0.0.0.0"
export MERCURY_PORT="8080"
export MERCURY_WEBSOCKET_PORT="8080"
export MERCURY_GRPC_PORT="8081"
export MERCURY_REST_PORT="8082"

# Authentication
export MERCURY_ADMIN_NPUBS="npub1admin1...,npub1admin2..."
export MERCURY_PRIVATE_KEY="nsec1private..."

# Database
export MERCURY_DATABASE_URL="postgres://user:pass@localhost:5432/mercury"
export MERCURY_DATABASE_MAX_CONNECTIONS="10"

# Cache
export MERCURY_CACHE_URL="redis://localhost:6379"
export MERCURY_CACHE_PASSWORD=""
export MERCURY_CACHE_DB="0"

# RabbitMQ
export MERCURY_RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export MERCURY_RABBITMQ_EXCHANGE_NAME="nostr_events"
export MERCURY_RABBITMQ_QUEUE_NAME="event_queue"
export MERCURY_RABBITMQ_DLX_NAME="nostr_dlx"
export MERCURY_RABBITMQ_TTL="28h"

# Quality Control
export MERCURY_QUALITY_ENABLED="true"
export MERCURY_QUALITY_SPAM_THRESHOLD="0.3"
export MERCURY_QUALITY_RATE_LIMIT_PER_MINUTE="10"
```

### Kind-Based Filtering

```bash
# Kind configuration directory
export MERCURY_KINDS_DIR="configs/kinds"

# Enable/disable kind-based filtering
export MERCURY_KIND_FILTERING_ENABLED="true"

# Moderation queue settings
export MERCURY_MODERATION_ENABLED="true"
export MERCURY_MODERATION_QUEUE_NAME="nostr_kind_moderation"
```

## Docker Configuration

### Docker Compose

**File**: `docker/docker-compose.yml`
```yaml
version: '3.8'

services:
  mercury-relay:
    build: ..
    ports:
      - "8080:8080"  # WebSocket
      - "8081:8081"  # gRPC
      - "8082:8082"  # REST API
    environment:
      - MERCURY_HOST=0.0.0.0
      - MERCURY_PORT=8080
      - MERCURY_WEBSOCKET_PORT=8080
      - MERCURY_GRPC_PORT=8081
      - MERCURY_REST_PORT=8082
      - MERCURY_DATABASE_URL=postgres://mercury:mercury@postgres:5432/mercury
      - MERCURY_CACHE_URL=redis://redis:6379
      - MERCURY_RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
    volumes:
      - ../configs:/app/configs:ro
      - ../config.local.yaml:/app/config.local.yaml:ro
    depends_on:
      - postgres
      - redis
      - rabbitmq

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: mercury
      POSTGRES_USER: mercury
      POSTGRES_PASSWORD: mercury
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  rabbitmq:
    image: rabbitmq:3-management-alpine
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    ports:
      - "15672:15672"  # Management UI
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

volumes:
  postgres_data:
  redis_data:
  rabbitmq_data:
```

### Docker Environment Variables

```bash
# Build arguments
export DOCKER_BUILDKIT=1

# Runtime environment
export MERCURY_ENV=production
export MERCURY_LOG_LEVEL=info
export MERCURY_CONFIG_FILE=config.yaml
```

## Production Configuration

### Security Settings

```yaml
# Production security configuration
server:
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/mercury.crt"
    key_file: "/etc/ssl/private/mercury.key"
  
auth:
  # Use strong, unique keys in production
  admin_npubs: ["npub1production_admin..."]
  private_key: "nsec1production_private..."
  
  # Enable additional security features
  require_auth: true
  auth_timeout: "1h"
  max_auth_attempts: 3

# Database security
database:
  ssl_mode: "require"
  max_connections: 50
  connection_timeout: "30s"
  
# Cache security
cache:
  password: "strong_redis_password"
  tls_enabled: true
```

### Performance Tuning

```yaml
# High-performance configuration
server:
  # Connection limits
  max_connections: 10000
  read_timeout: "60s"
  write_timeout: "60s"
  
  # Worker pools
  worker_pool_size: 100
  queue_size: 1000

# Database optimization
database:
  max_connections: 100
  max_idle_connections: 10
  connection_max_lifetime: "1h"
  
# Cache optimization
cache:
  max_connections: 100
  idle_timeout: "5m"
  read_timeout: "3s"
  write_timeout: "3s"

# RabbitMQ optimization
rabbitmq:
  # Connection pooling
  max_connections: 10
  max_channels: 100
  
  # Queue settings
  queue_ttl: "24h"
  message_ttl: "1h"
  
  # Performance tuning
  prefetch_count: 100
  consumer_timeout: "30s"
```

## Monitoring Configuration

### Logging

```yaml
# Logging configuration
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, file
  
  # File logging (if output: file)
  file:
    path: "/var/log/mercury/relay.log"
    max_size: "100MB"
    max_backups: 3
    max_age: "7d"
    compress: true

# Component-specific logging
components:
  websocket:
    log_level: "info"
  grpc:
    log_level: "warn"
  rest:
    log_level: "info"
  quality:
    log_level: "debug"
  rabbitmq:
    log_level: "warn"
```

### Metrics

```yaml
# Metrics configuration
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
  
  # Custom metrics
  custom_metrics:
    - name: "events_processed_total"
      type: "counter"
      help: "Total number of events processed"
    - name: "events_by_kind_total"
      type: "counter"
      help: "Total number of events by kind"
      labels: ["kind"]
    - name: "moderation_queue_size"
      type: "gauge"
      help: "Number of events in moderation queue"
```

## Troubleshooting

### Common Configuration Issues

**1. Kind files not loading:**
```bash
# Check file permissions
ls -la configs/kinds/

# Validate YAML syntax
yamllint configs/kinds/*.yml

# Check file naming
ls configs/kinds/ | grep -E '^[0-9]+\.yml$'
```

**2. RabbitMQ connection issues:**
```bash
# Test RabbitMQ connection
rabbitmqctl status

# Check queue creation
rabbitmqctl list_queues | grep nostr_kind

# Verify exchange creation
rabbitmqctl list_exchanges | grep nostr
```

**3. Database connection issues:**
```bash
# Test database connection
psql $MERCURY_DATABASE_URL -c "SELECT 1;"

# Check connection limits
psql $MERCURY_DATABASE_URL -c "SHOW max_connections;"
```

### Configuration Validation

```bash
# Validate main configuration
./mercury-relay --config-validate

# Test configuration loading
./mercury-relay --config-dump

# Check environment variables
./mercury-relay --env-dump
```

### Debug Mode

```bash
# Enable debug logging
export MERCURY_LOG_LEVEL=debug

# Enable component debugging
export MERCURY_DEBUG_WEBSOCKET=true
export MERCURY_DEBUG_GRPC=true
export MERCURY_DEBUG_REST=true
export MERCURY_DEBUG_QUALITY=true
export MERCURY_DEBUG_RABBITMQ=true

# Run with debug flags
./mercury-relay --debug --verbose
```

## Best Practices

### Configuration Management

1. **Version Control**: Keep all configuration files in version control
2. **Environment Separation**: Use different configs for dev/staging/prod
3. **Secrets Management**: Use environment variables for sensitive data
4. **Documentation**: Document all custom configurations
5. **Testing**: Test configurations in staging before production

### Security

1. **Access Control**: Limit admin access to trusted npubs
2. **Network Security**: Use TLS in production
3. **Database Security**: Use strong passwords and SSL
4. **Monitoring**: Set up alerts for configuration changes
5. **Backup**: Regularly backup configuration files

### Performance

1. **Resource Limits**: Set appropriate connection limits
2. **Monitoring**: Monitor resource usage and adjust accordingly
3. **Caching**: Use appropriate cache settings for your workload
4. **Queue Management**: Monitor queue depths and adjust TTL settings
5. **Scaling**: Plan for horizontal scaling with load balancers
# Configuration Guide

Mercury Relay uses YAML configuration files with environment variable overrides for maximum flexibility.

## Configuration File

The main configuration file is `config.yaml`. Here's a complete example:

```yaml
# Server Configuration
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

# Access Control
access:
  admin_npubs:
    - "npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j"
    - "npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx"
    - "npub1l5sga6xg72phsz5422ykujprejwud075ggrr3z2hwyrfgr7eylqstegx9z"
    - "npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl"
  update_interval: "1h"
  relay_url: "https://mercury-relay.imwald.eu"
  allow_public_read: true
  allow_public_write: false

# Quality Control
quality:
  enabled: true
  max_content_length: 10000
  spam_threshold: 0.7
  quarantine_suspicious: true

# SSH Configuration
ssh:
  enabled: false
  key_storage:
    key_dir: "./ssh-keys"
    private_key_ext: ".pem"
    public_key_ext: ".pub"
    key_size: 2048
    key_type: "rsa"
  connection:
    host: "localhost"
    port: 22
    username: "mercury"
    timeout: "30s"
    keep_alive: "30s"
    max_retries: 3
    retry_delay: "5s"
    compression: false
    banner: "Mercury Relay SSH Server"
  terminal_interface:
    enabled: true
    port: 2222
    host: "localhost"
    interactive: true
    log_level: "info"
  authentication:
    require_auth: true
    api_key: "admin-ssh-key-2024"
    basic_auth_user: "admin"
    basic_auth_pass: "mercury-ssh-2024"
    authorized_pubkeys: []
    allow_localhost: true

# Streaming Configuration
streaming:
  enabled: true
  upstream_relays:
    - url: "wss://theforest.nostr1.com"
      enabled: true
      priority: 1
    - url: "wss://orly-relay.imwald.eu"
      enabled: true
      priority: 2
    - url: "wss://nostr.land"
      enabled: true
      priority: 3
    - url: "wss://nostr21.com"
      enabled: true
      priority: 4
  transport_methods:
    websocket: true
    grpc: true
    tor: false
    i2p: false
    ssh: false

# gRPC Configuration
grpc:
  enabled: true
  host: "localhost"
  port: 9090
  tls_enabled: false
  cert_file: ""
  key_file: ""
  compression: true
  keep_alive: "30s"

# Tor Configuration
tor:
  enabled: false
  data_dir: "/var/lib/tor"
  control_port: 9051
  socks_port: 9050
  hidden_service_dir: "/var/lib/tor/mercury_relay"
  hidden_service_port: 80

# I2P Configuration
i2p:
  enabled: false
  sam_port: 7656
  sam_host: "127.0.0.1"
  tunnel_name: "mercury_relay"
  tunnel_port: 8080

# Database Configuration
database:
  type: "sqlite"
  path: "/app/data/mercury.db"
  # For PostgreSQL:
  # type: "postgres"
  # host: "postgres"
  # port: 5432
  # name: "mercury_relay"
  # user: "mercury"
  # password: "change_this_password"

# Redis Configuration
redis:
  enabled: false
  host: "redis"
  port: 6379
  password: ""
  db: 0
  max_retries: 3
  pool_size: 10
  min_idle_conns: 5
  dial_timeout: "5s"
  read_timeout: "3s"
  write_timeout: "3s"
  idle_timeout: "300s"
  idle_check_frequency: "60s"

# RabbitMQ Configuration
rabbitmq:
  enabled: false
  url: "amqp://guest:guest@localhost:5672/"
  exchange_name: "events"
  queue_name: "events_queue"
  dlx_name: "events_dlx"
  routing_key: "events"
  durable: true
  auto_delete: false
  internal: false
  no_wait: false
  exclusive: false
  auto_ack: false
  prefetch_count: 1
  prefetch_size: 0
  prefetch_global: false

# REST API Configuration
rest_api:
  enabled: true
  host: "0.0.0.0"
  port: 8082
  cors_enabled: true
  cors_origins: "*"
  rate_limit_enabled: true
  rate_limit_per_minute: 100
  rate_limit_burst_size: 20

# Admin Interface
admin:
  enabled: true
  host: "0.0.0.0"
  port: 8081
  cors_enabled: true
  cors_origins: "*"

# Logging Configuration
logging:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: ""
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true

# Monitoring Configuration
monitoring:
  enabled: true
  metrics_enabled: true
  health_check_interval: "30s"
  stats_interval: "60s"
  log_level: "info"
```

## Environment Variables

All configuration values can be overridden using environment variables:

### Server Configuration
- `NOSTR_RELAY_PORT`: WebSocket port (default: 8080)
- `ADMIN_PORT`: Admin API port (default: 8081)
- `REST_API_PORT`: REST API port (default: 8082)
- `LOG_LEVEL`: Log level (default: info)

### Access Control
- `MERCURY_ADMIN_NPUBS`: Comma-separated list of admin npubs (supports both hex and bech32 formats)
- `MERCURY_PRIVATE_KEY`: Private key for authentication (supports both hex and bech32 formats)
- `ACCESS_RELAY_URL`: Relay URL for follow list updates (default: https://mercury-relay.imwald.eu)
- `ACCESS_PUBLIC_READ`: Allow public read access (default: true)
- `ACCESS_PUBLIC_WRITE`: Allow public write access (default: false)

### SSH Configuration
- `SSH_ENABLED`: Enable SSH transport (default: false)
- `SSH_HOST`: SSH server host (default: localhost)
- `SSH_PORT`: SSH server port (default: 22)
- `SSH_USERNAME`: SSH username (default: mercury)
- `SSH_KEY_DIR`: SSH key storage directory (default: ./ssh-keys)
- `SSH_KEY_TYPE`: SSH key type (default: rsa)
- `SSH_KEY_SIZE`: SSH key size (default: 2048)
- `SSH_TIMEOUT`: SSH connection timeout (default: 30s)
- `SSH_TERMINAL_ENABLED`: Enable SSH terminal interface (default: true)
- `SSH_TERMINAL_PORT`: SSH terminal port (default: 2222)

### Streaming Configuration
- `STREAMING_ENABLED`: Enable streaming (default: true)
- `UPSTREAM_RELAYS`: Comma-separated list of upstream relays

### Transport Configuration
- `TOR_ENABLED`: Enable Tor transport (default: false)
- `I2P_ENABLED`: Enable I2P transport (default: false)
- `GRPC_ENABLED`: Enable gRPC transport (default: true)

### Database Configuration
- `DB_TYPE`: Database type (sqlite/postgres) (default: sqlite)
- `DB_PATH`: SQLite database path (default: /app/data/mercury.db)
- `DB_HOST`: PostgreSQL host (default: postgres)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_NAME`: Database name (default: mercury_relay)
- `DB_USER`: Database user (default: mercury)
- `DB_PASSWORD`: Database password (default: change_this_password)

### Redis Configuration
- `REDIS_ENABLED`: Enable Redis (default: false)
- `REDIS_HOST`: Redis host (default: redis)
- `REDIS_PORT`: Redis port (default: 6379)
- `REDIS_PASSWORD`: Redis password (default: "")

### RabbitMQ Configuration
- `RABBITMQ_ENABLED`: Enable RabbitMQ (default: false)
- `RABBITMQ_URL`: RabbitMQ connection URL (default: amqp://guest:guest@localhost:5672/)

## Key Formats

Mercury Relay supports both hex and bech32 formats for all Nostr keys:

### Supported Formats

- **Public Keys**: 
  - Hex: `a1b2c3d4e5f6...` (64 characters)
  - Bech32: `npub1...` (starts with npub1)
- **Private Keys**:
  - Hex: `a1b2c3d4e5f6...` (64 characters) 
  - Bech32: `nsec1...` (starts with nsec1)

### Environment Variables

All key-related environment variables accept both formats:

```bash
# Admin npubs (comma-separated)
MERCURY_ADMIN_NPUBS="npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j,npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx"

# Private key for authentication
MERCURY_PRIVATE_KEY="nsec1your-private-key-here"

# SSH configuration
SSH_ENABLED=true
SSH_HOST=localhost
SSH_PORT=22
SSH_USERNAME=mercury
SSH_KEY_DIR=./ssh-keys
SSH_KEY_TYPE=rsa
SSH_KEY_SIZE=2048
SSH_TIMEOUT=30s
SSH_TERMINAL_ENABLED=true
SSH_TERMINAL_PORT=2222
```

The system automatically detects the format and converts to hex internally for processing.

## Configuration Validation

The relay validates configuration on startup. Common validation errors:

- **Invalid npub format**: Ensure admin npubs are valid Nostr public keys
- **Port conflicts**: Ensure ports don't conflict with other services
- **Invalid URLs**: Check that relay URLs are properly formatted
- **Missing dependencies**: Ensure required services (Redis, PostgreSQL) are running

## Security Considerations

- **Admin npubs**: Use secure, well-known npubs for administrators
- **Database passwords**: Use strong, unique passwords for database connections
- **API keys**: Generate secure API keys for authentication
- **SSH keys**: Store SSH keys securely and use proper permissions
- **Network access**: Configure firewall rules appropriately
- **TLS certificates**: Use valid TLS certificates for production deployments

## Production Recommendations

1. **Use environment variables** for sensitive configuration
2. **Enable TLS** for all external connections
3. **Use strong passwords** for all services
4. **Configure proper logging** for monitoring and debugging
5. **Set up monitoring** for health checks and metrics
6. **Use external databases** (PostgreSQL) for production
7. **Configure backup strategies** for data persistence
8. **Use reverse proxies** (Apache/Nginx) for production deployments

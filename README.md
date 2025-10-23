# Mercury Relay

A high-performance, feature-rich Nostr relay built in Go with advanced authentication, quality control, and multi-transport support.

## Features

- **Multi-Transport Support**: WebSocket, gRPC, Tor, I2P, and SSH tunneling
- **Advanced Authentication**: Nostr NIP-42 authentication with multi-admin support
- **Dynamic Kind-Based Filtering**: Automatic routing of events by kind with quality control
- **Quality Control**: Spam detection, content filtering, and rate limiting
- **Modular Configuration**: Individual YAML files for each event kind
- **SSH Key Management**: Secure SSH key storage and management with Nostr authentication
- **Streaming**: Real-time event streaming with upstream relay support
- **Admin Interface**: Web-based administration with Nostr authentication
- **Docker Support**: Full containerization with Docker Compose

## Quick Start

### Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/your-org/mercury-relay.git
cd mercury-relay

# Start with Docker Compose
cd docker && docker-compose up -d

# The relay will be available at:
# - WebSocket: ws://localhost:8080
# - Admin API: http://localhost:8081
# - REST API: http://localhost:8082
```

### Manual Installation

```bash
# Build the relay
go build -o mercury-relay ./cmd/mercury-relay

# Run with default configuration
./mercury-relay
```

## Kind-Based Event Filtering

Mercury Relay features a dynamic kind-based filtering system that automatically routes events to appropriate topics based on their kind and quality:

### Event Routing Flow

1. **Event Validation**: All events are first validated for basic structure and content
2. **Invalid Events** → `moderation` topic (for manual review)
3. **Valid Known Kinds** → `kind.{number}` topic (e.g., `kind.1`, `kind.7`)
4. **Valid Unknown Kinds** → `undefined` topic

### Supported Event Kinds

The system dynamically loads event kinds from individual YAML configuration files in `configs/kinds/`:

- **Kind 0**: User metadata (`configs/kinds/0.yml`)
- **Kind 1**: Text notes (`configs/kinds/1.yml`)
- **Kind 3**: Follow lists (`configs/kinds/3.yml`)
- **Kind 7**: Reactions (`configs/kinds/7.yml`)
- **Kind 9**: Chat messages (`configs/kinds/9.yml`)
- **Kind 30**: Internal citations (`configs/kinds/30.yml`)
- **Kind 31**: External web citations (`configs/kinds/31.yml`)
- **Kind 10002**: Relay lists (`configs/kinds/10002.yml`)
- **Kind 30023**: Long-form content (`configs/kinds/30023.yml`)
- **Kind 30040**: Publication index (`configs/kinds/30040.yml`)
- **Kind 30041**: Publication content (`configs/kinds/30041.yml`)
- **Kind 30042**: Drive events (`configs/kinds/30042.yml`)
- **Kind 30043**: Traceback events (`configs/kinds/30043.yml`)

### Adding New Event Kinds

To add support for a new event kind:

1. Create a new YAML file: `configs/kinds/{kind_number}.yml`
2. Define the kind's structure and validation rules
3. Restart the relay - the new kind will be automatically detected

Example: `configs/kinds/1337.yml` for code snippets:

```yaml
# Kind 1337: Code Snippet
name: "Code Snippet"
description: "Code snippets with syntax highlighting"
required_tags: ["language"]
optional_tags: ["title", "description"]
content_validation:
  type: "text"
  max_length: 10000
  min_length: 1
quality_rules:
  - name: "valid_syntax"
    weight: 0.8
    description: "Code should have valid syntax"
replaceable: false
ephemeral: false
```

## Documentation

- **[Quick Start Guide](docs/quick-start.md)** - Get up and running in minutes
- **[Configuration Guide](docs/configuration.md)** - Complete configuration reference
- **[API Documentation](docs/api.md)** - REST API endpoints and examples
- **[Kind-Based Filtering](docs/kind-based-filtering.md)** - Dynamic event filtering system

## Configuration

The relay uses YAML configuration with environment variable overrides. See the [Configuration Guide](docs/configuration.md) for detailed setup instructions.

### Key Environment Variables

- `MERCURY_ADMIN_NPUBS`: Comma-separated list of admin npubs (supports hex/bech32)
- `MERCURY_PRIVATE_KEY`: Private key for authentication (supports hex/bech32)
- `NOSTR_RELAY_PORT`: WebSocket port (default: 8080)
- `ADMIN_PORT`: Admin API port (default: 8081)
- `REST_API_PORT`: REST API port (default: 8082)

## Documentation

- **[Configuration Guide](docs/configuration.md)** - Detailed configuration options
- **[Docker Setup](docs/docker.md)** - Docker deployment and configuration
- **[Environment Variables](docs/environment.md)** - Environment variable reference
- **[SSH Authentication](docs/ssh-authentication.md)** - SSH key management with Nostr auth
- **[Streaming Setup](docs/streaming.md)** - Upstream relay configuration
- **[Apache Setup](docs/apache.md)** - Apache reverse proxy configuration
- **[Directory Structure](docs/directory-structure.md)** - Project organization

## API Endpoints

- **WebSocket**: `ws://localhost:8080` - Nostr protocol
- **Admin API**: `http://localhost:8081` - Administration interface
- **REST API**: `http://localhost:8082` - REST endpoints
- **SSH Keys**: `http://localhost:8082/ssh-keys` - SSH key management

## Authentication

Mercury Relay uses Nostr NIP-42 authentication for all administrative functions:

- **Admin Access**: Configured via `MERCURY_ADMIN_NPUBS`
- **SSH Key Management**: Requires Nostr authentication
- **API Access**: All endpoints require authentication
- **Follow-based Access**: Admins can grant access by following users

## Development

```bash
# Run tests
go test ./...

# Run with development configuration
go run ./cmd/mercury-relay -config config.yaml

# Build for production
go build -ldflags="-s -w" -o mercury-relay ./cmd/mercury-relay
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## Support

- **Issues**: [GitHub Issues](https://github.com/your-org/mercury-relay/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/mercury-relay/discussions)
- **Documentation**: [docs/](docs/) folder
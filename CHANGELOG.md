# Changelog

All notable changes to Mercury Relay will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2024-01-15

### Added

#### Dynamic Kind-Based Filtering System
- **Modular Kind Configuration**: Individual YAML files for each event kind in `configs/kinds/`
- **Automatic Kind Discovery**: System automatically detects new kinds from configuration files
- **Quality Control Filtering**: Invalid events are routed to moderation queue for review
- **Topic-Based Routing**: Events are automatically routed to appropriate kind-specific topics

#### New API Endpoints
- `GET /api/v1/kind/{kind}/events` - Retrieve events from specific kind topics
- `GET /api/v1/kind/{kind}/stats` - Get statistics for specific kind topics  
- `GET /api/v1/kind/stats` - Get statistics for all kind topics

#### Supported Event Kinds
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

#### Special Topics
- **`nostr_kind_undefined`**: Valid events with unknown kinds
- **`nostr_kind_moderation`**: Invalid events requiring review

#### Configuration System
- **Individual Kind Files**: Each kind has its own YAML configuration file
- **Dynamic Loading**: System automatically loads kinds from configuration files
- **Validation Rules**: Configurable content validation and quality rules per kind
- **Tag Requirements**: Define required and optional tags for each kind

#### Documentation
- **[Quick Start Guide](docs/quick-start.md)** - Get up and running in minutes
- **[Configuration Guide](docs/configuration.md)** - Complete configuration reference
- **[API Documentation](docs/api.md)** - REST API endpoints and examples
- **[Kind-Based Filtering](docs/kind-based-filtering.md)** - Dynamic event filtering system

### Changed

#### RabbitMQ Integration
- **Enhanced Routing**: Events are now routed through quality control before topic assignment
- **Dynamic Topic Creation**: Topics are created dynamically based on configured kinds
- **Moderation Queue**: Invalid events are automatically routed to moderation queue

#### API Improvements
- **Health Endpoint**: `/api/v1/health` no longer requires authentication
- **Enhanced Statistics**: Kind-specific statistics with descriptions
- **Better Error Handling**: Improved error messages and status codes

#### Configuration
- **Modular Structure**: Replaced monolithic configuration with individual kind files
- **Environment Variables**: Added new environment variables for kind filtering
- **Docker Support**: Enhanced Docker configuration for kind-based filtering

### Technical Details

#### Event Flow
1. **Event Validation**: All events undergo basic validation (required fields, timestamps, etc.)
2. **Quality Control**: Invalid events → `moderation` topic
3. **Kind Routing**: Valid events → appropriate kind topic or `undefined` topic
4. **Dynamic Discovery**: New kinds are automatically detected from configuration files

#### Performance Improvements
- **Efficient Routing**: Optimized event routing with minimal overhead
- **Memory Management**: Better memory usage with topic-based organization
- **Scalability**: System automatically scales with new event kinds

#### Security Enhancements
- **Input Validation**: Enhanced validation for all event types
- **Quality Control**: Automatic filtering of invalid events
- **Moderation Support**: Built-in moderation queue for manual review

### Migration Guide

#### For Existing Installations

1. **Backup Configuration**: Backup your existing `config.local.yaml`
2. **Update Configuration**: Add RabbitMQ configuration if not present
3. **Create Kind Files**: Create individual kind configuration files in `configs/kinds/`
4. **Restart Services**: Restart Mercury Relay to pick up new configuration

#### Configuration Migration

**Before** (monolithic):
```yaml
# config.local.yaml
event_kinds:
  0: # User metadata
    name: "User Metadata"
    # ... configuration
  1: # Text note
    name: "Text Note"
    # ... configuration
```

**After** (modular):
```yaml
# config.local.yaml (main config)
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  # ... other settings

# configs/kinds/0.yml (individual kind)
name: "User Metadata"
description: "User profile information"
# ... kind-specific configuration

# configs/kinds/1.yml (individual kind)
name: "Text Note"
description: "Regular social media post"
# ... kind-specific configuration
```

### Breaking Changes

- **Configuration Structure**: Kind configurations are now in individual files
- **API Endpoints**: New authentication requirements for some endpoints
- **RabbitMQ**: RabbitMQ is now required for kind-based filtering

### Deprecations

- **Monolithic Configuration**: The old `event_kinds` section in main config is deprecated
- **Hardcoded Kinds**: Hardcoded kind lists in code are replaced with dynamic loading

## [1.0.0] - 2024-01-01

### Added

#### Core Features
- **Multi-Transport Support**: WebSocket, gRPC, Tor, I2P, and SSH tunneling
- **Advanced Authentication**: Nostr NIP-42 authentication with multi-admin support
- **Quality Control**: Spam detection, content filtering, and rate limiting
- **SSH Key Management**: Secure SSH key storage and management
- **Streaming**: Real-time event streaming with upstream relay support
- **Admin Interface**: Web-based administration with Nostr authentication
- **Docker Support**: Full containerization with Docker Compose

#### API Endpoints
- `GET /api/v1/health` - Health check endpoint
- `GET /api/v1/stats` - Relay statistics
- `GET /api/v1/events` - Query events
- `POST /api/v1/events` - Publish events
- `POST /api/v1/publish` - Publish single event
- `GET /api/v1/ebooks` - Get available ebooks
- `GET /api/v1/ebooks/{id}` - Get ebook content
- `GET /api/v1/ebooks/{id}/epub` - Generate EPUB

#### Configuration
- **YAML Configuration**: Main configuration in YAML format
- **Environment Variables**: Support for environment variable overrides
- **Docker Compose**: Complete Docker setup with all dependencies

#### Quality Control
- **Spam Detection**: Automatic spam detection and filtering
- **Rate Limiting**: Configurable rate limiting per user
- **Content Filtering**: Content-based filtering and validation
- **Blocking System**: User blocking and unblocking functionality

### Technical Details

#### Architecture
- **Go Implementation**: High-performance Go implementation
- **PostgreSQL**: Primary database for event storage
- **Redis**: Caching layer for improved performance
- **RabbitMQ**: Message queuing for event processing
- **Docker**: Full containerization support

#### Security
- **Nostr Authentication**: NIP-42 compliant authentication
- **Admin Controls**: Multi-admin support with proper access controls
- **Input Validation**: Comprehensive input validation and sanitization
- **Rate Limiting**: Protection against abuse and spam

#### Performance
- **High Throughput**: Optimized for high event throughput
- **Low Latency**: Minimal latency for real-time applications
- **Scalability**: Designed for horizontal scaling
- **Monitoring**: Built-in monitoring and statistics

### Documentation
- **README**: Comprehensive project documentation
- **Configuration Guide**: Detailed configuration instructions
- **API Documentation**: Complete API reference
- **Docker Guide**: Docker setup and deployment instructions

## Future Roadmap

### Planned Features

#### Enhanced Kind Support
- **Dynamic Reloading**: Hot-reload kind configurations without restart
- **Advanced Validation**: More sophisticated content validation rules
- **Analytics**: Detailed analytics for each kind topic
- **Auto-scaling**: Automatic topic scaling based on load

#### Integration Improvements
- **External Moderation**: Integration with external moderation tools
- **Webhook Support**: Webhook notifications for events
- **Metrics Export**: Prometheus metrics export
- **Logging**: Structured logging with correlation IDs

#### Performance Enhancements
- **Caching**: Advanced caching strategies
- **Compression**: Event compression for storage efficiency
- **Partitioning**: Event partitioning for better performance
- **Load Balancing**: Advanced load balancing strategies

#### Developer Experience
- **SDKs**: Official SDKs for multiple languages
- **CLI Tools**: Command-line tools for management
- **Testing**: Comprehensive testing framework
- **Documentation**: Enhanced documentation and examples

### Community Contributions

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to contribute.

### Support

- **Documentation**: Comprehensive documentation in the `docs/` directory
- **Issues**: Report issues on GitHub
- **Community**: Join our Discord/Matrix channels
- **Email**: support@mercury-relay.com

# Directory Structure

This document describes the organization of the Mercury Relay project.

## Root Directory

```
mercury-relay/
├── cmd/                    # Application entry points
├── internal/               # Internal packages
├── test/                   # Test files and fixtures
├── docs/                   # Documentation
├── docker/                 # Docker configuration files
├── configs/                # Configuration files
├── scripts/                # Deployment and setup scripts
├── examples/               # Example configurations
├── config.yaml            # Main configuration file
├── env.example            # Environment variables template
├── go.mod                  # Go module file
├── go.sum                  # Go module checksums
├── Makefile               # Build automation
├── LICENSE                 # License file
└── README.md              # Project overview
```

## Directory Details

### `/cmd/`
Contains the main entry points for different applications:
- `mercury-relay/` - Main relay server
- `mercury-admin/` - Admin CLI tool
- `test-data-gen/` - Test data generator
- `ssh-key-manager/` - SSH key management CLI
- `nostr-ssh-manager/` - Nostr-authenticated SSH manager

### `/internal/`
Internal packages organized by functionality:
- `access/` - Access control and admin management
- `admin/` - Admin interface
- `api/` - REST API handlers
- `auth/` - Authentication (Nostr, universal)
- `cache/` - Caching interfaces and implementations
- `config/` - Configuration management
- `models/` - Data models and generators
- `quality/` - Quality control and spam detection
- `queue/` - Message queue interfaces
- `relay/` - Core relay functionality
- `storage/` - Storage interfaces
- `streaming/` - Upstream relay streaming
- `testgen/` - Test data generation
- `transport/` - Transport layer (WebSocket, SSH, Tor, I2P)

### `/test/`
Test files and test data:
- `fixtures/` - Test data files
- `helpers/` - Test helper functions
- `integration/` - Integration tests
- `mocks/` - Mock implementations

### `/docs/`
Documentation files:
- `configuration.md` - Configuration guide
- `docker.md` - Docker setup
- `environment.md` - Environment variables
- `ssh-authentication.md` - SSH key management
- `streaming.md` - Upstream relay setup
- `apache.md` - Apache reverse proxy
- `directory-structure.md` - This file

### `/docker/`
Docker-related files:
- `docker-compose.yml` - Standard Docker Compose
- `docker-compose-tor.yml` - Tor-enabled Docker Compose
- `docker-compose.test.yml` - Test Docker Compose
- `Dockerfile` - Multi-stage Docker build
- `docker-run.sh` - Docker run script

### `/configs/`
Configuration files for various services:
- `apache.conf` - Apache virtual host
- `apache-docker.conf` - Apache Docker configuration
- `apache-setup.sh` - Apache setup script
- `nginx.conf` - Nginx configuration
- `init.sql` - Database initialization
- `streaming-config.yaml` - Streaming configuration
- `nostr-event-kinds.yaml` - Event kind definitions
- `tor/` - Tor configuration files
- `xftp/` - XFTP configuration files

### `/scripts/`
Deployment and setup scripts:
- `deploy.sh` - Standard deployment
- `deploy-tor.sh` - Tor deployment
- `streaming-setup.sh` - Streaming setup
- `tor-setup.sh` - Tor setup
- `xftp-setup.sh` - XFTP setup

### `/examples/`
Example configurations and usage:
- Example configuration files
- Usage examples
- Integration examples

## File Organization Principles

1. **Separation of Concerns**: Each directory has a specific purpose
2. **Docker Organization**: All Docker files are in `/docker/`
3. **Configuration Centralization**: All config files are in `/configs/`
4. **Script Organization**: All deployment scripts are in `/scripts/`
5. **Documentation**: All docs are in `/docs/`
6. **Clean Root**: Root directory contains only essential files

## Usage

### Development
```bash
# Run tests
go test ./...

# Build the relay
go build -o mercury-relay ./cmd/mercury-relay

# Run with config
./mercury-relay -config config.yaml
```

### Docker
```bash
# Standard deployment
cd docker && docker-compose up -d

# Tor deployment
cd docker && docker-compose -f docker-compose-tor.yml up -d
```

### Configuration
```bash
# Copy example environment
cp env.example .env

# Edit configuration
vim config.yaml
```

## Best Practices

1. **Keep Root Clean**: Only essential files in root directory
2. **Organize by Function**: Group related files together
3. **Use Subdirectories**: Create subdirectories for related files
4. **Document Structure**: Keep this file updated
5. **Consistent Naming**: Use consistent naming conventions

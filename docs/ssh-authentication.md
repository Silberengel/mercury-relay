# Mercury Relay SSH Support

This document describes the SSH functionality added to Mercury Relay, enabling WebSocket connections over TCP over SSH for enhanced security and tunneling capabilities.

## Features

- **SSH Key Management**: Secure generation, storage, and management of SSH keys
- **WebSocket over SSH**: Tunnel WebSocket connections through SSH for enhanced security
- **Terminal Interface**: Interactive SSH key management through a terminal interface
- **Port Forwarding**: SSH tunnel support for port forwarding
- **Key Storage**: Secure file-based key storage with proper permissions

## Configuration

### SSH Configuration in config.yaml

```yaml
ssh:
  enabled: true
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
```

### Environment Variables

You can override SSH configuration using environment variables:

```bash
export SSH_ENABLED=true
export SSH_HOST=your-ssh-server.com
export SSH_PORT=22
export SSH_USERNAME=mercury
export SSH_KEY_DIR=/path/to/ssh/keys
export SSH_TERMINAL_ENABLED=true
export SSH_TERMINAL_PORT=2222
```

## SSH Key Management

### Using the Terminal Interface

The SSH key manager provides an interactive terminal interface for managing SSH keys:

```bash
# Start the SSH key manager
go run cmd/ssh-key-manager/main.go
```

Available commands:
- `list` - List all SSH keys
- `add <name>` - Generate a new SSH key pair
- `remove <name>` - Remove an SSH key pair
- `show <name>` - Show details of a specific key
- `test` - Test SSH connection
- `help` - Show available commands
- `quit` - Exit the program

### Programmatic Key Management

```go
// Create SSH transport
sshTransport := transport.NewSSHTransport(sshConfig)

// Generate a new key
key, err := sshTransport.keyManager.GenerateKey("my-key", "user@example.com")

// List all keys
keys := sshTransport.keyManager.ListKeys()

// Remove a key
err = sshTransport.keyManager.RemoveKey("my-key")
```

## WebSocket over SSH

### Endpoints

When SSH is enabled, Mercury Relay provides an additional endpoint for WebSocket over SSH:

- **Standard WebSocket**: `ws://localhost:8080/` (regular WebSocket connection)
- **WebSocket over SSH**: `ws://localhost:8080/ssh` (WebSocket tunneled through SSH)

### Usage

Clients can connect to the SSH-tunneled WebSocket endpoint for enhanced security:

```javascript
// Connect to WebSocket over SSH
const ws = new WebSocket('ws://localhost:8080/ssh');

ws.onopen = function() {
    console.log('Connected to Mercury Relay via SSH tunnel');
    
    // Send Nostr protocol messages
    ws.send(JSON.stringify(['REQ', 'subscription-id', {kinds: [1]}]));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};
```

## SSH Tunnel Port Forwarding

The SSH transport also supports port forwarding for tunneling other services:

```go
// Create SSH tunnel for port forwarding
tunnel := transport.NewWebSocketSSHTunnel(sshTransport)

// Forward local port 8080 to remote host:port through SSH
err := tunnel.CreateSSHTunnel(8080, "remote-host", 8080)
```

## Security Considerations

### Key Storage

- SSH keys are stored with restricted permissions (600 for private keys, 644 for public keys)
- Keys are stored in the configured directory with proper file extensions
- Private keys are encrypted at rest using standard SSH key formats

### Connection Security

- SSH connections use standard SSH authentication methods
- Host key verification should be configured for production use
- Compression can be enabled for bandwidth optimization
- Connection timeouts and retry logic are configurable

### Best Practices

1. **Key Management**:
   - Use strong key sizes (2048+ bits for RSA)
   - Regularly rotate SSH keys
   - Store keys in secure locations
   - Use passphrase-protected keys in production

2. **Network Security**:
   - Use SSH over secure networks
   - Configure proper firewall rules
   - Monitor SSH connection logs
   - Use SSH key-based authentication instead of passwords

3. **Configuration**:
   - Disable SSH terminal interface in production if not needed
   - Use environment variables for sensitive configuration
   - Regularly update SSH server configurations
   - Monitor connection statistics

## Monitoring and Statistics

The SSH transport provides connection statistics:

```go
// Get SSH tunnel statistics
stats := tunnel.GetConnectionStats()
fmt.Printf("Total connections: %d\n", stats["total_connections"])
fmt.Printf("Active connections: %d\n", stats["active_connections"])
```

## Troubleshooting

### Common Issues

1. **SSH Connection Failures**:
   - Check SSH server configuration
   - Verify SSH keys are properly configured
   - Ensure network connectivity
   - Check SSH server logs

2. **Key Management Issues**:
   - Verify key directory permissions
   - Check key file formats
   - Ensure proper key ownership
   - Validate key generation parameters

3. **WebSocket over SSH Issues**:
   - Verify SSH tunnel is established
   - Check WebSocket upgrade process
   - Monitor connection statistics
   - Review SSH and WebSocket logs

### Debug Mode

Enable debug logging for SSH operations:

```yaml
logging:
  level: "debug"
```

## Examples

### Basic SSH Setup

1. Configure SSH in `config.yaml`:
```yaml
ssh:
  enabled: true
  key_storage:
    key_dir: "./ssh-keys"
  connection:
    host: "your-ssh-server.com"
    username: "mercury"
```

2. Generate SSH keys:
```bash
go run cmd/ssh-key-manager/main.go
# Use 'add my-key' to generate a key
```

3. Start Mercury Relay:
```bash
go run cmd/mercury-relay/main.go
```

4. Connect via WebSocket over SSH:
```javascript
const ws = new WebSocket('ws://localhost:8080/ssh');
```

### Advanced Configuration

For production deployments, consider:

- Using dedicated SSH servers
- Implementing key rotation policies
- Setting up monitoring and alerting
- Configuring backup and recovery procedures
- Implementing access control and auditing

## API Reference

### SSH Transport

```go
type SSHTransport struct {
    config     config.SSHConfig
    client     *ssh.Client
    keyManager *SSHKeyManager
    // ... other fields
}

func NewSSHTransport(config config.SSHConfig) *SSHTransport
func (s *SSHTransport) Start(ctx context.Context) error
func (s *SSHTransport) Stop() error
func (s *SSHTransport) IsHealthy() bool
func (s *SSHTransport) GetAddress() string
```

### SSH Key Manager

```go
type SSHKeyManager struct {
    config config.SSHKeyStorage
    keys   map[string]*SSHKey
    // ... other fields
}

func (km *SSHKeyManager) Initialize() error
func (km *SSHKeyManager) GenerateKey(name, comment string) (*SSHKey, error)
func (km *SSHKeyManager) ListKeys() []*SSHKey
func (km *SSHKeyManager) RemoveKey(name string) error
func (km *SSHKeyManager) GetAuthMethods() []ssh.AuthMethod
```

### WebSocket SSH Tunnel

```go
type WebSocketSSHTunnel struct {
    sshTransport *SSHTransport
    upgrader     websocket.Upgrader
    connections  map[*websocket.Conn]*SSHTunnelConnection
    // ... other fields
}

func NewWebSocketSSHTunnel(sshTransport *SSHTransport) *WebSocketSSHTunnel
func (wst *WebSocketSSHTunnel) HandleWebSocketOverSSH(w http.ResponseWriter, r *http.Request) error
func (wst *WebSocketSSHTunnel) GetConnectionStats() map[string]interface{}
func (wst *WebSocketSSHTunnel) CreateSSHTunnel(localPort int, remoteHost string, remotePort int) error
```

This SSH functionality provides Mercury Relay with enhanced security and tunneling capabilities, enabling secure WebSocket connections over SSH for Nostr protocol communication.

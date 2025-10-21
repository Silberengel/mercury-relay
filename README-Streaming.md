# Mercury Relay Streaming Configuration

This guide explains how to configure Mercury Relay for streaming from upstream Nostr relays.

## 🌊 Default Upstream Relays

Mercury Relay is configured to stream from these relays by default:

### **Primary Relays (Enabled)**
1. **`wss://theforest.nostr1.com`** - High performance relay
2. **`wss://orlay-relay.imwald.eu`** - European relay  
3. **`wss://nostr.land`** - Community relay
4. **`wss://nostr21.com`** - Alternative relay

### **Backup Relays (Disabled)**
- `wss://relay.damus.io` - Damus relay
- `wss://nostr.wine` - Nostr Wine
- `wss://nos.lol` - Nos.lol
- `wss://relay.snort.social` - Snort Social

## 🚀 Quick Setup

### 1. Configure Streaming

```bash
# Run the streaming setup script
./streaming-setup.sh
```

This script will:
- Test connectivity to all relays
- Configure working relays as upstream
- Update your configuration files
- Create Docker Compose overrides

### 2. Start Mercury Relay

```bash
# Start with streaming enabled
docker-compose up -d

# Check streaming status
docker-compose logs mercury-relay
```

## 🔧 Configuration

### Manual Configuration

Edit `config.yaml`:

```yaml
streaming:
  enabled: true
  upstream_relays:
    - url: "wss://theforest.nostr1.com"
      enabled: true
      priority: 1
    - url: "wss://orlay-relay.imwald.eu"
      enabled: true
      priority: 2
    - url: "wss://nostr.land"
      enabled: true
      priority: 3
    - url: "wss://nostr21.com"
      enabled: true
      priority: 4
```

### Advanced Configuration

Edit `streaming-config.yaml` for advanced settings:

```yaml
streaming:
  # Connection settings
  connection_pool_size: 10
  reconnect_interval: "30s"
  timeout: "60s"
  
  # Load balancing
  load_balancing:
    strategy: "round_robin"  # round_robin, least_connections, priority
    health_check_interval: "60s"
    failover_enabled: true
    
  # Rate limiting
  rate_limiting:
    enabled: true
    requests_per_minute: 100
    burst_size: 20
```

## 📊 Monitoring

### Check Streaming Status

```bash
# View streaming logs
docker-compose logs mercury-relay | grep -i stream

# Check relay connectivity
curl http://localhost:8082/api/v1/health

# View metrics
curl http://localhost:8082/api/v1/metrics
```

### Health Checks

```bash
# Test individual relays
./streaming-setup.sh

# Check connection pool
docker-compose exec mercury-relay netstat -an | grep :8080
```

## 🔄 Transport Methods

Mercury Relay supports multiple transport methods:

### **WebSocket (Primary)**
- Real-time bidirectional communication
- Low latency, high performance
- Default for Nostr protocol

### **HTTP Streaming**
- Server-side rendering support
- Fallback for WebSocket issues
- Good for proxy environments

### **Server-Sent Events (SSE)**
- Real-time updates to web clients
- One-way communication
- Good for monitoring dashboards

### **Tor Support**
- Anonymous connections
- Privacy-focused streaming
- Requires Tor daemon

### **I2P Support**
- Decentralized anonymous network
- Alternative to Tor
- Experimental support

## 🛠️ Troubleshooting

### Common Issues

1. **Relay Connection Failed**
   ```bash
   # Check relay status
   curl -I https://theforest.nostr1.com
   
   # Test WebSocket connection
   wscat -c wss://theforest.nostr1.com
   ```

2. **High Latency**
   ```bash
   # Check network connectivity
   ping theforest.nostr1.com
   
   # Test with different relays
   ./streaming-setup.sh
   ```

3. **Memory Usage**
   ```bash
   # Monitor container resources
   docker stats mercury-relay
   
   # Check connection pool
   docker-compose exec mercury-relay netstat -an
   ```

### Performance Tuning

```yaml
# Optimize for high throughput
streaming:
  connection_pool_size: 20
  timeout: "30s"
  rate_limiting:
    requests_per_minute: 200
    burst_size: 50
```

## 🔒 Security

### Rate Limiting

```yaml
rate_limiting:
  enabled: true
  requests_per_minute: 100
  burst_size: 20
  per_ip_limit: 50
```

### Access Control

```yaml
access_control:
  allowed_origins: ["*"]
  blocked_ips: []
  whitelist_mode: false
```

## 📈 Scaling

### Multiple Instances

```yaml
# Load balance across multiple Mercury Relay instances
services:
  mercury-relay-1:
    # ... configuration
  mercury-relay-2:
    # ... configuration
  nginx:
    # Load balance between instances
```

### High Availability

```yaml
# Configure failover
load_balancing:
  strategy: "least_connections"
  failover_enabled: true
  health_check_interval: "30s"
```

## 🔄 Updates

### Adding New Relays

1. Edit `config.yaml`:
   ```yaml
   upstream_relays:
     - url: "wss://new-relay.com"
       enabled: true
       priority: 5
   ```

2. Restart Mercury Relay:
   ```bash
   docker-compose restart mercury-relay
   ```

### Removing Relays

1. Set `enabled: false` in config
2. Restart Mercury Relay
3. Monitor for any issues

## 📞 Support

### Debugging

```bash
# Enable debug logging
export LOG_LEVEL=debug
docker-compose up -d

# View detailed logs
docker-compose logs -f mercury-relay
```

### Getting Help

1. Check the logs: `docker-compose logs mercury-relay`
2. Test connectivity: `./streaming-setup.sh`
3. Verify configuration: `docker-compose config`
4. Check system resources: `docker stats`

## 🎯 Best Practices

1. **Monitor Performance** - Use metrics endpoint to track performance
2. **Test Connectivity** - Regularly run `./streaming-setup.sh`
3. **Backup Configuration** - Keep backups of working configurations
4. **Update Regularly** - Keep relays and Mercury Relay updated
5. **Monitor Logs** - Watch for connection issues and errors

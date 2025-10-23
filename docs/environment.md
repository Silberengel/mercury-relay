# Mercury Relay Environment Variables

This guide explains how to configure Mercury Relay using environment variables for Docker deployment.

## 🐳 Docker Environment Variables

### **Basic Configuration**

```bash
# Ports
NOSTR_RELAY_PORT=8080
ADMIN_PORT=8081
REST_API_PORT=8082

# Logging
LOG_LEVEL=info

# Admin Configuration
MERCURY_ADMIN_NPUBS=npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j,npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx,npub1l5sga6xg72phsz5422ykujprejwud075ggrr3z2hwyrfgr7eylqstegx9z,npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl

# Authentication (supports both hex and bech32 formats)
MERCURY_PRIVATE_KEY=nsec1your-private-key-here
```

### **Streaming Configuration**

```bash
# Enable/disable streaming
STREAMING_ENABLED=true

# Upstream relays (comma-separated)
UPSTREAM_RELAYS=wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com
```

### **Tor and XFTP**

```bash
# Tor support
TOR_ENABLED=false
TOR_SOCKS_PORT=9050
TOR_CONTROL_PORT=9051

# XFTP support
XFTP_ENABLED=false
XFTP_PORT=8083
XFTP_MAX_FILE_SIZE=50MB
```

## 🚀 Quick Start Examples

### **1. Default Configuration**

```bash
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -e STREAMING_ENABLED=true \
  -e UPSTREAM_RELAYS="wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com" \
  -e MERCURY_ADMIN_NPUBS="npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j,npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx,npub1l5sga6xg72phsz5422ykujprejwud075ggrr3z2hwyrfgr7eylqstegx9z,npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl" \
  mercury-relay
```

### **2. Custom Relays**

```bash
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -e UPSTREAM_RELAYS="wss://relay.damus.io,wss://nos.lol,wss://relay.snort.social" \
  -e MERCURY_ADMIN_NPUBS="npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j,npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx,npub1l5sga6xg72phsz5422ykujprejwud075ggrr3z2hwyrfgr7eylqstegx9z,npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl" \
  mercury-relay
```

### **3. No Streaming**

```bash
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -e STREAMING_ENABLED=false \
  -e MERCURY_ADMIN_NPUBS="npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j,npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx,npub1l5sga6xg72phsz5422ykujprejwud075ggrr3z2hwyrfgr7eylqstegx9z,npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl" \
  mercury-relay
```

### **4. With Tor and XFTP**

```bash
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -e STREAMING_ENABLED=true \
  -e UPSTREAM_RELAYS="wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com" \
  -e TOR_ENABLED=true \
  -e XFTP_ENABLED=true \
  -e MERCURY_ADMIN_NPUBS="npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j,npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx,npub1l5sga6xg72phsz5422ykujprejwud075ggrr3z2hwyrfgr7eylqstegx9z,npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl" \
  mercury-relay
```

### **5. Production Configuration**

```bash
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  -e STREAMING_ENABLED=true \
  -e UPSTREAM_RELAYS="wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com" \
  -e LOG_LEVEL=warn \
  -e RATE_LIMIT_PER_MINUTE=200 \
  -e API_KEY="your-secret-api-key" \
  -e MERCURY_ADMIN_NPUBS="npub1flnpz46qtu3jwpsglzacmjrglnssyaxdvcfe5yf0hg3g4qad9xds2g784j,npub1v30tsz9vw6ylpz63g0a702nj3xa26t3m7p5us8f2y2sd8v6cnsvq465zjx,npub1l5sga6xg72phsz5422ykujprejwud075ggrr3z2hwyrfgr7eylqstegx9z,npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl" \
  mercury-relay
```

## 📁 Environment File

### **Using .env File**

1. Copy the example:
   ```bash
   cp env.example .env
   ```

2. Edit `.env` with your settings:
   ```bash
   # Edit .env file
   nano .env
   ```

3. Run with environment file:
   ```bash
   docker run -d --name mercury-relay \
     -p 8080:8080 -p 8081:8081 -p 8082:8082 \
     --env-file .env \
     mercury-relay
   ```

## 🔧 Docker Compose with Environment

### **docker-compose.yml**

```yaml
version: '3.8'

services:
  mercury-relay:
    build: .
    ports:
      - "8080:8080"
      - "8081:8081"
      - "8082:8082"
    environment:
      - STREAMING_ENABLED=${STREAMING_ENABLED:-true}
      - UPSTREAM_RELAYS=${UPSTREAM_RELAYS:-"wss://theforest.nostr1.com,wss://orlay-relay.imwald.eu,wss://nostr.land,wss://nostr21.com"}
      - TOR_ENABLED=${TOR_ENABLED:-false}
      - XFTP_ENABLED=${XFTP_ENABLED:-false}
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
```

### **Runtime Override**

```bash
# Override environment variables at runtime
STREAMING_ENABLED=true \
UPSTREAM_RELAYS="wss://custom-relay1.com,wss://custom-relay2.com" \
docker-compose up -d
```

## 📊 All Environment Variables

### **Core Settings**
- `NOSTR_RELAY_PORT` - Nostr WebSocket port (default: 8080)
- `ADMIN_PORT` - Admin API port (default: 8081)
- `REST_API_PORT` - REST API port (default: 8082)
- `LOG_LEVEL` - Log level (debug|info|warn|error)

### **Streaming**
- `STREAMING_ENABLED` - Enable streaming (true|false)
- `UPSTREAM_RELAYS` - Comma-separated relay URLs

### **Tor**
- `TOR_ENABLED` - Enable Tor support (true|false)
- `TOR_SOCKS_PORT` - Tor SOCKS port (default: 9050)
- `TOR_CONTROL_PORT` - Tor control port (default: 9051)

### **XFTP**
- `XFTP_ENABLED` - Enable XFTP support (true|false)
- `XFTP_PORT` - XFTP port (default: 8083)
- `XFTP_MAX_FILE_SIZE` - Max file size (default: 50MB)

### **Database**
- `DB_TYPE` - Database type (sqlite|postgres)
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_NAME` - Database name
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password

### **Security**
- `API_KEY` - Admin API key
- `CORS_ENABLED` - Enable CORS (true|false)
- `CORS_ORIGINS` - CORS origins (* for all)

### **Rate Limiting**
- `RATE_LIMIT_ENABLED` - Enable rate limiting (true|false)
- `RATE_LIMIT_PER_MINUTE` - Requests per minute (default: 100)
- `RATE_LIMIT_BURST_SIZE` - Burst size (default: 20)

## 🔄 Runtime Configuration Changes

### **Update Relays at Runtime**

```bash
# Method 1: Restart container with new environment
docker stop mercury-relay
docker rm mercury-relay
docker run -d --name mercury-relay \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -e UPSTREAM_RELAYS="wss://new-relay1.com,wss://new-relay2.com" \
  mercury-relay

# Method 2: Using Docker Compose
UPSTREAM_RELAYS="wss://new-relay1.com,wss://new-relay2.com" docker-compose up -d
```

### **View Current Configuration**

```bash
# View environment variables
docker exec mercury-relay env | grep -E "(STREAMING|UPSTREAM|TOR|XFTP)"

# View logs
docker logs mercury-relay
```

## 🛠️ Troubleshooting

### **Common Issues**

1. **Environment variables not working**:
   ```bash
   # Check if variables are set
   docker exec mercury-relay env | grep UPSTREAM_RELAYS
   
   # Verify container restart
   docker restart mercury-relay
   ```

2. **Relays not connecting**:
   ```bash
   # Test relay connectivity
   curl -I https://theforest.nostr1.com
   
   # Check logs
   docker logs mercury-relay | grep -i stream
   ```

3. **Port conflicts**:
   ```bash
   # Check port usage
   netstat -tulpn | grep :8080
   
   # Use different ports
   docker run -d --name mercury-relay \
     -p 9080:8080 -p 9081:8081 -p 9082:8082 \
     mercury-relay
   ```

## 📝 Best Practices

1. **Use environment files** for complex configurations
2. **Set secrets via environment** not in Dockerfiles
3. **Use Docker Compose** for multi-container setups
4. **Monitor logs** for configuration issues
5. **Test connectivity** before deploying
6. **Use health checks** for monitoring
7. **Backup configurations** before changes

## 🎯 Production Deployment

```bash
# Production example with all features
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

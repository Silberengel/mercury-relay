# Mercury Relay Docker Deployment

This guide will help you deploy Mercury Relay to your remote server using Docker.

## 🚀 Quick Start

### 1. Prerequisites

- Docker and Docker Compose installed on your server
- Domain name pointing to your server (optional but recommended)
- Basic knowledge of Linux command line

### 2. Deploy to Remote Server

```bash
# Clone the repository on your server
git clone <your-repo-url>
cd mercury-relay

# Run the deployment script
./deploy.sh
```

### 3. Manual Deployment

If you prefer manual deployment:

```bash
# Build the Docker image
docker build -t mercury-relay .

# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f
```

## 🔧 Configuration

### Environment Variables

Edit `docker-compose.yml` to customize:

```yaml
environment:
  - NOSTR_RELAY_PORT=8080
  - ADMIN_PORT=8081
  - REST_API_PORT=8082
  - LOG_LEVEL=info
```

### Domain Configuration

1. **Update nginx.conf**:
   ```nginx
   server_name your-domain.com;  # Change this
   ```

2. **Update deploy.sh**:
   ```bash
   DOMAIN="your-domain.com"  # Change this
   ```

### SSL/HTTPS Setup

#### Option 1: Let's Encrypt (Recommended)

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx

# Get SSL certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

#### Option 2: Self-signed Certificate

The deployment script automatically creates a self-signed certificate for development.

## 📊 Monitoring

### Health Checks

```bash
# Check if services are healthy
curl http://your-domain.com/health

# Check container status
docker-compose ps

# View logs
docker-compose logs -f mercury-relay
```

### Performance Monitoring

```bash
# Container resource usage
docker stats

# Disk usage
docker system df

# Clean up unused images
docker system prune
```

## 🔒 Security

### Firewall Setup

```bash
# Allow only necessary ports
sudo ufw allow 22    # SSH
sudo ufw allow 80    # HTTP
sudo ufw allow 443   # HTTPS
sudo ufw enable
```

### SSL Configuration

Update `nginx.conf` to enable HTTPS:

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    
    # Your location blocks here
}
```

## 🛠️ Maintenance

### Updates

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Backups

```bash
# Backup data
docker-compose exec postgres pg_dump -U mercury mercury_relay > backup.sql

# Backup configuration
cp config.yaml config.yaml.backup
```

### Logs

```bash
# View all logs
docker-compose logs

# View specific service logs
docker-compose logs mercury-relay
docker-compose logs nginx
docker-compose logs postgres
```

## 🚨 Troubleshooting

### Common Issues

1. **Port conflicts**:
   ```bash
   # Check what's using the ports
   sudo netstat -tulpn | grep :8080
   ```

2. **Permission issues**:
   ```bash
   # Fix permissions
   sudo chown -R $USER:$USER data/ logs/
   ```

3. **Container won't start**:
   ```bash
   # Check logs
   docker-compose logs mercury-relay
   
   # Check configuration
   docker-compose config
   ```

### Performance Issues

1. **High memory usage**:
   ```bash
   # Check memory usage
   docker stats
   
   # Restart services
   docker-compose restart
   ```

2. **Slow responses**:
   ```bash
   # Check nginx logs
   docker-compose logs nginx
   
   # Check relay logs
   docker-compose logs mercury-relay
   ```

## 📈 Scaling

### Multiple Instances

```yaml
# docker-compose.yml
services:
  mercury-relay-1:
    # ... configuration
  mercury-relay-2:
    # ... configuration
  nginx:
    # Load balance between instances
```

### Load Balancing

Update `nginx.conf`:

```nginx
upstream mercury_relay {
    server mercury-relay-1:8080;
    server mercury-relay-2:8080;
}
```

## 🔄 Updates and Maintenance

### Regular Maintenance

```bash
# Weekly maintenance script
#!/bin/bash
docker-compose pull
docker-compose up -d
docker system prune -f
```

### Monitoring

Set up monitoring with tools like:
- Prometheus + Grafana
- ELK Stack (Elasticsearch, Logstash, Kibana)
- Simple health checks

## 📞 Support

If you encounter issues:

1. Check the logs: `docker-compose logs`
2. Verify configuration: `docker-compose config`
3. Check system resources: `docker stats`
4. Review the troubleshooting section above

## 🎯 Production Checklist

- [ ] Domain configured and DNS pointing to server
- [ ] SSL certificate installed and working
- [ ] Firewall configured properly
- [ ] Monitoring set up
- [ ] Backup strategy in place
- [ ] Log rotation configured
- [ ] Security updates applied
- [ ] Performance testing completed

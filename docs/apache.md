# Mercury Relay with Apache

This guide shows you how to deploy Mercury Relay using Apache as the reverse proxy instead of Nginx.

## 🚀 Quick Start

### Option 1: Apache on Host System (Recommended)

If you already have Apache running on your server:

```bash
# 1. Start Mercury Relay (without Apache container)
docker-compose up -d mercury-relay redis postgres

# 2. Configure Apache on your host system
sudo ./apache-setup.sh

# 3. Update the Apache configuration
sudo nano /etc/apache2/sites-available/mercury-relay.conf
# Change ServerName to your domain

# 4. Restart Apache
sudo systemctl restart apache2
```

### Option 2: Apache in Docker Container

If you want Apache in a container:

```bash
# Start everything including Apache container
docker-compose up -d
```

## 🔧 Apache Configuration

### Virtual Host Setup

The Apache configuration includes:

- **WebSocket support** - For Nostr connections
- **Proxy configuration** - Routes to Mercury Relay
- **Security headers** - XSS protection, frame options, etc.
- **Rate limiting** - DDoS protection (if mod_evasive is available)
- **SSL support** - HTTPS configuration

### Key Features

```apache
# WebSocket upgrade for Nostr
RewriteCond %{HTTP:Upgrade} websocket [NC]
RewriteCond %{HTTP:Connection} upgrade [NC]
RewriteRule ^/?(.*) "ws://mercury-relay:8080/$1" [P,L]

# API endpoints
ProxyPass /api/ http://mercury-relay:8082/
ProxyPass /admin/ http://mercury-relay:8081/
```

## 📋 Required Apache Modules

Make sure these modules are enabled:

```bash
# Ubuntu/Debian
sudo a2enmod rewrite
sudo a2enmod proxy
sudo a2enmod proxy_wstunnel
sudo a2enmod headers
sudo a2enmod ssl
sudo a2enmod proxy_http

# CentOS/RHEL (usually enabled by default)
# Check with: httpd -M | grep -E "(rewrite|proxy|headers)"
```

## 🔒 SSL/HTTPS Setup

### Option 1: Let's Encrypt (Recommended)

```bash
# Install Certbot
sudo apt install certbot python3-certbot-apache

# Get SSL certificate
sudo certbot --apache -d your-domain.com

# Auto-renewal
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

### Option 2: Manual SSL Certificate

```bash
# Copy your certificates
sudo cp your-cert.crt /etc/ssl/certs/
sudo cp your-key.key /etc/ssl/private/

# Update Apache configuration
sudo nano /etc/apache2/sites-available/mercury-relay.conf
# Uncomment and configure the HTTPS VirtualHost section
```

## 🛠️ Configuration Files

### Main Configuration
- **`apache.conf`** - Virtual host configuration
- **`apache-docker.conf`** - Docker container configuration
- **`apache-setup.sh`** - Automated setup script

### Docker Compose
The `docker-compose.yml` includes an optional Apache container, but you can use your existing Apache installation instead.

## 📊 Monitoring

### Health Checks

```bash
# Check Mercury Relay health
curl http://your-domain.com/health

# Check Apache status
sudo systemctl status apache2

# Check Docker containers
docker-compose ps
```

### Logs

```bash
# Apache logs
sudo tail -f /var/log/apache2/mercury-relay_error.log
sudo tail -f /var/log/apache2/mercury-relay_access.log

# Mercury Relay logs
docker-compose logs -f mercury-relay
```

## 🔧 Troubleshooting

### Common Issues

1. **WebSocket not working**:
   ```bash
   # Check if proxy_wstunnel is enabled
   apache2 -M | grep proxy_wstunnel
   
   # Check Apache error logs
   sudo tail -f /var/log/apache2/error.log
   ```

2. **Proxy errors**:
   ```bash
   # Check if Mercury Relay is running
   docker-compose ps
   
   # Check container logs
   docker-compose logs mercury-relay
   ```

3. **Permission issues**:
   ```bash
   # Fix Apache permissions
   sudo chown -R www-data:www-data /var/www/html
   sudo chmod -R 755 /var/www/html
   ```

### Performance Tuning

```apache
# Add to your Apache configuration
<IfModule mod_deflate.c>
    AddOutputFilterByType DEFLATE text/plain
    AddOutputFilterByType DEFLATE text/html
    AddOutputFilterByType DEFLATE text/xml
    AddOutputFilterByType DEFLATE text/css
    AddOutputFilterByType DEFLATE application/xml
    AddOutputFilterByType DEFLATE application/xhtml+xml
    AddOutputFilterByType DEFLATE application/rss+xml
    AddOutputFilterByType DEFLATE application/javascript
    AddOutputFilterByType DEFLATE application/x-javascript
</IfModule>

# Enable KeepAlive
KeepAlive On
MaxKeepAliveRequests 100
KeepAliveTimeout 5
```

## 🔄 Updates

### Updating Mercury Relay

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Updating Apache Configuration

```bash
# Test configuration
sudo apache2ctl configtest

# If OK, restart Apache
sudo systemctl restart apache2
```

## 🎯 Production Checklist

- [ ] Apache modules enabled (rewrite, proxy, proxy_wstunnel, headers)
- [ ] SSL certificate installed and working
- [ ] Firewall configured (ports 80, 443)
- [ ] Mercury Relay containers running
- [ ] Health checks passing
- [ ] Logs being monitored
- [ ] Backup strategy in place

## 📞 Support

If you encounter issues:

1. Check Apache error logs: `sudo tail -f /var/log/apache2/error.log`
2. Check Mercury Relay logs: `docker-compose logs mercury-relay`
3. Test configuration: `sudo apache2ctl configtest`
4. Verify modules: `apache2 -M | grep proxy`

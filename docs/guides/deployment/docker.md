# Docker Deployment Guide

This guide covers deploying the WhatsApp Web API Multidevice application using Docker and Docker Compose.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation Methods](#installation-methods)
- [Docker Compose Setup](#docker-compose-setup)
- [Docker with Nginx Reverse Proxy](#docker-with-nginx-reverse-proxy)
- [Running Docker Containers](#running-docker-containers)
- [Docker Troubleshooting](#docker-troubleshooting)
- [Related Guides](#related-guides)

## Prerequisites

### Required

- **Docker**: Version 20.10 or higher
- **Docker Compose**: Version 2.0 or higher (optional, for compose deployment)
- **WhatsApp Account**: Active WhatsApp account with phone

### Optional

- **Reverse Proxy**: For HTTPS and production deployment (nginx, Caddy, Traefik)
- **Domain Name**: For SSL certificate and public access

### Installing Docker

**Ubuntu/Debian:**
```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo apt install docker-compose-plugin

# Add user to docker group
sudo usermod -aG docker $USER
```

**macOS:**
```bash
# Install Docker Desktop
brew install --cask docker
```

**Windows:**
Download Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop)

## Installation Methods

### Method 1: Pull from GitHub Container Registry

Pull the latest image:

```bash
docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
```

Available tags:
- `latest` - Latest stable release
- `v7.8.3` - Specific version tag
- `main` - Latest development build

### Method 2: Build from Source

Clone and build the image:

```bash
# Clone repository
git clone https://github.com/chatwoot-br/go-whatsapp-web-multidevice.git
cd go-whatsapp-web-multidevice

# Build image
docker build -t whatsapp-api .

# Run container
docker run -d \
  --name whatsapp-api \
  -p 3000:3000 \
  -v $(pwd)/storages:/app/storages \
  whatsapp-api
```

## Docker Compose Setup

### Basic Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  whatsapp:
    image: ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
    container_name: whatsapp-api
    ports:
      - "3000:3000"
    volumes:
      - ./storages:/app/storages
      - ./logs:/app/logs
    environment:
      - APP_PORT=3000
      - APP_DEBUG=false
      - APP_BASIC_AUTH=admin:secret123
      - WHATSAPP_WEBHOOK=https://your-webhook.com/handler
      - WHATSAPP_WEBHOOK_SECRET=your-secret-key
    restart: unless-stopped
```

Run the service:

```bash
# Start in detached mode
docker-compose up -d

# View logs
docker-compose logs -f

# Stop service
docker-compose down
```

### Docker Compose with PostgreSQL

For production deployments with PostgreSQL:

```yaml
version: '3.8'

services:
  whatsapp:
    image: ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
    container_name: whatsapp-api
    ports:
      - "3000:3000"
    volumes:
      - ./storages:/app/storages
    environment:
      - APP_PORT=3000
      - APP_BASIC_AUTH=admin:${WHATSAPP_PASSWORD}
      - DB_URI=postgresql://whatsapp:${DB_PASSWORD}@postgres:5432/whatsapp?sslmode=disable
      - WHATSAPP_WEBHOOK=${WEBHOOK_URL}
      - WHATSAPP_WEBHOOK_SECRET=${WEBHOOK_SECRET}
    depends_on:
      - postgres
    restart: unless-stopped
    networks:
      - whatsapp-net

  postgres:
    image: postgres:15-alpine
    container_name: whatsapp-postgres
    environment:
      - POSTGRES_DB=whatsapp
      - POSTGRES_USER=whatsapp
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped
    networks:
      - whatsapp-net

volumes:
  postgres_data:

networks:
  whatsapp-net:
    driver: bridge
```

Create `.env` file for secrets:

```bash
WHATSAPP_PASSWORD=your-strong-password
DB_PASSWORD=your-db-password
WEBHOOK_URL=https://your-webhook.com/handler
WEBHOOK_SECRET=your-webhook-secret
```

## Docker with Nginx Reverse Proxy

Complete setup with SSL termination and reverse proxy.

### docker-compose.yml

```yaml
version: '3.8'

services:
  whatsapp:
    image: ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
    container_name: whatsapp-api
    volumes:
      - ./storages:/app/storages
    environment:
      - APP_PORT=3000
      - APP_BASIC_AUTH=admin:${WHATSAPP_PASSWORD}
      - WHATSAPP_WEBHOOK=${WEBHOOK_URL}
      - WHATSAPP_WEBHOOK_SECRET=${WEBHOOK_SECRET}
    restart: unless-stopped
    networks:
      - whatsapp-net

  nginx:
    image: nginx:alpine
    container_name: whatsapp-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - whatsapp
    restart: unless-stopped
    networks:
      - whatsapp-net

networks:
  whatsapp-net:
    driver: bridge
```

### nginx.conf

Create `nginx.conf` for reverse proxy:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream whatsapp {
        server whatsapp:3000;
    }

    server {
        listen 80;
        server_name whatsapp.yourdomain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name whatsapp.yourdomain.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;

        location / {
            proxy_pass http://whatsapp;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # WebSocket support
        location /ws {
            proxy_pass http://whatsapp;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

### SSL Certificate Setup

**Option 1: Let's Encrypt with Certbot**

```bash
# Install certbot
sudo apt install certbot

# Obtain certificate
sudo certbot certonly --standalone -d whatsapp.yourdomain.com

# Copy certificates
sudo cp /etc/letsencrypt/live/whatsapp.yourdomain.com/fullchain.pem ./ssl/cert.pem
sudo cp /etc/letsencrypt/live/whatsapp.yourdomain.com/privkey.pem ./ssl/key.pem
```

**Option 2: Self-Signed Certificate (Development)**

```bash
# Create SSL directory
mkdir ssl

# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ssl/key.pem -out ssl/cert.pem \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=whatsapp.yourdomain.com"
```

### Start the Stack

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Check service status
docker-compose ps
```

Access your application:
- HTTP: `http://whatsapp.yourdomain.com` (redirects to HTTPS)
- HTTPS: `https://whatsapp.yourdomain.com`

## Running Docker Containers

### Basic Docker Run

Run a container with basic configuration:

```bash
docker run -d \
  --name whatsapp-api \
  -p 3000:3000 \
  -v $(pwd)/storages:/app/storages \
  -e APP_PORT=3000 \
  -e APP_BASIC_AUTH=admin:secret123 \
  -e WHATSAPP_WEBHOOK=https://webhook.site/your-id \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
```

### Run with All Options

```bash
docker run -d \
  --name whatsapp-api \
  -p 3000:3000 \
  -v $(pwd)/storages:/app/storages \
  -v $(pwd)/logs:/app/logs \
  -e APP_PORT=3000 \
  -e APP_DEBUG=false \
  -e APP_OS=MyAppName \
  -e APP_BASIC_AUTH=admin:secret123 \
  -e WHATSAPP_AUTO_REPLY="Thanks for your message!" \
  -e WHATSAPP_AUTO_MARK_READ=false \
  -e WHATSAPP_WEBHOOK=https://webhook.site/your-id \
  -e WHATSAPP_WEBHOOK_SECRET=super-secret \
  -e WHATSAPP_ACCOUNT_VALIDATION=true \
  -e WHATSAPP_CHAT_STORAGE=true \
  --restart unless-stopped \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
```

### Container Management

```bash
# View logs
docker logs -f whatsapp-api

# Stop container
docker stop whatsapp-api

# Start container
docker start whatsapp-api

# Restart container
docker restart whatsapp-api

# Remove container
docker rm -f whatsapp-api

# Execute command in container
docker exec -it whatsapp-api sh

# View container stats
docker stats whatsapp-api
```

### Access the Application

Once running, access the application:

1. **Web UI**: `http://localhost:3000`
2. **Health Check**: `curl http://localhost:3000/app/devices`
3. **Login**: Use QR code or pairing code via API

## Docker Troubleshooting

### Container Exits Immediately

**Problem**: Container starts but exits immediately

**Solution**:

```bash
# Check logs
docker logs whatsapp-api

# Run with debug mode
docker run -it --rm \
  -e APP_DEBUG=true \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest rest

# Check if port is already in use
netstat -tulpn | grep 3000

# Try a different port
docker run -d \
  --name whatsapp-api \
  -p 3001:3000 \
  -v $(pwd)/storages:/app/storages \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
```

### Volume Permission Issues

**Problem**: Cannot write to mounted volumes

**Solution**:

```bash
# Check volume permissions
docker run -it --rm \
  -v $(pwd)/storages:/app/storages \
  alpine ls -la /app/storages

# Fix permissions (Linux)
sudo chown -R $USER:$USER ./storages
chmod -R 755 ./storages

# Run with user ID (Linux)
docker run -d \
  --name whatsapp-api \
  --user $(id -u):$(id -g) \
  -p 3000:3000 \
  -v $(pwd)/storages:/app/storages \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
```

### Network Issues

**Problem**: Cannot connect to container

**Solution**:

```bash
# Check container is running
docker ps

# Check port mapping
docker port whatsapp-api

# Test from host
curl http://localhost:3000/app/devices

# Check container network
docker network inspect bridge

# Test from another container
docker run --rm curlimages/curl:latest \
  curl http://whatsapp:3000/app/devices
```

### Image Pull Issues

**Problem**: Cannot pull image from registry

**Solution**:

```bash
# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Pull specific version
docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.8.3

# Check image exists
docker images | grep whatsapp

# Clean up old images
docker image prune -a
```

### Database Issues in Docker

**Problem**: Database locked or not persisting

**Solution**:

```bash
# Stop container
docker stop whatsapp-api

# Backup database
cp storages/whatsapp.db storages/whatsapp.db.backup

# Check database file
sqlite3 storages/whatsapp.db "PRAGMA integrity_check;"

# Start container with fresh database
docker run -d \
  --name whatsapp-api \
  -p 3000:3000 \
  -v $(pwd)/storages:/app/storages \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest

# If using PostgreSQL, check connection
docker-compose logs postgres
```

### Memory/Resource Issues

**Problem**: Container using too much memory/CPU

**Solution**:

```bash
# Check resource usage
docker stats whatsapp-api

# Set memory limits
docker run -d \
  --name whatsapp-api \
  --memory="512m" \
  --cpus="1.0" \
  -p 3000:3000 \
  -v $(pwd)/storages:/app/storages \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest

# Disable chat storage if not needed
docker run -d \
  --name whatsapp-api \
  -p 3000:3000 \
  -v $(pwd)/storages:/app/storages \
  -e WHATSAPP_CHAT_STORAGE=false \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
```

## Backup and Restore

### Backup Docker Volumes

**Create backup:**

```bash
# Backup using tar
docker run --rm \
  -v whatsapp_storage:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/whatsapp-backup-$(date +%Y%m%d).tar.gz -C /data .

# Backup using Docker Compose volumes
docker-compose down
tar czf whatsapp-backup-$(date +%Y%m%d).tar.gz ./storages
docker-compose up -d
```

**Restore backup:**

```bash
# Stop container
docker-compose down

# Restore from tar
docker run --rm \
  -v whatsapp_storage:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/whatsapp-backup-20250105.tar.gz -C /data

# Or restore files directly
rm -rf ./storages/*
tar xzf whatsapp-backup-20250105.tar.gz -C ./storages

# Start container
docker-compose up -d
```

### Automated Backup Script

Create `backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/backup/whatsapp"
CONTAINER_NAME="whatsapp-api"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup using Docker
docker run --rm \
  -v whatsapp_storage:/data \
  -v "$BACKUP_DIR":/backup \
  alpine tar czf "/backup/whatsapp-$DATE.tar.gz" -C /data .

# Keep only last 7 backups
cd "$BACKUP_DIR"
ls -t whatsapp-*.tar.gz | tail -n +8 | xargs -r rm

echo "Backup completed: whatsapp-$DATE.tar.gz"
```

Schedule with cron:

```bash
# Add to crontab (daily at 2 AM)
0 2 * * * /path/to/backup.sh
```

## Related Guides

- **[Binary Deployment Guide](binary.md)** - Deploy using pre-built binaries
- **[Kubernetes Deployment Guide](kubernetes.md)** - Deploy on Kubernetes
- **[Production Checklist](production-checklist.md)** - Production deployment best practices
- **[Main Deployment Guide](../../deployment-guide.md)** - Overview of all deployment methods

## Additional Resources

- **API Documentation**: `docs/openapi.yaml` - Full REST API specification
- **Webhook Guide**: `docs/webhook-payload.md` - Webhook integration
- **GitHub Repository**: [go-whatsapp-web-multidevice](https://github.com/chatwoot-br/go-whatsapp-web-multidevice)

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-10-05

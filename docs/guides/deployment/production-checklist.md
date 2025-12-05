# Production Deployment Checklist

This guide provides a comprehensive checklist and best practices for deploying the WhatsApp Web API Multidevice application in production environments.

## Table of Contents

- [Pre-Deployment Checklist](#pre-deployment-checklist)
- [Security Best Practices](#security-best-practices)
- [Database Configuration](#database-configuration)
- [Monitoring and Logging](#monitoring-and-logging)
- [Scaling Considerations](#scaling-considerations)
- [Backup and Disaster Recovery](#backup-and-disaster-recovery)
- [Performance Optimization](#performance-optimization)
- [High Availability](#high-availability)
- [Security Hardening](#security-hardening)
- [Related Guides](#related-guides)

## Pre-Deployment Checklist

### Infrastructure

- [ ] **Server/Compute Resources**
  - [ ] Minimum 2GB RAM available
  - [ ] Minimum 2 CPU cores
  - [ ] 10GB+ disk space for database and media
  - [ ] Stable internet connection with low latency

- [ ] **Domain and DNS**
  - [ ] Domain name registered
  - [ ] DNS A/AAAA records configured
  - [ ] SSL certificate obtained (Let's Encrypt recommended)

- [ ] **Network Configuration**
  - [ ] Firewall rules configured (allow ports 80, 443)
  - [ ] Reverse proxy setup (nginx, Caddy, Traefik)
  - [ ] Load balancer configured (if using multiple accounts)

### Application Configuration

- [ ] **Authentication**
  - [ ] Strong basic auth credentials set
  - [ ] Credentials stored securely (environment variables, secrets manager)
  - [ ] Different credentials per environment (dev, staging, prod)

- [ ] **Database**
  - [ ] PostgreSQL installed and configured (recommended for production)
  - [ ] Database backups automated
  - [ ] Connection pooling configured
  - [ ] Database credentials secured

- [ ] **Webhook Configuration**
  - [ ] Webhook URL is HTTPS
  - [ ] Webhook secret generated (minimum 32 characters)
  - [ ] HMAC signature verification implemented
  - [ ] Webhook endpoint tested and monitored

- [ ] **Media Processing**
  - [ ] FFmpeg installed
  - [ ] Media size limits configured appropriately
  - [ ] Media storage path configured with sufficient space
  - [ ] Media cleanup strategy defined

### Deployment

- [ ] **Application Setup**
  - [ ] Latest stable version deployed
  - [ ] Environment variables configured
  - [ ] Log directory created with proper permissions
  - [ ] Systemd service configured (Linux) or equivalent

- [ ] **Testing**
  - [ ] Health check endpoint responding
  - [ ] QR/Pairing code login tested
  - [ ] Message sending tested
  - [ ] Media upload/download tested
  - [ ] Webhook delivery tested

- [ ] **Documentation**
  - [ ] API documentation accessible to developers
  - [ ] Deployment runbook created
  - [ ] Incident response procedures documented
  - [ ] Contact information for on-call team

## Security Best Practices

### 1. Use HTTPS Everywhere

**Always use HTTPS in production** to protect credentials and message content.

**Nginx reverse proxy with SSL:**

```nginx
server {
    listen 80;
    server_name whatsapp.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name whatsapp.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/whatsapp.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/whatsapp.yourdomain.com/privkey.pem;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. Strong Authentication

**Enable basic authentication with strong credentials:**

```bash
# Generate strong password
openssl rand -base64 32

# Set via environment variable
export APP_BASIC_AUTH="admin:$(openssl rand -base64 32)"

# Or via configuration
./whatsapp rest -b "admin:your-strong-password-here"
```

**Best practices:**
- Use passwords with at least 20 characters
- Use different credentials for each environment
- Rotate credentials regularly (quarterly)
- Store credentials in secrets manager (AWS Secrets Manager, HashiCorp Vault)
- Never commit credentials to version control

### 3. Webhook Security

**Secure webhook communication:**

```bash
# Generate strong webhook secret
openssl rand -hex 32

# Configure webhook with HTTPS and secret
./whatsapp rest \
  -w "https://your-secure-webhook.com/handler" \
  --webhook-secret "$(openssl rand -hex 32)"
```

**Webhook handler must verify signatures:**

```javascript
// Node.js example
const crypto = require('crypto');

function verifyWebhookSignature(payload, signature, secret) {
  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');

  return `sha256=${expectedSignature}` === signature;
}

app.post('/webhook', (req, res) => {
  const signature = req.headers['x-hub-signature-256'];

  if (!verifyWebhookSignature(req.rawBody, signature, WEBHOOK_SECRET)) {
    return res.status(401).send('Invalid signature');
  }

  // Process webhook
  res.status(200).send('OK');
});
```

### 4. Secure Environment Variables

**Never hardcode secrets:**

```bash
# Bad - hardcoded in script
./whatsapp rest -b "admin:password123"

# Good - use environment variables
export APP_BASIC_AUTH="admin:${STRONG_PASSWORD}"
export WHATSAPP_WEBHOOK_SECRET="${WEBHOOK_SECRET}"
./whatsapp rest

# Better - use secrets manager
export APP_BASIC_AUTH=$(aws secretsmanager get-secret-value --secret-id whatsapp/basic-auth --query SecretString --output text)
./whatsapp rest
```

### 5. Firewall Configuration

**Restrict access to necessary ports:**

```bash
# UFW (Ubuntu)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw deny 3000/tcp  # Don't expose application port directly
sudo ufw enable

# iptables
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 3000 -j DROP
```

### 6. File System Permissions

**Secure file and directory permissions:**

```bash
# Set restrictive permissions
chmod 700 /opt/whatsapp/storages
chmod 600 /opt/whatsapp/storages/*.db
chmod 700 /opt/whatsapp/.env

# Run as dedicated user
sudo useradd -r -s /bin/false whatsapp
sudo chown -R whatsapp:whatsapp /opt/whatsapp
```

### 7. Rate Limiting

**Implement rate limiting at reverse proxy:**

```nginx
# Nginx rate limiting
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

server {
    location /send/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://localhost:3000;
    }
}
```

## Database Configuration

### PostgreSQL (Recommended for Production)

**Setup PostgreSQL:**

```bash
# Install PostgreSQL
sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql <<EOF
CREATE DATABASE whatsapp;
CREATE USER whatsapp WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE whatsapp TO whatsapp;
ALTER DATABASE whatsapp OWNER TO whatsapp;
EOF
```

**Configure application:**

```bash
# Use PostgreSQL instead of SQLite
export DB_URI="postgresql://whatsapp:your-password@localhost:5432/whatsapp?sslmode=require"
./whatsapp rest
```

**PostgreSQL optimization:**

```sql
-- Tune PostgreSQL settings in postgresql.conf
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 4MB
min_wal_size = 1GB
max_wal_size = 4GB
```

### SQLite (Acceptable for Single Instance)

**If using SQLite, optimize configuration:**

```bash
# Use proper SQLite URI with optimizations
export DB_URI="file:storages/whatsapp.db?_journal_mode=WAL&_timeout=5000&_foreign_keys=on"
./whatsapp rest
```

**SQLite limitations:**
- Not suitable for high concurrency
- Single writer at a time
- Limited to single instance
- Requires regular database maintenance

### Database Maintenance

**PostgreSQL maintenance:**

```bash
# Schedule regular vacuum
0 2 * * 0 psql -U whatsapp -d whatsapp -c "VACUUM ANALYZE;"

# Monitor database size
SELECT pg_size_pretty(pg_database_size('whatsapp'));

# Check table sizes
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```

**SQLite maintenance:**

```bash
# Optimize database periodically
sqlite3 storages/whatsapp.db "VACUUM;"
sqlite3 storages/whatsapp.db "PRAGMA optimize;"

# Check integrity
sqlite3 storages/whatsapp.db "PRAGMA integrity_check;"
```

## Monitoring and Logging

### Application Monitoring

**Health check endpoint:**

```bash
# Monitor application health
curl -f http://localhost:3000/app/devices || alert "Application down"

# Add to monitoring system (Prometheus, Datadog, etc.)
```

**Systemd monitoring:**

```bash
# Check service status
sudo systemctl status whatsapp

# View live logs
sudo journalctl -u whatsapp -f

# Check for errors
sudo journalctl -u whatsapp -p err -n 50
```

### Log Management

**Configure structured logging:**

```bash
# Enable debug for troubleshooting
./whatsapp rest --debug true

# Production logging
./whatsapp rest --debug false > /var/log/whatsapp/app.log 2>&1
```

**Log rotation:**

Create `/etc/logrotate.d/whatsapp`:

```
/var/log/whatsapp/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 whatsapp whatsapp
    sharedscripts
    postrotate
        systemctl reload whatsapp > /dev/null 2>&1 || true
    endscript
}
```

### Monitoring Tools

**Prometheus metrics (if available):**

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'whatsapp-api'
    static_configs:
      - targets: ['localhost:3000']
    metrics_path: '/metrics'
```

**Uptime monitoring:**

```bash
# Simple uptime check script
#!/bin/bash
ENDPOINT="http://localhost:3000/app/devices"
AUTH="admin:secret123"

if ! curl -f -u "$AUTH" "$ENDPOINT" > /dev/null 2>&1; then
    echo "ALERT: WhatsApp API is down"
    # Send alert (email, Slack, PagerDuty, etc.)
fi
```

**Schedule with cron:**

```bash
# Check every 5 minutes
*/5 * * * * /opt/whatsapp/uptime-check.sh
```

### Webhook Monitoring

**Track webhook delivery:**

```javascript
// Monitor webhook failures
const webhookStats = {
  total: 0,
  success: 0,
  failed: 0,
  retries: 0
};

// Log and alert on failures
if (webhookStats.failed / webhookStats.total > 0.1) {
  alert('High webhook failure rate');
}
```

**Monitor webhook latency:**

```bash
# Check webhook response time
time curl -X POST https://your-webhook.com/handler \
  -H "Content-Type: application/json" \
  -d '{"test": "message"}'
```

## Scaling Considerations

### Vertical Scaling (Single Instance)

**Increase resources for single instance:**

```bash
# Monitor resource usage
top -p $(pgrep whatsapp)
htop -p $(pgrep whatsapp)

# Increase system limits
ulimit -n 10000  # Open files
ulimit -u 2048   # Processes

# Add to systemd service
[Service]
LimitNOFILE=10000
LimitNPROC=2048
```

**PostgreSQL for better performance:**

```bash
# Switch from SQLite to PostgreSQL
export DB_URI="postgresql://whatsapp:password@localhost/whatsapp"
./whatsapp rest
```

### Horizontal Scaling (Multiple Accounts)

⚠️ **Important**: Cannot scale single WhatsApp account horizontally due to protocol limitations.

**For multiple WhatsApp accounts:**

```bash
# Deploy separate instances per account
# Account 1
./whatsapp rest --port 3001 --db-uri "file:storages/account1.db"

# Account 2
./whatsapp rest --port 3002 --db-uri "file:storages/account2.db"

# Account 3
./whatsapp rest --port 3003 --db-uri "file:storages/account3.db"
```

**Load balancer configuration:**

```nginx
# Nginx load balancer for multiple accounts
upstream whatsapp_accounts {
    least_conn;
    server localhost:3001;
    server localhost:3002;
    server localhost:3003;
}

server {
    listen 80;
    location / {
        proxy_pass http://whatsapp_accounts;
    }
}
```

### Resource Optimization

**Disable chat storage if not needed:**

```bash
# Reduces memory and database usage
export WHATSAPP_CHAT_STORAGE=false
./whatsapp rest
```

**Configure media limits:**

```bash
# Limit media sizes to reduce memory usage
export MAX_IMAGE_SIZE=10485760    # 10MB
export MAX_VIDEO_SIZE=52428800    # 50MB
export MAX_FILE_SIZE=52428800     # 50MB
./whatsapp rest
```

## Backup and Disaster Recovery

### Backup Strategy

**Daily automated backups:**

Create `/opt/whatsapp/backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/backup/whatsapp"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup PostgreSQL
pg_dump -U whatsapp whatsapp | gzip > "$BACKUP_DIR/whatsapp-db-$DATE.sql.gz"

# Backup SQLite (if using)
cp /opt/whatsapp/storages/whatsapp.db "$BACKUP_DIR/whatsapp-db-$DATE.db"
cp /opt/whatsapp/storages/chatstorage.db "$BACKUP_DIR/chatstorage-$DATE.db"

# Backup media files
tar czf "$BACKUP_DIR/whatsapp-media-$DATE.tar.gz" \
  -C /opt/whatsapp/statics media/

# Upload to S3 (optional)
aws s3 cp "$BACKUP_DIR/whatsapp-db-$DATE.sql.gz" \
  s3://your-bucket/whatsapp-backups/

# Clean old backups
find "$BACKUP_DIR" -name "whatsapp-*" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: $DATE"
```

**Schedule backups:**

```bash
# Make executable
chmod +x /opt/whatsapp/backup.sh

# Add to crontab (daily at 2 AM)
0 2 * * * /opt/whatsapp/backup.sh
```

### Backup Verification

**Test backups regularly:**

```bash
#!/bin/bash
# restore-test.sh
BACKUP_FILE="$1"
TEST_DIR="/tmp/whatsapp-restore-test"

# Create test directory
mkdir -p "$TEST_DIR"

# Test PostgreSQL backup
gunzip -c "$BACKUP_FILE" | psql -U whatsapp -d whatsapp_test

# Test SQLite backup
sqlite3 "$BACKUP_FILE" "PRAGMA integrity_check;"

# Cleanup
rm -rf "$TEST_DIR"

echo "Backup verification completed"
```

### Disaster Recovery Plan

**Recovery procedures:**

1. **Complete system failure:**

```bash
# Stop application
sudo systemctl stop whatsapp

# Restore database from backup
gunzip -c whatsapp-db-20250105.sql.gz | psql -U whatsapp -d whatsapp

# Or for SQLite
cp whatsapp-db-20250105.db /opt/whatsapp/storages/whatsapp.db

# Restore media files
tar xzf whatsapp-media-20250105.tar.gz -C /opt/whatsapp/statics/

# Set permissions
sudo chown -R whatsapp:whatsapp /opt/whatsapp

# Start application
sudo systemctl start whatsapp

# Verify
curl -u admin:secret http://localhost:3000/app/devices
```

2. **Database corruption:**

```bash
# Stop application
sudo systemctl stop whatsapp

# Backup corrupted database
cp storages/whatsapp.db storages/whatsapp.db.corrupted

# Restore from latest backup
cp /backup/whatsapp/whatsapp-db-latest.db storages/whatsapp.db

# Start application
sudo systemctl start whatsapp
```

3. **Lost session (need to re-login):**

```bash
# Logout current session
curl -X GET http://localhost:3000/app/logout -u admin:secret

# Login with new QR/pairing code
curl -X GET http://localhost:3000/app/login -u admin:secret
```

### Backup to Cloud Storage

**AWS S3:**

```bash
# Install AWS CLI
pip install awscli

# Configure AWS credentials
aws configure

# Backup script with S3 upload
aws s3 sync /backup/whatsapp/ s3://your-bucket/whatsapp-backups/
```

**Google Cloud Storage:**

```bash
# Install gsutil
pip install gsutil

# Backup script with GCS upload
gsutil -m rsync -r /backup/whatsapp/ gs://your-bucket/whatsapp-backups/
```

## Performance Optimization

### Application Performance

**Optimize configuration:**

```bash
# Disable features not needed
export WHATSAPP_CHAT_STORAGE=false      # If you don't need chat history
export WHATSAPP_ACCOUNT_VALIDATION=false # If you trust input

# Tune for performance
./whatsapp rest --debug false
```

**Database query optimization:**

```sql
-- PostgreSQL: Add indexes
CREATE INDEX idx_messages_timestamp ON messages(timestamp);
CREATE INDEX idx_chats_jid ON chats(jid);

-- Analyze tables
ANALYZE messages;
ANALYZE chats;
```

### System Performance

**Optimize Linux kernel:**

```bash
# Increase file descriptors
sudo sysctl -w fs.file-max=100000
sudo sysctl -w fs.nr_open=100000

# Network tuning
sudo sysctl -w net.core.somaxconn=1024
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=2048

# Make permanent in /etc/sysctl.conf
```

**Monitor performance:**

```bash
# CPU and memory
top -p $(pgrep whatsapp)

# I/O performance
iostat -x 5

# Network performance
iftop -i eth0

# Database performance
pg_stat_statements  # PostgreSQL
```

## High Availability

### Application Availability

**Systemd automatic restart:**

```ini
[Service]
Restart=always
RestartSec=10
StartLimitBurst=5
StartLimitInterval=60
```

**Health monitoring:**

```bash
# Watchdog script
#!/bin/bash
while true; do
    if ! curl -f -u admin:secret http://localhost:3000/app/devices > /dev/null 2>&1; then
        echo "Application unhealthy, restarting..."
        systemctl restart whatsapp
    fi
    sleep 60
done
```

### Database Availability

**PostgreSQL replication (for read scaling):**

```bash
# Setup streaming replication
# On primary server
wal_level = replica
max_wal_senders = 3
wal_keep_size = 64

# On replica server
hot_standby = on
```

**Database connection pooling:**

```bash
# Use PgBouncer for connection pooling
sudo apt install pgbouncer

# Configure for WhatsApp database
[databases]
whatsapp = host=localhost port=5432 dbname=whatsapp

# Update application connection
export DB_URI="postgresql://whatsapp:pass@localhost:6432/whatsapp"
```

## Security Hardening

### System Hardening

**Disable root SSH login:**

```bash
# /etc/ssh/sshd_config
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes

sudo systemctl reload sshd
```

**Enable fail2ban:**

```bash
# Install fail2ban
sudo apt install fail2ban

# Create jail for nginx
# /etc/fail2ban/jail.local
[nginx-limit-req]
enabled = true
filter = nginx-limit-req
logpath = /var/log/nginx/error.log
```

**AppArmor/SELinux:**

```bash
# Ubuntu: AppArmor profile
sudo aa-enforce /etc/apparmor.d/whatsapp

# RHEL: SELinux policy
sudo semanage fcontext -a -t httpd_sys_rw_content_t "/opt/whatsapp/storages(/.*)?"
sudo restorecon -R /opt/whatsapp
```

### Application Hardening

**Restrict API access:**

```nginx
# Allow only specific IPs
location / {
    allow 192.168.1.0/24;
    allow 10.0.0.0/8;
    deny all;

    proxy_pass http://localhost:3000;
}
```

**Implement request validation:**

```nginx
# Block suspicious requests
if ($request_method !~ ^(GET|POST|PUT|DELETE)$ ) {
    return 444;
}

# Block malicious user agents
if ($http_user_agent ~* (bot|crawler|spider|scraper)) {
    return 403;
}
```

## Related Guides

- **[Docker Deployment Guide](docker.md)** - Deploy using Docker and Docker Compose
- **[Kubernetes Deployment Guide](kubernetes.md)** - Deploy on Kubernetes
- **[Binary Deployment Guide](binary.md)** - Deploy using pre-built binaries
- **[Main Deployment Guide](../../deployment-guide.md)** - Overview of all deployment methods

## Additional Resources

- **API Documentation**: `docs/openapi.yaml` - Full REST API specification
- **Webhook Guide**: `docs/webhook-payload.md` - Webhook integration
- **GitHub Repository**: [go-whatsapp-web-multidevice](https://github.com/chatwoot-br/go-whatsapp-web-multidevice)

---

**Version**: Compatible with v7.10.1+
**Last Updated**: 2025-12-05

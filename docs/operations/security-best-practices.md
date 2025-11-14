# Security Best Practices

Comprehensive security guide for production deployment of the WhatsApp Web API Multidevice application.

## Table of Contents

- [Overview](#overview)
- [Authentication Security](#authentication-security)
- [HTTPS and TLS](#https-and-tls)
- [Webhook Security](#webhook-security)
- [Database Security](#database-security)
- [API Security](#api-security)
- [Environment Variables Security](#environment-variables-security)
- [Network Security](#network-security)
- [System Hardening](#system-hardening)
- [Monitoring and Logging](#monitoring-and-logging)
- [Security Checklist](#security-checklist)

## Overview

Security should be a top priority when deploying WhatsApp API services in production. This guide covers essential security practices across all layers of your deployment.

**Security Layers:**
1. Network Security (Firewall, TLS)
2. Application Security (Authentication, Authorization)
3. Data Security (Encryption, Secrets Management)
4. Infrastructure Security (System Hardening, Monitoring)

## Authentication Security

### Basic Authentication

Always use strong credentials for API access.

#### Generate Strong Passwords

```bash
# Generate secure random password (32 characters)
openssl rand -base64 32

# Output example: 8xK9mPqR3wL5nT7vY2jC4zF6hN8bV1dE3xP9kQ5r
```

#### Configuration

```bash
# Single user with strong password
export APP_BASIC_AUTH="admin:$(openssl rand -base64 32)"

# Multiple users with different strong passwords
export APP_BASIC_AUTH="admin:8xK9mPqR3w,api:L5nT7vY2jC,support:4zF6hN8bV1"

# Command-line
./whatsapp rest -b "admin:$(openssl rand -base64 32)"
```

#### Best Practices

**✅ DO:**
- Use passwords with at least 20 characters
- Generate passwords using cryptographically secure random generators
- Use different credentials per user/service
- Rotate credentials regularly (every 90 days)
- Store credentials in secrets manager (AWS Secrets Manager, HashiCorp Vault, etc.)
- Use HTTPS always (never HTTP)

**❌ DON'T:**
- Use predictable passwords (password123, admin, etc.)
- Reuse passwords across services
- Store passwords in source code
- Share credentials between users
- Commit credentials to version control

#### Password Rotation Example

```bash
#!/bin/bash
# rotate-credentials.sh

# Generate new password
NEW_PASSWORD=$(openssl rand -base64 32)

# Update in secrets manager
aws secretsmanager update-secret \
  --secret-id whatsapp/basic-auth \
  --secret-string "admin:${NEW_PASSWORD}"

# Restart service with new credentials
sudo systemctl restart whatsapp

# Log rotation
echo "$(date): Credentials rotated" >> /var/log/whatsapp/security.log
```

### Admin API Authentication

Admin API uses Bearer token authentication.

#### Generate Secure Token

```bash
# Generate 256-bit secure token
openssl rand -hex 32

# Output example: a3d8f7e2b9c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9
```

#### Configuration

```bash
# Set Admin API token
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# Start Admin API
./whatsapp admin --port 8088
```

#### API Usage

```bash
# All requests must include Bearer token
curl -X POST "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer a3d8f7e2b9c1d4e5f6a7b8c9d0e1f2a3" \
  -H "Content-Type: application/json" \
  -d '{"port": 3001}'
```

## HTTPS and TLS

**NEVER use HTTP in production.** Always use HTTPS to encrypt all communications.

### Nginx Reverse Proxy with SSL

#### Install Certbot (Let's Encrypt)

```bash
# Ubuntu/Debian
sudo apt install certbot python3-certbot-nginx

# macOS
brew install certbot
```

#### Obtain SSL Certificate

```bash
# Obtain certificate
sudo certbot certonly --standalone -d whatsapp.yourdomain.com

# Certificates will be saved to:
# /etc/letsencrypt/live/whatsapp.yourdomain.com/fullchain.pem
# /etc/letsencrypt/live/whatsapp.yourdomain.com/privkey.pem
```

#### Nginx Configuration

Create `/etc/nginx/sites-available/whatsapp`:

```nginx
# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name whatsapp.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

# HTTPS server
server {
    listen 443 ssl http2;
    server_name whatsapp.yourdomain.com;

    # SSL Certificate
    ssl_certificate /etc/letsencrypt/live/whatsapp.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/whatsapp.yourdomain.com/privkey.pem;

    # SSL Configuration (Mozilla Intermediate)
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_stapling on;
    ssl_stapling_verify on;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Permissions-Policy "geolocation=(), microphone=(), camera=()" always;

    # Proxy Configuration
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Buffer sizes
        proxy_buffer_size 128k;
        proxy_buffers 4 256k;
        proxy_busy_buffers_size 256k;
    }

    # WebSocket support
    location /ws {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }

    # Rate Limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;

    # Access and Error Logs
    access_log /var/log/nginx/whatsapp_access.log;
    error_log /var/log/nginx/whatsapp_error.log;
}
```

Enable site and reload Nginx:

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/whatsapp /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload nginx
sudo systemctl reload nginx
```

#### Auto-Renew SSL Certificates

```bash
# Add renewal cron job
sudo crontab -e

# Add line (renew daily at 2 AM)
0 2 * * * certbot renew --quiet --post-hook "systemctl reload nginx"
```

### Caddy Reverse Proxy (Automatic HTTPS)

Caddy automatically obtains and renews SSL certificates:

#### Caddyfile Configuration

Create `/etc/caddy/Caddyfile`:

```
whatsapp.yourdomain.com {
    reverse_proxy localhost:3000

    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    # Rate limiting
    rate_limit {
        zone api {
            key {remote_host}
            events 10
            window 1s
        }
    }

    # Access logs
    log {
        output file /var/log/caddy/whatsapp_access.log
    }
}
```

Reload Caddy:

```bash
sudo systemctl reload caddy
```

### Application Configuration for Reverse Proxy

Enable trusted proxy mode:

```bash
# Environment variable
export APP_TRUSTED_PROXY=true

# CLI flag
./whatsapp rest --trusted-proxy=true

# .env file
APP_TRUSTED_PROXY=true
```

## Webhook Security

Webhooks must be secured to prevent unauthorized access and data tampering.

### HMAC Signature Verification

Always verify webhook signatures before processing events.

#### Configure Webhook Secret

```bash
# Generate strong secret (256-bit)
WEBHOOK_SECRET=$(openssl rand -hex 32)

# Configure application
export WHATSAPP_WEBHOOK_SECRET="$WEBHOOK_SECRET"
export WHATSAPP_WEBHOOK="https://yourapp.com/webhook"

./whatsapp rest
```

#### Verification Implementation

**Node.js Example:**

```javascript
const crypto = require('crypto');
const express = require('express');
const app = express();

// IMPORTANT: Use raw body parser for signature verification
app.use(express.raw({type: 'application/json'}));

function verifyWebhookSignature(payload, signature, secret) {
    // Calculate expected signature
    const expectedSignature = crypto
        .createHmac('sha256', secret)
        .update(payload, 'utf8')
        .digest('hex');

    // Extract received signature (remove 'sha256=' prefix)
    const receivedSignature = signature.replace('sha256=', '');

    // Use timing-safe comparison to prevent timing attacks
    return crypto.timingSafeEqual(
        Buffer.from(expectedSignature, 'hex'),
        Buffer.from(receivedSignature, 'hex')
    );
}

app.post('/webhook', (req, res) => {
    const signature = req.headers['x-hub-signature-256'];
    const payload = req.body; // Raw buffer
    const secret = process.env.WHATSAPP_WEBHOOK_SECRET;

    // Verify signature FIRST
    if (!verifyWebhookSignature(payload, signature, secret)) {
        console.error('Invalid webhook signature', {
            ip: req.ip,
            timestamp: new Date().toISOString()
        });
        return res.status(401).send('Unauthorized');
    }

    // Parse and process webhook data after verification
    const data = JSON.parse(payload.toString());
    console.log('Verified webhook:', data.event);

    // Process event
    processWebhookEvent(data);

    res.status(200).send('OK');
});
```

**Python Example:**

```python
import hmac
import hashlib
import json
from flask import Flask, request, abort

app = Flask(__name__)

def verify_webhook_signature(payload, signature, secret):
    """Verify webhook HMAC signature"""
    # Calculate expected signature
    expected_signature = hmac.new(
        secret.encode('utf-8'),
        payload,
        hashlib.sha256
    ).hexdigest()

    # Extract received signature
    received_signature = signature.replace('sha256=', '')

    # Use timing-safe comparison
    return hmac.compare_digest(expected_signature, received_signature)

@app.route('/webhook', methods=['POST'])
def webhook():
    signature = request.headers.get('X-Hub-Signature-256')
    payload = request.get_data()  # Raw bytes
    secret = os.getenv('WHATSAPP_WEBHOOK_SECRET')

    # Verify signature FIRST
    if not verify_webhook_signature(payload, signature, secret):
        print(f'Invalid webhook signature from {request.remote_addr}')
        abort(401)

    # Parse and process after verification
    data = json.loads(payload)
    print(f'Verified webhook: {data.get("event")}')

    # Process event
    process_webhook_event(data)

    return 'OK', 200
```

### Webhook Security Best Practices

**✅ DO:**
- Always use HTTPS webhook URLs
- Verify HMAC signatures on every request
- Use timing-safe comparison functions
- Log failed verification attempts
- Implement rate limiting
- Return `200 OK` quickly (process async if needed)
- Implement retry logic with exponential backoff

**❌ DON'T:**
- Use HTTP webhook URLs
- Skip signature verification
- Use direct string comparison for signatures
- Log full webhook payloads (may contain sensitive data)
- Process webhooks synchronously if it takes > 5 seconds
- Expose webhook endpoints without authentication

### IP Whitelisting (Optional)

If your webhook handler has a static IP, whitelist it:

```nginx
# Nginx configuration
location /webhook {
    # Allow only specific IPs
    allow 192.168.1.100;
    allow 10.0.0.0/8;
    deny all;

    proxy_pass http://localhost:3000;
}
```

## Database Security

### PostgreSQL Security

#### Secure PostgreSQL Setup

```sql
-- Create dedicated database and user
CREATE DATABASE whatsapp;
CREATE USER whatsapp WITH PASSWORD 'strong-password-here';

-- Grant minimal permissions
GRANT CONNECT ON DATABASE whatsapp TO whatsapp;
GRANT USAGE ON SCHEMA public TO whatsapp;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO whatsapp;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO whatsapp;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO whatsapp;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO whatsapp;
```

#### PostgreSQL Configuration

Edit `/etc/postgresql/14/main/postgresql.conf`:

```conf
# Listen only on localhost (if app is on same server)
listen_addresses = 'localhost'

# Enable SSL
ssl = on
ssl_cert_file = '/etc/postgresql/14/main/server.crt'
ssl_key_file = '/etc/postgresql/14/main/server.key'

# Password encryption
password_encryption = scram-sha-256

# Logging
log_connections = on
log_disconnections = on
log_line_prefix = '%t [%p] %u@%d '
```

#### Connection String with SSL

```bash
# Enable SSL mode
export DB_URI="postgresql://whatsapp:password@localhost:5432/whatsapp?sslmode=require"

# Verify certificate
export DB_URI="postgresql://whatsapp:password@localhost:5432/whatsapp?sslmode=verify-full&sslrootcert=/path/to/ca.crt"
```

### SQLite Security

#### File Permissions

```bash
# Restrict database file access
chmod 600 storages/whatsapp.db
chmod 600 storages/chatstorage.db

# Restrict directory access
chmod 700 storages/

# Set ownership
sudo chown whatsapp:whatsapp storages/
sudo chown whatsapp:whatsapp storages/*.db
```

#### SQLite WAL Mode

Use Write-Ahead Logging for better concurrency and integrity:

```bash
export DB_URI="file:storages/whatsapp.db?_journal_mode=WAL&_timeout=5000&_foreign_keys=on"
```

### Database Backup Security

```bash
#!/bin/bash
# secure-backup.sh

BACKUP_DIR="/secure/backups/whatsapp"
DATE=$(date +%Y%m%d_%H%M%S)
PGP_KEY="backup@yourcompany.com"

# Create encrypted backup
pg_dump -U whatsapp whatsapp | \
  gzip | \
  gpg --encrypt --recipient "$PGP_KEY" \
  > "$BACKUP_DIR/whatsapp-$DATE.sql.gz.gpg"

# Set restrictive permissions
chmod 400 "$BACKUP_DIR/whatsapp-$DATE.sql.gz.gpg"

# Upload to secure storage
aws s3 cp "$BACKUP_DIR/whatsapp-$DATE.sql.gz.gpg" \
  s3://your-secure-bucket/whatsapp-backups/ \
  --storage-class GLACIER \
  --server-side-encryption AES256

# Remove local backup after upload
rm "$BACKUP_DIR/whatsapp-$DATE.sql.gz.gpg"
```

## API Security

### Rate Limiting

Protect your API from abuse with rate limiting.

#### Nginx Rate Limiting

```nginx
# Define rate limit zones
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=send:10m rate=5r/s;

server {
    # General API rate limit
    location / {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://localhost:3000;
    }

    # Stricter limit for send endpoints
    location /send/ {
        limit_req zone=send burst=10 nodelay;
        proxy_pass http://localhost:3000;
    }
}
```

#### Application-Level Rate Limiting

For Node.js webhook handlers:

```javascript
const rateLimit = require('express-rate-limit');

// Rate limiter for webhook endpoint
const webhookLimiter = rateLimit({
    windowMs: 1 * 60 * 1000, // 1 minute
    max: 100, // Limit each IP to 100 requests per minute
    message: 'Too many requests, please try again later',
    standardHeaders: true,
    legacyHeaders: false,
});

app.post('/webhook', webhookLimiter, (req, res) => {
    // Handle webhook
});
```

### Input Validation

Always validate and sanitize user input:

```javascript
// Example: Validate phone number format
function validatePhone(phone) {
    // Only allow digits (country code + number)
    const phoneRegex = /^[1-9]\d{10,14}$/;
    return phoneRegex.test(phone);
}

app.post('/send/message', (req, res) => {
    const { phone, message } = req.body;

    // Validate inputs
    if (!validatePhone(phone)) {
        return res.status(400).json({
            error: 'Invalid phone number format'
        });
    }

    if (!message || message.length > 4096) {
        return res.status(400).json({
            error: 'Invalid message (max 4096 characters)'
        });
    }

    // Process request
    sendMessage(phone, message);
});
```

### CORS Configuration

If serving web clients, configure CORS properly:

```nginx
# Nginx CORS configuration
location / {
    # Only allow specific origins
    if ($http_origin ~* (https://app\.yourdomain\.com|https://admin\.yourdomain\.com)) {
        add_header 'Access-Control-Allow-Origin' "$http_origin" always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type' always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
    }

    # Handle preflight requests
    if ($request_method = 'OPTIONS') {
        return 204;
    }

    proxy_pass http://localhost:3000;
}
```

## Environment Variables Security

### Secrets Management

**Never hardcode secrets in code or configuration files.**

#### AWS Secrets Manager

```bash
#!/bin/bash
# load-secrets.sh

# Fetch secrets from AWS Secrets Manager
BASIC_AUTH=$(aws secretsmanager get-secret-value \
  --secret-id whatsapp/basic-auth \
  --query SecretString \
  --output text)

WEBHOOK_SECRET=$(aws secretsmanager get-secret-value \
  --secret-id whatsapp/webhook-secret \
  --query SecretString \
  --output text)

DB_PASSWORD=$(aws secretsmanager get-secret-value \
  --secret-id whatsapp/db-password \
  --query SecretString \
  --output text)

# Export as environment variables
export APP_BASIC_AUTH="$BASIC_AUTH"
export WHATSAPP_WEBHOOK_SECRET="$WEBHOOK_SECRET"
export DB_URI="postgresql://whatsapp:${DB_PASSWORD}@localhost:5432/whatsapp"

# Start application
./whatsapp rest
```

#### HashiCorp Vault

```bash
#!/bin/bash
# load-secrets-vault.sh

# Login to Vault
vault login -method=token token="$VAULT_TOKEN"

# Fetch secrets
export APP_BASIC_AUTH=$(vault kv get -field=basic_auth secret/whatsapp)
export WHATSAPP_WEBHOOK_SECRET=$(vault kv get -field=webhook_secret secret/whatsapp)
export DB_URI=$(vault kv get -field=db_uri secret/whatsapp)

# Start application
./whatsapp rest
```

#### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: whatsapp-secrets
  namespace: whatsapp
type: Opaque
stringData:
  basic-auth: "admin:secret123"
  webhook-secret: "super-secret-key"
  db-password: "database-password"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: whatsapp-api
spec:
  template:
    spec:
      containers:
      - name: whatsapp
        env:
        - name: APP_BASIC_AUTH
          valueFrom:
            secretKeyRef:
              name: whatsapp-secrets
              key: basic-auth
        - name: WHATSAPP_WEBHOOK_SECRET
          valueFrom:
            secretKeyRef:
              name: whatsapp-secrets
              key: webhook-secret
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: whatsapp-secrets
              key: db-password
```

### Environment File Security

If using `.env` files:

```bash
# Set restrictive permissions
chmod 600 src/.env

# Never commit to git
echo ".env" >> .gitignore

# Use encryption for storing .env
gpg --encrypt --recipient your@email.com src/.env

# Decrypt when needed
gpg --decrypt src/.env.gpg > src/.env
```

## Network Security

### Firewall Configuration

#### UFW (Ubuntu/Debian)

```bash
# Default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS (if using reverse proxy)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Deny direct access to application ports
sudo ufw deny 3000/tcp
sudo ufw deny 8088/tcp

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status verbose
```

#### iptables

```bash
# Flush existing rules
sudo iptables -F

# Default policies
sudo iptables -P INPUT DROP
sudo iptables -P FORWARD DROP
sudo iptables -P OUTPUT ACCEPT

# Allow loopback
sudo iptables -A INPUT -i lo -j ACCEPT

# Allow established connections
sudo iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Allow SSH
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Allow HTTP/HTTPS
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Drop all other incoming
sudo iptables -A INPUT -j DROP

# Save rules
sudo iptables-save > /etc/iptables/rules.v4
```

### SSH Hardening

```bash
# Edit SSH config
sudo nano /etc/ssh/sshd_config

# Recommended settings:
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
X11Forwarding no
AllowUsers youruser
Port 2222  # Non-standard port

# Reload SSH
sudo systemctl reload sshd
```

### Network Isolation

Use Docker networks for isolation:

```yaml
version: '3.8'

services:
  whatsapp:
    networks:
      - internal
      - external
    # Only expose via reverse proxy

  postgres:
    networks:
      - internal
    # Not exposed externally

  nginx:
    networks:
      - external
    ports:
      - "80:80"
      - "443:443"

networks:
  internal:
    internal: true  # No external access
  external:
    driver: bridge
```

## System Hardening

### User Permissions

Run application as dedicated user:

```bash
# Create dedicated user
sudo useradd -r -s /bin/false whatsapp

# Create application directory
sudo mkdir -p /opt/whatsapp
sudo chown whatsapp:whatsapp /opt/whatsapp

# Set restrictive permissions
sudo chmod 755 /opt/whatsapp
sudo chmod 700 /opt/whatsapp/storages
```

### Systemd Security

Enhance systemd service security:

```ini
[Unit]
Description=WhatsApp Web API
After=network.target

[Service]
Type=simple
User=whatsapp
Group=whatsapp
WorkingDirectory=/opt/whatsapp

ExecStart=/opt/whatsapp/whatsapp rest

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/whatsapp/storages /opt/whatsapp/logs

# Resource limits
LimitNOFILE=10000
LimitNPROC=2048

# Restart policy
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### AppArmor Profile (Ubuntu)

Create `/etc/apparmor.d/whatsapp`:

```
#include <tunables/global>

/opt/whatsapp/whatsapp {
  #include <abstractions/base>

  # Allow execution
  /opt/whatsapp/whatsapp mr,

  # Allow storage access
  /opt/whatsapp/storages/ rw,
  /opt/whatsapp/storages/** rw,
  /opt/whatsapp/logs/ rw,
  /opt/whatsapp/logs/** rw,

  # Allow network access
  network inet stream,
  network inet6 stream,

  # Deny everything else
  deny /etc/** w,
  deny /home/** rw,
  deny /root/** rw,
}
```

Enable profile:

```bash
sudo aa-enforce /etc/apparmor.d/whatsapp
```

## Monitoring and Logging

### Security Logging

Log all security-relevant events:

```bash
# Create security log file
sudo mkdir -p /var/log/whatsapp
sudo touch /var/log/whatsapp/security.log
sudo chown whatsapp:whatsapp /var/log/whatsapp/security.log
sudo chmod 640 /var/log/whatsapp/security.log
```

### Failed Authentication Monitoring

Monitor failed authentication attempts:

```bash
# Monitor nginx logs for 401 errors
sudo tail -f /var/log/nginx/whatsapp_access.log | grep ' 401 '

# Count failed attempts
sudo grep ' 401 ' /var/log/nginx/whatsapp_access.log | wc -l
```

### Automated Security Alerts

```bash
#!/bin/bash
# security-monitor.sh

LOG_FILE="/var/log/nginx/whatsapp_access.log"
THRESHOLD=10
ALERT_EMAIL="security@yourcompany.com"

# Count 401 errors in last hour
COUNT=$(sudo grep ' 401 ' "$LOG_FILE" | \
  grep "$(date -d '1 hour ago' '+%d/%b/%Y:%H')" | \
  wc -l)

if [ "$COUNT" -gt "$THRESHOLD" ]; then
    echo "ALERT: $COUNT failed authentication attempts in the last hour" | \
    mail -s "WhatsApp API Security Alert" "$ALERT_EMAIL"
fi
```

Schedule with cron:

```bash
# Run every hour
0 * * * * /opt/whatsapp/security-monitor.sh
```

### Audit Logging

Enable audit logging for compliance:

```bash
# Install auditd
sudo apt install auditd

# Add audit rules
sudo auditctl -w /opt/whatsapp/storages/ -p wa -k whatsapp_db
sudo auditctl -w /opt/whatsapp/.env -p wa -k whatsapp_config

# View audit logs
sudo ausearch -k whatsapp_db
sudo ausearch -k whatsapp_config
```

## Security Checklist

### Pre-Deployment Security Checklist

- [ ] **Authentication**
  - [ ] Strong passwords configured (20+ characters)
  - [ ] Credentials stored in secrets manager
  - [ ] Different credentials per environment
  - [ ] Admin API token generated (32+ characters)

- [ ] **HTTPS/TLS**
  - [ ] SSL certificate obtained and installed
  - [ ] HTTP redirects to HTTPS
  - [ ] TLS 1.2+ enabled
  - [ ] Security headers configured
  - [ ] Certificate auto-renewal configured

- [ ] **Webhook Security**
  - [ ] HTTPS webhook URL configured
  - [ ] Strong webhook secret generated (32+ characters)
  - [ ] HMAC signature verification implemented
  - [ ] Webhook rate limiting enabled

- [ ] **Database Security**
  - [ ] Database credentials secured
  - [ ] Minimal permissions granted
  - [ ] SSL/TLS enabled for database connections
  - [ ] Regular backups configured
  - [ ] Backup encryption enabled

- [ ] **Network Security**
  - [ ] Firewall configured
  - [ ] Unnecessary ports closed
  - [ ] Application not directly exposed to internet
  - [ ] Reverse proxy configured
  - [ ] SSH hardened (key-only, non-standard port)

- [ ] **System Security**
  - [ ] Application running as dedicated user
  - [ ] File permissions restricted
  - [ ] System hardening applied (AppArmor/SELinux)
  - [ ] Resource limits configured
  - [ ] Security updates automated

- [ ] **Monitoring**
  - [ ] Security logging enabled
  - [ ] Failed authentication monitoring active
  - [ ] Automated alerts configured
  - [ ] Audit logging enabled (if required)
  - [ ] Log retention policy defined

### Post-Deployment Security Checklist

- [ ] **Testing**
  - [ ] SSL/TLS tested (SSL Labs, testssl.sh)
  - [ ] Authentication tested
  - [ ] Webhook signature verification tested
  - [ ] Rate limiting tested
  - [ ] Penetration testing completed

- [ ] **Documentation**
  - [ ] Incident response plan documented
  - [ ] Security contacts defined
  - [ ] Backup restoration procedure documented
  - [ ] Security policies communicated to team

- [ ] **Ongoing**
  - [ ] Security updates monitored and applied
  - [ ] Credentials rotated regularly
  - [ ] Logs reviewed regularly
  - [ ] Security audits scheduled
  - [ ] Backup restoration tested regularly

### Security Testing Tools

```bash
# Test SSL/TLS configuration
testssl.sh https://whatsapp.yourdomain.com

# Test security headers
curl -I https://whatsapp.yourdomain.com

# Test rate limiting
ab -n 1000 -c 10 https://whatsapp.yourdomain.com/send/message

# Scan for vulnerabilities
nmap -sV -sC whatsapp.yourdomain.com
```

## Incident Response

### Security Incident Procedure

1. **Detect**: Monitor logs and alerts
2. **Contain**: Isolate affected systems
3. **Investigate**: Determine scope and impact
4. **Remediate**: Fix vulnerabilities
5. **Recover**: Restore normal operations
6. **Learn**: Document and improve

### Emergency Response Commands

```bash
# Immediately block all traffic
sudo iptables -P INPUT DROP
sudo iptables -P FORWARD DROP

# Stop application
sudo systemctl stop whatsapp

# Backup current state for forensics
sudo tar czf /tmp/incident-$(date +%Y%m%d).tar.gz \
  /opt/whatsapp/storages/ \
  /var/log/whatsapp/ \
  /var/log/nginx/

# Review recent access logs
sudo tail -n 1000 /var/log/nginx/whatsapp_access.log

# Review authentication attempts
sudo grep ' 401 ' /var/log/nginx/whatsapp_access.log | tail -n 100
```

## Related Documentation

- **[Configuration Reference](../reference/configuration.md)** - Complete configuration options
- **[Webhook Security Guide](../guides/webhooks/security.md)** - Detailed webhook security
- **[Production Checklist](../guides/deployment/production-checklist.md)** - Production deployment
- **[Admin API Guide](../guides/admin-api.md)** - Multi-instance security

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-11-14

# Troubleshooting Guide

Comprehensive guide to diagnosing and resolving common issues with the WhatsApp Web API Multidevice application.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Connection Issues](#connection-issues)
- [Authentication Issues](#authentication-issues)
- [Message Sending Issues](#message-sending-issues)
- [Media Issues](#media-issues)
- [Webhook Issues](#webhook-issues)
- [Database Issues](#database-issues)
- [Performance Issues](#performance-issues)
- [Docker Issues](#docker-issues)
- [Common Error Messages](#common-error-messages)
- [Debug Mode](#debug-mode)
- [Health Checks](#health-checks)
- [Log Analysis](#log-analysis)
- [Getting Help](#getting-help)

## Quick Diagnostics

### First Steps Checklist

When experiencing issues, start with these basic checks:

**1. Check application status:**
```bash
# Is the application running?
ps aux | grep whatsapp

# Docker container status
docker ps | grep whatsapp

# Systemd service status
sudo systemctl status whatsapp
```

**2. Check connectivity:**
```bash
# Can you reach the application?
curl -I http://localhost:3000

# Check network connectivity
ping -c 4 whatsapp.com
```

**3. Check logs:**
```bash
# Application logs
tail -n 50 /var/log/whatsapp/app.log

# Docker logs
docker logs --tail 50 whatsapp-api

# Systemd logs
journalctl -u whatsapp -n 50
```

**4. Verify configuration:**
```bash
# Check environment variables
env | grep -E 'APP_|WHATSAPP_|DB_'

# Check .env file
cat src/.env
```

**5. Test health endpoint:**
```bash
# Check device connection status
curl http://localhost:3000/app/devices
```

### Quick Fixes

**Try these first:**

```bash
# 1. Restart application
sudo systemctl restart whatsapp
# OR
docker restart whatsapp-api

# 2. Reconnect to WhatsApp
curl http://localhost:3000/app/reconnect

# 3. Check disk space
df -h

# 4. Check memory usage
free -h

# 5. Clear media cache
rm -rf /app/statics/media/*
```

## Connection Issues

### Issue: Cannot Connect to WhatsApp

**Symptoms:**
- Application starts but shows "disconnected"
- QR code doesn't appear
- Messages fail to send

**Diagnostic Commands:**
```bash
# Check connection status
curl http://localhost:3000/app/devices

# Check logs for connection errors
grep -i "connect\|disconnect" /var/log/whatsapp/app.log

# Check network connectivity
ping web.whatsapp.com
```

**Common Causes & Solutions:**

**1. WhatsApp session expired**
```bash
# Solution: Logout and login again
curl http://localhost:3000/app/logout
curl http://localhost:3000/app/login
# Scan new QR code
```

**2. Network connectivity issues**
```bash
# Check DNS resolution
nslookup web.whatsapp.com

# Check firewall rules
sudo iptables -L -n | grep -i drop

# Test HTTPS connectivity
curl -I https://web.whatsapp.com
```

**3. Database corruption**
```bash
# Check database integrity
sqlite3 storages/whatsapp.db "PRAGMA integrity_check;"

# If corrupted, restore from backup
cp /backup/whatsapp.db storages/whatsapp.db
systemctl restart whatsapp
```

**4. Rate limiting or IP block**
```bash
# Wait 1-2 hours before retrying
# Use different IP if possible
# Check for excessive reconnection attempts in logs
grep -c "reconnect" /var/log/whatsapp/app.log
```

### Issue: Frequent Disconnections

**Symptoms:**
- Connection drops every few minutes
- Messages intermittently fail
- Constant reconnection attempts in logs

**Diagnostic Commands:**
```bash
# Monitor connection stability
watch -n 5 'curl -s http://localhost:3000/app/devices | jq'

# Check reconnection frequency
grep "reconnect\|disconnect" /var/log/whatsapp/app.log | tail -20

# Monitor network stability
ping -c 100 web.whatsapp.com | grep loss
```

**Solutions:**

**1. Enable auto-reconnect (should be enabled by default)**
```bash
# Auto-reconnect is enabled by default in v7.x+
# Check logs to verify it's working
grep "auto.*reconnect" /var/log/whatsapp/app.log
```

**2. Check network stability**
```bash
# Monitor for packet loss
mtr web.whatsapp.com

# Check for DNS issues
cat /etc/resolv.conf

# Use Google DNS if needed
echo "nameserver 8.8.8.8" | sudo tee -a /etc/resolv.conf
```

**3. Resource constraints**
```bash
# Check memory usage
free -h

# Check CPU usage
top -bn1 | grep whatsapp

# Increase memory if needed (Docker)
docker update --memory="2g" whatsapp-api
```

**4. WhatsApp rate limiting**
```bash
# Reduce message sending frequency
# Add delays between messages (2-3 seconds)
# Monitor for blocks in logs
grep -i "rate\|limit\|block" /var/log/whatsapp/app.log
```

### Issue: Remote Logout

**Symptoms:**
- Session ends unexpectedly
- "Logged out remotely" in logs
- Need to re-authenticate

**Common Causes:**
1. Logged out from mobile WhatsApp
2. Multiple instances using same account
3. WhatsApp security detection
4. Account compromised

**Solutions:**

**Immediate action:**
```bash
# Clear existing session
curl http://localhost:3000/app/logout

# Login again
curl http://localhost:3000/app/login

# Check no other instances are running
ps aux | grep whatsapp
docker ps | grep whatsapp
```

**Prevention:**
```bash
# Never run multiple instances with same account
# Ensure only one active session
# Enable debug mode to see detailed logs
./whatsapp rest --debug=true
```

## Authentication Issues

### Issue: 401 Unauthorized

**Symptoms:**
- API requests return 401 error
- "Unauthorized" message
- Cannot access any endpoints

**Diagnostic Commands:**
```bash
# Test without auth
curl -I http://localhost:3000/app/devices

# Test with auth
curl -I -u admin:password http://localhost:3000/app/devices

# Check auth configuration
env | grep APP_BASIC_AUTH
```

**Solutions:**

**1. Verify credentials**
```bash
# Check configured credentials
echo $APP_BASIC_AUTH

# Test with correct credentials
curl -u admin:correct-password http://localhost:3000/send/message
```

**2. Encode credentials correctly**
```bash
# Base64 encode for Authorization header
echo -n "admin:password" | base64

# Use in header
curl -H "Authorization: Basic $(echo -n admin:password | base64)" \
  http://localhost:3000/app/devices
```

**3. If basic auth not needed, disable it**
```bash
# Start without authentication
./whatsapp rest
# No --basic-auth or APP_BASIC_AUTH set
```

### Issue: Basic Auth Not Working

**Symptoms:**
- Credentials correct but still 401
- Authentication randomly fails
- Works in browser but not in code

**Common Causes:**
1. Incorrect credentials format
2. Special characters in password
3. URL encoding issues

**Solutions:**

**1. Test credentials format**
```bash
# Format: username:password
export APP_BASIC_AUTH="admin:mypassword"

# Multiple users format
export APP_BASIC_AUTH="admin:pass1,user:pass2"

# Restart application
systemctl restart whatsapp
```

**2. Special characters in password**
```bash
# Use URL encoding for special characters
# Or use simple alphanumeric passwords

# Generate simple password
openssl rand -base64 32 | tr -dc 'a-zA-Z0-9'
```

**3. Test with curl**
```bash
# Method 1: -u flag
curl -u admin:password http://localhost:3000/app/devices

# Method 2: Header
curl -H "Authorization: Basic $(echo -n admin:password | base64)" \
  http://localhost:3000/app/devices
```

## Message Sending Issues

### Issue: Messages Not Sending

**Symptoms:**
- API returns success but message not delivered
- Recipient doesn't receive message
- No error in response

**Diagnostic Commands:**
```bash
# Check connection status
curl http://localhost:3000/app/devices

# Verify recipient exists
curl "http://localhost:3000/user/check?phone=5511999998888"

# Check recent errors
grep -i "error\|fail" /var/log/whatsapp/app.log | tail -20
```

**Solutions:**

**1. Verify phone number format**
```bash
# Correct format (no +, no spaces, no dashes)
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Test"
  }'

# See phone-number-format.md for details
```

**2. Check if user is on WhatsApp**
```bash
# Validate user exists
curl "http://localhost:3000/user/check?phone=5511999998888"

# If exists: true, user is valid
# If exists: false, number not on WhatsApp
```

**3. Disable account validation (not recommended)**
```bash
# Only if you're sure number is valid
export WHATSAPP_ACCOUNT_VALIDATION=false
./whatsapp rest
```

**4. Check for rate limiting**
```bash
# Add delays between messages
for phone in 5511999998888 5511999997777; do
  curl -X POST http://localhost:3000/send/message \
    -H "Content-Type: application/json" \
    -d "{\"phone\": \"$phone\", \"message\": \"Hello\"}"
  sleep 2  # Wait 2 seconds
done
```

### Issue: Invalid Phone Number Error

**Symptoms:**
- "Invalid JID" error
- "Phone number format invalid"
- Message send fails immediately

**Common Causes:**
1. Missing country code
2. Contains formatting characters
3. Starts with zero
4. Wrong length

**Solutions:**

**Check phone number:**
```bash
# Invalid formats
❌ "+55 11 99999-8888"  # Has formatting
❌ "011999998888"        # Missing country code
❌ "+5511999998888"      # Has + sign
❌ "0055119999988 88"     # Has leading zeros

# Valid formats
✅ "5511999998888"       # Correct
✅ "5511999998888@s.whatsapp.net"  # Also correct
```

**Validation script:**
```bash
#!/bin/bash
# validate-phone.sh

PHONE="$1"
CLEAN=$(echo "$PHONE" | tr -cd '0-9')

echo "Original: $PHONE"
echo "Cleaned:  $CLEAN"
echo "Length:   ${#CLEAN}"

if [ ${#CLEAN} -lt 10 ] || [ ${#CLEAN} -gt 15 ]; then
    echo "❌ Invalid length (must be 10-15 digits)"
    exit 1
fi

if [[ "$CLEAN" =~ ^0 ]]; then
    echo "❌ Cannot start with 0"
    exit 1
fi

echo "✅ Valid format"
```

See [Phone Number Format Guide](phone-number-format.md) for complete details.

### Issue: Message Appears Sent But Not Delivered

**Symptoms:**
- API returns success
- Message ID returned
- Webhook shows "sent" but not "delivered"

**Diagnostic Steps:**

**1. Setup webhook to track status:**
```bash
# Configure webhook
export WHATSAPP_WEBHOOK="https://your-webhook.site/handler"
./whatsapp rest

# Monitor for delivery events
# message.sent → message.delivered → message.read
```

**2. Check recipient status:**
```bash
# Verify number is active
curl "http://localhost:3000/user/check?phone=5511999998888"

# Check user info
curl "http://localhost:3000/user/info?phone=5511999998888"
```

**3. Common reasons for non-delivery:**
- Recipient blocked you
- Recipient's phone is off/no internet
- Recipient has WhatsApp notifications disabled
- Network issues between you and recipient

**4. Test with your own number:**
```bash
# Send to yourself to verify system works
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "YOUR_NUMBER",
    "message": "Test message"
  }'
```

## Media Issues

### Issue: Media Not Sending

**Symptoms:**
- Images/videos fail to send
- "FFmpeg not found" error
- Media processing timeout

**Diagnostic Commands:**
```bash
# Check FFmpeg installation
ffmpeg -version

# Check FFmpeg in PATH
which ffmpeg

# Test media processing
ffmpeg -i test.jpg -q:v 5 output.jpg

# Check media storage permissions
ls -la /app/statics/media/
```

**Solutions:**

**1. Install FFmpeg**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Verify installation
ffmpeg -version
```

**2. Fix PATH issues**
```bash
# Add FFmpeg to PATH
export PATH=$PATH:/usr/local/bin

# Make permanent
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc
```

**3. Check file size limits**
```bash
# Check current limits
env | grep MAX_.*_SIZE

# Increase limits if needed
export MAX_IMAGE_SIZE=20971520    # 20MB
export MAX_VIDEO_SIZE=104857600   # 100MB
export MAX_FILE_SIZE=52428800     # 50MB

./whatsapp rest
```

**4. Verify media storage permissions**
```bash
# Create media directory
mkdir -p /app/statics/media

# Set permissions
chmod 755 /app/statics/media
chown -R whatsapp:whatsapp /app/statics

# Check disk space
df -h /app/statics
```

### Issue: Image Compression Fails

**Symptoms:**
- Images send as documents
- No compression applied
- Original size sent

**Solutions:**

**1. Enable compression**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/image.jpg",
    "compress": true
  }'
```

**2. Check FFmpeg codecs**
```bash
# Verify JPEG support
ffmpeg -codecs | grep mjpeg

# Should show:
# DEV.LS mjpeg       Motion JPEG
```

**3. Pre-compress images**
```bash
# Manual compression with FFmpeg
ffmpeg -i large.jpg -q:v 5 -s 1920x1920 compressed.jpg

# Or use ImageMagick
convert large.jpg -quality 75 -resize 1920x1920\> compressed.jpg
```

### Issue: Video Processing Timeout

**Symptoms:**
- Large videos fail to send
- Processing takes too long
- Timeout errors

**Solutions:**

**1. Reduce video size before sending**
```bash
# Compress video with FFmpeg
ffmpeg -i large.mp4 \
  -vcodec h264 -crf 28 \
  -acodec aac -b:a 128k \
  -vf scale=1280:-2 \
  compressed.mp4

# Check file size
ls -lh compressed.mp4
```

**2. Disable compression for small files**
```bash
# If video is already optimized
curl -X POST http://localhost:3000/send/video \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "video": "https://example.com/video.mp4",
    "compress": false
  }'
```

**3. Split long videos**
```bash
# Split into 5-minute segments
ffmpeg -i long.mp4 -ss 00:00:00 -t 00:05:00 part1.mp4
ffmpeg -i long.mp4 -ss 00:05:00 -t 00:05:00 part2.mp4
```

### Issue: Audio Conversion Fails

**Symptoms:**
- Audio files won't send
- "Opus codec not found"
- Audio appears as document

**Solutions:**

**1. Check Opus support**
```bash
# Verify Opus codec available
ffmpeg -codecs | grep opus

# Should show:
# DEA.L. opus      Opus
```

**2. Install Opus library**
```bash
# Ubuntu/Debian
sudo apt install libopus-dev ffmpeg

# Rebuild FFmpeg if needed
sudo apt install --reinstall ffmpeg
```

**3. Disable auto-conversion temporarily**
```bash
# If issues persist
export WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=false
./whatsapp rest

# Send audio without conversion
curl -X POST http://localhost:3000/send/audio \
  -F "phone=5511999998888" \
  -F "audio=@audio.mp3"
```

**4. Manual conversion**
```bash
# Convert MP3 to Opus manually
ffmpeg -i audio.mp3 -c:a libopus -b:a 64k audio.ogg

# Send converted file
curl -X POST http://localhost:3000/send/audio \
  -F "phone=5511999998888" \
  -F "audio=@audio.ogg"
```

## Webhook Issues

### Issue: Webhooks Not Receiving Events

**Symptoms:**
- Webhook endpoint not being called
- No webhook events in logs
- Messages received but webhook silent

**Diagnostic Commands:**
```bash
# Check webhook configuration
env | grep WEBHOOK

# Test webhook URL manually
curl -X POST https://your-webhook.com/handler \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# Check application logs for webhook attempts
grep -i "webhook" /var/log/whatsapp/app.log | tail -20
```

**Solutions:**

**1. Verify webhook URL is accessible**
```bash
# Test from application server
curl -I https://your-webhook.com/handler

# Should return 200 or 405 (Method Not Allowed is OK for GET)
```

**2. Check firewall/network**
```bash
# Allow outbound HTTPS
sudo ufw allow out 443/tcp

# Check network connectivity
traceroute your-webhook.com
```

**3. Use webhook testing service**
```bash
# Use webhook.site for testing
export WHATSAPP_WEBHOOK="https://webhook.site/your-unique-id"
./whatsapp rest

# Send test message and check webhook.site
```

**4. Verify webhook is configured**
```bash
# Check configuration
curl http://localhost:3000/app/devices

# Logs should show webhook URL if configured
grep "webhook" /var/log/whatsapp/app.log
```

**5. Enable debug mode**
```bash
# See detailed webhook activity
./whatsapp rest --debug=true

# Check logs for webhook delivery
tail -f /var/log/whatsapp/app.log | grep webhook
```

### Issue: Webhook Signature Verification Fails

**Symptoms:**
- Webhook called but returns 401
- "Invalid signature" errors
- HMAC verification fails

**Diagnostic Steps:**

**1. Verify secret matches**
```bash
# Application side
echo $WHATSAPP_WEBHOOK_SECRET

# Webhook handler side
echo $WEBHOOK_SECRET

# Must be identical
```

**2. Check signature calculation**
```javascript
// Node.js - verify signature correctly
const crypto = require('crypto');

app.use(express.raw({type: 'application/json'})); // Important!

app.post('/webhook', (req, res) => {
    const signature = req.headers['x-hub-signature-256'];
    const payload = req.body; // Raw buffer

    const expectedSig = crypto
        .createHmac('sha256', process.env.WEBHOOK_SECRET)
        .update(payload)
        .digest('hex');

    const receivedSig = signature.replace('sha256=', '');

    if (expectedSig !== receivedSig) {
        console.log('Expected:', expectedSig);
        console.log('Received:', receivedSig);
        return res.status(401).send('Invalid signature');
    }

    // Process webhook
    res.status(200).send('OK');
});
```

**3. Common mistakes:**
```javascript
// ❌ Wrong - using parsed JSON
app.use(express.json());
const payload = JSON.stringify(req.body); // Signature will fail

// ✅ Correct - using raw body
app.use(express.raw({type: 'application/json'}));
const payload = req.body; // Buffer
```

**4. Debug signature mismatch:**
```bash
# Enable debug mode
./whatsapp rest --debug=true

# Check webhook payload and signature in logs
grep "webhook.*signature" /var/log/whatsapp/app.log
```

See [Webhook Security Guide](../guides/webhooks/security.md) for complete implementation.

### Issue: Webhook Timeouts

**Symptoms:**
- Webhook returns 503/504
- Timeout errors in logs
- Webhook retries exhausted

**Solutions:**

**1. Increase endpoint timeout**
```javascript
// Node.js Express
app.post('/webhook', (req, res) => {
    // Send 200 immediately
    res.status(200).send('OK');

    // Process asynchronously
    setImmediate(() => {
        processWebhook(req.body);
    });
});
```

**2. Optimize webhook processing**
```javascript
// Process heavy operations in background
app.post('/webhook', async (req, res) => {
    res.status(200).send('OK');

    // Queue for background processing
    await jobQueue.add('webhook-processing', req.body);
});
```

**3. Check webhook endpoint performance**
```bash
# Test response time
time curl -X POST https://your-webhook.com/handler \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# Should respond in < 5 seconds
```

## Database Issues

### Issue: Database Locked

**Symptoms:**
- "Database is locked" error
- Operations timeout
- SQLite busy errors

**Diagnostic Commands:**
```bash
# Check if database is in use
lsof storages/whatsapp.db

# Check for deadlocks
sqlite3 storages/whatsapp.db "PRAGMA wal_checkpoint;"

# Verify database integrity
sqlite3 storages/whatsapp.db "PRAGMA integrity_check;"
```

**Solutions:**

**1. Stop application and check database**
```bash
# Stop application
systemctl stop whatsapp

# Check database integrity
sqlite3 storages/whatsapp.db "PRAGMA integrity_check;"

# If OK, restart
systemctl start whatsapp
```

**2. Enable WAL mode**
```bash
# Write-Ahead Logging improves concurrency
export DB_URI="file:storages/whatsapp.db?_journal_mode=WAL&_timeout=5000"
./whatsapp rest
```

**3. Migrate to PostgreSQL (recommended for production)**
```bash
# Install PostgreSQL
sudo apt install postgresql

# Create database
sudo -u postgres createdb whatsapp

# Configure application
export DB_URI="postgresql://user:pass@localhost/whatsapp"
./whatsapp rest
```

### Issue: Database Corruption

**Symptoms:**
- "Database disk image is malformed"
- Integrity check fails
- Application crashes on startup

**Recovery Steps:**

**1. Stop application immediately**
```bash
systemctl stop whatsapp
```

**2. Backup corrupted database**
```bash
cp storages/whatsapp.db storages/whatsapp.db.corrupted
cp storages/chatstorage.db storages/chatstorage.db.corrupted
```

**3. Try recovery**
```bash
# Attempt SQLite recovery
sqlite3 storages/whatsapp.db.corrupted ".recover" | \
  sqlite3 storages/whatsapp.db.recovered

# Check integrity
sqlite3 storages/whatsapp.db.recovered "PRAGMA integrity_check;"

# If OK, use recovered database
mv storages/whatsapp.db.recovered storages/whatsapp.db
```

**4. Restore from backup**
```bash
# If recovery fails, restore latest backup
cp /backup/whatsapp/whatsapp.db storages/whatsapp.db
```

**5. Last resort: Start fresh**
```bash
# Warning: Loses WhatsApp session
rm storages/whatsapp.db
systemctl start whatsapp
# Need to login again
```

### Issue: Chat Storage Growing Too Large

**Symptoms:**
- Database file very large (> 1GB)
- Slow queries
- Disk space issues

**Solutions:**

**1. Check database size**
```bash
du -h storages/chatstorage.db
```

**2. Disable chat storage if not needed**
```bash
export WHATSAPP_CHAT_STORAGE=false
./whatsapp rest
```

**3. Clean old messages**
```bash
# Backup first
cp storages/chatstorage.db storages/chatstorage.db.backup

# Delete messages older than 30 days
sqlite3 storages/chatstorage.db "
DELETE FROM messages
WHERE timestamp < datetime('now', '-30 days');
"

# Vacuum to reclaim space
sqlite3 storages/chatstorage.db "VACUUM;"

# Check new size
du -h storages/chatstorage.db
```

**4. Regular maintenance**
```bash
# Create cleanup script
cat > /opt/whatsapp/cleanup-chat.sh <<'EOF'
#!/bin/bash
sqlite3 /opt/whatsapp/storages/chatstorage.db "
DELETE FROM messages WHERE timestamp < datetime('now', '-30 days');
VACUUM;
"
EOF

chmod +x /opt/whatsapp/cleanup-chat.sh

# Schedule monthly cleanup
echo "0 2 1 * * /opt/whatsapp/cleanup-chat.sh" | crontab -
```

## Performance Issues

### Issue: High Memory Usage

**Symptoms:**
- Memory usage > 2GB
- Out of memory errors
- System slowdown

**Diagnostic Commands:**
```bash
# Check memory usage
free -h

# Application memory usage
ps aux | grep whatsapp | awk '{print $6/1024 " MB"}'

# Docker memory usage
docker stats whatsapp-api --no-stream
```

**Solutions:**

**1. Disable chat storage**
```bash
export WHATSAPP_CHAT_STORAGE=false
./whatsapp rest
```

**2. Clear media cache**
```bash
# Remove temporary media files
rm -rf /app/statics/media/*

# Remove old QR codes
find /app/statics/images/qrcode -mtime +1 -delete
```

**3. Optimize database**
```bash
# Vacuum database
sqlite3 storages/whatsapp.db "VACUUM;"
sqlite3 storages/chatstorage.db "VACUUM;"
```

**4. Set memory limits (Docker)**
```bash
# Limit to 1GB
docker run --memory="1g" --memory-swap="1g" \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest
```

**5. Restart periodically**
```bash
# Schedule daily restart (2 AM)
echo "0 2 * * * systemctl restart whatsapp" | crontab -
```

### Issue: High CPU Usage

**Symptoms:**
- CPU usage > 80%
- System unresponsive
- Slow API responses

**Diagnostic Commands:**
```bash
# Check CPU usage
top -bn1 | grep whatsapp

# Application threads
ps -eLf | grep whatsapp | wc -l

# System load
uptime
```

**Solutions:**

**1. Check for infinite loops in logs**
```bash
grep -i "error\|panic\|loop" /var/log/whatsapp/app.log | tail -50
```

**2. Reduce media processing**
```bash
# Disable compression temporarily
export WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=false

# Reduce concurrent operations
# Send messages with delays
```

**3. Limit concurrent connections**
```bash
# Nginx rate limiting
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
```

**4. Optimize FFmpeg usage**
```bash
# Use hardware acceleration if available
ffmpeg -hwaccel auto -i input.mp4 output.mp4
```

### Issue: Slow API Responses

**Symptoms:**
- Requests take > 5 seconds
- Timeout errors
- Poor user experience

**Diagnostic Steps:**

**1. Measure response time**
```bash
# Test send message endpoint
time curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{"phone":"5511999998888","message":"Test"}'

# Should complete in < 2 seconds
```

**2. Check database performance**
```bash
# PostgreSQL slow queries
# /etc/postgresql/14/main/postgresql.conf
log_min_duration_statement = 1000  # Log queries > 1s

# Check slow query log
tail -f /var/log/postgresql/postgresql-14-main.log
```

**3. Enable connection pooling (PostgreSQL)**
```bash
# Install PgBouncer
sudo apt install pgbouncer

# Configure
# /etc/pgbouncer/pgbouncer.ini
[databases]
whatsapp = host=localhost port=5432 dbname=whatsapp

# Use pooler
export DB_URI="postgresql://user:pass@localhost:6432/whatsapp"
```

**4. Add database indexes**
```sql
-- PostgreSQL optimization
CREATE INDEX IF NOT EXISTS idx_messages_timestamp
  ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_chats_jid
  ON chats(jid);

ANALYZE messages;
ANALYZE chats;
```

**5. Optimize media sizes**
```bash
# Reduce max sizes
export MAX_IMAGE_SIZE=10485760   # 10MB instead of 20MB
export MAX_VIDEO_SIZE=52428800   # 50MB instead of 100MB
```

## Docker Issues

### Issue: Container Exits Immediately

**Symptoms:**
- Container starts then stops
- No logs visible
- Exit code 1 or 139

**Diagnostic Commands:**
```bash
# Check container status
docker ps -a | grep whatsapp

# View container logs
docker logs whatsapp-api

# Check exit code
docker inspect whatsapp-api | grep ExitCode
```

**Solutions:**

**1. Run in foreground to see errors**
```bash
# Remove -d flag
docker run -it --rm \
  -p 3000:3000 \
  -e APP_DEBUG=true \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest
```

**2. Check volume permissions**
```bash
# Inspect volume
docker run -it --rm \
  -v whatsapp-data:/data \
  alpine ls -la /data

# Fix permissions if needed
docker run -it --rm \
  -v whatsapp-data:/data \
  alpine chown -R 1000:1000 /data
```

**3. Check command syntax**
```bash
# Verify command format
docker run ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest --debug=true
#                                                         ^^^^ mode required
```

**4. Check environment variables**
```bash
# Verify env vars
docker run -it --rm \
  -e APP_DEBUG=true \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice \
  sh -c 'env | grep APP'
```

### Issue: Cannot Access Container Port

**Symptoms:**
- Container running but port not accessible
- Connection refused errors
- Cannot reach http://localhost:3000

**Solutions:**

**1. Check port mapping**
```bash
# Verify port is mapped
docker ps | grep whatsapp
# Should show: 0.0.0.0:3000->3000/tcp

# Test from host
curl http://localhost:3000/app/devices
```

**2. Check container IP**
```bash
# Get container IP
docker inspect whatsapp-api | grep IPAddress

# Test container directly
curl http://172.17.0.2:3000/app/devices
```

**3. Check firewall**
```bash
# Allow port 3000
sudo ufw allow 3000/tcp

# Or use different port
docker run -p 8080:3000 ...
```

**4. Check application binding**
```bash
# Application should bind to 0.0.0.0, not 127.0.0.1
docker exec whatsapp-api netstat -tlnp | grep 3000

# Should show: 0.0.0.0:3000
```

### Issue: Volume Permissions

**Symptoms:**
- "Permission denied" errors
- Cannot write to database
- Media upload fails

**Solutions:**

**1. Check volume ownership**
```bash
# List volume contents
docker run -it --rm \
  -v whatsapp-data:/data \
  alpine ls -la /data

# Fix ownership
docker run -it --rm \
  -v whatsapp-data:/data \
  alpine sh -c "chown -R 1000:1000 /data && chmod -R 755 /data"
```

**2. Use bind mounts with correct permissions**
```bash
# Create directory with correct ownership
mkdir -p ./storages
sudo chown -R 1000:1000 ./storages
chmod -R 755 ./storages

# Mount with bind
docker run -v $(pwd)/storages:/app/storages ...
```

**3. Run as specific user**
```bash
docker run --user 1000:1000 \
  -v $(pwd)/storages:/app/storages \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest
```

## Common Error Messages

### "Invalid JID" Error

**Error:**
```
{"code":"INVALID_JID","message":"your JID is invalid"}
```

**Cause:** Phone number format incorrect

**Solution:**
```bash
# Fix phone number format
# Remove: +, spaces, dashes, parentheses
# Include: country code

# Wrong: +55 11 99999-8888
# Right: 5511999998888
```

See [Phone Number Format](phone-number-format.md) for details.

### "User is not registered" Error

**Error:**
```
{"code":"INVALID_JID","message":"user is not registered"}
```

**Cause:** Number doesn't have WhatsApp or wrong format

**Solution:**
```bash
# Verify number exists on WhatsApp
curl "http://localhost:3000/user/check?phone=5511999998888"

# Try with/without leading 9 (Brazil mobile)
curl "http://localhost:3000/user/check?phone=5511999998888"
curl "http://localhost:3000/user/check?phone=551199998888"
```

### "FFmpeg not found" Error

**Error:**
```
exec: "ffmpeg": executable file not found in $PATH
```

**Cause:** FFmpeg not installed or not in PATH

**Solution:**
```bash
# Install FFmpeg
sudo apt install ffmpeg  # Ubuntu/Debian
brew install ffmpeg      # macOS

# Verify
ffmpeg -version

# Add to PATH if needed
export PATH=$PATH:/usr/local/bin
```

### "Database is locked" Error

**Error:**
```
database is locked (code: 5)
```

**Cause:** Multiple processes accessing SQLite database

**Solution:**
```bash
# Stop all instances
systemctl stop whatsapp
killall whatsapp

# Enable WAL mode
export DB_URI="file:storages/whatsapp.db?_journal_mode=WAL&_timeout=5000"

# Or migrate to PostgreSQL
export DB_URI="postgresql://user:pass@localhost/whatsapp"
```

### "Connection timeout" Error

**Error:**
```
dial tcp: i/o timeout
```

**Cause:** Cannot reach WhatsApp servers

**Solution:**
```bash
# Check internet connectivity
ping web.whatsapp.com

# Check DNS
nslookup web.whatsapp.com

# Check firewall
sudo iptables -L OUTPUT -n

# Try different DNS
echo "nameserver 8.8.8.8" | sudo tee -a /etc/resolv.conf
```

### "Out of memory" Error

**Error:**
```
fatal error: out of memory
```

**Cause:** Insufficient memory

**Solution:**
```bash
# Increase Docker memory
docker update --memory="2g" whatsapp-api

# Or disable chat storage
export WHATSAPP_CHAT_STORAGE=false

# Clear media cache
rm -rf /app/statics/media/*

# Restart application
systemctl restart whatsapp
```

### "Profile picture panic" Error

**Error:**
```
panic: unsupported payload type: *store.PrivacyToken
```

**Cause:** Outdated whatsmeow library

**Solution:**
```bash
# Update to latest version
# Check current version (should be v7.10.1 or later)
./whatsapp --version

# Download latest release
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-amd64

# Or rebuild from source
cd src && go get -u go.mau.fi/whatsmeow && go build
```

See [Postmortem 001](../postmortems/001-profile-picture-panic.md) for details.

## Debug Mode

### Enabling Debug Mode

**Method 1: Command line**
```bash
./whatsapp rest --debug=true
```

**Method 2: Environment variable**
```bash
export APP_DEBUG=true
./whatsapp rest
```

**Method 3: .env file**
```bash
echo "APP_DEBUG=true" >> src/.env
./whatsapp rest
```

**Docker:**
```bash
docker run -e APP_DEBUG=true \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest
```

### Debug Mode Output

**What debug mode shows:**
- Detailed connection logs
- Message send/receive details
- Webhook delivery attempts
- Database queries
- Media processing steps
- Error stack traces

**Example debug output:**
```
DEBUG[2025-11-14T10:30:00Z] Message send requested  phone=5511999998888
DEBUG[2025-11-14T10:30:00Z] Validating recipient...
DEBUG[2025-11-14T10:30:01Z] User exists on WhatsApp  jid=5511999998888@s.whatsapp.net
DEBUG[2025-11-14T10:30:01Z] Sending message...
DEBUG[2025-11-14T10:30:02Z] Message sent successfully  message_id=3EB0C431
DEBUG[2025-11-14T10:30:02Z] Calling webhook...  url=https://webhook.site/xxx
DEBUG[2025-11-14T10:30:03Z] Webhook delivered  status=200
```

### Filtering Debug Logs

**View only errors:**
```bash
./whatsapp rest --debug=true 2>&1 | grep ERROR
```

**View only webhook activity:**
```bash
./whatsapp rest --debug=true 2>&1 | grep -i webhook
```

**View connection events:**
```bash
./whatsapp rest --debug=true 2>&1 | grep -i "connect\|disconnect"
```

**Save debug logs:**
```bash
./whatsapp rest --debug=true > /var/log/whatsapp/debug.log 2>&1
```

### Debug Mode Performance Impact

**Warning:** Debug mode generates significant log output and may:
- Slow down application (5-10%)
- Use more disk space
- Expose sensitive information in logs

**Recommendations:**
- Use only for troubleshooting
- Don't enable in production permanently
- Rotate logs frequently
- Secure log files (chmod 600)

## Health Checks

### Quick Health Check

```bash
# Check application status
curl http://localhost:3000/app/devices

# Expected response:
{
  "code": "SUCCESS",
  "message": "Success",
  "results": [{
    "device": "Chrome",
    "platform": "linux",
    "connected": true
  }]
}
```

### Comprehensive Health Check Script

```bash
#!/bin/bash
# health-check.sh

echo "=== WhatsApp API Health Check ==="

# 1. Application running?
if pgrep -x "whatsapp" > /dev/null; then
    echo "✅ Application is running"
else
    echo "❌ Application is NOT running"
    exit 1
fi

# 2. Port accessible?
if curl -s -f http://localhost:3000/app/devices > /dev/null; then
    echo "✅ Port 3000 is accessible"
else
    echo "❌ Port 3000 is NOT accessible"
    exit 1
fi

# 3. WhatsApp connected?
CONNECTED=$(curl -s http://localhost:3000/app/devices | jq -r '.results[0].connected')
if [ "$CONNECTED" = "true" ]; then
    echo "✅ WhatsApp is connected"
else
    echo "❌ WhatsApp is NOT connected"
    exit 1
fi

# 4. Database accessible?
if [ -f "storages/whatsapp.db" ]; then
    if sqlite3 storages/whatsapp.db "PRAGMA quick_check;" > /dev/null 2>&1; then
        echo "✅ Database is healthy"
    else
        echo "❌ Database has issues"
        exit 1
    fi
else
    echo "❌ Database file not found"
    exit 1
fi

# 5. Disk space?
DISK_USAGE=$(df /app/storages | awk 'NR==2 {print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -lt 90 ]; then
    echo "✅ Disk space OK ($DISK_USAGE% used)"
else
    echo "⚠️  Disk space low ($DISK_USAGE% used)"
fi

# 6. Memory usage?
MEM_USAGE=$(free | awk 'NR==2 {printf "%.0f", $3/$2*100}')
if [ "$MEM_USAGE" -lt 90 ]; then
    echo "✅ Memory usage OK ($MEM_USAGE% used)"
else
    echo "⚠️  Memory usage high ($MEM_USAGE% used)"
fi

echo "=== Health check complete ==="
```

### Automated Health Monitoring

**With systemd:**
```bash
# Run health check every 5 minutes
cat > /etc/systemd/system/whatsapp-health.service <<'EOF'
[Unit]
Description=WhatsApp API Health Check

[Service]
Type=oneshot
ExecStart=/opt/whatsapp/health-check.sh
EOF

cat > /etc/systemd/system/whatsapp-health.timer <<'EOF'
[Unit]
Description=Run WhatsApp health check every 5 minutes

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min

[Install]
WantedBy=timers.target
EOF

systemctl enable --now whatsapp-health.timer
```

**With cron:**
```bash
# Add to crontab
*/5 * * * * /opt/whatsapp/health-check.sh || echo "WhatsApp health check failed" | mail -s "Alert" admin@example.com
```

### Monitoring Integration

**Prometheus (if available):**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'whatsapp'
    static_configs:
      - targets: ['localhost:3000']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

**Uptime monitoring services:**
```bash
# Configure with UptimeRobot, Pingdom, or StatusCake
# Monitor: http://your-domain.com/app/devices
# Expected: HTTP 200 response
```

## Log Analysis

### Log Locations

**Binary installation:**
```bash
# Application logs
/var/log/whatsapp/app.log

# System logs
journalctl -u whatsapp

# Nginx logs
/var/log/nginx/whatsapp_access.log
/var/log/nginx/whatsapp_error.log
```

**Docker:**
```bash
# Container logs
docker logs whatsapp-api

# Follow logs
docker logs -f whatsapp-api

# Last 100 lines
docker logs --tail 100 whatsapp-api

# Since specific time
docker logs --since 1h whatsapp-api
```

### Common Log Patterns

**Find errors:**
```bash
grep -i "error\|fail\|panic" /var/log/whatsapp/app.log | tail -20
```

**Find connection issues:**
```bash
grep -i "connect\|disconnect\|timeout" /var/log/whatsapp/app.log | tail -20
```

**Find webhook failures:**
```bash
grep -i "webhook.*fail\|webhook.*error" /var/log/whatsapp/app.log | tail -20
```

**Find authentication failures:**
```bash
grep " 401 " /var/log/nginx/whatsapp_access.log | tail -20
```

**Count message sends:**
```bash
grep "message.*sent" /var/log/whatsapp/app.log | wc -l
```

### Log Analysis Script

```bash
#!/bin/bash
# analyze-logs.sh

LOG_FILE="/var/log/whatsapp/app.log"

echo "=== Log Analysis ==="
echo ""

# Error count
ERROR_COUNT=$(grep -c "ERROR" "$LOG_FILE")
echo "Total errors: $ERROR_COUNT"

# Warning count
WARN_COUNT=$(grep -c "WARN" "$LOG_FILE")
echo "Total warnings: $WARN_COUNT"

# Connection events
DISCONNECT_COUNT=$(grep -c "disconnect" "$LOG_FILE")
RECONNECT_COUNT=$(grep -c "reconnect" "$LOG_FILE")
echo "Disconnections: $DISCONNECT_COUNT"
echo "Reconnections: $RECONNECT_COUNT"

# Message stats
MSG_SENT=$(grep -c "message.*sent" "$LOG_FILE")
MSG_FAILED=$(grep -c "message.*fail" "$LOG_FILE")
echo "Messages sent: $MSG_SENT"
echo "Messages failed: $MSG_FAILED"

# Top errors
echo ""
echo "Top 5 errors:"
grep "ERROR" "$LOG_FILE" | awk -F'] ' '{print $2}' | sort | uniq -c | sort -rn | head -5

# Recent errors
echo ""
echo "Recent errors (last 10):"
grep "ERROR" "$LOG_FILE" | tail -10
```

### Log Rotation

**Configure logrotate:**
```bash
# Create /etc/logrotate.d/whatsapp
cat > /etc/logrotate.d/whatsapp <<'EOF'
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
EOF
```

**Test log rotation:**
```bash
# Test configuration
logrotate -d /etc/logrotate.d/whatsapp

# Force rotation
logrotate -f /etc/logrotate.d/whatsapp
```

## Getting Help

### Before Asking for Help

**Gather this information:**

1. **Version information:**
```bash
./whatsapp --version
# Or check logs
grep "version" /var/log/whatsapp/app.log | head -1
```

2. **Environment details:**
```bash
# OS version
uname -a
lsb_release -a

# Docker version (if using)
docker --version

# Go version (if building from source)
go version
```

3. **Configuration:**
```bash
# Sanitized environment (remove secrets)
env | grep -E 'APP_|WHATSAPP_|DB_' | sed 's/=.*/=***/'
```

4. **Recent logs:**
```bash
# Last 50 log lines
tail -n 50 /var/log/whatsapp/app.log

# Or for Docker
docker logs --tail 50 whatsapp-api
```

5. **Error messages:**
```bash
# Any errors in logs
grep -i "error\|panic\|fatal" /var/log/whatsapp/app.log | tail -10
```

### Where to Get Help

**1. Check Documentation:**
- [Getting Started Guide](../getting-started/)
- [Deployment Guides](../guides/deployment/)
- [API Documentation](api/openapi.md)
- [Postmortems](../postmortems/) - Known issues

**2. Search GitHub Issues:**
- [Existing Issues](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues)
- [Closed Issues](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues?q=is%3Aissue+is%3Aclosed)

**3. Create GitHub Issue:**
- [New Issue](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues/new)
- Include all information from "Before Asking for Help" section
- Use issue template if available

**4. Community Discussions:**
- [GitHub Discussions](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/discussions)

### Issue Template

When creating an issue, use this template:

```markdown
## Issue Description
Brief description of the problem

## Environment
- OS: Ubuntu 22.04
- Installation Method: Docker / Binary / Source
- Application Version: v7.10.1
- Go Version (if applicable): 1.24.0
- Docker Version (if applicable): 24.0.0

## Configuration
```bash
APP_PORT=3000
APP_DEBUG=false
WHATSAPP_WEBHOOK=https://***
DB_URI=file:storages/whatsapp.db
```

## Steps to Reproduce
1. Start application with...
2. Send message to...
3. Error occurs...

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Logs
```
[Paste relevant log lines]
```

## Screenshots (if applicable)
[Attach screenshots]

## Additional Context
Any other relevant information
```

## Related Documentation

- **[Phone Number Format](phone-number-format.md)** - JID format guide
- **[Configuration Reference](configuration.md)** - All configuration options
- **[Monitoring Guide](../operations/monitoring.md)** - Monitoring and alerts
- **[Security Best Practices](../operations/security-best-practices.md)** - Security hardening
- **[Performance Tuning](../operations/performance-tuning.md)** - Optimization guide
- **[Media Handling Guide](../guides/media-handling.md)** - Media issues
- **[Webhook Guides](../guides/webhooks/)** - Webhook troubleshooting
- **[Postmortems](../postmortems/)** - Lessons learned from production issues

---

**Version**: Compatible with v7.10.1+
**Last Updated**: 2025-12-05

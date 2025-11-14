# Webhook Setup Guide

This guide walks you through setting up and configuring webhooks for the Go WhatsApp Web Multidevice application.

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Testing Webhook Delivery](#testing-webhook-delivery)
- [Retry Logic and Error Handling](#retry-logic-and-error-handling)
- [Common Setup Patterns](#common-setup-patterns)
- [Troubleshooting](#troubleshooting)

## Overview

The webhook system sends HTTP POST requests to configured URLs whenever WhatsApp events occur. Each webhook request includes event data in JSON format and security headers for verification.

Webhooks enable real-time integration with your application by notifying you of:
- Incoming messages (text, media, contacts, locations)
- Message receipts (delivered, read)
- Group events (join, leave, promote, demote)
- Protocol events (message edits, deletions, revocations)

## Configuration

### Environment Variables

You can configure webhooks using environment variables:

```bash
# Single webhook URL
WHATSAPP_WEBHOOK=https://yourapp.com/webhook

# Multiple webhook URLs (comma-separated)
WHATSAPP_WEBHOOK=https://app1.com/webhook,https://app2.com/webhook

# Webhook secret for HMAC verification
WHATSAPP_WEBHOOK_SECRET=your-super-secret-key
```

### Command Line Flags

Alternatively, use command line flags when starting the application:

```bash
# Single webhook
./whatsapp rest --webhook="https://yourapp.com/webhook"

# Multiple webhooks
./whatsapp rest --webhook="https://app1.com/webhook,https://app2.com/webhook"

# Custom secret
./whatsapp rest --webhook-secret="your-secret-key"
```

### Docker Compose Configuration

If running with Docker, add these environment variables to your `docker-compose.yml`:

```yaml
services:
  whatsapp:
    image: your-whatsapp-image
    environment:
      - WHATSAPP_WEBHOOK=https://yourapp.com/webhook
      - WHATSAPP_WEBHOOK_SECRET=your-super-secret-key
    ports:
      - "3000:3000"
```

### Using .env File

Create a `.env` file in the `src/` directory:

```bash
# Webhook Configuration
WHATSAPP_WEBHOOK=https://yourapp.com/webhook
WHATSAPP_WEBHOOK_SECRET=your-super-secret-key

# Other settings
APP_PORT=3000
APP_DEBUG=true
```

## Testing Webhook Delivery

### Using a Local Tunnel

For local development, use a tunneling service to expose your local server:

#### Using ngrok

```bash
# Start ngrok
ngrok http 3001

# Use the generated URL
./whatsapp rest --webhook="https://abc123.ngrok.io/webhook"
```

#### Using localtunnel

```bash
# Install localtunnel
npm install -g localtunnel

# Start tunnel
lt --port 3001

# Use the generated URL
./whatsapp rest --webhook="https://your-subdomain.loca.lt/webhook"
```

### Test Webhook Endpoint

Create a simple test endpoint to verify webhook delivery:

```javascript
const express = require('express');
const app = express();

app.use(express.raw({type: 'application/json'}));

app.post('/webhook', (req, res) => {
    console.log('Headers:', req.headers);
    console.log('Body:', req.body.toString());
    res.status(200).send('OK');
});

app.listen(3001, () => {
    console.log('Test webhook server listening on port 3001');
});
```

### Testing with curl

Send a test request to your webhook endpoint:

```bash
curl -X POST https://yourapp.com/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=test" \
  -d '{"test": "data"}'
```

## Retry Logic and Error Handling

The webhook system includes automatic retry logic with exponential backoff to handle temporary failures.

### Retry Configuration

- **Timeout**: 10 seconds per request
- **Max Attempts**: 5 retries
- **Backoff Strategy**: Exponential (1s, 2s, 4s, 8s, 16s)
- **Success Criteria**: HTTP 2xx status code

### Retry Behavior

1. **Initial Request**: Sent immediately when event occurs
2. **First Retry**: After 1 second if initial request fails
3. **Second Retry**: After 2 seconds if first retry fails
4. **Third Retry**: After 4 seconds if second retry fails
5. **Fourth Retry**: After 8 seconds if third retry fails
6. **Final Retry**: After 16 seconds if fourth retry fails

After all retries are exhausted, the event is dropped.

### Best Practices for Webhook Endpoints

To ensure reliable webhook delivery:

1. **Respond Quickly**: Return HTTP 200 within 10 seconds
2. **Process Asynchronously**: Queue events for background processing
3. **Handle Duplicates**: Use message IDs to detect duplicate events
4. **Validate Signatures**: Always verify HMAC signatures (see [Security Guide](security.md))
5. **Log Errors**: Record failed events for debugging
6. **Use HTTPS**: Ensure webhook URLs use HTTPS for secure transmission

### Example: Asynchronous Processing

```javascript
const express = require('express');
const queue = require('./queue'); // Your queue implementation

app.post('/webhook', async (req, res) => {
    // Verify signature first
    if (!verifySignature(req)) {
        return res.status(401).send('Unauthorized');
    }

    // Respond immediately
    res.status(200).send('OK');

    // Process asynchronously
    const data = JSON.parse(req.body);
    await queue.add('process-webhook', data);
});
```

## Common Setup Patterns

### Single Application

Direct webhook to your main application:

```bash
WHATSAPP_WEBHOOK=https://myapp.com/api/webhooks/whatsapp
```

### Multiple Applications

Send events to multiple endpoints simultaneously:

```bash
WHATSAPP_WEBHOOK=https://app1.com/webhook,https://app2.com/webhook,https://app3.com/webhook
```

All configured webhooks receive the same events.

### Webhook Relay Service

Use a webhook relay or message queue service:

```bash
# Using Zapier
WHATSAPP_WEBHOOK=https://hooks.zapier.com/hooks/catch/12345/abcde/

# Using n8n
WHATSAPP_WEBHOOK=https://your-n8n.com/webhook/whatsapp

# Using AWS API Gateway
WHATSAPP_WEBHOOK=https://abc123.execute-api.us-east-1.amazonaws.com/prod/webhook
```

### Load Balancer

Point webhook to a load balancer for high availability:

```bash
WHATSAPP_WEBHOOK=https://lb.yourapp.com/webhook
```

### Reverse Proxy with Subpath

If your webhook is behind a reverse proxy with a subpath:

```bash
# Application at https://yourapp.com/whatsapp/webhook
WHATSAPP_WEBHOOK=https://yourapp.com/whatsapp/webhook

# Make sure your proxy passes the path correctly
```

## Troubleshooting

### Webhook Not Receiving Events

**Problem**: Webhook endpoint is not receiving any events.

**Solutions**:
1. Verify webhook URL is accessible from the internet
2. Check firewall and network settings
3. Test URL with curl: `curl -I https://yourapp.com/webhook`
4. Enable debug logging to see delivery attempts:
   ```bash
   ./whatsapp rest --debug=true --webhook="https://yourapp.com/webhook"
   ```
5. Check webhook configuration:
   ```bash
   # Verify environment variable
   echo $WHATSAPP_WEBHOOK
   ```

### Signature Verification Fails

**Problem**: Webhook signature verification is failing.

**Solutions**:
1. Ensure webhook secret matches configuration
2. Use raw request body (not parsed JSON) for verification
3. Check HMAC implementation (see [Security Guide](security.md))
4. Verify header name: `X-Hub-Signature-256`
5. Check signature format: `sha256={hash}`

### Timeouts

**Problem**: Webhook requests are timing out.

**Solutions**:
1. Optimize webhook processing speed
2. Implement asynchronous processing (respond immediately, process later)
3. Check endpoint performance (should respond within 10 seconds)
4. Reduce processing in webhook handler
5. Use a queue system for background processing

### Missing Media Files

**Problem**: Media files referenced in webhook are not accessible.

**Solutions**:
1. Check media storage path configuration (default: `statics/media/`)
2. Ensure sufficient disk space
3. Verify file permissions
4. Download media immediately using the `/message/:message_id/download` endpoint
5. Store media in your own storage system

### Duplicate Events

**Problem**: Receiving the same event multiple times.

**Solutions**:
1. Implement idempotency using message IDs:
   ```javascript
   const processedIds = new Set();

   app.post('/webhook', (req, res) => {
       const data = JSON.parse(req.body);
       const messageId = data.message?.id;

       if (processedIds.has(messageId)) {
           return res.status(200).send('Already processed');
       }

       processedIds.add(messageId);
       // Process event...
   });
   ```
2. Use a database to track processed events
3. Accept that duplicates may occur (design idempotent handlers)

### SSL/TLS Certificate Errors

**Problem**: Webhook delivery fails due to SSL certificate issues.

**Solutions**:
1. Ensure your webhook URL uses a valid SSL certificate
2. Use Let's Encrypt for free SSL certificates
3. Check certificate expiration
4. Verify certificate chain is complete

### Network Connectivity Issues

**Problem**: Webhook delivery fails intermittently.

**Solutions**:
1. Check network stability
2. Monitor application logs for connection errors
3. Verify DNS resolution
4. Consider using a webhook relay service for reliability
5. Implement health checks on your webhook endpoint

## Debug Logging

Enable debug mode to see detailed webhook logs:

```bash
./whatsapp rest --debug=true --webhook="https://yourapp.com/webhook"
```

Debug logs include:
- Webhook delivery attempts
- HTTP status codes
- Error messages
- Retry attempts
- Response times

## Health Check Endpoint

Implement a health check endpoint for monitoring:

```javascript
app.get('/webhook/health', (req, res) => {
    res.status(200).json({
        status: 'healthy',
        timestamp: new Date().toISOString()
    });
});
```

## Next Steps

- Learn about [Webhook Security](security.md) to properly verify signatures
- Explore [Integration Examples](examples.md) for complete implementation examples
- Review [Event Types](../../reference/webhooks/event-types.md) to understand available events
- Check [Payload Schemas](../../reference/webhooks/payload-schemas.md) for detailed payload structures

# Webhook Security Guide

This guide covers security best practices for webhooks, including HMAC signature verification and implementation examples in multiple languages.

## Table of Contents

- [Overview](#overview)
- [HMAC Signature Verification](#hmac-signature-verification)
- [Verification Examples](#verification-examples)
- [Best Practices](#best-practices)
- [Common Security Issues](#common-security-issues)

## Overview

All webhook requests include an HMAC SHA256 signature for security verification. This ensures that:
- The request actually came from your WhatsApp service
- The payload has not been tampered with
- Unauthorized parties cannot send fake webhook events

**Always verify webhook signatures before processing events.**

## HMAC Signature Verification

### How It Works

1. The WhatsApp service generates an HMAC SHA256 hash of the request body using your secret key
2. The hash is sent in the `X-Hub-Signature-256` header
3. Your webhook endpoint recalculates the hash using the same secret
4. Compare the calculated hash with the received hash to verify authenticity

### Signature Format

- **Header Name**: `X-Hub-Signature-256`
- **Format**: `sha256={signature}`
- **Algorithm**: HMAC SHA256
- **Default Secret**: `secret` (configurable via `--webhook-secret` or `WHATSAPP_WEBHOOK_SECRET`)

### Configuration

Set your webhook secret using environment variables or command line flags:

```bash
# Environment Variable
export WHATSAPP_WEBHOOK_SECRET=your-super-secret-key

# Command Line Flag
./whatsapp rest --webhook-secret="your-super-secret-key"

# Docker Compose
environment:
  - WHATSAPP_WEBHOOK_SECRET=your-super-secret-key
```

**Important**: Use a strong, randomly generated secret in production:

```bash
# Generate a strong secret (Linux/macOS)
openssl rand -hex 32

# Example output: a3d8f7e2b9c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9
```

## Verification Examples

### Node.js

Using the built-in `crypto` module:

```javascript
const crypto = require('crypto');

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

// Express.js Example
const express = require('express');
const app = express();

// Important: Use raw body parser for signature verification
app.use(express.raw({type: 'application/json'}));

app.post('/webhook', (req, res) => {
    const signature = req.headers['x-hub-signature-256'];
    const payload = req.body; // Raw buffer
    const secret = process.env.WHATSAPP_WEBHOOK_SECRET || 'secret';

    // Verify signature
    if (!verifyWebhookSignature(payload, signature, secret)) {
        console.error('Invalid webhook signature');
        return res.status(401).send('Unauthorized');
    }

    // Parse and process webhook data
    const data = JSON.parse(payload.toString());
    console.log('Verified webhook:', data);

    res.status(200).send('OK');
});
```

### Python

Using the `hmac` and `hashlib` modules:

```python
import hmac
import hashlib

def verify_webhook_signature(payload, signature, secret):
    """
    Verify webhook HMAC signature

    Args:
        payload: Raw request body (bytes)
        signature: Value of X-Hub-Signature-256 header
        secret: Webhook secret key

    Returns:
        bool: True if signature is valid, False otherwise
    """
    # Calculate expected signature
    expected_signature = hmac.new(
        secret.encode('utf-8'),
        payload,
        hashlib.sha256
    ).hexdigest()

    # Extract received signature (remove 'sha256=' prefix)
    received_signature = signature.replace('sha256=', '')

    # Use timing-safe comparison
    return hmac.compare_digest(expected_signature, received_signature)

# Flask Example
from flask import Flask, request, abort
import os

app = Flask(__name__)

@app.route('/webhook', methods=['POST'])
def webhook():
    signature = request.headers.get('X-Hub-Signature-256')
    payload = request.get_data()  # Raw bytes
    secret = os.getenv('WHATSAPP_WEBHOOK_SECRET', 'secret')

    # Verify signature
    if not verify_webhook_signature(payload, signature, secret):
        print('Invalid webhook signature')
        abort(401)

    # Parse and process webhook data
    import json
    data = json.loads(payload)
    print('Verified webhook:', data)

    return 'OK', 200

# Django Example
from django.http import HttpResponse, HttpResponseForbidden
from django.views.decorators.csrf import csrf_exempt
import json
import os

@csrf_exempt
def webhook_view(request):
    if request.method != 'POST':
        return HttpResponseForbidden()

    signature = request.headers.get('X-Hub-Signature-256')
    payload = request.body  # Raw bytes
    secret = os.getenv('WHATSAPP_WEBHOOK_SECRET', 'secret')

    # Verify signature
    if not verify_webhook_signature(payload, signature, secret):
        print('Invalid webhook signature')
        return HttpResponseForbidden()

    # Parse and process webhook data
    data = json.loads(payload)
    print('Verified webhook:', data)

    return HttpResponse('OK')
```

### Go

Using the `crypto/hmac` package:

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
)

func verifyWebhookSignature(payload []byte, signature string, secret string) bool {
    // Calculate expected signature
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expectedSignature := hex.EncodeToString(mac.Sum(nil))

    // Extract received signature (remove 'sha256=' prefix)
    receivedSignature := strings.TrimPrefix(signature, "sha256=")

    // Use constant-time comparison to prevent timing attacks
    return hmac.Equal([]byte(expectedSignature), []byte(receivedSignature))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Read raw body
    payload, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Get signature from header
    signature := r.Header.Get("X-Hub-Signature-256")
    secret := os.Getenv("WHATSAPP_WEBHOOK_SECRET")
    if secret == "" {
        secret = "secret"
    }

    // Verify signature
    if !verifyWebhookSignature(payload, signature, secret) {
        fmt.Println("Invalid webhook signature")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse and process webhook data
    var data map[string]interface{}
    if err := json.Unmarshal(payload, &data); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    fmt.Printf("Verified webhook: %+v\n", data)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func main() {
    http.HandleFunc("/webhook", webhookHandler)
    fmt.Println("Webhook server listening on :3001")
    http.ListenAndServe(":3001", nil)
}
```

### PHP

Using the `hash_hmac` function:

```php
<?php

function verifyWebhookSignature($payload, $signature, $secret) {
    // Calculate expected signature
    $expectedSignature = hash_hmac('sha256', $payload, $secret);

    // Extract received signature (remove 'sha256=' prefix)
    $receivedSignature = str_replace('sha256=', '', $signature);

    // Use timing-safe comparison
    return hash_equals($expectedSignature, $receivedSignature);
}

// Get raw POST data
$payload = file_get_contents('php://input');

// Get signature from header
$headers = getallheaders();
$signature = $headers['X-Hub-Signature-256'] ?? '';

// Get secret from environment or use default
$secret = getenv('WHATSAPP_WEBHOOK_SECRET') ?: 'secret';

// Verify signature
if (!verifyWebhookSignature($payload, $signature, $secret)) {
    error_log('Invalid webhook signature');
    http_response_code(401);
    echo 'Unauthorized';
    exit;
}

// Parse and process webhook data
$data = json_decode($payload, true);
error_log('Verified webhook: ' . print_r($data, true));

http_response_code(200);
echo 'OK';
```

### Ruby

Using the `openssl` library:

```ruby
require 'openssl'
require 'json'
require 'sinatra'

def verify_webhook_signature(payload, signature, secret)
  # Calculate expected signature
  expected_signature = OpenSSL::HMAC.hexdigest(
    OpenSSL::Digest.new('sha256'),
    secret,
    payload
  )

  # Extract received signature (remove 'sha256=' prefix)
  received_signature = signature.sub('sha256=', '')

  # Use timing-safe comparison
  Rack::Utils.secure_compare(expected_signature, received_signature)
end

# Sinatra Example
post '/webhook' do
  # Get raw body
  request.body.rewind
  payload = request.body.read

  # Get signature from header
  signature = request.env['HTTP_X_HUB_SIGNATURE_256'] || ''

  # Get secret from environment
  secret = ENV['WHATSAPP_WEBHOOK_SECRET'] || 'secret'

  # Verify signature
  unless verify_webhook_signature(payload, signature, secret)
    puts 'Invalid webhook signature'
    halt 401, 'Unauthorized'
  end

  # Parse and process webhook data
  data = JSON.parse(payload)
  puts "Verified webhook: #{data}"

  status 200
  body 'OK'
end
```

### Java

Using the `javax.crypto.Mac` class:

```java
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.InvalidKeyException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;

public class WebhookSignatureVerifier {

    public static boolean verifyWebhookSignature(
            byte[] payload,
            String signature,
            String secret
    ) throws NoSuchAlgorithmException, InvalidKeyException {
        // Calculate expected signature
        Mac mac = Mac.getInstance("HmacSHA256");
        SecretKeySpec secretKeySpec = new SecretKeySpec(
            secret.getBytes(StandardCharsets.UTF_8),
            "HmacSHA256"
        );
        mac.init(secretKeySpec);
        byte[] hash = mac.doFinal(payload);

        // Convert to hex string
        StringBuilder expectedSignature = new StringBuilder();
        for (byte b : hash) {
            expectedSignature.append(String.format("%02x", b));
        }

        // Extract received signature (remove 'sha256=' prefix)
        String receivedSignature = signature.replace("sha256=", "");

        // Use timing-safe comparison
        return MessageDigest.isEqual(
            expectedSignature.toString().getBytes(StandardCharsets.UTF_8),
            receivedSignature.getBytes(StandardCharsets.UTF_8)
        );
    }

    // Spring Boot Example
    @PostMapping("/webhook")
    public ResponseEntity<String> handleWebhook(
            @RequestBody byte[] payload,
            @RequestHeader("X-Hub-Signature-256") String signature
    ) {
        String secret = System.getenv().getOrDefault(
            "WHATSAPP_WEBHOOK_SECRET",
            "secret"
        );

        try {
            // Verify signature
            if (!verifyWebhookSignature(payload, signature, secret)) {
                System.err.println("Invalid webhook signature");
                return ResponseEntity.status(401).body("Unauthorized");
            }

            // Parse and process webhook data
            String payloadStr = new String(payload, StandardCharsets.UTF_8);
            // Process JSON here...
            System.out.println("Verified webhook: " + payloadStr);

            return ResponseEntity.ok("OK");

        } catch (Exception e) {
            System.err.println("Error verifying signature: " + e.getMessage());
            return ResponseEntity.status(500).body("Internal Server Error");
        }
    }
}
```

## Best Practices

### 1. Use Strong Secrets

- **Length**: Use at least 32 characters
- **Randomness**: Generate using cryptographically secure random generators
- **Rotation**: Rotate secrets periodically (e.g., every 90 days)
- **Storage**: Store secrets securely (environment variables, secret managers)

```bash
# Good: Strong random secret
openssl rand -hex 32

# Bad: Weak or predictable secrets
secret="password123"
secret="my-app-secret"
```

### 2. Always Use Raw Body

The signature is calculated on the raw request body. Parse JSON only after verification:

```javascript
// Correct: Use raw body for verification
app.use(express.raw({type: 'application/json'}));

app.post('/webhook', (req, res) => {
    const rawBody = req.body; // Buffer
    verifySignature(rawBody, ...);
    const data = JSON.parse(rawBody.toString());
});

// Incorrect: Using parsed JSON
app.use(express.json()); // Parses body automatically

app.post('/webhook', (req, res) => {
    const data = req.body; // Already parsed!
    // Verification will fail because body is modified
});
```

### 3. Use Timing-Safe Comparison

Always use timing-safe comparison functions to prevent timing attacks:

```javascript
// Correct: Timing-safe comparison
crypto.timingSafeEqual(Buffer.from(a), Buffer.from(b));

// Incorrect: Direct string comparison (vulnerable to timing attacks)
if (expectedSignature === receivedSignature) { ... }
```

### 4. Validate Before Processing

Always verify the signature before any processing:

```javascript
app.post('/webhook', (req, res) => {
    // 1. Verify signature FIRST
    if (!verifySignature(...)) {
        return res.status(401).send('Unauthorized');
    }

    // 2. Then process the event
    processWebhook(data);
});
```

### 5. Log Security Events

Log all signature verification failures for security monitoring:

```javascript
if (!verifySignature(...)) {
    console.error('Invalid webhook signature', {
        ip: req.ip,
        timestamp: new Date().toISOString(),
        signature: signature,
        // Don't log the payload or secret!
    });
    return res.status(401).send('Unauthorized');
}
```

### 6. Use HTTPS

Always use HTTPS for webhook URLs to prevent man-in-the-middle attacks:

```bash
# Correct: HTTPS
WHATSAPP_WEBHOOK=https://yourapp.com/webhook

# Incorrect: HTTP (insecure)
WHATSAPP_WEBHOOK=http://yourapp.com/webhook
```

### 7. Implement Rate Limiting

Protect your webhook endpoint from abuse:

```javascript
const rateLimit = require('express-rate-limit');

const webhookLimiter = rateLimit({
    windowMs: 1 * 60 * 1000, // 1 minute
    max: 100, // Limit each IP to 100 requests per windowMs
    message: 'Too many requests'
});

app.post('/webhook', webhookLimiter, (req, res) => {
    // Handle webhook
});
```

### 8. Don't Log Sensitive Data

Never log secrets, full payloads, or personal information:

```javascript
// Bad: Logging sensitive data
console.log('Secret:', secret);
console.log('Full payload:', data);

// Good: Log only necessary information
console.log('Webhook received:', {
    event: data.event,
    messageId: data.message?.id,
    timestamp: data.timestamp
});
```

## Common Security Issues

### Issue 1: Using Parsed JSON for Verification

**Problem**: Signature verification fails because the body was parsed before verification.

**Solution**: Use raw body parser for webhook endpoint:

```javascript
// Correct
app.use('/webhook', express.raw({type: 'application/json'}));

// Incorrect
app.use(express.json()); // Applies to all routes
```

### Issue 2: Direct String Comparison

**Problem**: Using `===` for signature comparison is vulnerable to timing attacks.

**Solution**: Use timing-safe comparison functions:

```javascript
// Use crypto.timingSafeEqual() in Node.js
// Use hmac.compare_digest() in Python
// Use hmac.Equal() in Go
```

### Issue 3: Weak Secrets

**Problem**: Using predictable or short secrets.

**Solution**: Generate strong random secrets:

```bash
# Generate 32-byte (256-bit) random secret
openssl rand -hex 32
```

### Issue 4: Signature Not Checked

**Problem**: Processing webhooks without verifying signatures.

**Solution**: Always verify signatures:

```javascript
if (!verifySignature(...)) {
    return res.status(401).send('Unauthorized');
}
```

### Issue 5: Secret Exposed in Code

**Problem**: Hardcoding secret in source code.

**Solution**: Use environment variables:

```javascript
const secret = process.env.WHATSAPP_WEBHOOK_SECRET;
```

### Issue 6: HTTP Instead of HTTPS

**Problem**: Using HTTP URLs for webhooks.

**Solution**: Always use HTTPS:

```bash
WHATSAPP_WEBHOOK=https://yourapp.com/webhook
```

## Testing Signature Verification

### Generate Test Signature

```bash
# Using OpenSSL
echo -n '{"test":"data"}' | openssl dgst -sha256 -hmac "your-secret" -hex

# Output: sha256={signature}
```

### Test with curl

```bash
PAYLOAD='{"test":"data"}'
SECRET="your-secret"
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')

curl -X POST https://yourapp.com/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

## Next Steps

- Learn about [Webhook Setup](setup.md) for configuration details
- Explore [Integration Examples](examples.md) for complete implementations
- Review [Event Types](../../reference/webhooks/event-types.md) for available events
- Check [Payload Schemas](../../reference/webhooks/payload-schemas.md) for payload structures

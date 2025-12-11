# Webhook Integration Examples

This guide provides complete, working examples of webhook integrations in multiple languages and frameworks.

## Table of Contents

- [Overview](#overview)
- [Complete Integration Examples](#complete-integration-examples)
- [Event Type Handlers](#event-type-handlers)
- [Error Handling Patterns](#error-handling-patterns)
- [Idempotency Handling](#idempotency-handling)
- [Practical Use Cases](#practical-use-cases)

## Overview

These examples demonstrate real-world webhook integration patterns, including:
- Signature verification
- Event type handling
- Asynchronous processing
- Error handling
- Idempotency
- Database integration

## Complete Integration Examples

### Node.js with Express

A production-ready Express.js webhook server:

```javascript
const express = require('express');
const crypto = require('crypto');
const app = express();

// Configuration
const WEBHOOK_SECRET = process.env.WHATSAPP_WEBHOOK_SECRET || 'secret';
const PORT = process.env.PORT || 3001;

// Use raw body parser for signature verification
app.use(express.raw({type: 'application/json'}));

// Signature verification function
function verifyWebhookSignature(payload, signature, secret) {
    const expectedSignature = crypto
        .createHmac('sha256', secret)
        .update(payload, 'utf8')
        .digest('hex');

    const receivedSignature = signature.replace('sha256=', '');

    return crypto.timingSafeEqual(
        Buffer.from(expectedSignature, 'hex'),
        Buffer.from(receivedSignature, 'hex')
    );
}

// Message handler
function handleMessage(data) {
    console.log('New message:', {
        from: data.pushname,
        sender: data.sender_id,
        chat: data.chat_id,
        text: data.message?.text || '(media)',
        timestamp: data.timestamp
    });

    // Process text messages
    if (data.message?.text) {
        console.log('Text message:', data.message.text);
    }

    // Process media messages
    if (data.image) {
        console.log('Image received:', data.image.media_path);
    }
    if (data.video) {
        console.log('Video received:', data.video.media_path);
    }
    if (data.audio) {
        console.log('Audio received:', data.audio.media_path);
    }
    if (data.document) {
        console.log('Document received:', data.document.media_path);
    }

    // Process reactions
    if (data.reaction) {
        console.log('Reaction:', data.reaction.message, 'on message:', data.reaction.id);
    }
}

// Receipt handler
function handleReceipt(data) {
    console.log('Receipt event:', {
        type: data.payload.receipt_type,
        description: data.payload.receipt_type_description,
        chat: data.payload.chat_id,
        messageIds: data.payload.ids
    });

    switch (data.payload.receipt_type) {
        case 'delivered':
            console.log('Messages delivered:', data.payload.ids);
            break;
        case 'read':
            console.log('Messages read:', data.payload.ids);
            break;
        case 'played':
            console.log('View-once media played:', data.payload.ids);
            break;
    }
}

// Group event handler
function handleGroupEvent(data) {
    console.log('Group event:', {
        type: data.payload.type,
        group: data.payload.chat_id,
        users: data.payload.jids
    });

    switch (data.payload.type) {
        case 'join':
            console.log(`${data.payload.jids.length} users joined group`);
            data.payload.jids.forEach(jid => {
                console.log(`- ${jid} joined`);
                // Send welcome message
            });
            break;
        case 'leave':
            console.log(`${data.payload.jids.length} users left group`);
            break;
        case 'promote':
            console.log(`${data.payload.jids.length} users promoted to admin`);
            break;
        case 'demote':
            console.log(`${data.payload.jids.length} users demoted from admin`);
            break;
    }
}

// Protocol event handler
function handleProtocolEvent(data) {
    switch (data.action) {
        case 'event.delete_for_me':
            console.log('Message deleted for me:', data.deleted_message_id);
            break;
        case 'message_revoked':
            console.log('Message revoked:', data.revoked_message_id);
            break;
        case 'message_edited':
            console.log('Message edited:', data.message.id, 'new text:', data.edited_text);
            break;
    }
}

// Main webhook endpoint
app.post('/webhook', (req, res) => {
    const signature = req.headers['x-hub-signature-256'];
    const payload = req.body;

    // Verify signature
    if (!signature || !verifyWebhookSignature(payload, signature, WEBHOOK_SECRET)) {
        console.error('Invalid webhook signature');
        return res.status(401).send('Unauthorized');
    }

    // Parse webhook data
    const data = JSON.parse(payload.toString());
    console.log('Webhook received:', data.event || data.action || 'message');

    // Route to appropriate handler
    if (data.event === 'message.ack') {
        handleReceipt(data);
    } else if (data.event === 'group.participants') {
        handleGroupEvent(data);
    } else if (data.action) {
        handleProtocolEvent(data);
    } else {
        handleMessage(data);
    }

    res.status(200).send('OK');
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.status(200).json({
        status: 'healthy',
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`Webhook server listening on port ${PORT}`);
    console.log(`POST /webhook - Main webhook endpoint`);
    console.log(`GET /health - Health check endpoint`);
});
```

### Python with Flask

A production-ready Flask webhook server:

```python
from flask import Flask, request, abort
import hmac
import hashlib
import json
import os
from datetime import datetime

app = Flask(__name__)

# Configuration
WEBHOOK_SECRET = os.getenv('WHATSAPP_WEBHOOK_SECRET', 'secret')

def verify_webhook_signature(payload, signature, secret):
    """Verify webhook HMAC signature"""
    expected_signature = hmac.new(
        secret.encode('utf-8'),
        payload,
        hashlib.sha256
    ).hexdigest()

    received_signature = signature.replace('sha256=', '')
    return hmac.compare_digest(expected_signature, received_signature)

def handle_message(data):
    """Handle incoming message events"""
    print(f"New message from {data.get('pushname')}:")
    print(f"  Sender: {data.get('sender_id')}")
    print(f"  Chat: {data.get('chat_id')}")
    print(f"  Timestamp: {data.get('timestamp')}")

    # Process text messages
    message = data.get('message', {})
    if message.get('text'):
        print(f"  Text: {message['text']}")

    # Process media messages
    if 'image' in data:
        print(f"  Image: {data['image']['media_path']}")
    if 'video' in data:
        print(f"  Video: {data['video']['media_path']}")
    if 'audio' in data:
        print(f"  Audio: {data['audio']['media_path']}")
    if 'document' in data:
        print(f"  Document: {data['document']['media_path']}")

    # Process reactions
    if 'reaction' in data:
        print(f"  Reaction: {data['reaction']['message']} on {data['reaction']['id']}")

def handle_receipt(data):
    """Handle message receipt events"""
    payload = data.get('payload', {})
    receipt_type = payload.get('receipt_type')
    description = payload.get('receipt_type_description')

    print(f"Receipt event: {receipt_type}")
    print(f"  Description: {description}")
    print(f"  Chat: {payload.get('chat_id')}")
    print(f"  Message IDs: {payload.get('ids')}")

    if receipt_type == 'delivered':
        print(f"  Messages delivered")
    elif receipt_type == 'read':
        print(f"  Messages read")
    elif receipt_type == 'played':
        print(f"  View-once media played")

def handle_group_event(data):
    """Handle group events"""
    payload = data.get('payload', {})
    event_type = payload.get('type')
    chat_id = payload.get('chat_id')
    jids = payload.get('jids', [])

    print(f"Group event: {event_type}")
    print(f"  Group: {chat_id}")
    print(f"  Users: {jids}")

    if event_type == 'join':
        print(f"  {len(jids)} users joined")
        for jid in jids:
            print(f"    - {jid}")
            # Send welcome message
    elif event_type == 'leave':
        print(f"  {len(jids)} users left")
    elif event_type == 'promote':
        print(f"  {len(jids)} users promoted to admin")
    elif event_type == 'demote':
        print(f"  {len(jids)} users demoted from admin")

def handle_protocol_event(data):
    """Handle protocol events"""
    action = data.get('action')

    if action == 'event.delete_for_me':
        print(f"Message deleted: {data.get('deleted_message_id')}")
    elif action == 'message_revoked':
        print(f"Message revoked: {data.get('revoked_message_id')}")
    elif action == 'message_edited':
        print(f"Message edited: {data.get('message', {}).get('id')}")
        print(f"  New text: {data.get('edited_text')}")

@app.route('/webhook', methods=['POST'])
def webhook():
    """Main webhook endpoint"""
    # Get signature from header
    signature = request.headers.get('X-Hub-Signature-256')
    if not signature:
        print('Missing signature header')
        abort(401)

    # Get raw payload
    payload = request.get_data()

    # Verify signature
    if not verify_webhook_signature(payload, signature, WEBHOOK_SECRET):
        print('Invalid webhook signature')
        abort(401)

    # Parse webhook data
    data = json.loads(payload)
    event_type = data.get('event') or data.get('action') or 'message'
    print(f"\nWebhook received: {event_type}")

    # Route to appropriate handler
    if data.get('event') == 'message.ack':
        handle_receipt(data)
    elif data.get('event') == 'group.participants':
        handle_group_event(data)
    elif data.get('action'):
        handle_protocol_event(data)
    else:
        handle_message(data)

    return 'OK', 200

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return {
        'status': 'healthy',
        'timestamp': datetime.utcnow().isoformat() + 'Z'
    }, 200

if __name__ == '__main__':
    port = int(os.getenv('PORT', 3001))
    print(f"Webhook server listening on port {port}")
    print(f"POST /webhook - Main webhook endpoint")
    print(f"GET /health - Health check endpoint")
    app.run(host='0.0.0.0', port=port)
```

### Go with Standard Library

A production-ready Go webhook server:

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

const defaultSecret = "secret"

// Webhook data structures
type WebhookMessage struct {
    SenderID  string  `json:"sender_id"`
    ChatID    string  `json:"chat_id"`
    From      string  `json:"from"`
    Timestamp string  `json:"timestamp"`
    Pushname  string  `json:"pushname"`
    Message   Message `json:"message"`
    Image     *Media  `json:"image,omitempty"`
    Video     *Media  `json:"video,omitempty"`
    Audio     *Media  `json:"audio,omitempty"`
    Document  *Media  `json:"document,omitempty"`
    Reaction  *Reaction `json:"reaction,omitempty"`
}

type Message struct {
    Text          string `json:"text"`
    ID            string `json:"id"`
    RepliedID     string `json:"replied_id"`
    QuotedMessage string `json:"quoted_message"`
}

type Media struct {
    MediaPath string `json:"media_path"`
    MimeType  string `json:"mime_type"`
    Caption   string `json:"caption"`
}

type Reaction struct {
    Message string `json:"message"`
    ID      string `json:"id"`
}

type ReceiptEvent struct {
    Event     string        `json:"event"`
    Payload   ReceiptPayload `json:"payload"`
    Timestamp string        `json:"timestamp"`
}

type ReceiptPayload struct {
    ChatID      string   `json:"chat_id"`
    From        string   `json:"from"`
    IDs         []string `json:"ids"`
    ReceiptType string   `json:"receipt_type"`
    Description string   `json:"receipt_type_description"`
    SenderID    string   `json:"sender_id"`
}

type GroupEvent struct {
    Event     string       `json:"event"`
    Payload   GroupPayload `json:"payload"`
    Timestamp string       `json:"timestamp"`
}

type GroupPayload struct {
    ChatID string   `json:"chat_id"`
    Type   string   `json:"type"`
    JIDs   []string `json:"jids"`
}

func verifyWebhookSignature(payload []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expectedSignature := hex.EncodeToString(mac.Sum(nil))
    receivedSignature := strings.TrimPrefix(signature, "sha256=")
    return hmac.Equal([]byte(expectedSignature), []byte(receivedSignature))
}

func handleMessage(data map[string]interface{}) {
    fmt.Printf("New message from %v:\n", data["pushname"])
    fmt.Printf("  Sender: %v\n", data["sender_id"])
    fmt.Printf("  Chat: %v\n", data["chat_id"])

    if message, ok := data["message"].(map[string]interface{}); ok {
        if text, ok := message["text"].(string); ok && text != "" {
            fmt.Printf("  Text: %s\n", text)
        }
    }

    if image, ok := data["image"].(map[string]interface{}); ok {
        fmt.Printf("  Image: %v\n", image["media_path"])
    }
    if video, ok := data["video"].(map[string]interface{}); ok {
        fmt.Printf("  Video: %v\n", video["media_path"])
    }
}

func handleReceipt(data map[string]interface{}) {
    payload, ok := data["payload"].(map[string]interface{})
    if !ok {
        return
    }

    receiptType := payload["receipt_type"]
    description := payload["receipt_type_description"]

    fmt.Printf("Receipt event: %v\n", receiptType)
    fmt.Printf("  Description: %v\n", description)
    fmt.Printf("  Chat: %v\n", payload["chat_id"])
    fmt.Printf("  Message IDs: %v\n", payload["ids"])
}

func handleGroupEvent(data map[string]interface{}) {
    payload, ok := data["payload"].(map[string]interface{})
    if !ok {
        return
    }

    eventType := payload["type"]
    chatID := payload["chat_id"]
    jids := payload["jids"]

    fmt.Printf("Group event: %v\n", eventType)
    fmt.Printf("  Group: %v\n", chatID)
    fmt.Printf("  Users: %v\n", jids)
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
    if signature == "" {
        log.Println("Missing signature header")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Get secret from environment
    secret := os.Getenv("WHATSAPP_WEBHOOK_SECRET")
    if secret == "" {
        secret = defaultSecret
    }

    // Verify signature
    if !verifyWebhookSignature(payload, signature, secret) {
        log.Println("Invalid webhook signature")
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse webhook data
    var data map[string]interface{}
    if err := json.Unmarshal(payload, &data); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Route to appropriate handler
    event := ""
    if e, ok := data["event"].(string); ok {
        event = e
    } else if a, ok := data["action"].(string); ok {
        event = a
    } else {
        event = "message"
    }

    log.Printf("Webhook received: %s\n", event)

    if event == "message.ack" {
        handleReceipt(data)
    } else if event == "group.participants" {
        handleGroupEvent(data)
    } else {
        handleMessage(data)
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := map[string]string{
        "status":    "healthy",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/webhook", webhookHandler)
    http.HandleFunc("/health", healthHandler)

    port := os.Getenv("PORT")
    if port == "" {
        port = "3001"
    }

    log.Printf("Webhook server listening on port %s\n", port)
    log.Println("POST /webhook - Main webhook endpoint")
    log.Println("GET /health - Health check endpoint")

    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}
```

## Event Type Handlers

### Handling Different Message Types

```javascript
function handleMessage(data) {
    // Text message
    if (data.message?.text) {
        handleTextMessage(data);
    }

    // Media messages
    if (data.image) handleImageMessage(data);
    if (data.video) handleVideoMessage(data);
    if (data.audio) handleAudioMessage(data);
    if (data.document) handleDocumentMessage(data);
    if (data.sticker) handleStickerMessage(data);

    // Special messages
    if (data.contact) handleContactMessage(data);
    if (data.location) handleLocationMessage(data);
    if (data.reaction) handleReactionMessage(data);

    // Message flags
    if (data.view_once) console.log('View-once message');
    if (data.forwarded) console.log('Forwarded message');
}

function handleTextMessage(data) {
    const text = data.message.text;
    const isReply = data.message.replied_id !== '';

    console.log(`Text: ${text}`);

    if (isReply) {
        console.log(`Reply to: ${data.message.quoted_message}`);
    }

    // Auto-reply to specific commands
    if (text.startsWith('/help')) {
        // Send help message
    }
}

function handleImageMessage(data) {
    const imagePath = data.image.media_path;
    const caption = data.image.caption;

    console.log(`Image received: ${imagePath}`);
    if (caption) console.log(`Caption: ${caption}`);

    // Process image (e.g., upload to cloud storage)
}
```

## Error Handling Patterns

### Graceful Error Handling

```javascript
app.post('/webhook', async (req, res) => {
    try {
        const signature = req.headers['x-hub-signature-256'];
        const payload = req.body;

        // Verify signature
        if (!verifyWebhookSignature(payload, signature, WEBHOOK_SECRET)) {
            console.error('Invalid signature');
            return res.status(401).send('Unauthorized');
        }

        // Parse data
        const data = JSON.parse(payload.toString());

        // Process asynchronously
        processWebhookAsync(data).catch(err => {
            console.error('Error processing webhook:', err);
            // Log to error tracking service
        });

        // Respond quickly
        res.status(200).send('OK');

    } catch (error) {
        console.error('Webhook error:', error);
        res.status(500).send('Internal Server Error');
    }
});

async function processWebhookAsync(data) {
    try {
        // Handle different event types
        if (data.event === 'message.ack') {
            await handleReceipt(data);
        } else if (data.message) {
            await handleMessage(data);
        }
    } catch (error) {
        // Log error with context
        console.error('Processing error:', {
            error: error.message,
            event: data.event || 'message',
            messageId: data.message?.id,
            timestamp: data.timestamp
        });
        throw error;
    }
}
```

## Idempotency Handling

### Using In-Memory Cache

```javascript
const processedMessages = new Set();
const CACHE_TTL = 24 * 60 * 60 * 1000; // 24 hours

function isProcessed(messageId) {
    return processedMessages.has(messageId);
}

function markAsProcessed(messageId) {
    processedMessages.add(messageId);

    // Clean up after TTL
    setTimeout(() => {
        processedMessages.delete(messageId);
    }, CACHE_TTL);
}

function handleMessage(data) {
    const messageId = data.message?.id;

    if (!messageId) {
        console.warn('Message without ID');
        return;
    }

    if (isProcessed(messageId)) {
        console.log('Duplicate message, skipping:', messageId);
        return;
    }

    // Process message
    console.log('Processing message:', messageId);
    // ... process message ...

    markAsProcessed(messageId);
}
```

### Using Redis

```javascript
const redis = require('redis');
const client = redis.createClient();

async function isProcessed(messageId) {
    const exists = await client.exists(`processed:${messageId}`);
    return exists === 1;
}

async function markAsProcessed(messageId) {
    // Store with 24-hour expiration
    await client.setex(`processed:${messageId}`, 86400, '1');
}

async function handleMessage(data) {
    const messageId = data.message?.id;

    if (!messageId) {
        console.warn('Message without ID');
        return;
    }

    if (await isProcessed(messageId)) {
        console.log('Duplicate message, skipping:', messageId);
        return;
    }

    // Process message
    console.log('Processing message:', messageId);
    // ... process message ...

    await markAsProcessed(messageId);
}
```

### Using Database

```javascript
const db = require('./database');

async function handleMessage(data) {
    const messageId = data.message?.id;

    if (!messageId) return;

    try {
        // Try to insert message ID
        await db.query(
            'INSERT INTO processed_messages (message_id, processed_at) VALUES (?, NOW())',
            [messageId]
        );

        // Process message
        console.log('Processing message:', messageId);
        // ... process message ...

    } catch (error) {
        if (error.code === 'ER_DUP_ENTRY') {
            console.log('Duplicate message, skipping:', messageId);
        } else {
            throw error;
        }
    }
}
```

## Practical Use Cases

### Auto-Reply Bot

```javascript
function handleMessage(data) {
    const text = data.message?.text;
    const from = data.sender_id;

    if (!text) return;

    // FAQ responses
    const faq = {
        '/help': 'Available commands: /help, /status, /info',
        '/status': 'All systems operational',
        '/info': 'This is an automated WhatsApp bot'
    };

    if (faq[text.toLowerCase()]) {
        sendMessage(from, faq[text.toLowerCase()]);
    }
}

async function sendMessage(to, text) {
    const response = await fetch('http://localhost:3000/send/message', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            user_id: to,
            message: text
        })
    });

    console.log('Auto-reply sent:', text);
}
```

### Group Welcome Messages

```javascript
async function handleGroupEvent(data) {
    if (data.payload.type === 'join') {
        const groupId = data.payload.chat_id;
        const newMembers = data.payload.jids;

        for (const member of newMembers) {
            const welcomeMessage = `Welcome to the group, @${member}!`;
            await sendGroupMessage(groupId, welcomeMessage);
        }
    }
}

async function sendGroupMessage(groupId, text) {
    await fetch('http://localhost:3000/send/message', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            user_id: groupId,
            message: text
        })
    });
}
```

### Message Logging to Database

```javascript
const db = require('./database');

async function handleMessage(data) {
    try {
        await db.query(
            `INSERT INTO messages (
                message_id, sender_id, chat_id, text,
                media_type, media_path, timestamp
            ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
            [
                data.message.id,
                data.sender_id,
                data.chat_id,
                data.message.text || null,
                getMediaType(data),
                getMediaPath(data),
                data.timestamp
            ]
        );

        console.log('Message logged:', data.message.id);
    } catch (error) {
        console.error('Failed to log message:', error);
    }
}

function getMediaType(data) {
    if (data.image) return 'image';
    if (data.video) return 'video';
    if (data.audio) return 'audio';
    if (data.document) return 'document';
    return null;
}

function getMediaPath(data) {
    return data.image?.media_path ||
           data.video?.media_path ||
           data.audio?.media_path ||
           data.document?.media_path ||
           null;
}
```

### Integration with External APIs

```javascript
async function handleMessage(data) {
    const text = data.message?.text;

    if (text && text.startsWith('/translate ')) {
        const textToTranslate = text.replace('/translate ', '');
        const translation = await translateText(textToTranslate);
        await sendMessage(data.sender_id, translation);
    }
}

async function translateText(text) {
    const response = await fetch('https://api.translation-service.com/translate', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${process.env.TRANSLATION_API_KEY}`
        },
        body: JSON.stringify({
            text: text,
            target: 'en'
        })
    });

    const data = await response.json();
    return data.translatedText;
}
```

## Next Steps

- Review [Webhook Setup](setup.md) for configuration details
- Learn about [Webhook Security](security.md) for signature verification
- Check [Event Types](../../reference/webhooks/event-types.md) for all available events
- Explore [Payload Schemas](../../reference/webhooks/payload-schemas.md) for detailed payload structures

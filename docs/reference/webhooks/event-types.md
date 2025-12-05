# Webhook Event Types Reference

This document provides a comprehensive reference of all webhook event types sent by the Go WhatsApp Web Multidevice application.

## Table of Contents

- [Overview](#overview)
- [Event Categories](#event-categories)
- [Message Events](#message-events)
- [Receipt Events](#receipt-events)
- [Group Events](#group-events)
- [Protocol Events](#protocol-events)
- [Event Summary Table](#event-summary-table)

## Overview

The webhook system sends different event types based on WhatsApp activities. Each event type has a specific structure and purpose. Understanding these event types helps you build robust webhook integrations.

### Event Identification

Events can be identified by:
- `event` field (for receipt and group events)
- `action` field (for protocol events)
- Message structure (for message events)

## Event Categories

### 1. Message Events

Regular incoming messages from users, including text, media, and special message types.

**Identification**: Presence of `message` object without `event` or `action` field.

### 2. Receipt Events

Message acknowledgment events (delivered, read, played).

**Identification**: `event: "message.ack"`

### 3. Group Events

Group membership and administration changes.

**Identification**: `event: "group.participants"`

### 4. Protocol Events

WhatsApp protocol-level events (edits, deletions, revocations).

**Identification**: `action` field with values like `message_edited`, `message_revoked`, etc.

## Message Events

### Text Messages

**Description**: Plain text messages from users.

**Key Fields**:
- `message.text` - Message content
- `sender_id` - Sender's phone number
- `chat_id` - Chat identifier
- `pushname` - Sender's display name

**Example Use Case**: Chatbot command processing, customer support tickets

---

### Reply Messages

**Description**: Messages that reply to previous messages.

**Key Fields**:
- `message.text` - Reply text
- `message.replied_id` - ID of the message being replied to
- `message.quoted_message` - Text of the quoted message

**Example Use Case**: Thread tracking, conversation context

---

### Reaction Messages

**Description**: Emoji reactions to messages.

**Key Fields**:
- `reaction.message` - Emoji reaction
- `reaction.id` - ID of the message being reacted to

**Example Use Case**: Sentiment analysis, message feedback

---

### Image Messages

**Description**: Image messages with optional captions.

**Key Fields**:
- `image.media_path` - Local path to the image file
- `image.mime_type` - Image MIME type
- `image.caption` - Optional caption text

**Example Use Case**: Image analysis, content moderation, media archival

---

### Video Messages

**Description**: Video messages with optional captions.

**Key Fields**:
- `video.media_path` - Local path to the video file
- `video.mime_type` - Video MIME type
- `video.caption` - Optional caption text

**Example Use Case**: Video processing, content storage

---

### Audio Messages

**Description**: Audio messages including voice notes.

**Key Fields**:
- `audio.media_path` - Local path to the audio file
- `audio.mime_type` - Audio MIME type
- `audio.caption` - Optional caption text

**Example Use Case**: Voice transcription, audio analysis

---

### Document Messages

**Description**: Document/file attachments with optional captions.

**Key Fields**:
- `document.media_path` - Local path to the document
- `document.mime_type` - Document MIME type
- `document.caption` - Optional caption text

**Example Use Case**: Document processing, file storage

---

### Sticker Messages

**Description**: Sticker messages (static or animated).

**Key Fields**:
- `sticker.media_path` - Local path to the sticker file (WebP format)
- `sticker.mime_type` - Always `image/webp`
- `sticker.caption` - Usually empty

**Example Use Case**: Sticker analytics, content filtering

---

### Contact Messages

**Description**: Shared contact information.

**Key Fields**:
- `contact.displayName` - Contact's display name
- `contact.vcard` - vCard formatted contact data

**Example Use Case**: Contact extraction, CRM integration

---

### Location Messages

**Description**: Location sharing messages.

**Key Fields**:
- `location.degreesLatitude` - Latitude coordinate
- `location.degreesLongitude` - Longitude coordinate
- `location.name` - Location name
- `location.address` - Location address

**Example Use Case**: Location tracking, delivery routing

---

### Live Location Messages

**Description**: Real-time location sharing updates.

**Key Fields**:
- `location.degreesLatitude` - Current latitude
- `location.degreesLongitude` - Current longitude
- `location.JPEGThumbnail` - Base64 encoded map thumbnail

**Example Use Case**: Real-time tracking, delivery monitoring

**Note**: Contains raw WhatsApp protocol structure.

---

### List Messages

**Description**: Interactive list messages.

**Key Fields**:
- `list.title` - List title
- `list.description` - List description
- `list.sections` - Array of list sections with options

**Example Use Case**: Menu navigation, option selection

**Note**: Contains raw WhatsApp protocol structure.

---

### Order Messages

**Description**: E-commerce order information.

**Key Fields**:
- `order.orderTitle` - Order title
- `order.itemCount` - Number of items
- `order.sellerJid` - Seller's WhatsApp ID

**Example Use Case**: Order processing, e-commerce integration

**Note**: Contains raw WhatsApp protocol structure.

---

### View-Once Messages

**Description**: Media messages that can only be viewed once.

**Key Fields**:
- `view_once` - Flag set to `true`
- Media fields (`image`, `video`, etc.)

**Example Use Case**: Temporary content handling, privacy compliance

---

### Forwarded Messages

**Description**: Messages that were forwarded from another chat.

**Key Fields**:
- `forwarded` - Flag set to `true`
- `message.text` or media fields

**Example Use Case**: Viral content tracking, forward chain analysis

## Receipt Events

All receipt events use `event: "message.ack"` with different `receipt_type` values.

### Delivered Receipt

**Description**: Message successfully delivered to recipient's device.

**Receipt Type**: `delivered`

**Key Fields**:
- `payload.ids` - Array of delivered message IDs
- `payload.chat_id` - Chat where delivery occurred
- `payload.receipt_type_description` - Human-readable description

**Example Use Case**: Delivery tracking, message status updates

---

### Read Receipt

**Description**: User opened the chat and saw the message.

**Receipt Type**: `read`

**Key Fields**:
- `payload.ids` - Array of read message IDs
- `payload.chat_id` - Chat where message was read
- `payload.receipt_type_description` - Human-readable description

**Example Use Case**: Read confirmation, engagement metrics

---

### Sender Receipt

**Description**: Sent by your other devices when a message you sent is delivered to them.

**Receipt Type**: `sender`

**Example Use Case**: Multi-device synchronization tracking

---

### Retry Receipt

**Description**: Message delivered but decryption failed.

**Receipt Type**: `retry`

**Example Use Case**: Error handling, message retry logic

---

### Played Receipt

**Description**: View-once media was opened by recipient or by you on another device.

**Receipt Type**: `played`

**Example Use Case**: View-once media tracking

---

### Played Self Receipt

**Description**: Current user opened a view-once media message from a different device, with read receipts disabled.

**Receipt Type**: `played_self`

**Example Use Case**: Self-device synchronization tracking

---

### Read Self Receipt

**Description**: Current user read a message from a different device, with read receipts disabled in privacy settings.

**Receipt Type**: `read_self`

**Example Use Case**: Multi-device read status tracking

## Group Events

All group events use `event: "group.participants"` with different `type` values.

### Group Member Join

**Description**: Users joined or were added to a group.

**Event Type**: `join`

**Key Fields**:
- `payload.chat_id` - Group ID
- `payload.jids` - Array of user JIDs who joined

**Example Use Case**: Welcome messages, member tracking, analytics

---

### Group Member Leave

**Description**: Users left or were removed from a group.

**Event Type**: `leave`

**Key Fields**:
- `payload.chat_id` - Group ID
- `payload.jids` - Array of user JIDs who left

**Example Use Case**: Member tracking, exit surveys, analytics

---

### Group Member Promotion

**Description**: Users promoted to group admin.

**Event Type**: `promote`

**Key Fields**:
- `payload.chat_id` - Group ID
- `payload.jids` - Array of user JIDs who were promoted

**Example Use Case**: Admin notifications, permission updates

---

### Group Member Demotion

**Description**: Users demoted from group admin.

**Event Type**: `demote`

**Key Fields**:
- `payload.chat_id` - Group ID
- `payload.jids` - Array of user JIDs who were demoted

**Example Use Case**: Admin notifications, permission updates

## Protocol Events

Protocol events are identified by the `action` field.

### Delete For Me

**Description**: Message deleted by sender for themselves only.

**Action**: `event.delete_for_me`

**Key Fields**:
- `deleted_message_id` - ID of deleted message
- `sender_id` - Who deleted the message
- `original_content` - Original message text (if found in database)
- `original_timestamp` - When message was originally sent
- `was_from_me` - Boolean indicating if it was your message

**Example Use Case**: Message deletion tracking, audit logging

**Note**: `original_*` fields only present if message was in local database.

---

### Message Revoked

**Description**: Message deleted for everyone in the chat.

**Action**: `message_revoked`

**Key Fields**:
- `revoked_message_id` - ID of revoked message
- `revoked_chat` - Chat where revocation occurred
- `revoked_from_me` - Boolean indicating if you revoked it
- `sender_id` - Who revoked the message

**Example Use Case**: Message deletion tracking, compliance, moderation

---

### Message Edited

**Description**: Message edited by sender.

**Action**: `message_edited`

**Key Fields**:
- `message.id` - Message ID
- `edited_text` - New message text
- `sender_id` - Who edited the message

**Example Use Case**: Edit history tracking, audit logging

## Event Summary Table

| **Category** | **Event/Action** | **Description** | **Identifier** |
|--------------|------------------|-----------------|----------------|
| **Messages** | Text Message | Plain text messages | `message.text` present |
| **Messages** | Reply Message | Reply to previous message | `message.replied_id` present |
| **Messages** | Reaction | Emoji reaction | `reaction` object present |
| **Messages** | Image | Image with optional caption | `image` object present |
| **Messages** | Video | Video with optional caption | `video` object present |
| **Messages** | Audio | Audio/voice note | `audio` object present |
| **Messages** | Document | File/document attachment | `document` object present |
| **Messages** | Sticker | Sticker message | `sticker` object present |
| **Messages** | Contact | Shared contact | `contact` object present |
| **Messages** | Location | Location sharing | `location` object present |
| **Messages** | Live Location | Real-time location | `location` with live data |
| **Messages** | List Message | Interactive list | `list` object present |
| **Messages** | Order Message | E-commerce order | `order` object present |
| **Messages** | View-Once | One-time viewable media | `view_once: true` |
| **Messages** | Forwarded | Forwarded message | `forwarded: true` |
| **Receipts** | Delivered | Message delivered | `event: message.ack`, `receipt_type: delivered` |
| **Receipts** | Read | Message read | `event: message.ack`, `receipt_type: read` |
| **Receipts** | Sender | Delivered to sender's device | `event: message.ack`, `receipt_type: sender` |
| **Receipts** | Retry | Decryption failed | `event: message.ack`, `receipt_type: retry` |
| **Receipts** | Played | View-once media opened | `event: message.ack`, `receipt_type: played` |
| **Receipts** | Played Self | Opened on another device | `event: message.ack`, `receipt_type: played_self` |
| **Receipts** | Read Self | Read on another device | `event: message.ack`, `receipt_type: read_self` |
| **Groups** | Member Join | Users joined group | `event: group.participants`, `type: join` |
| **Groups** | Member Leave | Users left group | `event: group.participants`, `type: leave` |
| **Groups** | Promote | Users promoted to admin | `event: group.participants`, `type: promote` |
| **Groups** | Demote | Users demoted from admin | `event: group.participants`, `type: demote` |
| **Protocol** | Delete For Me | Message deleted (sender only) | `action: event.delete_for_me` |
| **Protocol** | Message Revoked | Message deleted for everyone | `action: message_revoked` |
| **Protocol** | Message Edited | Message edited | `action: message_edited` |

## Event Filtering and Handling

### Filtering by Event Type

```javascript
function routeWebhook(data) {
    // Receipt events
    if (data.event === 'message.ack') {
        handleReceipt(data);
    }
    // Group events
    else if (data.event === 'group.participants') {
        handleGroupEvent(data);
    }
    // Protocol events
    else if (data.action) {
        handleProtocolEvent(data);
    }
    // Message events
    else {
        handleMessage(data);
    }
}
```

### Filtering by Message Type

```javascript
function handleMessage(data) {
    // Text messages
    if (data.message?.text) {
        handleTextMessage(data);
    }

    // Media messages
    if (data.image) handleImageMessage(data);
    if (data.video) handleVideoMessage(data);
    if (data.audio) handleAudioMessage(data);
    if (data.document) handleDocumentMessage(data);

    // Special messages
    if (data.contact) handleContactMessage(data);
    if (data.location) handleLocationMessage(data);
    if (data.reaction) handleReactionMessage(data);
}
```

### Filtering by Receipt Type

```javascript
function handleReceipt(data) {
    const receiptType = data.payload.receipt_type;

    switch (receiptType) {
        case 'delivered':
            handleDeliveredReceipt(data);
            break;
        case 'read':
            handleReadReceipt(data);
            break;
        case 'played':
            handlePlayedReceipt(data);
            break;
        default:
            console.log('Other receipt type:', receiptType);
    }
}
```

### Filtering by Group Event Type

```javascript
function handleGroupEvent(data) {
    const eventType = data.payload.type;

    switch (eventType) {
        case 'join':
            handleMemberJoin(data);
            break;
        case 'leave':
            handleMemberLeave(data);
            break;
        case 'promote':
            handleMemberPromote(data);
            break;
        case 'demote':
            handleMemberDemote(data);
            break;
    }
}
```

## Common Fields Across All Events

All webhook events include these common fields when applicable:

| **Field**   | **Type** | **Description** |
|-------------|----------|-----------------|
| `sender_id` | string   | User part of sender JID (phone number) |
| `chat_id`   | string   | User part of chat JID |
| `from`      | string   | Full JID of the sender |
| `from_lid`  | string   | (Optional) LID for identity-hidden accounts |
| `timestamp` | string   | RFC3339 formatted timestamp |
| `pushname`  | string   | Display name of the sender |

## Next Steps

- View detailed [Payload Schemas](payload-schemas.md) for complete JSON examples
- Learn about [Webhook Setup](../../guides/webhooks/setup.md)
- Review [Security Guidelines](../../guides/webhooks/security.md)
- Explore [Integration Examples](../../guides/webhooks/examples.md)

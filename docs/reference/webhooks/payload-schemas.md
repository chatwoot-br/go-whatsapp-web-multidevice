# Webhook Payload Schemas Reference

This document provides detailed payload schemas for all webhook event types with complete JSON examples.

## Table of Contents

- [Common Payload Fields](#common-payload-fields)
- [Message Events](#message-events)
- [Receipt Events](#receipt-events)
- [Group Events](#group-events)
- [Media Messages](#media-messages)
- [Special Message Types](#special-message-types)
- [Protocol Events](#protocol-events)

## Common Payload Fields

All webhook payloads share these common fields:

| **Field**   | **Type** | **Description**                                                                                     |
|-------------|----------|-----------------------------------------------------------------------------------------------------|
| `sender_id` | string   | User part of sender JID (phone number, without `@s.whatsapp.net`)                                   |
| `chat_id`   | string   | User part of chat JID                                                                               |
| `from`      | string   | Full JID of the sender (e.g., `628123456789@s.whatsapp.net`)                                        |
| `from_lid`  | string   | (Optional) LID (Lidded ID) of sender when using identity-hidden accounts (e.g., `20036609675500@lid`) |
| `timestamp` | string   | RFC3339 formatted timestamp (e.g., `2023-10-15T10:30:00Z`)                                          |
| `pushname`  | string   | Display name of the sender                                                                          |

## Message Events

### Text Message

Plain text message from a user.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T10:30:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "Hello, how are you?",
    "id": "3EB0C127D7BACC83D6A1",
    "replied_id": "",
    "quoted_message": ""
  }
}
```

**Field Descriptions**:
- `message.text`: The text content of the message
- `message.id`: Unique message identifier
- `message.replied_id`: ID of message being replied to (empty if not a reply)
- `message.quoted_message`: Text of the quoted message (empty if not a reply)

---

### Reply Message

Message that replies to a previous message.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T10:35:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "I'm doing great, thanks!",
    "id": "3EB0C127D7BACC83D6A2",
    "replied_id": "3EB0C127D7BACC83D6A1",
    "quoted_message": "Hello, how are you?"
  }
}
```

**Field Descriptions**:
- `message.replied_id`: Contains the ID of the message being replied to
- `message.quoted_message`: Contains the text of the message being replied to

---

### Reaction Message

Emoji reaction to a message.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T10:40:00Z",
  "pushname": "John Doe",
  "reaction": {
    "message": "👍",
    "id": "3EB0C127D7BACC83D6A1"
  },
  "message": {
    "text": "",
    "id": "88760C69D1F35FEB239102699AE9XXXX",
    "replied_id": "",
    "quoted_message": ""
  }
}
```

**Field Descriptions**:
- `reaction.message`: The emoji reaction
- `reaction.id`: ID of the message being reacted to
- `message.text`: Empty for reaction messages
- `message.id`: ID of the reaction event itself

---

## Receipt Events

Receipt events are triggered when messages receive acknowledgments such as delivery confirmations and read receipts.

### Message Delivered

Triggered when a message is successfully delivered to the recipient's device.

```json
{
  "event": "message.ack",
  "payload": {
    "chat_id": "120363402106XXXXX@g.us",
    "from": "6289685XXXXXX@s.whatsapp.net in 120363402106XXXXX@g.us",
    "ids": [
      "3EB00106E8BE0F407E88EC"
    ],
    "receipt_type": "delivered",
    "receipt_type_description": "means the message was delivered to the device (but the user might not have noticed).",
    "sender_id": "6289685XXXXXX@s.whatsapp.net"
  },
  "timestamp": "2025-07-18T22:44:20Z"
}
```

**Field Descriptions**:
- `event`: Always `"message.ack"` for receipt events
- `payload.chat_id`: Chat identifier (group or individual chat)
- `payload.from`: Sender information with chat context
- `payload.ids`: Array of message IDs that received the acknowledgment
- `payload.receipt_type`: Type of receipt (see types below)
- `payload.receipt_type_description`: Human-readable description
- `payload.sender_id`: JID of the message sender
- `timestamp`: When the receipt was received

---

### Message Read

Triggered when a message is read by the recipient (they opened the chat and saw the message).

```json
{
  "event": "message.ack",
  "payload": {
    "chat_id": "120363402106XXXXX@g.us",
    "from": "6289685XXXXXX@s.whatsapp.net in 120363402106XXXXX@g.us",
    "ids": [
      "3EB00106E8BE0F407E88EC"
    ],
    "receipt_type": "read",
    "receipt_type_description": "the user opened the chat and saw the message.",
    "sender_id": "6289685XXXXXX@s.whatsapp.net"
  },
  "timestamp": "2025-07-18T22:44:44Z"
}
```

---

### Receipt Types

| **Receipt Type** | **Description** |
|------------------|-----------------|
| `delivered` | Message was delivered to the device (but the user might not have noticed) |
| `read` | User opened the chat and saw the message |
| `sender` | Sent by your other devices when a message you sent is delivered to them |
| `retry` | Message was delivered to the device, but decrypting the message failed |
| `read_self` | Current user read a message from a different device, and has read receipts disabled in privacy settings |
| `played` | View-once media was opened (by recipient or by you on another device) |
| `played_self` | Current user opened a view-once media message from a different device, and has read receipts disabled |

---

## Group Events

Group events are triggered when group metadata changes, including member join/leave events, admin promotions/demotions, and group settings updates.

### Group Member Join

Triggered when users join or are added to a group.

```json
{
  "event": "group.participants",
  "payload": {
    "chat_id": "120363402106XXXXX@g.us",
    "type": "join",
    "jids": [
      "6289685XXXXXX@s.whatsapp.net",
      "6289686YYYYYY@s.whatsapp.net"
    ]
  },
  "timestamp": "2025-07-28T10:30:00Z"
}
```

**Field Descriptions**:
- `event`: Always `"group.participants"` for group events
- `payload.chat_id`: Group identifier (ends with `@g.us`)
- `payload.type`: Action type (`join`, `leave`, `promote`, or `demote`)
- `payload.jids`: Array of user JIDs affected by this action
- `timestamp`: When the group event occurred

---

### Group Member Leave

Triggered when users leave or are removed from a group.

```json
{
  "event": "group.participants",
  "payload": {
    "chat_id": "120363402106XXXXX@g.us",
    "type": "leave",
    "jids": [
      "6289687ZZZZZZ@s.whatsapp.net"
    ]
  },
  "timestamp": "2025-07-28T10:32:00Z"
}
```

---

### Group Member Promotion

Triggered when users are promoted to admin.

```json
{
  "event": "group.participants",
  "payload": {
    "chat_id": "120363402106XXXXX@g.us",
    "type": "promote",
    "jids": [
      "6289688AAAAAA@s.whatsapp.net"
    ]
  },
  "timestamp": "2025-07-28T10:33:00Z"
}
```

---

### Group Member Demotion

Triggered when users are demoted from admin.

```json
{
  "event": "group.participants",
  "payload": {
    "chat_id": "120363402106XXXXX@g.us",
    "type": "demote",
    "jids": [
      "6289689BBBBBB@s.whatsapp.net"
    ]
  },
  "timestamp": "2025-07-28T10:34:00Z"
}
```

---

## Media Messages

### Image Message

Image message with optional caption.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628123456789",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2025-07-13T11:05:51Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6A3",
    "replied_id": "",
    "quoted_message": ""
  },
  "image": {
    "media_path": "statics/media/1752404751-ad9e37ac-c658-4fe5-8d25-ba4a3f4d58fd.jpe",
    "mime_type": "image/jpeg",
    "caption": "Check out this photo!"
  }
}
```

**Field Descriptions**:
- `image.media_path`: Local file path where the image is stored
- `image.mime_type`: MIME type of the image (e.g., `image/jpeg`, `image/png`)
- `image.caption`: Optional caption text (empty string if no caption)
- `message.text`: Empty for media messages

---

### Video Message

Video message with optional caption.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628123456789",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2025-07-13T11:07:24Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6A4",
    "replied_id": "",
    "quoted_message": ""
  },
  "video": {
    "media_path": "statics/media/1752404845-b9393cd1-8546-4df9-8a60-ee3276036aba.m4v",
    "mime_type": "video/mp4",
    "caption": "Watch this video"
  }
}
```

**Field Descriptions**:
- `video.media_path`: Local file path where the video is stored
- `video.mime_type`: MIME type of the video (e.g., `video/mp4`, `video/3gpp`)
- `video.caption`: Optional caption text

---

### Audio Message

Audio message including voice notes.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T10:55:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6A5",
    "replied_id": "",
    "quoted_message": ""
  },
  "audio": {
    "media_path": "statics/media/1752404905-b9393cd1-8546-4df9-8a60-ee3276036aba.m4v",
    "mime_type": "audio/ogg",
    "caption": ""
  }
}
```

**Field Descriptions**:
- `audio.media_path`: Local file path where the audio is stored
- `audio.mime_type`: MIME type of the audio (e.g., `audio/ogg`, `audio/mp4`)
- `audio.caption`: Usually empty for audio messages

---

### Document Message

Document/file attachment with optional caption.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T11:00:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6A6",
    "replied_id": "",
    "quoted_message": ""
  },
  "document": {
    "media_path": "statics/media/1752404965-b9393cd1-8546-4df9-8a60-ee3276036aba.pdf",
    "mime_type": "application/pdf",
    "caption": "Important document"
  }
}
```

**Field Descriptions**:
- `document.media_path`: Local file path where the document is stored
- `document.mime_type`: MIME type of the document (e.g., `application/pdf`, `application/vnd.ms-excel`)
- `document.caption`: Optional caption text

---

### Sticker Message

Sticker message (static or animated WebP format).

```json
{
  "chat_id": "628968XXXXXXXX",
  "from": "628968XXXXXXXX@s.whatsapp.net",
  "message": {
    "text": "",
    "id": "446AC2BAF2061B53E24CA526DBDFBD4E",
    "replied_id": "",
    "quoted_message": ""
  },
  "pushname": "John Doe",
  "sender_id": "628968XXXXXXXX",
  "sticker": {
    "media_path": "statics/media/1752404986-ff2464a6-c54c-4e6c-afde-c4c925ce3573.webp",
    "mime_type": "image/webp",
    "caption": ""
  },
  "timestamp": "2025-07-13T11:09:45Z"
}
```

**Field Descriptions**:
- `sticker.media_path`: Local file path where the sticker is stored
- `sticker.mime_type`: Always `image/webp`
- `sticker.caption`: Always empty for stickers

---

## Special Message Types

### Contact Message

Shared contact information in vCard format.

```json
{
  "chat_id": "6289XXXXXXXXX",
  "contact": {
    "displayName": "Jane Smith",
    "vcard": "BEGIN:VCARD\nVERSION:3.0\nN:;Jane Smith;;;\nFN:Jane Smith\nTEL;type=Mobile:+62 812 3456 7890\nEND:VCARD",
    "contextInfo": {
      "expiration": 7776000,
      "ephemeralSettingTimestamp": 1751808692,
      "disappearingMode": {
        "initiator": 0,
        "trigger": 1,
        "initiatedByMe": true
      }
    }
  },
  "from": "6289XXXXXXXXX@s.whatsapp.net",
  "message": {
    "text": "",
    "id": "56B3DFF4994284634E7AAFEEF6F1A0A2",
    "replied_id": "",
    "quoted_message": ""
  },
  "pushname": "John Doe",
  "sender_id": "6289XXXXXXXXX",
  "timestamp": "2025-07-13T11:10:19Z"
}
```

**Field Descriptions**:
- `contact.displayName`: The display name of the shared contact
- `contact.vcard`: vCard formatted contact data (version 3.0)
- `contact.contextInfo`: Additional context (ephemeral settings, etc.)

**vCard Format**: Standard vCard 3.0 format containing:
- `N`: Name components (surname, given name, etc.)
- `FN`: Formatted name
- `TEL`: Phone number(s)

---

### Location Message

Location sharing with coordinates and address information.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T11:15:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6A9",
    "replied_id": "",
    "quoted_message": ""
  },
  "location": {
    "degreesLatitude": -6.2088,
    "degreesLongitude": 106.8456,
    "name": "Jakarta, Indonesia",
    "address": "Central Jakarta, DKI Jakarta, Indonesia"
  }
}
```

**Field Descriptions**:
- `location.degreesLatitude`: Latitude coordinate (decimal degrees)
- `location.degreesLongitude`: Longitude coordinate (decimal degrees)
- `location.name`: Location name or title
- `location.address`: Full address description

---

### Live Location Message

Real-time location sharing updates.

**Note**: Live location messages contain the raw WhatsApp protocol structure, sent as-is from the WhatsApp protocol.

```json
{
  "chat_id": "6289XXXXXXXXX",
  "from": "6289XXXXXXXXX@s.whatsapp.net",
  "location": {
    "degreesLatitude": -7.8050297,
    "degreesLongitude": 110.4549165,
    "JPEGThumbnail": "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBD...(base64 encoded image)",
    "contextInfo": {
      "expiration": 7776000,
      "ephemeralSettingTimestamp": 1751808692,
      "disappearingMode": {
        "initiator": 0,
        "trigger": 1,
        "initiatedByMe": true
      }
    }
  },
  "message": {
    "text": "",
    "id": "94D13237B4D7F33EE4A63228BBD79EC0",
    "replied_id": "",
    "quoted_message": ""
  },
  "pushname": "John Doe",
  "sender_id": "6289685XXXXXX",
  "timestamp": "2025-07-13T11:11:22Z"
}
```

**Field Descriptions**:
- `location.degreesLatitude`: Current latitude
- `location.degreesLongitude`: Current longitude
- `location.JPEGThumbnail`: Base64 encoded map thumbnail image
- `location.contextInfo`: Additional context including ephemeral settings

---

### List Message

Interactive list message with selectable options.

**Note**: The WhatsApp list message structure is sent as-is from the WhatsApp protocol.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T11:20:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6AA",
    "replied_id": "",
    "quoted_message": ""
  },
  "list": {
    "title": "Choose an option",
    "description": "Please select from the following options",
    "buttonText": "View Options",
    "listType": 1,
    "sections": [
      {
        "title": "Section 1",
        "rows": [
          {
            "title": "Option 1",
            "description": "Description for option 1",
            "rowId": "option_1"
          },
          {
            "title": "Option 2",
            "description": "Description for option 2",
            "rowId": "option_2"
          }
        ]
      }
    ]
  }
}
```

**Field Descriptions**:
- `list.title`: Main title of the list
- `list.description`: Description text
- `list.buttonText`: Text shown on the button to open the list
- `list.listType`: Type of list (1 for single select)
- `list.sections`: Array of sections containing selectable rows
- `list.sections[].rows[].rowId`: Unique identifier for each option

---

### Order Message

E-commerce order information.

**Note**: The WhatsApp order message structure is sent as-is from the WhatsApp protocol.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T11:25:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6AB",
    "replied_id": "",
    "quoted_message": ""
  },
  "order": {
    "orderTitle": "Order #12345",
    "itemCount": 2,
    "message": "Thank you for your order!",
    "orderSurface": "CATALOG",
    "sellerJid": "628987654321@s.whatsapp.net"
  }
}
```

**Field Descriptions**:
- `order.orderTitle`: Title or identifier of the order
- `order.itemCount`: Number of items in the order
- `order.message`: Custom message from seller
- `order.orderSurface`: Order source (e.g., `CATALOG`)
- `order.sellerJid`: WhatsApp JID of the seller

---

## Protocol Events

### Delete For Me

Triggered when a message is deleted by the sender for themselves only.

```json
{
  "action": "event.delete_for_me",
  "deleted_message_id": "3EB0C127D7BACC83D6A8",
  "sender_id": "6289685XXXXXX",
  "from": "6289685XXXXXX@s.whatsapp.net",
  "chat_id": "6289685XXXXXX@s.whatsapp.net",
  "original_content": "This was the original message text",
  "original_sender": "6289685XXXXXX@s.whatsapp.net",
  "original_timestamp": "2025-07-13T10:00:00Z",
  "was_from_me": true,
  "original_media_type": "image",
  "original_filename": "photo.jpg",
  "timestamp": "2025-07-13T11:13:30Z"
}
```

**Field Descriptions**:
- `action`: Always `"event.delete_for_me"`
- `deleted_message_id`: ID of the deleted message
- `sender_id`: Who deleted the message
- `chat_id`: Chat where deletion occurred
- `original_content`: Original message text (only if found in database)
- `original_sender`: Original sender JID (only if found in database)
- `original_timestamp`: When message was originally sent (only if found in database)
- `was_from_me`: Whether it was your message (only if found in database)
- `original_media_type`: Type of media if it was a media message (only if found in database)
- `original_filename`: Original filename for media (only if found in database)
- `timestamp`: When the deletion occurred

**Note**: Fields prefixed with `original_*` and `was_from_me` are only present if the message was found in the local database.

---

### Message Revoked

Message deleted for everyone in the chat.

```json
{
  "action": "message_revoked",
  "chat_id": "6289XXXXXXXXX",
  "from": "6289XXXXXXXXX@s.whatsapp.net",
  "message": {
    "text": "",
    "id": "F4062F2BBCB19B7432195AD7080DA4E2",
    "replied_id": "",
    "quoted_message": ""
  },
  "pushname": "John Doe",
  "revoked_chat": "6289XXXXXXXXX@s.whatsapp.net",
  "revoked_from_me": true,
  "revoked_message_id": "94D13237B4D7F33EE4A63228BBD79EC0",
  "sender_id": "6289XXXXXXXXX",
  "timestamp": "2025-07-13T11:13:30Z"
}
```

**Field Descriptions**:
- `action`: Always `"message_revoked"`
- `revoked_message_id`: ID of the revoked message
- `revoked_chat`: Chat where revocation occurred
- `revoked_from_me`: Boolean indicating if you revoked the message
- `sender_id`: Who revoked the message
- `timestamp`: When the revocation occurred

---

### Message Edited

Message edited by the sender.

```json
{
  "action": "message_edited",
  "chat_id": "6289XXXXXXXXX",
  "edited_text": "This is the edited message text",
  "from": "6289XXXXXXXXX@s.whatsapp.net",
  "message": {
    "text": "This is the edited message text",
    "id": "D6271D8223A05B4DA6AE9FE3CD632543",
    "replied_id": "",
    "quoted_message": ""
  },
  "pushname": "John Doe",
  "sender_id": "6289XXXXXXXXX",
  "timestamp": "2025-07-13T11:14:19Z"
}
```

**Field Descriptions**:
- `action`: Always `"message_edited"`
- `edited_text`: The new message text after editing
- `message.text`: Same as `edited_text`
- `message.id`: ID of the edited message
- `sender_id`: Who edited the message
- `timestamp`: When the edit occurred

---

## Special Flags

### View Once Message

Media messages that can only be viewed once.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T11:40:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "",
    "id": "3EB0C127D7BACC83D6B2",
    "replied_id": "",
    "quoted_message": ""
  },
  "image": {
    "media_path": "statics/media/1752405060-b9393cd1-8546-4df9-8a60-ee3276036aba.jpg",
    "mime_type": "image/jpeg",
    "caption": "View this once"
  },
  "view_once": true
}
```

**Field Descriptions**:
- `view_once`: Boolean flag set to `true` for view-once messages
- Combined with media fields (`image`, `video`, etc.)

---

### Forwarded Message

Messages that were forwarded from another chat.

```json
{
  "sender_id": "628123456789",
  "chat_id": "628987654321",
  "from": "628123456789@s.whatsapp.net",
  "timestamp": "2023-10-15T11:45:00Z",
  "pushname": "John Doe",
  "message": {
    "text": "This is a forwarded message",
    "id": "3EB0C127D7BACC83D6B3",
    "replied_id": "",
    "quoted_message": ""
  },
  "forwarded": true
}
```

**Field Descriptions**:
- `forwarded`: Boolean flag set to `true` for forwarded messages
- Can be combined with any message type (text, media, etc.)

---

## Media Types Summary

The following media types are supported in webhook payloads:

| **Media Type** | **Object** | **MIME Types** | **Caption Support** |
|----------------|------------|----------------|---------------------|
| Image | `image` | `image/jpeg`, `image/png`, `image/webp`, `image/gif` | Yes |
| Video | `video` | `video/mp4`, `video/3gpp`, `video/quicktime` | Yes |
| Audio | `audio` | `audio/ogg`, `audio/mp4`, `audio/mpeg` | Usually empty |
| Document | `document` | Various (PDF, Excel, Word, etc.) | Yes |
| Sticker | `sticker` | `image/webp` | No (always empty) |

Each media object contains:
- `media_path`: Local file path where media is stored (relative to application root)
- `mime_type`: MIME type of the media file
- `caption`: Optional caption text (empty string if not present)

## Next Steps

- Learn about [Event Types](event-types.md) for a quick reference of all events
- Review [Webhook Setup](../../guides/webhooks/setup.md) for configuration
- Check [Security Guidelines](../../guides/webhooks/security.md) for HMAC verification
- Explore [Integration Examples](../../guides/webhooks/examples.md) for working code

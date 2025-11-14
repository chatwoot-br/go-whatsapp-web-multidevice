# Send Your First Message

Complete guide to logging in and sending your first WhatsApp message via the API.

## Table of Contents

- [Before You Begin](#before-you-begin)
- [Login Methods](#login-methods)
- [Send Text Messages](#send-text-messages)
- [Send Images](#send-images)
- [Send Other Media Types](#send-other-media-types)
- [Verify Message Delivery](#verify-message-delivery)
- [Common Issues](#common-issues)

## Before You Begin

Ensure you have:

1. **Application running** on `http://localhost:3000` (or your configured port)
2. **FFmpeg installed** for media processing
3. **Active WhatsApp account** ready to connect

**Check if application is running:**
```bash
# Test endpoint
curl -I http://localhost:3000

# Should return HTTP/1.1 200 OK
```

## Login Methods

You must login to WhatsApp before sending messages. Choose one of these methods:

### Method 1: QR Code via Web Interface

The easiest method for first-time setup.

**Steps:**

1. **Open browser:**
   ```bash
   open http://localhost:3000
   ```

2. **Navigate to Login page:**
   - Click on **Login** in the menu
   - Or go directly to: `http://localhost:3000/app/login`

3. **Scan QR code:**
   - A QR code will be displayed on screen
   - Open WhatsApp on your phone
   - Go to **Settings** → **Linked Devices** → **Link a Device**
   - Scan the QR code displayed on your screen

4. **Wait for confirmation:**
   - You'll see a success message when connected
   - Device name will appear as "Chrome (MyApp)" or your configured device name

### Method 2: QR Code via API

For automation or headless environments.

**Request:**
```bash
curl -X GET http://localhost:3000/app/login
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "qr_link": "http://localhost:3000/statics/images/qrcode/scan-qr-xxx.png",
    "qr_duration": 30
  }
}
```

**Steps:**

1. **Get QR code URL** from response
2. **Open the image URL** in browser or display it
3. **Scan with WhatsApp mobile app:**
   - Open WhatsApp → **Settings** → **Linked Devices** → **Link a Device**
   - Scan the QR code

**Note:** QR code expires after 30 seconds. Request a new one if expired.

### Method 3: Pairing Code

Link device using an 8-character code instead of QR scanning.

**Request:**
```bash
# Replace with your phone number (include country code, no + sign)
curl -X GET "http://localhost:3000/app/login-with-code?phone=5511999998888"
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "pair_code": "ABCD-1234"
  }
}
```

**Steps:**

1. **Get pairing code** from response (e.g., `ABCD-1234`)
2. **Open WhatsApp on your phone**
3. Go to **Settings** → **Linked Devices** → **Link a Device**
4. Tap **Link with phone number instead**
5. **Enter the pairing code**: `ABCD-1234`
6. **Confirm** on your phone

**Phone Number Format:**
- Include country code, no `+` sign
- Examples:
  - Brazil: `5511999998888` (55 = country, 11 = area, 999998888 = number)
  - USA: `14155552671` (1 = country)
  - India: `919876543210` (91 = country)

### Verify Connection

After logging in, verify your connection:

```bash
curl http://localhost:3000/app/devices
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": [
    {
      "device": "Chrome (MyApp)",
      "platform": "Chrome",
      "connected": true
    }
  ]
}
```

**Check your user info:**
```bash
curl http://localhost:3000/user/info
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "phone": "5511999998888",
    "name": "Your Name",
    "status": "Hey there! I am using WhatsApp.",
    "connected": true
  }
}
```

## Send Text Messages

### Basic Text Message

Send a simple text message to a phone number.

**Request:**
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Hello from WhatsApp API!"
  }'
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "message_id": "3EB0C431D4D2E2D2F3E8",
    "status": "sent",
    "timestamp": "2025-11-14T10:30:00Z"
  }
}
```

**Parameters:**
- `phone` (required): Recipient's phone number with country code
- `message` (required): Text message content

### Message with Line Breaks

Include newlines in your messages:

**Request:**
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Hello!\n\nThis is a multi-line message.\n\nBest regards,\nAPI Bot"
  }'
```

### Message with Mentions

Mention other WhatsApp users in your message:

**Request:**
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Hello @5511999998888, how are you?"
  }'
```

**Format:** Use `@` followed by the phone number (with country code).

**Multiple mentions:**
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Team meeting: @5511999998888, @5511999997777, @5511999996666"
  }'
```

### Reply to a Message

Reply to a specific message:

**Request:**
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Thanks for your message!",
    "reply_message_id": "3EB0C431D4D2E2D2F3E8"
  }'
```

**Parameters:**
- `reply_message_id`: The ID of the message you're replying to

**Note:** Get message IDs from webhook events or chat history.

### Send to Group

Send message to a WhatsApp group:

**Request:**
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "120363028943211686@g.us",
    "message": "Hello everyone in the group!"
  }'
```

**Group JID Format:** `[group-id]@g.us`

**Get your group IDs:**
```bash
curl http://localhost:3000/user/my/groups
```

## Send Images

### Send Image from URL

Send an image from a publicly accessible URL:

**Request:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/image.jpg",
    "caption": "Check out this image!"
  }'
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "message_id": "3EB0C431D4D2E2D2F3E9",
    "status": "sent"
  }
}
```

**Parameters:**
- `phone` (required): Recipient's phone number
- `image` (required): URL or base64 encoded image
- `caption` (optional): Image caption text
- `compress` (optional): Compress image (default: true)
- `view_once` (optional): View once mode (default: false)

### Send Image from File

Send an image file from your local machine:

**Request:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "caption=Beautiful photo!" \
  -F "image=@/path/to/image.jpg"
```

### Send Compressed Image

By default, images are compressed. To send original quality:

**Request:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/image.jpg",
    "caption": "Original quality image",
    "compress": false
  }'
```

### Send View Once Image

Send image that can only be viewed once:

**Request:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/secret.jpg",
    "caption": "View once only!",
    "view_once": true
  }'
```

### Send Base64 Encoded Image

Send image as base64 string:

**Request:**
```bash
# First, encode image to base64
base64_image=$(base64 -i image.jpg)

curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d "{
    \"phone\": \"5511999998888\",
    \"image\": \"data:image/jpeg;base64,$base64_image\",
    \"caption\": \"Base64 encoded image\"
  }"
```

## Send Other Media Types

### Send Video

**Request:**
```bash
curl -X POST http://localhost:3000/send/video \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "video": "https://example.com/video.mp4",
    "caption": "Check out this video!",
    "compress": true
  }'
```

**Or from file:**
```bash
curl -X POST http://localhost:3000/send/video \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "caption=Amazing video!" \
  -F "video=@/path/to/video.mp4"
```

### Send Audio

**Request:**
```bash
curl -X POST http://localhost:3000/send/audio \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "audio": "https://example.com/audio.mp3"
  }'
```

**Or from file:**
```bash
curl -X POST http://localhost:3000/send/audio \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "audio=@/path/to/audio.mp3"
```

### Send Document/File

**Request:**
```bash
curl -X POST http://localhost:3000/send/file \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "file": "https://example.com/document.pdf",
    "filename": "report.pdf"
  }'
```

**Or from file:**
```bash
curl -X POST http://localhost:3000/send/file \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "file=@/path/to/document.pdf"
```

### Send Sticker

The API automatically converts images to WebP sticker format.

**Request:**
```bash
curl -X POST http://localhost:3000/send/sticker \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "sticker": "https://example.com/sticker.png"
  }'
```

**Supported formats:** JPG, JPEG, PNG, WebP, GIF
**Auto-resize:** 512x512 pixels
**Transparency:** Preserved for PNG images

### Send Contact

**Request:**
```bash
curl -X POST http://localhost:3000/send/contact \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "contact_name": "John Doe",
    "contact_phone": "5511988887777"
  }'
```

### Send Location

**Request:**
```bash
curl -X POST http://localhost:3000/send/location \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "latitude": -23.550520,
    "longitude": -46.633308,
    "name": "São Paulo Cathedral",
    "address": "Praça da Sé, São Paulo - SP, Brazil"
  }'
```

### Send Link with Preview

**Request:**
```bash
curl -X POST http://localhost:3000/send/link \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "link": "https://github.com",
    "caption": "Check out this website!"
  }'
```

## Verify Message Delivery

### Check Message Status

Use webhooks to receive message status updates:

**Webhook events:**
- `message.sent` - Message sent successfully
- `message.delivered` - Message delivered to recipient
- `message.read` - Message read by recipient
- `message.failed` - Message failed to send

**Setup webhook:**
```bash
./whatsapp rest --webhook="https://your-webhook.site/handler"
```

**Example webhook payload:**
```json
{
  "event": "message.delivered",
  "data": {
    "message_id": "3EB0C431D4D2E2D2F3E8",
    "from": "5511999998888@s.whatsapp.net",
    "timestamp": "2025-11-14T10:30:00Z",
    "status": "delivered"
  }
}
```

See [Webhook Documentation](../webhook-payload.md) for complete details.

### Query Message History

Get messages from a chat:

**Request:**
```bash
curl -X GET "http://localhost:3000/chat/5511999998888@s.whatsapp.net/messages?limit=10"
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "messages": [
      {
        "message_id": "3EB0C431D4D2E2D2F3E8",
        "from": "me",
        "text": "Hello from WhatsApp API!",
        "timestamp": "2025-11-14T10:30:00Z"
      }
    ]
  }
}
```

## Common Issues

### Message Not Sending

**Problem:** Message fails with error

**Solutions:**

**1. Check phone number format:**
```bash
# Correct format (with country code, no + or spaces)
✅ 5511999998888
❌ +55 11 99999-8888
❌ 11999998888
❌ +5511999998888
```

**2. Verify number is on WhatsApp:**
```bash
curl -X GET "http://localhost:3000/user/check?phone=5511999998888"
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "exists": true,
    "jid": "5511999998888@s.whatsapp.net"
  }
}
```

**3. Check connection status:**
```bash
curl http://localhost:3000/app/devices
```

**4. Disable account validation (not recommended):**
```bash
./whatsapp rest --account-validation=false
```

### Connection Lost

**Problem:** Lost connection to WhatsApp

**Solutions:**

**1. Check connection status:**
```bash
curl http://localhost:3000/app/devices
```

**2. Reconnect:**
```bash
curl http://localhost:3000/app/reconnect
```

**3. Logout and login again:**
```bash
# Logout
curl http://localhost:3000/app/logout

# Login with QR code
curl http://localhost:3000/app/login
```

### Image/Media Not Sending

**Problem:** Media files fail to send

**Solutions:**

**1. Verify FFmpeg is installed:**
```bash
ffmpeg -version
```

**2. Check file size limits:**
- Images: Max 20MB
- Videos: Max 100MB
- Files: Max 50MB
- Audio: Max 16MB

**3. Check URL accessibility:**
```bash
# Test if URL is accessible
curl -I https://example.com/image.jpg
```

**4. Try base64 encoding:**
```bash
base64_image=$(base64 -i image.jpg)
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d "{
    \"phone\": \"5511999998888\",
    \"image\": \"data:image/jpeg;base64,$base64_image\"
  }"
```

**5. Check image format:**
Supported formats: JPG, JPEG, PNG, WebP, GIF

**6. Disable compression:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/image.jpg",
    "compress": false
  }'
```

### Authentication Errors

**Problem:** 401 Unauthorized

**Solution:**

If you enabled basic authentication:

```bash
# Include credentials in request
curl -u username:password \
  -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Hello!"
  }'

# Or use Authorization header
curl -X POST http://localhost:3000/send/message \
  -H "Authorization: Basic $(echo -n username:password | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Hello!"
  }'
```

### Rate Limiting

**Problem:** WhatsApp blocking or rate limiting messages

**Solution:**

**1. Add delays between messages:**
```bash
# Send messages with 2-second delay
for phone in 5511999998888 5511999997777; do
  curl -X POST http://localhost:3000/send/message \
    -H "Content-Type: application/json" \
    -d "{\"phone\": \"$phone\", \"message\": \"Hello!\"}"
  sleep 2
done
```

**2. Monitor for blocks:**
- Watch for error messages
- Reduce sending frequency
- Use official WhatsApp Business API for high volume

### Invalid JID Format

**Problem:** Error about invalid JID

**Solution:**

JID (Jabber ID) is WhatsApp's internal identifier format:

**For individuals:**
```
5511999998888@s.whatsapp.net
```

**For groups:**
```
120363028943211686@g.us
```

**For newsletters:**
```
25220266111770696@newsletter
```

Most endpoints accept just the phone number, and the API adds `@s.whatsapp.net` automatically. For groups and newsletters, use the full JID.

**Get group JIDs:**
```bash
curl http://localhost:3000/user/my/groups
```

## Next Steps

Now that you can send messages, explore:

1. **[Configuration Basics](configuration-basics.md)** - Advanced configuration options
2. **[Webhook Integration](../webhook-payload.md)** - Receive WhatsApp events
3. **[API Documentation](../reference/openapi.yaml)** - Complete API reference
4. **[Group Management](../guides/)** - Create and manage groups
5. **[Chat History](../guides/)** - Query chat messages and history

## Testing Checklist

Use this checklist to verify your setup:

- [ ] Application is running and accessible
- [ ] Successfully logged in to WhatsApp
- [ ] Connection status shows "connected"
- [ ] Text message sent successfully
- [ ] Image sent successfully
- [ ] Message appears in WhatsApp mobile app
- [ ] Can reply to messages from mobile app
- [ ] Webhook receiving events (if configured)

## Quick Reference

### Essential Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/app/login` | GET | Get QR code |
| `/app/login-with-code?phone=` | GET | Get pairing code |
| `/app/devices` | GET | Connection status |
| `/user/info` | GET | Your user info |
| `/user/check?phone=` | GET | Check if number exists |
| `/send/message` | POST | Send text message |
| `/send/image` | POST | Send image |
| `/send/video` | POST | Send video |
| `/send/audio` | POST | Send audio |
| `/send/file` | POST | Send document |
| `/send/sticker` | POST | Send sticker |
| `/send/contact` | POST | Send contact |
| `/send/location` | POST | Send location |
| `/send/link` | POST | Send link |

### Phone Number Format

Always use international format without `+` or spaces:

| Country | Format | Example |
|---------|--------|---------|
| Brazil | 55AAXXXXXXXXX | 5511999998888 |
| USA | 1AAANNNNNNNN | 14155552671 |
| UK | 44AAANNNNNNNN | 447911123456 |
| India | 91AAAAAAAAAA | 919876543210 |

Where:
- AA = Area/region code
- N = Number
- X = Digits

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-11-14

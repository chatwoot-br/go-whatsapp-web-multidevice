# ISSUE-002: LID chat_id Not Normalized Causes Duplicate Contacts in Chatwoot

## Status: Fixed

## Summary
When a message arrives from a contact whose JID is in LID format (`@lid`), the `chat_id` field in the webhook payload was not being normalized to the phone number format (`@s.whatsapp.net`). This caused Chatwoot to create duplicate contacts instead of matching existing contacts by phone number.

## Environment
- **go-whatsapp-web-multidevice**: v4.9.1+2
- **Chatwoot**: chatwoot-br fork
- **Test scenario**: Message "ping acme" from contact 552140402221 (LID: 174852883411040@lid)

## Problem Description

### Observed Behavior
When a message was received from a contact whose chat JID was in LID format:

1. go-whatsapp correctly resolved the LID to phone number in logs
2. The `from` field was correctly normalized to phone number
3. **BUG**: The `chat_id` field was NOT normalized (still contained `@lid`)
4. Chatwoot received `chat_id` ending with `@lid` and created a new LID-based contact
5. Existing contact with phone +552140402221 was not found, resulting in duplicate

### Log Evidence (2026-01-15 20:22:59)

```
Received message 3EB0BB53CA368F89047FC6 from 174852883411040:28@lid in 174852883411040@lid
Resolved LID 174852883411040@lid to phone number 552140402221@s.whatsapp.net
Resolved LID 174852883411040:28@lid to phone number 552140402221:28@s.whatsapp.net
Forwarding message event to 1 configured webhook(s)
```

**Webhook payload sent (BEFORE FIX)**:
```json
{
  "event": "message",
  "device_id": "5521998762522@s.whatsapp.net",
  "payload": {
    "id": "3EB0BB53CA368F89047FC6",
    "chat_id": "174852883411040@lid",        // NOT normalized (BUG)
    "from": "552140402221@s.whatsapp.net",   // Correctly normalized
    "from_lid": "174852883411040:28@lid",
    "from_name": "Chatwoot Brasil",
    "body": "ping acme"
  }
}
```

### Visual Evidence

Two conversations were created in Chatwoot for the same contact:

| Conversation | Contact Name | Phone | Avatar | Source ID |
|--------------|--------------|-------|--------|-----------|
| 41 | Chatwoot Brasil | +552140402221 | Green logo | 552140402221@s.whatsapp.net |
| 42 | Chatwoot Brasil | (none) | CB initials | 174852883411040@lid |

### Expected Behavior
The `chat_id` should be normalized to phone number format when the LID-to-phone mapping is known, ensuring Chatwoot can match existing contacts.

## Root Cause Analysis

### Code Location
**File**: `src/infrastructure/whatsapp/event_message.go`
**Function**: `buildFromFields()`

### The Bug
```go
func buildFromFields(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) {
    // BUG: chat_id was NOT normalized
    payload["chat_id"] = evt.Info.Chat.ToNonAD().String()  // → 174852883411040@lid

    // from WAS correctly normalized
    normalizedSenderJID := NormalizeJIDFromLID(ctx, senderJID, client)
    payload["from"] = normalizedSenderJID.ToNonAD().String()  // → 552140402221@s.whatsapp.net
}
```

### Chatwoot's Handling
In `incoming_message_whatsapp_web_service.rb`, Chatwoot checks:
```ruby
def lid_based_chat?
  webhook_params.dig(:payload, :chat_id).to_s.end_with?('@lid')
end
```

When this returns `true`, Chatwoot creates a new LID-based contact instead of looking up by phone number.

## Fix Applied

### Commit
**Date**: 2026-01-15

### Change
Normalize `chat_id` the same way `from` is normalized, and preserve the original LID in a new `chat_lid` field:

```go
func buildFromFields(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) {
    // Save original LID if chat was @lid (for reference/debugging)
    chatJID := evt.Info.Chat
    if chatJID.Server == "lid" {
        payload["chat_lid"] = chatJID.ToNonAD().String()
    }

    // Normalize chat JID (convert LID to phone number if known)
    // This ensures Chatwoot can match contacts by phone number
    normalizedChatJID := NormalizeJIDFromLID(ctx, chatJID, client)
    payload["chat_id"] = normalizedChatJID.ToNonAD().String()

    // ... rest unchanged
}
```

### Webhook Payload (AFTER FIX)
```json
{
  "event": "message",
  "device_id": "5521998762522@s.whatsapp.net",
  "payload": {
    "id": "3EB0BB53CA368F89047FC6",
    "chat_id": "552140402221@s.whatsapp.net",  // Now normalized
    "chat_lid": "174852883411040@lid",          // Original LID preserved
    "from": "552140402221@s.whatsapp.net",
    "from_lid": "174852883411040:28@lid",
    "from_name": "Chatwoot Brasil",
    "body": "ping acme"
  }
}
```

## Impact

### Before Fix
- Duplicate contacts created for LID-based chats
- Contact phone numbers not populated
- Avatar not synced for duplicate contacts
- Conversation history split across multiple contacts

### After Fix
- Existing contacts matched by phone number
- Single contact per phone number
- Avatar and phone number correctly associated
- Unified conversation history

## Testing

1. Send a message from a contact whose JID is in LID format
2. Verify webhook payload contains normalized `chat_id` (ending with `@s.whatsapp.net`)
3. Verify `chat_lid` contains the original LID for reference
4. Verify Chatwoot matches existing contact instead of creating duplicate

## Related Files
- `src/infrastructure/whatsapp/event_message.go` - Fixed file
- `src/infrastructure/whatsapp/jid_utils.go` - `NormalizeJIDFromLID()` function
- Chatwoot: `app/services/whatsapp/incoming_message_whatsapp_web_service.rb` - Consumer of webhook

## Related Issues
- ISSUE-001: Linked device messages not synced (different issue, same LID context)

## Timeline
- **2026-01-15**: Issue identified from Chatwoot duplicate contact investigation
- **2026-01-15**: Root cause analysis completed
- **2026-01-15**: Fix implemented and tested

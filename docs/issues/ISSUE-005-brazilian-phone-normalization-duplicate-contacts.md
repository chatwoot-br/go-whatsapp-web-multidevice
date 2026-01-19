# ISSUE-005: Brazilian Phone Number Normalization Causes Duplicate Contacts

## Status: Open

## Summary
When Chatwoot sends messages to a Brazilian phone number with the 9-digit mobile format (e.g., `5566996679626`), the system resolves it to a different LID than the contact's actual LID. This occurs because WhatsApp normalizes Brazilian numbers to the 8-digit format internally, but the system ignores this normalization and uses the original input number for LID lookup.

## Environment
- **go-whatsapp-web-multidevice**: v8.1.0+6
- **Chatwoot**: chatwoot-br fork
- **Test scenario**: Contact "Lucas Adriano" with phone 556696679626 (LID: 138336618467351@lid)

## Problem Description

### Phone Number Formats
Brazilian mobile numbers transitioned from 8-digit to 9-digit format (adding a leading '9'):
- **Old format (8 digits)**: `5566 96679626` (12 total digits)
- **New format (9 digits)**: `5566 996679626` (13 total digits)

Both formats refer to the same contact, but the system treats them as different contacts.

### Observed Behavior

| Input Phone | WhatsApp Normalized JID | LID Resolved | Result |
|-------------|-------------------------|--------------|--------|
| 556696679626 | 556696679626@s.whatsapp.net | 138336618467351@lid | Correct (incoming messages) |
| 5566996679626 | 556696679626@s.whatsapp.net | 1099729797120@lid | **Wrong LID** (outgoing messages) |

### Log Evidence (2026-01-19 11:20:27)

**1. Chatwoot sends message to 5566996679626:**
```
11:20:27 | 200 | 1.492450744s | POST | /send/message | -
```

**2. System queries WhatsApp usync with the number:**
```xml
<usync context="interactive">
  <list><user><contact>+5566996679626@c.us</contact></user></list>
</usync>
```

**3. WhatsApp returns NORMALIZED JID (12 digits, not 13):**
```xml
<list>
  <user jid="556696679626@s.whatsapp.net">
    <contact type="in">+5566996679626@c.us</contact>
  </user>
</list>
```

**4. BUG: System ignores normalized JID and resolves LID with original number:**
```
Resolved phone 5566996679626@s.whatsapp.net to LID 1099729797120@lid for sending
```

**5. Message sent to wrong LID:**
```xml
<message id="3EB0FF73D78112CE92D058" to="1099729797120@lid" type="text">
```

### Expected Behavior
The system should use WhatsApp's normalized JID (`556696679626@s.whatsapp.net`) for LID lookup, not the original input (`5566996679626@s.whatsapp.net`).

## Root Cause Analysis

### Code Location
**File**: `src/pkg/utils/whatsapp.go`

### The Bug

**Function `IsOnWhatsapp` (line 611):**
```go
func IsOnWhatsapp(client *whatsmeow.Client, jid string) bool {
    // ...
    data, err := client.IsOnWhatsApp(ctx, []string{phone})
    // ...
    for _, v := range data {
        if !v.IsIn {
            return false
        }
    }
    return true  // BUG: Returns bool, discards v.JID (normalized JID)
}
```

The `IsOnWhatsAppResponse` struct from whatsmeow contains:
- `Query string` - The query string used
- `JID JID` - **The canonical (normalized) user ID** ← This is being discarded!
- `IsIn bool` - Whether the phone is registered

**Function `ValidateJidWithLogin` (line 655):**
```go
func ValidateJidWithLogin(client *whatsmeow.Client, jid string) (types.JID, error) {
    MustLogin(client)
    if config.WhatsappAccountValidation && !IsOnWhatsapp(client, jid) {
        return types.JID{}, pkgError.InvalidJID(...)
    }
    return ParseJID(jid)  // BUG: Uses original jid, not normalized
}
```

### Data Flow

```
Chatwoot Request: 5566996679626
         │
         ▼
ValidateJidWithLogin("5566996679626@s.whatsapp.net")
         │
         ├──► IsOnWhatsapp() ──► WhatsApp usync
         │         │                    │
         │         │    Returns: JID=556696679626@s.whatsapp.net (normalized)
         │         │                    │
         │         ◄── returns: true (discards normalized JID) ◄──┘
         │
         ▼
ParseJID("5566996679626@s.whatsapp.net")  ← Uses ORIGINAL, not normalized
         │
         ▼
ResolveJIDForSend(5566996679626@s.whatsapp.net)
         │
         ├──► client.Store.LIDs.GetLIDForPN(5566996679626@s.whatsapp.net)
         │                    │
         │    Returns: 1099729797120@lid  ← WRONG LID!
         │
         ▼
client.SendMessage(to: 1099729797120@lid)  ← Message sent to wrong LID
```

## Impact

### Before Fix
- Outgoing messages to Brazilian numbers with 9-digit format go to wrong LID
- Duplicate contacts created in Chatwoot (one for incoming, one for outgoing)
- Conversation history split across multiple contacts
- Contact matching by phone number fails

### After Fix
- Outgoing messages will use the normalized phone number
- Same LID used for both incoming and outgoing messages
- Single contact per phone number
- Unified conversation history

## Proposed Fix

### Option 1: Modify `ValidateJidWithLogin` to return normalized JID

Create a new function that returns the normalized JID from WhatsApp:

```go
// ValidateAndNormalizeJID validates JID and returns WhatsApp's normalized JID
func ValidateAndNormalizeJID(client *whatsmeow.Client, jid string) (types.JID, error) {
    MustLogin(client)

    if strings.Contains(jid, "@s.whatsapp.net") {
        phone := strings.TrimSuffix(jid, "@s.whatsapp.net")
        if !strings.HasPrefix(phone, "+") {
            phone = "+" + phone
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        data, err := client.IsOnWhatsApp(ctx, []string{phone})
        if err != nil || len(data) == 0 {
            if config.WhatsappAccountValidation {
                return types.JID{}, pkgError.InvalidJID(...)
            }
            return ParseJID(jid)
        }

        for _, v := range data {
            if v.IsIn && !v.JID.IsEmpty() {
                // Return WhatsApp's normalized JID
                return v.JID, nil
            }
        }

        if config.WhatsappAccountValidation {
            return types.JID{}, pkgError.InvalidJID(...)
        }
    }

    return ParseJID(jid)
}
```

### Option 2: Normalize at the entry point

Add Brazilian phone normalization logic before any JID operations.

## Testing

1. Send a message from Chatwoot to a Brazilian number with 9-digit format (e.g., `5566996679626`)
2. Verify the system uses the normalized phone number (`556696679626`) for LID lookup
3. Verify the message is sent to the correct LID (same as incoming messages)
4. Verify no duplicate contacts are created

## Related Files
- `src/pkg/utils/whatsapp.go` - `IsOnWhatsapp()`, `ValidateJidWithLogin()`, `ResolveJIDForSend()`
- `src/usecase/send.go` - Message sending entry point
- `src/usecase/message.go` - Other message operations

## Related Issues
- ISSUE-002: LID chat_id Not Normalized Causes Duplicate Contacts (similar LID normalization issue)
- ISSUE-004: History Sync LID Context Canceled Duplicate Chats (related LID handling)

## Timeline
- **2026-01-19**: Issue identified from log analysis (`gowa-5bb8ddbb57-vtklm.log`)
- **2026-01-19**: Root cause analysis completed

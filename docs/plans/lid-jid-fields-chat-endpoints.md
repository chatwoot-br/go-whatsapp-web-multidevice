# Plan: Add LID/JID Fields to All Chat-Related Endpoints

## Goal
Enable external systems to identify and merge duplicate contacts by exposing both LID (Linked ID) and JID (phone number) in all chat-related API responses.

## Implementation Phases

### Phase 1: Domain Struct Changes

**File: `src/domains/chat/chat.go`**

1. Add `LID` field to `ChatInfo` (line 47-54):
```go
type ChatInfo struct {
    JID                 string `json:"jid"`
    LID                 string `json:"lid,omitempty"`  // NEW
    Name                string `json:"name"`
    ...
}
```

2. Add `ChatLID` and `SenderLID` to `MessageInfo` (line 56-70):
```go
type MessageInfo struct {
    ID         string `json:"id"`
    ChatJID    string `json:"chat_jid"`
    ChatLID    string `json:"chat_lid,omitempty"`    // NEW
    SenderJID  string `json:"sender_jid"`
    SenderLID  string `json:"sender_lid,omitempty"`  // NEW
    ...
}
```

3. Add `ChatLID` to `PinChatResponse` (line 40-45):
```go
type PinChatResponse struct {
    ...
    ChatJID string `json:"chat_jid"`
    ChatLID string `json:"chat_lid,omitempty"`  // NEW
    ...
}
```

4. Add `ChatLID` to `SetDisappearingTimerResponse` (line 84-89):
```go
type SetDisappearingTimerResponse struct {
    ...
    ChatJID      string `json:"chat_jid"`
    ChatLID      string `json:"chat_lid,omitempty"`  // NEW
    ...
}
```

**File: `src/domains/user/account.go`**

5. Add `LID` to `MyListContactsResponseData`:
```go
type MyListContactsResponseData struct {
    JID  types.JID `json:"jid"`
    LID  string    `json:"lid,omitempty"`  // NEW
    Name string    `json:"name"`
}
```

**File: `src/domains/group/group.go`**

6. Add `LID` to `GetGroupRequestParticipantsResponse`:
```go
type GetGroupRequestParticipantsResponse struct {
    JID         string    `json:"jid"`
    LID         string    `json:"lid,omitempty"`  // NEW
    PhoneNumber string    `json:"phone_number"`
    ...
}
```

---

### Phase 2: Usecase Layer - LID Resolution

**File: `src/usecase/chat.go`**

Add import: `"go.mau.fi/whatsmeow/types"`

7. `ListChats` (after line 65) - resolve LID for each chat:
```go
resolver := whatsapp.GetLIDResolver()
if resolver != nil {
    if jid, err := types.ParseJID(chat.JID); err == nil {
        lidJID := resolver.ResolveToLID(ctx, jid)
        if lidJID.Server == "lid" {
            chatInfo.LID = lidJID.String()
        }
    }
}
```

8. `GetChatMessages` (after line 172) - resolve LID for messages:
```go
resolver := whatsapp.GetLIDResolver()
if resolver != nil {
    if chatJID, err := types.ParseJID(message.ChatJID); err == nil {
        lidJID := resolver.ResolveToLID(ctx, chatJID)
        if lidJID.Server == "lid" {
            messageInfo.ChatLID = lidJID.String()
        }
    }
    if senderJID, err := types.ParseJID(message.Sender); err == nil {
        lidJID := resolver.ResolveToLID(ctx, senderJID)
        if lidJID.Server == "lid" {
            messageInfo.SenderLID = lidJID.String()
        }
    }
}
```

9. `GetChatMessages` - also resolve LID for chatInfo (after line 184)

10. `PinChat` (after line 233) - resolve LID:
```go
resolver := whatsapp.GetLIDResolver()
if resolver != nil {
    lidJID := resolver.ResolveToLID(ctx, targetJID)
    if lidJID.Server == "lid" {
        response.ChatLID = lidJID.String()
    }
}
```

11. `SetDisappearingTimer` (after line 278) - same pattern

**File: `src/usecase/user.go`**

12. `MyListContacts` - resolve LID for each contact

**File: `src/usecase/group.go`**

13. `GetGroupRequestParticipants` - resolve LID for each participant

---

### Phase 3: Webhook Event Handlers

**File: `src/infrastructure/whatsapp/event_receipt.go`**

14. Add LID resolution to `createReceiptPayload`:
```go
resolver := GetLIDResolver()
if resolver != nil {
    chatPN, chatLID := resolver.ResolveToPNForWebhook(context.Background(), evt.Chat)
    payload["chat_jid"] = chatPN.String()
    if !chatLID.IsEmpty() {
        payload["chat_lid"] = chatLID.String()
    }
    // Same for sender
}
```

**File: `src/infrastructure/whatsapp/event_group.go`**

15. Add LIDs array for participant JIDs in `createGroupInfoPayload`

**File: `src/infrastructure/whatsapp/event_delete.go`**

16. Add LID resolution to `createDeletePayload`

---

### Phase 4: MCP Tools (Automatic)

MCP tools use the same usecases, so they inherit LID fields automatically. No changes needed.

---

## Files to Modify

| File | Changes |
|------|---------|
| `src/domains/chat/chat.go` | Add LID fields to ChatInfo, MessageInfo, PinChatResponse, SetDisappearingTimerResponse |
| `src/domains/user/account.go` | Add LID field to MyListContactsResponseData |
| `src/domains/group/group.go` | Add LID field to GetGroupRequestParticipantsResponse |
| `src/usecase/chat.go` | Add LIDResolver calls in ListChats, GetChatMessages, PinChat, SetDisappearingTimer |
| `src/usecase/user.go` | Add LIDResolver calls in MyListContacts |
| `src/usecase/group.go` | Add LIDResolver calls in GetGroupRequestParticipants |
| `src/infrastructure/whatsapp/event_receipt.go` | Add LID fields to receipt webhook |
| `src/infrastructure/whatsapp/event_group.go` | Add participant LIDs to group webhook |
| `src/infrastructure/whatsapp/event_delete.go` | Add LID fields to delete webhook |

## Reference Implementation

Follow existing patterns in:
- `src/infrastructure/whatsapp/lid_resolver.go` - LIDResolver API
- `src/infrastructure/whatsapp/event_message.go` - Webhook LID resolution pattern
- `src/usecase/group.go:162-166` - Group participants LID pattern

## Backward Compatibility

- All existing fields remain unchanged
- New LID fields use `omitempty` - omitted when resolution fails
- Webhooks include both formats for backward compatibility

## Testing

After implementation:
1. `go test ./...` - Run all tests
2. Test REST endpoints return LID when available
3. Test webhooks include LID fields
4. Verify MCP tools show LID in responses

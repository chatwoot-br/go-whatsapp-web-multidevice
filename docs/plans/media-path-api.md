# Plan: Add media_path to Chat Messages API

## Problem
The `/chat/{jid}/messages` API returns encrypted WhatsApp CDN URLs (`mmg.whatsapp.net/xxx.enc`) that external consumers like Chatwoot cannot access. The gateway UI works because it downloads on-demand using the active session, but API consumers get 403 errors.

## Solution
Add `media_path` field to track downloaded media and return it in API responses.

---

## Implementation Steps

### 1. Database Migration - Add media_path column
**File**: `src/infrastructure/chatstorage/sqlite_repository.go`

Add Migration 3 to `getMigrations()`:
```sql
ALTER TABLE messages ADD COLUMN media_path TEXT;
```

### 2. Update Message Domain Model
**File**: `src/domains/chatstorage/chatstorage.go`

Add to `Message` struct:
```go
MediaPath string `db:"media_path"` // Local path to downloaded media
```

### 3. Update Repository Queries
**File**: `src/infrastructure/chatstorage/sqlite_repository.go`

- Update `scanMessage()` to include `media_path`
- Update INSERT/UPDATE queries in `StoreMessage()` and `StoreMessagesBatch()`
- Add new method `UpdateMessageMediaPath(messageID, chatJID, mediaPath string) error`

### 4. Update Repository Interface
**File**: `src/domains/chatstorage/interfaces.go`

Add method:
```go
UpdateMessageMediaPath(messageID, chatJID, mediaPath string) error
```

### 5. Update API Response
**File**: `src/domains/chat/chat.go`

Add to `MessageInfo` struct:
```go
MediaPath string `json:"media_path,omitempty"`
```

### 6. Include media_path in GetChatMessages
**File**: `src/usecase/chat.go`

Map `message.MediaPath` to `messageInfo.MediaPath` in the response.

### 7. Add Caching to DownloadMedia
**File**: `src/usecase/message.go`

In `DownloadMedia()`:
1. Check if `message.MediaPath` exists and file is present on disk
2. If cached, return existing path (skip download)
3. After successful download, call `UpdateMessageMediaPath()` to store path

### 8. Track media_path on Auto-Download
**File**: `src/infrastructure/whatsapp/init.go`

In `downloadMedia()` handler (called when `WhatsappAutoDownloadMedia` is enabled):
- After successfully downloading incoming message media
- Store the media_path in the database via chat storage repository

---

## Files to Modify

| File | Change |
|------|--------|
| `src/infrastructure/chatstorage/sqlite_repository.go` | Migration + queries + new method |
| `src/domains/chatstorage/chatstorage.go` | Add MediaPath to Message struct |
| `src/domains/chatstorage/interfaces.go` | Add UpdateMessageMediaPath interface |
| `src/domains/chat/chat.go` | Add MediaPath to MessageInfo |
| `src/usecase/chat.go` | Map MediaPath in GetChatMessages |
| `src/usecase/message.go` | Add caching + DB update in DownloadMedia |
| `src/infrastructure/whatsapp/init.go` | Store media_path on auto-download |

---

## Expected API Response After Implementation

```json
{
  "id": "ABC123",
  "media_type": "image",
  "url": "https://mmg.whatsapp.net/...",
  "media_path": "/statics/media/5521999999999/2025-12-10/image.jpg"
}
```

Chatwoot checks `media_path` first (already implemented on their side). If present, uses it. If not, shows placeholder.

---

## Testing
1. Run `go test ./...`
2. Verify existing messages get `media_path: null` (no regression)
3. Download a message media via `/message/{id}/download`
4. Query `/chat/{jid}/messages` - verify `media_path` is populated
5. Download same message again - verify cache hit (no re-download)
6. Send new message with media (auto-download enabled) - verify `media_path` is set

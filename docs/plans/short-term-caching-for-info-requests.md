# Plan: Short-Term Caching for Info Requests

## Problem
WhatsApp returns 429 rate-limit errors when the same info endpoint is called repeatedly in quick succession. Currently, errors are converted to panics via `PanicIfNeeded`, causing poor UX.

## Solution
Implement short-term in-memory caching for info requests to reduce WhatsApp API calls.

## Info Endpoints to Cache

| Endpoint | Route | Cache Key | TTL |
|----------|-------|-----------|-----|
| User Info | `GET /user/info` | `userinfo:{phone}` | 30s |
| User Avatar | `GET /user/avatar` | `avatar:{phone}:{preview}:{community}` | 60s |
| Business Profile | `GET /user/business-profile` | `bizprofile:{phone}` | 60s |
| Group Info | `GET /group/info` | `groupinfo:{groupJID}` | 30s |
| Group Info from Link | `GET /group/info-from-link` | `grouplink:{linkHash}` | 60s |

## Implementation

### 1. Create Generic Cache Module
**File**: `src/pkg/cache/cache.go`

```go
package cache

type Cache struct {
    data map[string]cacheEntry
    mu   sync.RWMutex
    ttl  time.Duration
}

type cacheEntry struct {
    value     any
    expiresAt time.Time
}

func New(ttl time.Duration) *Cache
func (c *Cache) Get(key string) (any, bool)
func (c *Cache) Set(key string, value any)
func (c *Cache) Delete(key string)
```

### 2. Create Info Cache Service
**File**: `src/infrastructure/whatsapp/info_cache.go`

- Device-scoped caches (each device has its own cache)
- Separate caches per info type with appropriate TTLs
- Thread-safe operations
- **Error caching**: Cache error responses (401, 403, 404) to prevent repeated lookups

### 3. Modify Usecase Functions
Add cache check before WhatsApp API calls:

**Files to modify:**
- `src/usecase/user.go` - `Info()`, `Avatar()`, `BusinessProfile()`
- `src/usecase/group.go` - `GroupInfo()`, `GetGroupInfoFromLink()`

Pattern:
```go
func (service serviceUser) Avatar(ctx context.Context, request domainUser.AvatarRequest) (...) {
    // 1. Check cache (including cached errors)
    if cached, ok := infoCache.GetUserAvatar(phone, isPreview, isCommunity); ok {
        if cached.ErrorMsg != "" {
            return response, errors.New(cached.ErrorMsg)
        }
        return cached, nil
    }

    // 2. Call WhatsApp API
    result, err := client.GetProfilePictureInfo(...)
    if err != nil {
        // 3. Cache the error to prevent repeated lookups
        infoCache.SetUserAvatarError(phone, isPreview, isCommunity, err.Error())
        return response, err
    }

    // 4. Store successful result in cache
    infoCache.SetUserAvatar(phone, isPreview, isCommunity, result)

    return result, nil
}
```

## Files to Create/Modify

| File | Action |
|------|--------|
| `src/pkg/cache/cache.go` | Create - Generic TTL cache |
| `src/infrastructure/whatsapp/info_cache.go` | Create - Info-specific caching with error support |
| `src/usecase/user.go` | Modify - Add cache to Info, Avatar, BusinessProfile |
| `src/usecase/group.go` | Modify - Add cache to GroupInfo, GetGroupInfoFromLink |

## Cache Configuration

```go
const (
    UserInfoTTL        = 30 * time.Second
    UserAvatarTTL      = 60 * time.Second
    BusinessProfileTTL = 60 * time.Second
    GroupInfoTTL       = 30 * time.Second
    GroupLinkInfoTTL   = 60 * time.Second
)
```

## Error Caching

**Problem**: Without error caching, when WhatsApp returns errors like:
- 401 "not-authorized" (hidden profile picture)
- 404 "item-not-found" (no profile picture)
- 403 "forbidden" (not participating in group)

The code returns early without caching, causing repeated API calls for the same failed request.

**Solution**: Cache error responses with the same TTL as successful responses.

```go
// Cache result structs include error field
type UserAvatarResult struct {
    URL      string
    ID       string
    Type     string
    HasURL   bool
    ErrorMsg string // Cached error message
}

type GroupInfoResult struct {
    Data     *types.GroupInfo
    ErrorMsg string // Cached error message
}
```

## Verification

1. **Build**: `go build ./...`
2. **Test**: Call `/user/avatar` for a user with hidden profile - should only hit WhatsApp once
3. **Check logs**: Should see `Cache HIT` and `Cache SET (error)` messages
4. **Verify TTL**: Wait 60+ seconds, call again - should hit WhatsApp API

## Implementation Checklist

- [x] Create generic TTL cache module (`src/pkg/cache/cache.go`)
- [x] Create info cache service (`src/infrastructure/whatsapp/info_cache.go`)
- [x] Add caching to `Info()` in user.go
- [x] Add caching to `Avatar()` in user.go
- [x] Add caching to `BusinessProfile()` in user.go
- [x] Add caching to `GroupInfo()` in group.go
- [x] Add caching to `GetGroupInfoFromLink()` in group.go
- [x] Add error caching for Avatar (401, 404 errors)
- [x] Add error caching for GroupInfo (403 errors)
- [x] Build and verify no compilation errors
- [x] Run tests
- [ ] Manual testing with debug logs

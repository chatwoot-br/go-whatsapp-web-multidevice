# Plan: Short-Term Caching for Info Requests

## Problem
WhatsApp returns 429 rate-limit errors when the same info endpoint is called repeatedly in quick succession. Currently, errors are converted to panics via `PanicIfNeeded`, causing poor UX.

## Solution
Implement short-term in-memory caching for info requests to reduce WhatsApp API calls.

## Info Endpoints to Cache

| Endpoint | Route | Cache Key | TTL |
|----------|-------|-----------|-----|
| User Info | `GET /user/info` | `userinfo:{phone}` | 30s |
| User Avatar | `GET /user/avatar` | `avatar:{phone}:{preview}` | 60s |
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

### 3. Modify Usecase Functions
Add cache check before WhatsApp API calls:

**Files to modify:**
- `src/usecase/user.go` - `Info()`, `Avatar()`, `BusinessProfile()`
- `src/usecase/group.go` - `GroupInfo()`, `GetGroupInfoFromLink()`

Pattern:
```go
func (service serviceUser) Info(ctx context.Context, request domainUser.InfoRequest) (...) {
    // 1. Check cache
    if cached, ok := infocache.GetUserInfo(deviceID, request.Phone); ok {
        return cached, nil
    }

    // 2. Call WhatsApp API
    result, err := client.GetUserInfo(...)
    if err != nil {
        return response, err
    }

    // 3. Store in cache
    infocache.SetUserInfo(deviceID, request.Phone, result)

    return result, nil
}
```

## Files to Create/Modify

| File | Action |
|------|--------|
| `src/pkg/cache/cache.go` | Create - Generic TTL cache |
| `src/infrastructure/whatsapp/info_cache.go` | Create - Info-specific caching |
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

## Verification

1. **Build**: `go build ./...`
2. **Test**: Call `/group/info` rapidly 5+ times - should only hit WhatsApp once
3. **Check logs**: Should see cache hits after first request
4. **Verify TTL**: Wait 30+ seconds, call again - should hit WhatsApp API

## Implementation Checklist

- [x] Create generic TTL cache module (`src/pkg/cache/cache.go`)
- [x] Create info cache service (`src/infrastructure/whatsapp/info_cache.go`)
- [x] Add caching to `Info()` in user.go
- [x] Add caching to `Avatar()` in user.go
- [x] Add caching to `BusinessProfile()` in user.go
- [x] Add caching to `GroupInfo()` in group.go
- [x] Add caching to `GetGroupInfoFromLink()` in group.go
- [x] Build and verify no compilation errors
- [x] Run tests
- [ ] Manual testing with debug logs

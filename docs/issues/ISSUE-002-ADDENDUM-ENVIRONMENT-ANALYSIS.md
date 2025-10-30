# ISSUE-002 Addendum: Why This Bug Only Manifested in Production

**Date**: 2025-10-30
**Related Issue**: [ISSUE-002-MEDIA-FILENAME-MIME-POLLUTION.md](./ISSUE-002-MEDIA-FILENAME-MIME-POLLUTION.md)

## Critical Finding: Alpine Linux MIME Database Absence

### Root Cause of Environment-Specific Behavior

The bug **only manifested in production** (not in dev) because of a critical difference in the runtime environment:

#### Production Environment (Alpine Linux)
**Dockerfile**: `docker/golang.Dockerfile:17`
```dockerfile
FROM alpine:3.20
RUN apk add --no-cache ffmpeg supervisor curl python3 py3-pip net-tools
# ❌ MISSING: No MIME types database installed!
```

**Issue**: Alpine Linux **does not include `/etc/mime.types`** by default, and the production Dockerfile doesn't install the `mailcap` package that provides it.

#### Development Environment (macOS/Debian)
**Platform**: macOS (darwin/arm64) or Debian-based devcontainer
```bash
✓ MIME database found: /etc/apache2/mime.types (macOS)
✓ MIME database found: /etc/mime.types (Debian)
```

### Code Flow Analysis

#### OLD CODE (Before Fix)

```go
func determineMediaExtension(originalFilename, mimeType string) string {
    // ... earlier checks ...

    // Step 1: Try mime.ExtensionsByType
    if ext, err := mime.ExtensionsByType(mimeType); err == nil && len(ext) > 0 {
        return ext[0]  // ← SUCCESS on macOS/Debian, FAILS on Alpine
    }

    // Step 2: Fallback (only reached on Alpine)
    if parts := strings.Split(mimeType, "/"); len(parts) > 1 {
        return "." + parts[len(parts)-1]  // ← BUG: Includes "; codecs=opus"
    }

    return ""
}
```

**Production (Alpine - NO MIME database):**
```
Input: "audio/ogg; codecs=opus"
↓
mime.ExtensionsByType("audio/ogg; codecs=opus")
  → Result: [] (empty - no database!)
  → Falls through ❌
↓
strings.Split("audio/ogg; codecs=opus", "/")
  → ["audio", "ogg; codecs=opus"]
  → Returns: ".ogg; codecs=opus" ❌ BUG!
↓
File saved as: "1761817628-uuid.ogg; codecs=opus"
```

**Development (macOS/Debian - HAS MIME database):**
```
Input: "audio/ogg; codecs=opus"
↓
mime.ExtensionsByType("audio/ogg; codecs=opus")
  → Result: [.oga .ogg .opus .spx] ✓
  → Returns: ".oga" immediately ✓
↓
File saved as: "1761817628-uuid.oga"
```

### Why mime.ExtensionsByType() Behaves Differently

Go's `mime` package behavior:

1. **With MIME database** (macOS/Debian):
   - Reads from `/etc/mime.types` or `/etc/apache2/mime.types`
   - Handles parameterized MIME types gracefully (strips parameters internally on some platforms)
   - Returns correct extensions: `[.oga .ogg .opus .spx]`

2. **Without MIME database** (Alpine):
   - No `/etc/mime.types` file exists
   - Uses only built-in hardcoded types (very limited)
   - `audio/ogg` might not be in the limited built-in list
   - Returns empty: `[]`

### Test Proof

**macOS (Development):**
```bash
$ go run test_mime_platform.go
Platform MIME Database Test
============================
OS: darwin
Arch: arm64
Go Version: go1.24.6

MIME: "audio/ogg; codecs=opus"
  Extensions: [.oga .ogg .opus .spx]  ← Works!
  Error: <nil>
  Empty? false

Checking for MIME database files:
  ✓ Found: /etc/apache2/mime.types
```

**Alpine (Production - simulated):**
```bash
$ docker run --rm alpine:3.20 sh -c "ls -la /etc/mime.types"
ls: /etc/mime.types: No such file or directory  ← Missing!

$ docker run --rm alpine:3.20 sh -c "apk info | grep mime"
# (no mime-related packages installed)
```

## NEW CODE (After Fix) - Works on Both!

```go
func determineMediaExtension(originalFilename, mimeType string) string {
    // ... earlier checks ...

    // ✅ NEW: Strip MIME parameters FIRST
    baseType := strings.Split(mimeType, ";")[0]
    baseType = strings.TrimSpace(baseType)

    // Step 1: Try mime.ExtensionsByType with clean type
    if ext, err := mime.ExtensionsByType(baseType); err == nil && len(ext) > 0 {
        return ext[0]  // ← SUCCESS on macOS/Debian
    }

    // Step 2: Fallback (Alpine without database)
    if parts := strings.Split(baseType, "/"); len(parts) > 1 {
        return "." + parts[len(parts)-1]  // ← Now returns clean ".ogg" ✓
    }

    return ""
}
```

**Production (Alpine - NO MIME database):**
```
Input: "audio/ogg; codecs=opus"
↓
Strip parameters: "audio/ogg" ✓
↓
mime.ExtensionsByType("audio/ogg")
  → Result: [] (still empty - no database)
  → Falls through, but with CLEAN type ✓
↓
strings.Split("audio/ogg", "/")
  → ["audio", "ogg"]
  → Returns: ".ogg" ✓ FIXED!
↓
File saved as: "1761817628-uuid.ogg" ✓
```

**Development (macOS/Debian - HAS MIME database):**
```
Input: "audio/ogg; codecs=opus"
↓
Strip parameters: "audio/ogg" ✓
↓
mime.ExtensionsByType("audio/ogg")
  → Result: [.oga .ogg .opus .spx] ✓
  → Returns: ".oga" immediately ✓
↓
File saved as: "1761817628-uuid.oga" ✓
```

## Why You Couldn't Reproduce in Dev

### Scenario 1: Testing BEFORE the fix
**Dev (macOS):**
- `mime.ExtensionsByType("audio/ogg; codecs=opus")` → Returns `[.oga ...]`
- Never falls through to the buggy split
- File saved as: `.oga` ✓
- **No bug visible**

**Prod (Alpine):**
- `mime.ExtensionsByType("audio/ogg; codecs=opus")` → Returns `[]`
- Falls through to buggy split
- File saved as: `.ogg; codecs=opus` ❌
- **Bug manifests**

### Scenario 2: Testing AFTER the fix
**Both Environments:**
- Parameters stripped before any processing
- Even if MIME database missing, clean type is used
- Both environments work correctly ✓

## Recommendations

### Immediate: Fix Already Applied ✓
The parameter stripping fix ensures consistent behavior across all environments.

### ✅ IMPLEMENTED: Install MIME Database in Production

To make Alpine behavior match development more closely (and potentially get better extension choices):

**Updated `docker/golang.Dockerfile:18-20`:**
```dockerfile
FROM alpine:3.20
# Install runtime dependencies including mailcap for MIME types database (/etc/mime.types)
# mailcap provides proper MIME type -> file extension mapping for media files
RUN apk add --no-cache ffmpeg supervisor curl python3 py3-pip net-tools mailcap
```

**Benefits:**
- ✅ More accurate extension determination
- ✅ Consistent behavior with dev environment
- ✅ Better handling of edge cases
- ✅ Production will now use proper MIME database like dev

**Trade-offs:**
- Minimal increase in image size (~50KB)
- Not strictly necessary with the parameter stripping fix, but provides defense in depth

### Testing Recommendation

To test environment-specific bugs in the future:

1. **Test in Alpine container:**
   ```bash
   docker run --rm -v $(pwd):/app -w /app golang:1.24-alpine3.20 sh -c "
     cd src && go test ./pkg/utils -run TestDetermineMediaExtension
   "
   ```

2. **Test without MIME database:**
   ```bash
   # Remove MIME database temporarily
   sudo mv /etc/mime.types /etc/mime.types.bak
   go test ./pkg/utils
   sudo mv /etc/mime.types.bak /etc/mime.types
   ```

3. **Add CI test for Alpine:**
   ```yaml
   # .github/workflows/test.yml
   test-alpine:
     runs-on: ubuntu-latest
     container: golang:1.24-alpine3.20
     steps:
       - uses: actions/checkout@v3
       - run: cd src && go test ./...
   ```

## Evidence from Production

**Actual file from production `/app/instances/3001/statics/media/`:**
```
-rw------- 1 root root  10892 Oct 30 09:47 1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg; codecs=opus
                                                                                            ^^^^^^^^^^^^^^^^^
                                                                                            BUG: Semicolon in filename!
```

**Why this proves Alpine lacks MIME database:**
- If MIME database existed, `mime.ExtensionsByType()` would have succeeded
- File would have been saved as `.oga` or `.ogg` (no semicolon)
- The presence of `; codecs=opus` proves it fell through to the string split fallback

## Lessons Learned

### 1. Environmental Parity is Critical
- Dev and prod environments should be as similar as possible
- Test in production-like containers (Alpine) not just local dev (macOS/Debian)

### 2. Don't Trust Platform-Specific Behavior
- Go's `mime` package behavior varies by platform
- Always have robust fallbacks that work without platform dependencies

### 3. Minimal Docker Images Have Trade-offs
- Alpine is great for small images
- But missing standard files (like MIME database) can cause subtle bugs
- Document what's missing and why

### 4. Test Edge Cases
- Test with parameterized MIME types
- Test without system dependencies
- Test on target production platform

## Related Files

- **Production Dockerfile**: `docker/golang.Dockerfile` (Alpine 3.20, no MIME database)
- **Dev Dockerfile**: `.devcontainer/Dockerfile` (Debian bookworm, has MIME database)
- **Bug Location**: `src/pkg/utils/whatsapp.go:69-95`
- **Fix Applied**: Strips MIME parameters before all processing

## Testing Checklist

When deploying the fix:

- [x] Test on macOS (dev environment)
- [ ] Test in Alpine Docker container (production simulation)
- [ ] Verify no semicolons in new media filenames
- [ ] Run migration script on existing files
- [ ] Monitor production after deployment

---

**Document Created**: 2025-10-30
**Status**: CRITICAL FINDING
**Impact**: Explains why bug was environment-specific
**Action Required**: Already fixed, but consider adding MIME database to Alpine image

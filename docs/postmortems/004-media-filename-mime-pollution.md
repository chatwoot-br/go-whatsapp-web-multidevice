# Postmortem: Media Files Inaccessible Due to MIME Type Parameters in Filenames

**Date**: 2025-10-30
**Severity**: High
**Status**: Resolved
**Version Affected**: All versions with original `determineMediaExtension()` implementation
**Version Fixed**: 2025-10-30 (same day)

## Incident Summary

Media files downloaded from WhatsApp were saved with MIME type parameters (e.g., `; codecs=opus`) embedded in filenames, causing 404 errors when downstream systems attempted to access them. Audio files were saved as `file.ogg; codecs=opus` instead of `file.ogg`, making them inaccessible via HTTP requests. The webhook payload contained clean filenames, creating a mismatch between the actual filename on disk and the URL sent to consumers.

This issue only manifested in production (Alpine Linux) and not in development environments (macOS/Debian) due to differences in MIME database availability.

## Impact

**Severity: HIGH**

- **Media Accessibility**: Audio files completely inaccessible via webhook URLs (404 errors)
- **User Experience**: Media failed to load in downstream systems (Chatwoot, integrations)
- **Data Integrity**: Files existed on disk but were unreachable via API
- **Webhook Reliability**: Webhook payloads contained incorrect media paths

### Affected Operations

1. Working: Media download from WhatsApp and local storage
2. Failing: HTTP access to stored media files (404 errors)
3. Affected: All downstream systems consuming webhook media URLs
4. Affected Media Types: Primarily audio files with `audio/ogg; codecs=opus`, potentially any media with MIME type parameters

### Timeline

- **2025-10-29**: Audio messages received and saved with malformed filenames
- **2025-10-30 09:47:08**: Chatwoot reported 404 errors downloading media
- **2025-10-30**: Root cause identified - MIME parameters in filenames
- **2025-10-30**: Fix implemented with parameter stripping
- **2025-10-30**: 39 unit tests added and passing
- **2025-10-30**: Alpine MIME database fix added for defense in depth

## Root Cause

### Technical Details

**Error from Downstream System (Chatwoot)**:
```
ERROR: Error downloading WhatsApp Web media: 404 Not Found
Failed media URL: http://...statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg
Attachment payload: {"mime_type" => "audio/ogg; codecs=opus", ...}
```

**Actual Files on Disk**:
```bash
-rw------- 1 root root  77983 Oct 29 18:21 1761762116-uuid.ogg; codecs=opus
-rw------- 1 root root  15893 Oct 29 18:22 1761762141-uuid.ogg; codecs=opus
-rw------- 1 root root  10892 Oct 30 09:47 1761817628-uuid.ogg; codecs=opus
```

**File Mismatch**:
- Requested: `.../1761817628-uuid.ogg`
- Actual: `1761817628-uuid.ogg; codecs=opus`
- Result: **404 Not Found**

### Root Cause Analysis

**Location**: `src/pkg/utils/whatsapp.go:69-89`

**Vulnerable Code**:
```go
func determineMediaExtension(originalFilename, mimeType string) string {
    if originalFilename != "" {
        if ext := filepath.Ext(originalFilename); ext != "" {
            return ext
        }
    }

    if ext, ok := resolveKnownDocumentExtension(mimeType); ok {
        return ext
    }

    if ext, err := mime.ExtensionsByType(mimeType); err == nil && len(ext) > 0 {
        return ext[0]
    }

    // BUG: This doesn't strip MIME type parameters!
    // When mimeType = "audio/ogg; codecs=opus"
    // parts = ["audio", "ogg; codecs=opus"]
    // Returns: ".ogg; codecs=opus" ❌
    if parts := strings.Split(mimeType, "/"); len(parts) > 1 {
        return "." + parts[len(parts)-1]
    }

    return ""
}
```

**The Bug**:
1. WhatsApp sends audio with MIME type `audio/ogg; codecs=opus` (includes codec parameter)
2. Function naively splits by `/` without stripping parameters
3. Returns `.ogg; codecs=opus` as the file extension
4. File saved as: `1761817628-uuid.ogg; codecs=opus`
5. Webhook sends clean filename: `1761817628-uuid.ogg`
6. HTTP requests for clean filename get 404

### Environment-Specific Behavior

**Why This Only Happened in Production:**

**Production (Alpine Linux)**:
- Alpine doesn't include `/etc/mime.types` by default
- `mime.ExtensionsByType("audio/ogg; codecs=opus")` returns empty `[]`
- Falls through to buggy string split
- Result: `.ogg; codecs=opus` (BROKEN)

**Development (macOS/Debian)**:
- Has MIME database at `/etc/mime.types` or `/etc/apache2/mime.types`
- `mime.ExtensionsByType("audio/ogg; codecs=opus")` returns `[.oga .ogg .opus .spx]`
- Never reaches the buggy fallback
- Result: `.oga` (WORKS)

This explains why the bug wasn't caught during development testing.

## Resolution

### Fix Applied: Strip MIME Parameters

**Date**: 2025-10-30 (same day as discovery)

**Modified File**: `src/pkg/utils/whatsapp.go:69-89`

**Before (Broken)**:
```go
func determineMediaExtension(originalFilename, mimeType string) string {
    // ... earlier checks ...

    if ext, err := mime.ExtensionsByType(mimeType); err == nil && len(ext) > 0 {
        return ext[0]  // Fails on Alpine
    }

    // Fallback without parameter stripping
    if parts := strings.Split(mimeType, "/"); len(parts) > 1 {
        return "." + parts[len(parts)-1]  // Returns ".ogg; codecs=opus"
    }

    return ""
}
```

**After (Fixed)**:
```go
func determineMediaExtension(originalFilename, mimeType string) string {
    // ... earlier checks ...

    // Strip MIME type parameters BEFORE processing
    // "audio/ogg; codecs=opus" → "audio/ogg"
    baseType := strings.Split(mimeType, ";")[0]
    baseType = strings.TrimSpace(baseType)

    if ext, err := mime.ExtensionsByType(baseType); err == nil && len(ext) > 0 {
        return ext[0]
    }

    // Now split the cleaned MIME type
    if parts := strings.Split(baseType, "/"); len(parts) > 1 {
        return "." + parts[len(parts)-1]  // Returns ".ogg" ✓
    }

    return ""
}
```

### Additional Fix: Alpine MIME Database

**File**: `docker/golang.Dockerfile:18-20`

Added `mailcap` package to provide MIME type database in Alpine:

```dockerfile
FROM alpine:3.20
# Install runtime dependencies including mailcap for MIME types database
RUN apk add --no-cache ffmpeg supervisor curl python3 py3-pip net-tools mailcap
```

**Benefits**:
- Makes Alpine behavior consistent with development
- Provides proper MIME database (~50KB)
- Defense in depth - better extension choices

### Testing

**Unit Tests Added**: 39 test cases covering:
- Audio with codec parameters
- Video with multiple parameters
- Images with charset
- Simple MIME types
- Edge cases with unusual parameters
- Verification no semicolons in results

**All Tests Passing**: ✓

## Prevention

### Steps Taken to Prevent Recurrence

1. **RFC Compliance**:
   - Implemented proper MIME type parsing per RFC 2045
   - Strip parameters before processing base type
   - Handle both primary type and parameters correctly

2. **Environment Parity**:
   - Added MIME database to Alpine production image
   - Makes production behavior consistent with development
   - Reduces environment-specific bugs

3. **Comprehensive Testing**:
   - 39 unit tests for MIME type handling
   - Test with parameterized MIME types
   - Test on target production platform (Alpine)

4. **Migration Script** (pending):
   - Script to rename existing malformed files
   - Clean up production media directories
   - Document migration process

### Testing Strategy for Environment-Specific Bugs

**For Future Development**:

1. Test in Alpine container:
   ```bash
   docker run --rm -v $(pwd):/app -w /app golang:1.24-alpine3.20 \
     sh -c "cd src && go test ./pkg/utils"
   ```

2. Add CI tests for Alpine:
   ```yaml
   test-alpine:
     runs-on: ubuntu-latest
     container: golang:1.24-alpine3.20
     steps:
       - uses: actions/checkout@v3
       - run: cd src && go test ./...
   ```

3. Document platform differences in architecture guide

## Lessons Learned

### What Went Well

1. **Fast Diagnosis**: Log analysis quickly identified the file mismatch
2. **Root Cause Clear**: Stack trace of file operations revealed the bug
3. **Comprehensive Fix**: Both symptom (parameter stripping) and environment (MIME database) fixed
4. **Same-Day Resolution**: Issue identified and fixed within hours
5. **Thorough Testing**: 39 unit tests ensure fix works correctly

### What Could Be Improved

1. **Environmental Parity**: Should have tested in Alpine during development
2. **MIME Handling**: Should have followed RFC standards from the start
3. **Production Testing**: Should have caught malformed filenames earlier
4. **Migration Planning**: Should have file cleanup script ready with fix

### Action Items

- [x] Fix MIME parameter stripping
- [x] Add MIME database to Alpine image
- [x] Add comprehensive unit tests
- [x] Run migration script on production media directories
- [x] Verify all malformed files cleaned up
- [x] Monitor for any remaining 404 errors
- [ ] Document MIME type handling in architecture guide
- [ ] Add CI tests for Alpine environment

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - Project architecture and development guide
- [Webhook Documentation](../webhook-payload.md) - Media message payloads
- [Troubleshooting](../reference/troubleshooting.md) - Common issues

## External References

- **RFC 2045**: MIME Part One - Format of Internet Message Bodies
  - Section 5.1: Syntax of the Content-Type Header Field
  - https://www.rfc-editor.org/rfc/rfc2045#section-5.1
- **Go mime package**: https://pkg.go.dev/mime
- **Alpine mailcap package**: https://pkgs.alpinelinux.org/package/edge/main/x86_64/mailcap
- **Original Issue**: `docs/issues/ISSUE-002-MEDIA-FILENAME-MIME-POLLUTION.md` (archived)
- **Environment Analysis**: `docs/issues/ISSUE-002-ADDENDUM-ENVIRONMENT-ANALYSIS.md` (archived)

## Migration Guide

### For Operators

**Affected Files**:
```bash
# Check for malformed filenames
find /app/instances -path "*/statics/media/*" -name "*;*"
```

**Migration Script** (`scripts/fix-media-filenames.sh`):
```bash
#!/bin/bash
# Fix malformed media filenames by removing MIME type parameters

INSTANCES_DIR="${INSTANCES_DIR:-/app/instances}"

echo "Scanning for malformed media files..."

find "$INSTANCES_DIR" -type f -path "*/statics/media/*" -name "*;*" | while read -r file; do
    dir=$(dirname "$file")
    filename=$(basename "$file")
    clean_filename=$(echo "$filename" | cut -d';' -f1)

    echo "Renaming: $filename → $clean_filename"
    mv "$file" "$dir/$clean_filename"
done

echo "Migration complete!"
```

**Deployment Steps**:
1. Backup media directories
2. Deploy updated code
3. Run migration script
4. Verify no semicolons in filenames
5. Test media access from Chatwoot
6. Monitor for 24 hours

---

**Postmortem Author**: Development Team
**Last Updated**: 2025-12-05
**Resolution Time**: Same day (< 8 hours)
**Status**: Resolved and deployed (v7.8.0+)

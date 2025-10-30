# Critical Issue: Media Files Inaccessible Due to MIME Type Parameters in Filenames

**Issue Type**: Critical Bug
**Status**: 🔴 OPEN
**Severity**: High (causes 404 errors and breaks media access in downstream systems)
**Date Reported**: 2025-10-30
**Date Resolved**: -
**Affected Versions**: All versions with current `determineMediaExtension()` implementation
**Fixed Version**: -

## Summary

Media files downloaded from WhatsApp are being saved with MIME type parameters (e.g., `; codecs=opus`) embedded in the filename, causing 404 errors when downstream systems like Chatwoot attempt to download them. The webhook payload contains the filename without these parameters, creating a mismatch between the actual filename on disk and the URL sent to consumers.

## Impact

**Severity: HIGH**

- 🔴 **Media Accessibility**: Audio files completely inaccessible via webhook URLs
- 🔴 **User Experience**: Media fails to load in downstream systems (Chatwoot, integrations)
- 🟡 **Data Integrity**: Files exist on disk but are unreachable via API
- 🟡 **Webhook Reliability**: Webhook payloads contain incorrect media paths

### Affected Operations

1. ✅ **Working**: Media download from WhatsApp and local storage
2. ❌ **Failing**: HTTP access to stored media files (404 errors)
3. ❌ **Affected**: All downstream systems consuming webhook media URLs
4. ❌ **Affected Media Types**: Audio files with `audio/ogg; codecs=opus` MIME type (potentially others)

## Root Cause Analysis

### Technical Details

**Error Message** (from Chatwoot logs):
```
E, [2025-10-30T09:47:08.903677 #1] ERROR -- : [ActiveJob] [Webhooks::WhatsappEventsJob] [a02a9c7b-d5cf-46fc-8939-ab0a5fb2e43b] Error downloading WhatsApp Web media: 404 Not Found
E, [2025-10-30T09:47:08.903773 #1] ERROR -- : [ActiveJob] [Webhooks::WhatsappEventsJob] [a02a9c7b-d5cf-46fc-8939-ab0a5fb2e43b] WhatsApp Web: Failed media URL: http://gowa.chatwoot-atendimento.svc.cluster.local:3001/552140402221/statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg
E, [2025-10-30T09:47:08.903819 #1] ERROR -- : [ActiveJob] [Webhooks::WhatsappEventsJob] [a02a9c7b-d5cf-46fc-8939-ab0a5fb2e43b] WhatsApp Web: Attachment payload: {"id" => "statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg", "mime_type" => "audio/ogg; codecs=opus", "caption" => ""}
```

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

**Root Cause**:
1. WhatsApp sends audio messages with MIME type `audio/ogg; codecs=opus` (includes codec parameter)
2. The `determineMediaExtension()` function naively splits the MIME type by `/` without stripping parameters
3. The function returns `.ogg; codecs=opus` as the file extension
4. `ExtractMedia()` saves the file with this malformed extension: `1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg; codecs=opus`
5. Webhook payload construction logic strips the parameter, sending `1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg`
6. When consumers request the file via HTTP, they get a 404 because the actual filename has the parameter suffix

### File Flow Analysis

**Step 1: WhatsApp Message Received**
- MIME type: `audio/ogg; codecs=opus`
- Message downloaded via `whatsmeow` library

**Step 2: Extension Determination** (`src/pkg/utils/whatsapp.go:579`)
```go
extension := determineMediaExtension(originalFilename, extractedMedia.MimeType)
// extension = ".ogg; codecs=opus" ❌
```

**Step 3: File Saved** (`src/pkg/utils/whatsapp.go:581`)
```go
extractedMedia.MediaPath = fmt.Sprintf("%s/%d-%s%s",
    storageLocation,           // "statics/media"
    time.Now().Unix(),         // 1761817628
    uuid.NewString(),          // "85830071-9253-4294-8a64-bcaff59119ae"
    extension)                 // ".ogg; codecs=opus"
// Result: "statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg; codecs=opus"
```

**Step 4: Webhook Payload Created** (`src/infrastructure/whatsapp/event_message.go:138`)
```go
body["audio"] = path
// path.MediaPath = "statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg; codecs=opus"
// But somewhere this gets stripped to: "statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg"
```

**Step 5: HTTP Request (404 Error)**
- Requested: `http://.../statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg`
- Actual file: `1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg; codecs=opus`
- Result: **404 Not Found**

## Evidence

### Observed Behavior

**Actual filenames on disk** (`/app/instances/3001/statics/media/`):
```bash
-rw------- 1 root root  77983 Oct 29 18:21 1761762116-15edbef0-e044-48f6-b37d-a80b1593d2b2.ogg; codecs=opus
-rw------- 1 root root  15893 Oct 29 18:22 1761762141-a852c704-95a6-4a35-b90b-8d5644544735.ogg; codecs=opus
-rw------- 1 root root  10892 Oct 30 09:47 1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg; codecs=opus
```

**Note the pattern**: All audio files have `; codecs=opus` **literally in the filename** instead of just the `.ogg` extension.

**Webhook payload sent to Chatwoot**:
```json
{
  "audio": {
    "media_path": "statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg",
    "mime_type": "audio/ogg; codecs=opus",
    "caption": ""
  }
}
```

**URL constructed by Chatwoot**:
```
http://gowa.chatwoot-atendimento.svc.cluster.local:3001/552140402221/statics/media/1761817628-85830071-9253-4294-8a64-bcaff59119ae.ogg
```

**Mismatch**:
- Expected: `...ae.ogg`
- Actual: `...ae.ogg; codecs=opus` ❌

### Impact on Other Media Types

**Currently affected**: Audio files with codec parameters
**Potentially affected**: Any media with MIME type parameters:
- `video/mp4; codecs="avc1.42E01E, mp4a.40.2"`
- `image/webp; charset=binary`
- `application/pdf; charset=UTF-8`

The bug affects **any MIME type with parameters** that falls through to the final fallback in `determineMediaExtension()`.

## Proposed Solutions

### Solution 1: Strip MIME Type Parameters (IMMEDIATE - P0) ✅ RECOMMENDED

**Priority**: P0 (Must fix immediately)
**Timeline**: Same day
**Risk**: Low
**Backward Compatibility**: Breaking (requires file renaming or migration)

**Implementation** (`src/pkg/utils/whatsapp.go:69-89`):

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

	// Strip MIME type parameters before processing
	// Example: "audio/ogg; codecs=opus" → "audio/ogg"
	baseType := strings.Split(mimeType, ";")[0]
	baseType = strings.TrimSpace(baseType)

	if ext, err := mime.ExtensionsByType(baseType); err == nil && len(ext) > 0 {
		return ext[0]
	}

	// Now split the cleaned MIME type
	if parts := strings.Split(baseType, "/"); len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}

	return ""
}
```

**Benefits**:
- ✅ Fixes root cause
- ✅ Prevents future occurrences
- ✅ Simple, surgical fix
- ✅ No performance impact
- ✅ Follows MIME type standards (RFC 2045)

**Considerations**:
- ⚠️ Existing malformed files remain inaccessible
- ⚠️ Requires migration strategy for existing files

### Solution 2: File Cleanup/Migration Script (IMMEDIATE - P0)

**Priority**: P0 (Must run after Solution 1)
**Timeline**: Same day
**Risk**: Low

Create a migration script to rename existing files with malformed extensions.

**Implementation** (`scripts/fix-media-filenames.sh`):

```bash
#!/bin/bash
# Fix malformed media filenames by removing MIME type parameters

INSTANCES_DIR="${INSTANCES_DIR:-/app/instances}"

echo "Scanning for malformed media files..."

find "$INSTANCES_DIR" -type f -path "*/statics/media/*" -name "*;*" | while read -r file; do
    # Extract directory and filename
    dir=$(dirname "$file")
    filename=$(basename "$file")

    # Remove everything from semicolon onward
    clean_filename=$(echo "$filename" | cut -d';' -f1)

    echo "Renaming: $filename → $clean_filename"

    # Rename file
    mv "$file" "$dir/$clean_filename"
done

echo "Migration complete!"
```

**Usage**:
```bash
# Dry run
bash scripts/fix-media-filenames.sh --dry-run

# Execute
bash scripts/fix-media-filenames.sh

# For specific instance
INSTANCES_DIR=/app/instances/3001 bash scripts/fix-media-filenames.sh
```

### Solution 3: Backward-Compatible Media Access (SHORT-TERM - P1)

**Priority**: P1 (Should implement)
**Timeline**: 1-2 days
**Risk**: Medium

Add a fallback mechanism to try both clean and malformed filenames when serving static files.

**Implementation**: Custom Fiber middleware to handle 404s

```go
// In src/cmd/rest.go
app.Use(config.AppBasePath+"/statics/media", func(c *fiber.Ctx) error {
    path := c.Path()

    // Try normal file first
    if err := c.Next(); err == nil {
        return nil
    }

    // If 404, try appending common MIME parameters
    commonSuffixes := []string{
        "; codecs=opus",
        "; codecs=\"avc1.42E01E, mp4a.40.2\"",
    }

    for _, suffix := range commonSuffixes {
        altPath := filepath.Join(".", "statics", "media", filepath.Base(path)+suffix)
        if _, err := os.Stat(altPath); err == nil {
            return c.SendFile(altPath)
        }
    }

    return fiber.ErrNotFound
})
```

**Benefits**:
- ✅ Maintains access to existing malformed files
- ✅ Provides graceful degradation
- ✅ Buys time for full migration

**Drawbacks**:
- ⚠️ Performance overhead (multiple file checks)
- ⚠️ Technical debt (workaround for underlying issue)

## Recommended Fix Priority

### Phase 1: IMMEDIATE (Today) ✅ CRITICAL

1. **Solution 1**: Fix `determineMediaExtension()` function ✅ COMPLETED
   - ✅ Strip MIME type parameters before extension extraction
   - ✅ Add unit tests (39 test cases, all passing)
   - Ready to deploy to all instances

2. **Solution 1b**: Add MIME database to Alpine ✅ COMPLETED
   - ✅ Added `mailcap` package to `docker/golang.Dockerfile`
   - ✅ Provides `/etc/mime.types` (73KB) with proper MIME mappings
   - ✅ Makes production behavior consistent with development
   - See: [ISSUE-002-ADDENDUM-ENVIRONMENT-ANALYSIS.md](./ISSUE-002-ADDENDUM-ENVIRONMENT-ANALYSIS.md)

3. **Solution 2**: Run migration script (PENDING)
   - Rename existing malformed files
   - Verify all instances are migrated
   - Document renamed files

### Phase 2: SHORT-TERM (This Week)

3. **Monitoring**: Add metrics for media access
   - Track 404 errors on media endpoints
   - Alert on abnormal patterns

4. **Validation**: Add integration tests
   - Test various MIME types with parameters
   - Verify clean filenames
   - Verify HTTP accessibility

### Phase 3: OPTIONAL

5. **Solution 3**: Add fallback mechanism (if needed)
   - Only if migration is incomplete
   - Remove after full migration

## Testing Strategy

### Unit Tests

**File**: `src/pkg/utils/whatsapp_test.go`

```go
func TestDetermineMediaExtension_StripsMimeParameters(t *testing.T) {
	tests := []struct {
		name         string
		originalFile string
		mimeType     string
		expected     string
	}{
		{
			name:         "Audio with codec parameter",
			originalFile: "",
			mimeType:     "audio/ogg; codecs=opus",
			expected:     ".ogg",
		},
		{
			name:         "Video with multiple parameters",
			originalFile: "",
			mimeType:     "video/mp4; codecs=\"avc1.42E01E, mp4a.40.2\"",
			expected:     ".mp4",
		},
		{
			name:         "Image with charset",
			originalFile: "",
			mimeType:     "image/webp; charset=binary",
			expected:     ".webp",
		},
		{
			name:         "Simple MIME type",
			originalFile: "",
			mimeType:     "image/jpeg",
			expected:     ".jpeg",
		},
		{
			name:         "Original filename takes precedence",
			originalFile: "test.pdf",
			mimeType:     "application/octet-stream; charset=UTF-8",
			expected:     ".pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineMediaExtension(tt.originalFile, tt.mimeType)
			if result != tt.expected {
				t.Errorf("determineMediaExtension() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetermineMediaExtension_NoSemicolonInResult(t *testing.T) {
	// Ensure no MIME type produces an extension with semicolon
	problematicMimeTypes := []string{
		"audio/ogg; codecs=opus",
		"video/mp4; codecs=\"avc1.42E01E\"",
		"application/pdf; charset=UTF-8",
	}

	for _, mimeType := range problematicMimeTypes {
		ext := determineMediaExtension("", mimeType)
		if strings.Contains(ext, ";") {
			t.Errorf("Extension contains semicolon: %s (from MIME: %s)", ext, mimeType)
		}
	}
}
```

### Integration Tests

**File**: `src/integration_test.go`

```go
func TestMediaDownloadAndAccess(t *testing.T) {
	// 1. Mock WhatsApp audio message with parameterized MIME type
	// 2. Trigger message event
	// 3. Verify file is saved with clean extension
	// 4. Verify HTTP access returns 200
	// 5. Verify webhook payload contains accessible path
}
```

### Manual Testing

1. **Send test audio message** to WhatsApp
2. **Verify filename** on disk has clean extension:
   ```bash
   ls -la /app/instances/*/statics/media/*.ogg
   # Should NOT contain "; codecs=opus"
   ```
3. **Check webhook payload** sent to consumers
4. **Verify HTTP access** to media URL returns 200
5. **Test with various media types**:
   - Audio (ogg, mp3, m4a)
   - Video (mp4, avi)
   - Images (webp with parameters)

## Migration Guide

### For Operators

**Before deploying the fix**:

1. **Backup media directories**:
   ```bash
   tar -czf media-backup-$(date +%Y%m%d).tar.gz /app/instances/*/statics/media/
   ```

2. **Count affected files**:
   ```bash
   find /app/instances -path "*/statics/media/*" -name "*;*" | wc -l
   ```

**After deploying the fix**:

1. **Deploy updated code** with Solution 1
2. **Run migration script** (Solution 2):
   ```bash
   bash scripts/fix-media-filenames.sh
   ```
3. **Verify migration**:
   ```bash
   # Should return 0
   find /app/instances -path "*/statics/media/*" -name "*;*" | wc -l
   ```
4. **Test media access** from Chatwoot/integrations
5. **Monitor for 24 hours** for any 404 errors

### Rollback Plan

If issues occur:

1. **Restore from backup**:
   ```bash
   tar -xzf media-backup-YYYYMMDD.tar.gz -C /
   ```
2. **Revert code deployment**
3. **Implement Solution 3** (fallback mechanism) as temporary fix

## Monitoring & Alerts

### Metrics to Add

```go
// Prometheus metrics
var (
	mediaAccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_media_access_total",
			Help: "Total number of media access attempts",
		},
		[]string{"status", "extension"}, // 200, 404, etc.
	)

	mediaFilenameIssuesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "whatsapp_media_filename_issues_total",
			Help: "Total number of media files saved with malformed names",
		},
	)
)
```

### Alerts

```yaml
- alert: WhatsAppMediaHighNotFoundRate
  expr: rate(whatsapp_media_access_total{status="404"}[5m]) > 0.1
  severity: warning
  annotations:
    summary: "High rate of 404 errors on media access"
    description: "More than 10% of media requests are returning 404"

- alert: WhatsAppMediaMalformedFilenames
  expr: rate(whatsapp_media_filename_issues_total[5m]) > 0
  severity: critical
  annotations:
    summary: "Media files still being saved with malformed names"
    description: "MIME parameter stripping is not working correctly"
```

### Logging Improvements

```go
// When saving media
logrus.WithFields(logrus.Fields{
	"mime_type":        mimeType,
	"original_mime":    originalMimeType,
	"extension":        extension,
	"has_semicolon":    strings.Contains(extension, ";"),
	"filename":         filepath.Base(extractedMedia.MediaPath),
}).Debug("Media file saved")

// When serving media (if 404)
logrus.WithFields(logrus.Fields{
	"requested_path":   c.Path(),
	"actual_files":     listMediaFiles(),
	"similar_files":    findSimilarFiles(c.Path()),
}).Warn("Media file not found")
```

## Related Issues

- **Original Report**: Chatwoot ActiveJob error logs (2025-10-30)
- **Related Code**: `src/pkg/utils/whatsapp.go:69-89`
- **Related Docs**: `docs/webhook-payload.md` (Media Messages section)

## External References

- **RFC 2045**: MIME Part One - Format of Internet Message Bodies
  - Section 5.1: Syntax of the Content-Type Header Field
  - States that parameters follow the type/subtype, separated by semicolon
- **Go mime package**: `mime.ExtensionsByType()` expects base MIME type without parameters

## Communication Plan

### Internal Team

- [x] Document issue (this file)
- [ ] Implement Solution 1 (fix `determineMediaExtension()`)
- [ ] Create migration script (Solution 2)
- [ ] Add unit tests
- [ ] Test in staging environment
- [ ] Deploy to production
- [ ] Run migration script on all instances
- [ ] Monitor for 24 hours

### Users/Integrators

**If users report media access issues**:
- Acknowledge the issue
- Explain that it's a filename encoding bug
- Provide ETA for fix (same day)
- Notify when fix is deployed and migration is complete

## Next Steps

### Immediate (Today)

- [ ] Implement fix in `determineMediaExtension()` function
- [ ] Add unit tests for MIME parameter stripping
- [ ] Create file migration script
- [ ] Test fix locally with various MIME types
- [ ] Prepare deployment plan

### Short-term (This Week)

- [ ] Deploy fix to staging
- [ ] Run migration script on staging
- [ ] Verify all media accessible via HTTP
- [ ] Deploy to production
- [ ] Run migration script on all production instances
- [ ] Monitor media access metrics

### Medium-term (Next Week)

- [ ] Add integration tests
- [ ] Add monitoring and alerts
- [ ] Document in changelog
- [ ] Optional: Implement fallback mechanism if needed

### Optional Improvements

- [ ] Add webhook payload validation
- [ ] Add media filename sanitization tests
- [ ] Review all MIME type handling code
- [ ] Add pre-commit hook to check for MIME parameter handling

---

**Issue Created**: 2025-10-30
**Last Updated**: 2025-10-30
**Date Resolved**: -
**Priority**: P0 - CRITICAL
**Fix Status**: 🔴 OPEN - Awaiting implementation
**Estimated Resolution**: Same day
**Related Files**:
- `src/pkg/utils/whatsapp.go:69-89` (Bug location)
- `src/pkg/utils/whatsapp.go:581` (File save location)
- `src/infrastructure/whatsapp/event_message.go:133-195` (Webhook payload creation)
- `src/cmd/rest.go:47` (Static file serving)

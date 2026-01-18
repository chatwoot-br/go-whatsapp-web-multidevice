# ISSUE-003: S3 Presigned URL Image Download Fails with "unsupported file type"

## Status: Fixed

## Summary
When sending images via Chatwoot in production environments using S3/Object Storage, the `DownloadImageFromURL()` function fails with error "unsupported file type: " because S3 presigned URLs use random keys without file extensions. The function extracted the filename from URL path and validated the extension, which doesn't exist in S3 URLs.

## Environment
- **go-whatsapp-web-multidevice**: v8.1.0+4
- **Chatwoot**: chatwoot-br fork with S3/Object Storage
- **Storage**: Hetzner Object Storage (S3-compatible)
- **Test scenario**: Send image attachment from Chatwoot conversation

## Problem Description

### Observed Behavior
When sending an image from Chatwoot in production:

1. Chatwoot sends image attachment URL via `SendReplyJob`
2. gowa receives URL like: `https://acme-woot.nbg1.your-objectstorage.com/chatwoot/rptnjp5fjmnjaawhfbe995pxd5hs?...`
3. `DownloadImageFromURL()` validates Content-Type header (passes: `image/png`)
4. Function extracts filename from URL path: `rptnjp5fjmnjaawhfbe995pxd5hs`
5. `filepath.Ext()` returns empty string (no extension)
6. **BUG**: Empty string fails `allowedExtensions` check
7. Returns error: "unsupported file type: "

### Why It Works in Development
| Environment | Storage | URL Format | Has Extension? |
|------------|---------|------------|----------------|
| Development | Local disk | `/rails/active_storage/blobs/.../image.png` | Yes |
| Production | S3/Object Storage | `https://s3.../rptnjp5fjmnjaawhfbe995pxd5hs?...` | No |

### Log Evidence (2026-01-17)

**gowa log**:
```
Panic recovered in middleware: failed to download image from URL unsupported file type:
```

**Chatwoot worker log**:
```json
{
  "message_type": 2,
  "content_type": "image",
  "content_attributes": {
    "items": [{
      "thumb_url": "https://acme-woot.nbg1.your-objectstorage.com/chatwoot/variants/rptnjp5fjmnjaawhfbe995pxd5hs/...",
      "data_url": "https://acme-woot.nbg1.your-objectstorage.com/chatwoot/rptnjp5fjmnjaawhfbe995pxd5hs?response-content-type=image%2Fpng"
    }]
  }
}
```

### Expected Behavior
The function should derive the file extension from the `Content-Type` HTTP header (already validated) rather than relying on URL path, which may not contain an extension for S3/Object Storage URLs.

## Root Cause Analysis

### Code Location
**File**: `src/pkg/utils/general.go`
**Function**: `DownloadImageFromURL()` (lines 235-285)

### The Bug
```go
func DownloadImageFromURL(url string) ([]byte, string, error) {
    // Content-Type validation (CORRECT)
    contentType := response.Header.Get("Content-Type")
    if !strings.HasPrefix(contentType, "image/") {
        return nil, "", fmt.Errorf("invalid content type: %s", contentType)
    }

    // Extract filename from URL (PROBLEMATIC for S3)
    segments := strings.Split(url, "/")
    fileName := segments[len(segments)-1]
    fileName = strings.Split(fileName, "?")[0]  // → "rptnjp5fjmnjaawhfbe995pxd5hs"

    // BUG: Extension check fails for extensionless URLs
    allowedExtensions := map[string]bool{
        ".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
    }
    extension := strings.ToLower(filepath.Ext(fileName))  // → ""
    if !allowedExtensions[extension] {  // allowedExtensions[""] is false
        return nil, "", fmt.Errorf("unsupported file type: %s", extension)  // ERROR
    }
}
```

### Inconsistency with Other Functions
| Function | Content-Type Check | Extension Check |
|----------|-------------------|-----------------|
| `DownloadImageFromURL()` | Yes | Yes (BUG) |
| `DownloadAudioFromURL()` | Yes | No |
| `DownloadVideoFromURL()` | Yes | No |

Only `DownloadImageFromURL()` had the redundant and problematic extension check.

## Fix Applied

### Branch
`milesibastos/fix-s3-image-extension`

### Commits
| SHA | Message |
|-----|---------|
| `b773863` | test: add failing test for S3 presigned URL image download |
| `c5f600b` | test: update GIF test to expect MIME type validation error |
| `36f23b5` | fix(utils): derive image extension from Content-Type for S3 URLs |

### Change
Use Content-Type header as the source of truth for MIME type validation and file extension derivation, consistent with `DownloadAudioFromURL()` and `DownloadVideoFromURL()`:

```go
func DownloadImageFromURL(url string) ([]byte, string, error) {
    // Extract MIME type without parameters (e.g., "image/png; charset=utf-8" -> "image/png")
    contentType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])

    // Map allowed MIME types to file extensions
    mimeToExt := map[string]string{
        "image/jpeg": ".jpg",
        "image/jpg":  ".jpg",
        "image/png":  ".png",
        "image/webp": ".webp",
    }

    extension, ok := mimeToExt[contentType]
    if !ok {
        return nil, "", fmt.Errorf("unsupported image type: %s", contentType)
    }

    // ... content length check ...

    // Extract filename from URL
    segments := strings.Split(url, "/")
    fileName := segments[len(segments)-1]
    fileName = strings.Split(fileName, "?")[0]

    // Add extension if filename doesn't have one (common for S3 presigned URLs)
    if filepath.Ext(fileName) == "" {
        fileName = fileName + extension
    }
}
```

## Impact

### Before Fix
- Image sending fails in production with S3/Object Storage
- Error: "unsupported file type: "
- Users cannot send images from Chatwoot

### After Fix
- S3 presigned URLs work correctly
- Extension derived from Content-Type header
- Consistent behavior with audio/video download functions
- Works with both traditional URLs (with extension) and S3 URLs (without extension)

## Testing

### Unit Tests Added
```go
// Test S3-style URL without file extension (derives extension from Content-Type)
s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if strings.HasPrefix(r.URL.Path, "/bucket/") {
        w.Header().Set("Content-Type", "image/png")
        w.Write([]byte("s3 image data"))
    }
}))
defer s3Server.Close()

data, filename, err = utils.DownloadImageFromURL(s3Server.URL + "/bucket/rptnjp5fjmnjaawhfbe995pxd5hs")
assert.NoError(suite.T(), err)
assert.Equal(suite.T(), "rptnjp5fjmnjaawhfbe995pxd5hs.png", filename)
```

### Test Results
- All 61 tests pass
- Build completes without errors

### Manual Verification
1. Deploy new version to production
2. Send image from Chatwoot conversation
3. Verify image is received on WhatsApp

## Related Files
- `src/pkg/utils/general.go` - Fixed file (DownloadImageFromURL function)
- `src/pkg/utils/general_test.go` - Test file (new S3 URL tests)
- Chatwoot: `app/models/attachment.rb` - Source of S3 URLs via `file.blob.url`
- Chatwoot: `app/services/whatsapp/providers/whatsapp_web_service.rb` - `accessible_download_url` method

## Related Issues
- None (this is a gowa-specific issue)

## Timeline
- **2026-01-17**: Issue reported from production image sending failures
- **2026-01-17**: Root cause analysis completed (S3 URLs lack file extensions)
- **2026-01-17**: TDD implementation (failing tests first, then fix)
- **2026-01-17**: All tests passing, fix verified

## Deployment Notes
After merging:
1. Create new release tag (e.g., `v8.1.0+5`)
2. GitHub Actions will build and push new Docker image
3. Update Helm chart or deployment to use new image tag

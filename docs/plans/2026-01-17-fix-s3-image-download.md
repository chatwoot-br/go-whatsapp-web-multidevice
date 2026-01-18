# Fix S3 Image Download - "unsupported file type" Error

> **Status:** ✅ COMPLETED (2026-01-17)

**Goal:** Fix image sending failures when using S3/Object Storage by deriving file extension from Content-Type header instead of URL path.

**Architecture:** Modify `DownloadImageFromURL()` to use Content-Type header as the source of truth for MIME type validation and file extension derivation, consistent with how `DownloadAudioFromURL()` and `DownloadVideoFromURL()` already work.

**Tech Stack:** Go, httptest for testing

---

## Implementation Summary

**Branch:** `milesibastos/fix-s3-image-extension`

**Commits:**
| SHA | Message |
|-----|---------|
| `b773863` | test: add failing test for S3 presigned URL image download |
| `c5f600b` | test: update GIF test to expect MIME type validation error |
| `36f23b5` | fix(utils): derive image extension from Content-Type for S3 URLs |

**Files Modified:**
- `src/pkg/utils/general.go` - Main fix (MIME-based extension derivation)
- `src/pkg/utils/general_test.go` - S3 URL tests + updated assertions

**Test Results:** All 61 tests pass ✅

---

## Context

**Problem:** Images fail to send in production with error `failed to download image from URL unsupported file type: `

**Root Cause:** S3 presigned URLs use random keys without file extensions (e.g., `/chatwoot/rptnjp5fjmnjaawhfbe995pxd5hs`). The `DownloadImageFromURL()` function extracts filename from URL path and checks extension, which fails for S3 URLs.

**Why it works in development:** Local storage URLs include filenames like `/image.png` with extensions.

---

### Task 1: Add Test for S3 Presigned URL (No Extension) ✅

**Files:**
- Modify: `src/pkg/utils/general_test.go` (after line 403)

**Step 1: Write the failing test**

Add this test case inside `TestDownloadImageFromURL()` function, after the "Test filename extraction with query parameters" section:

```go
	// Test S3-style URL without file extension (derives extension from Content-Type)
	s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate S3 presigned URL path (random key, no extension)
		if strings.HasPrefix(r.URL.Path, "/bucket/") {
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("s3 image data"))
		}
	}))
	defer s3Server.Close()

	data, filename, err = utils.DownloadImageFromURL(s3Server.URL + "/bucket/rptnjp5fjmnjaawhfbe995pxd5hs")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "rptnjp5fjmnjaawhfbe995pxd5hs.png", filename)
	assert.Equal(suite.T(), []byte("s3 image data"), data)

	// Test S3-style URL with JPEG content type
	s3JpegServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("jpeg data"))
	}))
	defer s3JpegServer.Close()

	data, filename, err = utils.DownloadImageFromURL(s3JpegServer.URL + "/bucket/randomkey123")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "randomkey123.jpg", filename)

	// Test S3-style URL with WebP content type
	s3WebpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		w.Write([]byte("webp data"))
	}))
	defer s3WebpServer.Close()

	data, filename, err = utils.DownloadImageFromURL(s3WebpServer.URL + "/bucket/anotherkey456")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "anotherkey456.webp", filename)
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src && go test ./pkg/utils/... -run TestDownloadImageFromURL -v`

Expected: FAIL with "unsupported file type: " because the S3 URL has no extension.

**Step 3: Commit failing test**

```bash
cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice
git add src/pkg/utils/general_test.go
git commit -m "test: add failing test for S3 presigned URL image download"
```

---

### Task 2: Update GIF Test to Check MIME Type Error ✅

**Files:**
- Modify: `src/pkg/utils/general_test.go` (lines 357-368)

**Step 1: Update the GIF test assertion**

The current test expects "unsupported file type" error from extension check. After our fix, it will fail on MIME type validation instead. Update the test to check for the new error message:

Find this code (around line 368):
```go
	assert.Contains(suite.T(), err.Error(), "unsupported file type")
```

Replace with:
```go
	assert.Contains(suite.T(), err.Error(), "unsupported image type")
```

**Step 2: Commit the test update**

```bash
cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice
git add src/pkg/utils/general_test.go
git commit -m "test: update GIF test to expect MIME type validation error"
```

---

### Task 3: Implement Content-Type Based Extension Derivation ✅

**Files:**
- Modify: `src/pkg/utils/general.go` (lines 255-279)

**Step 1: Replace extension validation with MIME-based approach**

Find this code block (lines 255-279):
```go
	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("invalid content type: %s", contentType)
	}
	// Check content length if available
	if contentLength := response.ContentLength; contentLength > int64(config.WhatsappSettingMaxImageSize) {
		return nil, "", fmt.Errorf("image size %d exceeds maximum allowed size %d", contentLength, config.WhatsappSettingMaxImageSize)
	}
	// Limit the size from config
	reader := io.LimitReader(response.Body, int64(config.WhatsappSettingMaxImageSize))
	// Extract the file name from the URL and remove query parameters if present
	segments := strings.Split(url, "/")
	fileName := segments[len(segments)-1]
	fileName = strings.Split(fileName, "?")[0]
	// Check if the file extension is supported
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	if !allowedExtensions[extension] {
		return nil, "", fmt.Errorf("unsupported file type: %s", extension)
	}
```

Replace with:
```go
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

	// Check content length if available
	if contentLength := response.ContentLength; contentLength > int64(config.WhatsappSettingMaxImageSize) {
		return nil, "", fmt.Errorf("image size %d exceeds maximum allowed size %d", contentLength, config.WhatsappSettingMaxImageSize)
	}
	// Limit the size from config
	reader := io.LimitReader(response.Body, int64(config.WhatsappSettingMaxImageSize))

	// Extract the file name from the URL and remove query parameters if present
	segments := strings.Split(url, "/")
	fileName := segments[len(segments)-1]
	fileName = strings.Split(fileName, "?")[0]

	// Add extension if filename doesn't have one (common for S3 presigned URLs)
	if filepath.Ext(fileName) == "" {
		fileName = fileName + extension
	}
```

**Step 2: Run tests to verify all pass**

Run: `cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src && go test ./pkg/utils/... -run TestDownloadImageFromURL -v`

Expected: All tests PASS including the new S3 presigned URL tests.

**Step 3: Run full test suite**

Run: `cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src && go test ./... -v`

Expected: All tests PASS.

**Step 4: Commit the fix**

```bash
cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice
git add src/pkg/utils/general.go
git commit -m "fix(utils): derive image extension from Content-Type for S3 URLs

S3/Object Storage presigned URLs use random keys without file extensions.
This caused 'unsupported file type' errors when sending images in production.

The fix uses Content-Type header as the source of truth for MIME type
validation and file extension derivation, consistent with DownloadAudioFromURL
and DownloadVideoFromURL.

Fixes production image sending failures with S3 storage."
```

---

### Task 4: Verify Build and Final Tests ✅

**Step 1: Build the application**

Run: `cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src && go build -o /tmp/whatsapp`

Expected: Build succeeds with no errors.

**Step 2: Run all tests one final time**

Run: `cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src && go test ./... -v`

Expected: All tests PASS.

---

## Verification Checklist

- [x] All existing tests pass
- [x] New S3 presigned URL tests pass
- [x] Build completes without errors
- [x] GIF test updated to check MIME type error (not extension error)

## Deployment Notes

After merging:
1. Create new release tag (e.g., `v8.1.0+5`)
2. GitHub Actions will build and push new Docker image
3. Update Helm chart or deployment to use new image tag

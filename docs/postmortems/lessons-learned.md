# Lessons Learned from Minor Issues

This document captures key learnings from issues that didn't warrant full postmortems but contain valuable insights for development and operations.

## Configuration Issues

### Audio URL Validation Failure (ISSUE-005)

**Date**: 2025-10-30
**Severity**: High (but not a bug in go-whatsapp-web-multidevice)
**Status**: Configuration Issue - Chatwoot Side

#### Summary

Chatwoot-generated audio URLs failed validation because they used `http://0.0.0.0:3000` as the base URL. The hostname `0.0.0.0` is a bind-all address that:
1. Fails standard URL validation (`is.URL` check)
2. Cannot be accessed from the WhatsApp service container

#### Key Learning

**The Issue**: Chatwoot's Rails configuration used `0.0.0.0` as the host in URL generation:
```ruby
config.action_controller.default_url_options = {
  host: '0.0.0.0',  # INVALID for URLs
  port: 3000
}
```

**Why It Fails**:
- `0.0.0.0` is valid for binding servers (listen on all interfaces)
- `0.0.0.0` is **invalid** as a destination hostname in URLs
- Cannot be used in URLs for client connections
- Fails RFC-compliant URL validation

**The Fix**: Update Chatwoot configuration to use a proper hostname:
```ruby
config.action_controller.default_url_options = {
  host: 'host.docker.internal',  # or container name on shared network
  port: 3000
}
```

#### Lessons

1. **Separation of Concerns**:
   - Bind addresses (`0.0.0.0`) are for servers
   - Hostnames in URLs are for clients
   - These are fundamentally different concepts

2. **Docker Networking**:
   - Use `host.docker.internal` to access host from containers
   - Use container names on shared Docker networks
   - Don't expose internal bind addresses in URLs

3. **URL Validation**:
   - Strict validation is correct behavior
   - The service should not rewrite malformed URLs
   - Fix URL generation at the source, not in consumers

4. **Environment Variables**:
   - Make host/port configurable via environment
   - Different environments may need different values
   - Document valid values in deployment guides

#### Resolution

**Not a Bug**: This was a configuration issue in Chatwoot, not a bug in go-whatsapp-web-multidevice. The URL validation working as intended caught a misconfiguration.

**User Action Required**: Update Chatwoot's Rails configuration to use a proper hostname.

**References**:
- Original issue: `docs/issues/ISSUE-005-AUDIO-URL-VALIDATION-FAILURE.md`
- Docker networking: https://docs.docker.com/desktop/networking/

---

## Integration Issues

### Chatwoot PDF Upload Failure (ISSUE-002-CHATWOOT)

**Date**: 2025-10-30
**Severity**: High (Chatwoot integration issue)
**Status**: Resolved on Chatwoot Side

#### Summary

PDF file attachments failed to send through the WhatsApp channel in Chatwoot with error:
```
Message: request Content-Type has bad boundary or is not multipart/form-data
```

Images, videos, and audio files worked correctly - only PDF/document files failed.

#### Key Learning

**The Issue**: Chatwoot's `SendReplyJob` was constructing the HTTP multipart/form-data request incorrectly for PDF files compared to image/video files.

**Root Cause**:
- Different code paths for `file_type: "file"` vs `file_type: "image"`
- Multipart form boundary not properly set in Content-Type header for documents
- Or form field name was incorrect (`document` instead of `file`)

**The Fix**: Chatwoot team fixed their multipart form construction to handle PDF files correctly.

#### Lessons

1. **File Type Handling**:
   - Don't treat document files differently from media files
   - Use the same multipart construction for all file types
   - Test all file type code paths, not just the common ones

2. **HTTP Multipart Forms**:
   - Must include proper boundary in Content-Type header
   - Field names must match API expectations
   - Form construction should be consistent across file types

3. **Error Messages**:
   - "bad boundary or is not multipart/form-data" clearly indicates malformed request
   - Validates at the HTTP level before even processing
   - Good error handling catches issues early

4. **Integration Testing**:
   - Test all file types in integrations, not just images
   - PDF/documents have different MIME types and handling
   - Don't assume all media types behave the same

#### Resolution

**Status**: Resolved by Chatwoot team fixing their multipart form construction.

**Verification**: PDF files now upload and send successfully through Chatwoot's WhatsApp channel.

**References**:
- Original issue: `docs/issues/ISSUE-002-CHATWOOT-PDF-UPLOAD-FAILURE.md`

---

## Platform-Specific Behaviors

### Alpine Linux Missing MIME Database

**Date**: 2025-10-30
**Component**: Environment Analysis for ISSUE-002
**Impact**: Contributed to MIME pollution bug

#### Summary

The MIME filename pollution bug (see Postmortem 004) only manifested in production because Alpine Linux doesn't include a MIME types database by default, while development environments (macOS/Debian) do.

#### Key Learning

**The Difference**:
- **Development (macOS/Debian)**: Has `/etc/mime.types` or `/etc/apache2/mime.types`
  - `mime.ExtensionsByType()` returns proper extensions
  - Bug never triggered because fallback code wasn't reached

- **Production (Alpine)**: No MIME database by default
  - `mime.ExtensionsByType()` returns empty array
  - Falls through to buggy string split code
  - Bug manifested as malformed filenames

#### Lessons

1. **Environment Parity is Critical**:
   - Dev and prod should be as similar as possible
   - Test in production-like containers (Alpine) not just local dev
   - Platform differences can hide bugs

2. **Don't Trust Platform-Specific Behavior**:
   - Go's `mime` package behavior varies by platform
   - Always have robust fallbacks that work without platform dependencies
   - Test without system dependencies

3. **Minimal Docker Images Have Trade-offs**:
   - Alpine is great for small images
   - Missing standard files can cause subtle bugs
   - Document what's missing and why
   - Consider adding essential dependencies

4. **Testing Strategy**:
   - Test on target production platform
   - Test with missing system dependencies
   - Use CI to test multiple platforms
   - Don't assume dev environment behavior matches prod

#### Resolution

**Immediate Fix**: Strip MIME parameters before processing (makes code work regardless of database)

**Long-term Fix**: Added `mailcap` package to Alpine image for MIME database (~50KB)

**Benefits**:
- More accurate extension determination
- Consistent behavior with dev environment
- Defense in depth

**Testing Recommendation**:
```bash
# Test in Alpine container
docker run --rm -v $(pwd):/app -w /app golang:1.24-alpine3.20 \
  sh -c "cd src && go test ./pkg/utils"
```

**References**:
- Environment analysis: `docs/issues/ISSUE-002-ADDENDUM-ENVIRONMENT-ANALYSIS.md`
- Main postmortem: `docs/postmortems/004-media-filename-mime-pollution.md`

---

## General Development Principles

### Patterns Observed Across Issues

1. **Defensive Programming**:
   - Always check for nil before dereferencing
   - Add panic recovery to goroutines
   - Validate input from external systems
   - Don't trust library behavior to be stable

2. **Environment Consistency**:
   - Test in production-like environments
   - Document platform differences
   - Add essential dependencies even to minimal images
   - Use Docker for consistent testing

3. **Configuration Over Code**:
   - Don't hard-code environment-specific values
   - Use environment variables for deployment-specific settings
   - Provide sensible defaults
   - Document required configuration

4. **Error Handling**:
   - Validate early and fail fast
   - Provide clear error messages
   - Log errors with context
   - Don't hide errors or fail silently

5. **Integration Testing**:
   - Test all file types and edge cases
   - Test with real downstream systems
   - Simulate network issues and delays
   - Don't assume happy path always works

6. **Library Management**:
   - Monitor upstream for fixes and breaking changes
   - Test library updates in staging
   - Have rollback plan for updates
   - Document library-specific issues

### Questions to Ask When Investigating Issues

1. **Is this environment-specific?**
   - Test on target platform
   - Check for missing dependencies
   - Verify configuration differences

2. **Is this a race condition?**
   - Look at timing patterns
   - Check for shared mutable state
   - Examine goroutine interactions

3. **Is this a configuration issue?**
   - Check environment variables
   - Verify service-to-service addressing
   - Look at integration settings

4. **Is this an upstream issue?**
   - Check library issue trackers
   - Look for recent commits
   - Test with different library versions

5. **Is this a protocol change?**
   - Check if external APIs changed
   - Look for version mismatches
   - Verify protocol documentation

---

## Related Documentation

- [Postmortem: Profile Picture Panic](001-profile-picture-panic.md)
- [Postmortem: Multi-Device Encryption](002-multidevice-encryption.md)
- [Postmortem: Auto-Reconnect Panic](003-auto-reconnect-panic.md)
- [Postmortem: MIME Filename Pollution](004-media-filename-mime-pollution.md)
- [Troubleshooting Guide](../reference/troubleshooting.md)
- [Architecture Documentation](../developer/architecture.md)

---

**Last Updated**: 2025-10-30
**Purpose**: Capture learnings from issues that don't warrant full postmortems
**Audience**: Developers, operators, integrators

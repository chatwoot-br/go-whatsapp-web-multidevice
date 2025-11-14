# Postmortems

Lessons learned from critical issues and their resolutions.

## Critical Postmortems

### Service Crashes

1. **[Profile Picture Panic](001-profile-picture-panic.md)** - Service crash on profile picture fetch
   - **Date**: 2025-10-07
   - **Severity**: Critical
   - **Status**: Resolved (library update)
   - **Impact**: Complete service crash, potential message loss
   - **Root Cause**: Unsupported `*store.PrivacyToken` payload in whatsmeow library
   - **Resolution**: Updated whatsmeow to v0.0.0-20251005083110-4fe97da162dc

2. **[Auto-Reconnect Panic](003-auto-reconnect-panic.md)** - Nil pointer dereference in auto-reconnect goroutine
   - **Date**: 2025-10-30
   - **Severity**: Critical
   - **Status**: Resolved (architecture fix)
   - **Impact**: Predictable crash 5 minutes after startup
   - **Root Cause**: Goroutine held stale client reference with nil Store
   - **Resolution**: Changed to use global client pattern with nil checks

### Message Delivery Issues

3. **[Multi-Device Encryption Failure](002-multidevice-encryption.md)** - Messages not reaching recipient's main phone
   - **Date**: 2025-10-30
   - **Severity**: Medium-High
   - **Status**: Fix Available (library update required)
   - **Impact**: Messages only reached linked devices, not main phone (Device 0)
   - **Root Cause**: Device cache not invalidated on participant hash mismatch, LID migration complexity
   - **Resolution**: Update to whatsmeow with Oct 28, 2025 fix

### Media Handling Issues

4. **[Media Filename MIME Pollution](004-media-filename-mime-pollution.md)** - Media files inaccessible due to MIME parameters in filenames
   - **Date**: 2025-10-30
   - **Severity**: High
   - **Status**: Resolved (code fix + Alpine enhancement)
   - **Impact**: Audio files saved with semicolons in filenames, causing 404 errors
   - **Root Cause**: MIME parameters not stripped, Alpine lacks MIME database
   - **Resolution**: Strip parameters before processing + added mailcap to Alpine

## Lessons Learned

5. **[Lessons Learned](lessons-learned.md)** - Key insights from minor issues
   - Configuration issues (Audio URL validation)
   - Integration issues (Chatwoot PDF upload)
   - Platform-specific behaviors (Alpine MIME database)
   - General development principles

## Statistics

- **Total Critical Issues**: 4
- **Total Lessons Learned**: 3 minor issues
- **Average Resolution Time**: Same day to 2 days
- **Library-Related Issues**: 3 of 4 (whatsmeow)
- **Architecture Issues**: 1 of 4 (global client pattern)
- **Environment-Specific**: 1 of 4 (Alpine MIME database)

## About This Section

This section documents significant issues encountered in production, their root causes, and resolutions. The purpose is to:

- Share knowledge about complex problems
- Prevent similar issues in the future
- Document system behavior and edge cases
- Build institutional knowledge
- Track patterns across issues

## Format

Each postmortem includes:

- **Incident Summary**: What happened (2-3 sentences)
- **Impact**: Severity, scope, and user experience
- **Timeline**: Key events from detection to resolution
- **Root Cause**: Technical analysis of what went wrong
- **Resolution**: How it was fixed
- **Prevention**: Steps to prevent recurrence
- **Lessons Learned**: What went well, what could be improved
- **Related Documentation**: Links to relevant docs and external references

## Common Themes

### Library Management
- Monitor upstream repositories for fixes
- Test library updates in staging before production
- Document library-specific issues and workarounds
- Keep dependencies reasonably up-to-date

### Environment Parity
- Test in production-like containers (Alpine)
- Document platform differences
- Add essential dependencies to minimal images
- Use CI to test multiple platforms

### Defensive Programming
- Check for nil before dereferencing
- Add panic recovery to goroutines
- Validate input from external systems
- Use global patterns for shared resources

### Configuration Management
- Use environment variables for deployment-specific settings
- Validate configuration early
- Provide clear error messages for misconfigurations
- Document valid configuration values

## Related Documentation

- **[Troubleshooting Guide](../reference/troubleshooting.md)** - Common issues and solutions
- **[Architecture](../developer/architecture.md)** - System design and patterns
- **[Operations Guide](../operations/)** - Production best practices
- **[Release Process](../developer/release-process.md)** - How to release new versions
- **[CLAUDE.md](../../CLAUDE.md)** - Development guide

## Archive

Original issue documents have been archived from `docs/issues/` after conversion to postmortem format. The structured postmortem format provides better insights for future reference and prevention.

---

**Last Updated**: 2025-11-14
**Maintained By**: Development Team
**Purpose**: Document production issues and learnings

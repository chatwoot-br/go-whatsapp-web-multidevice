# Postmortem: Service Crash on Profile Picture Fetch

**Date**: 2025-10-07
**Severity**: Critical
**Status**: Resolved
**Version Affected**: whatsmeow `v0.0.0-20251003111114-4479f300784e`
**Version Fixed**: whatsmeow `v0.0.0-20251005083110-4fe97da162dc`

## Incident Summary

The go-whatsapp-web-multidevice service crashed with a panic when attempting to fetch profile pictures from WhatsApp contacts. The panic was caused by an unsupported payload type (`*store.PrivacyToken`) in the whatsmeow library, causing complete service restarts and potential message loss in downstream systems like Chatwoot.

The issue occurred when WhatsApp updated their protocol to include privacy tokens in profile picture requests, but the whatsmeow library version in use did not support this new payload type.

## Impact

**Severity: CRITICAL**

- **Service Availability**: Complete service crash requiring restart
- **Message Loss**: Messages could be lost during service restart in downstream systems
- **User Experience**: Profile pictures failed to load
- **Reliability**: Cascading failures affecting message processing

### Affected Operations

1. Working: Message webhook delivery
2. Failing: Profile picture fetch (caused panic)
3. Affected: All downstream systems dependent on service availability

### Timeline

- **2025-10-07 19:01:36Z**: Service successfully processed messages and forwarded to webhook
- **2025-10-07 19:01:36Z**: Profile picture fetch attempted, service crashed with panic
- **2025-10-07 19:01:41Z**: Service automatically restarted
- **2025-10-07 (same day)**: Issue diagnosed and whatsmeow library updated
- **2025-10-07**: Fix verified with tests, pending production deployment

## Root Cause

### Technical Details

**Panic Message**:
```
panic: unsupported payload type: *store.PrivacyToken

goroutine [running]:
go.mau.fi/whatsmeow/binary.(*binaryEncoder).write(...)
go.mau.fi/whatsmeow.(*Client).GetProfilePictureInfo(...)
github.com/aldinokemal/go-whatsapp-web-multidevice/usecase.serviceUser.Avatar.func1()
```

**Location**: `src/usecase/user.go:88`

**Root Cause Analysis**:
1. WhatsApp updated their protocol to include privacy tokens in profile picture requests
2. The whatsmeow library version `v0.0.0-20251003111114-4479f300784e` (Oct 3) did not support the `*store.PrivacyToken` payload type
3. The Avatar function in user.go ran in a goroutine without panic recovery
4. When `GetProfilePictureInfo` panicked, it crashed the entire service instead of returning an error

**Vulnerable Code**:
```go
func (service serviceUser) Avatar(ctx context.Context, request domainUser.AvatarRequest) (response domainUser.AvatarResponse, err error) {
    // ...
    go func() {
        // ... validation ...

        // LINE 88: This call panics with PrivacyToken error
        pic, err := whatsapp.GetClient().GetProfilePictureInfo(dataWaRecipient, &whatsmeow.GetProfilePictureParams{
            Preview:     request.IsPreview,
            IsCommunity: request.IsCommunity,
        })

        if err != nil {
            chanErr <- err
        } else if pic == nil {
            chanErr <- errors.New("no avatar found")
        } else {
            response.URL = pic.URL
            response.ID = pic.ID
            response.Type = pic.Type
            chanResp <- response
        }
    }()
    // ...
}
```

The goroutine lacked panic recovery, so when the library panicked, the entire service crashed.

## Resolution

### Fix Applied: Update whatsmeow Library

**Date**: 2025-10-07 (same day as discovery)

Updated the whatsmeow library from `v0.0.0-20251003111114-4479f300784e` to `v0.0.0-20251005083110-4fe97da162dc` (October 5, 2025).

**Changes Made**:
```bash
# Updated from:
go.mau.fi/whatsmeow v0.0.0-20251003111114-4479f300784e

# Updated to:
go.mau.fi/whatsmeow v0.0.0-20251005083110-4fe97da162dc
go.mau.fi/libsignal v0.2.1-0.20251004173110-6e0a3f2435ed
```

**Upstream Fixes** (included in updated version):
- Commit 1: "user: Add PrivacyToken to GetProfilePictureInfo (#950)" by purpshell (Oct 3, 2025)
- Commit 2: "user: fix including privacy token in GetProfilePictureInfo" by tulir (Oct 3, 2025)

These commits added proper support for the `*store.PrivacyToken` payload type that WhatsApp now includes in profile picture requests.

**Verification**:
- Code compiles successfully
- All tests pass (internal/admin, pkg/utils, usecase, validations)
- No breaking changes detected

### Deployment Status

- Staging: Deployed
- Production: Deployed (v7.7.1+)
- Monitoring: Completed - no recurrence observed

## Prevention

### Steps Taken to Prevent Recurrence

1. **Library Updates**: Established process for monitoring whatsmeow releases
   - Subscribe to whatsmeow repository notifications
   - Review changelog for each update
   - Test updates in staging before production

2. **Defensive Programming** (recommended for future):
   - Add panic recovery to all goroutines that interact with external libraries
   - Implement graceful error handling in async operations
   - Add monitoring for panic events

3. **Documentation Updates**:
   - Created this postmortem for future reference
   - Updated CLAUDE.md with troubleshooting section
   - Documented in CHANGELOG.md

### Future Improvements Considered

**Optional Enhancement: Add Panic Recovery**:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.WithFields(log.Fields{
                "panic":   r,
                "phone":   request.Phone,
                "preview": request.IsPreview,
            }).Error("Panic recovered in Avatar function")
            chanErr <- fmt.Errorf("failed to get profile picture: %v", r)
        }
    }()

    // ... existing code ...
}()
```

This would provide an additional safety net if similar library issues occur in the future.

## Lessons Learned

### What Went Well

1. **Quick Detection**: Issue was detected immediately through service crash
2. **Fast Resolution**: Root cause identified and fixed within hours on the same day
3. **Upstream Fix Available**: The issue was already fixed in whatsmeow upstream
4. **No Breaking Changes**: Library update was straightforward with no compatibility issues

### What Could Be Improved

1. **Proactive Monitoring**: Need to monitor whatsmeow repository more actively
2. **Panic Recovery**: Should add panic recovery to all goroutines as defensive measure
3. **Testing**: Need better integration tests that simulate WhatsApp protocol changes
4. **Staging Environment**: Should have caught this in staging before production

### Action Items

- [x] Update whatsmeow library to latest version
- [x] Verify no breaking changes
- [x] Run test suite
- [x] Deploy to staging environment
- [x] Deploy to production
- [x] Monitor for 24-48 hours
- [ ] Consider implementing panic recovery as defensive measure
- [ ] Set up automated alerts for whatsmeow releases

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - Project architecture and development guide
- [Release Process](../developer/release-process.md) - How to release new versions
- [Troubleshooting](../reference/troubleshooting.md) - Common issues and solutions

## External References

- **whatsmeow Repository**: https://github.com/tulir/whatsmeow
- **Issue #672**: https://github.com/tulir/whatsmeow/issues/672 - GetProfilePictureInfo WebSocket disconnection
- **PR #950**: https://github.com/tulir/whatsmeow/pull/950 - Add PrivacyToken to GetProfilePictureInfo
- **Original Issue**: `docs/features/ADR-001/ISSUE-001-message-loss-on-profile-picture-fetch.md`
- **Duplicate Issue**: `docs/issues/ISSUE-001-PROFILE-PICTURE-PANIC.md` (archived)

---

**Postmortem Author**: Development Team
**Last Updated**: 2025-12-05
**Resolution Time**: Same day (< 12 hours)
**Status**: Resolved and deployed

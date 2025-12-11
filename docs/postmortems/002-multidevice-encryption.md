# Postmortem: WhatsApp Multi-Device Encryption Failure

**Date**: 2025-10-30
**Severity**: Medium-High
**Status**: Resolved
**Version Affected**: whatsmeow `v0.0.0-20251024191251-088fa33fb87f` (Oct 24, 2025)
**Version with Fix**: whatsmeow `v0.0.0-20251028165006-ad7a618ba42f` (Oct 28, 2025+)

## Incident Summary

Messages sent to WhatsApp contacts with multiple devices (main phone + linked devices like WhatsApp Web) failed to reach the recipient's main phone (Device 0) while successfully delivering to linked devices. The issue was caused by the WhatsApp server not returning prekey bundles for Device 0, combined with whatsmeow's device cache not being invalidated on participant hash mismatches.

The root cause was traced to incomplete LID (Linked Identity Device) migration and a library bug where device caches weren't cleared when the server indicated a participant list hash mismatch.

## Impact

**Severity: MEDIUM-HIGH**

- **Message Delivery**: Messages only reached linked devices (e.g., WhatsApp Web), not the main phone
- **User Experience**: Recipients with linked devices saw messages on web/desktop but not on their primary phone
- **Silent Failure**: Senders received success confirmation despite incomplete delivery
- **Scope**: Affected recipients with multiple devices, particularly those with privacy settings enabled

### Affected Operations

1. Working: Message delivery to linked devices (Device 14, etc.)
2. Working: Message delivery to sender's devices (for sync)
3. Failing: Message delivery to recipient's main phone (Device 0)
4. Failing: Prekey bundle retrieval for Device 0

### Timeline

- **2025-10-30 11:42:07**: PDF document sent to contact with 2 devices
- **2025-10-30 11:42:07**: Device query returned: Device 0 + Device 14
- **2025-10-30 11:42:07**: Prekey request sent (missing Device 0)
- **2025-10-30 11:42:07**: Message encrypted for Device 14 only
- **2025-10-30 11:42:07**: Server returned participant hash mismatch warning
- **2025-10-30**: Issue diagnosed, upstream fix identified

## Root Cause

### Technical Details

**Error Message**:
```
Failed to encrypt 3EB0F0EFC85FA8B39070F9 for 5521998762522@s.whatsapp.net:
can't encrypt message for device: no signal session established with 151474956939293_1:0
```

**Warning from Server**:
```
Server returned different participant list hash when sending to 5521998762522@s.whatsapp.net.
Some devices may not have received the message.
```

**Root Cause Analysis**:

1. **Device Cache Issue**: The whatsmeow library cached device information, but when participant hash mismatches occurred, it didn't automatically invalidate and refresh the device cache

2. **LID Migration Complexity**: WhatsApp's transition from Phone Number (PN) JIDs to Linked Identity Device (LID) JIDs created confusion in device addressing:
   - Device query returned PN format: `5521998762522:0`
   - Library tried to use LID format: `151474956939293_1:0`
   - Prekey requests didn't properly map between formats

3. **Missing Prekey Request**: The library didn't request prekey bundle for Device 0:
   ```xml
   <iq id="252.188-91" to="s.whatsapp.net" type="get">
     <key>
       <user jid="5521998762522:14@s.whatsapp.net" reason="identity"/>  <!-- Device 14 ✓ -->
       <!-- MISSING: Device 0 prekey request -->
     </key>
   </iq>
   ```

4. **Incomplete Encryption**: Without a prekey bundle, Signal Protocol couldn't establish a session for Device 0:
   ```xml
   <message id="3EB0F0EFC85FA8B39070F9" to="5521998762522@s.whatsapp.net">
     <participants>
       <to jid="5521998762522:14@s.whatsapp.net">
         <enc type="pkmsg" v="2"><!-- Device 14 encrypted ✓ --></enc>
       </to>
       <!-- MISSING: Device 0 encryption -->
     </participants>
   </message>
   ```

5. **Library Version**: The whatsmeow version in use (Oct 24, 2025) predated the fix committed on Oct 28, 2025

### Upstream Fix Available

**whatsmeow commit ad7a618 (October 28, 2025)**:
> "send: clear device cache for DMs on phash mismatch"

This commit directly addresses the issue by automatically clearing the device cache when participant hash mismatches are detected, forcing a fresh device query and session re-establishment.

## Resolution

### Fix Required: Update whatsmeow Library

**Current Version**: `v0.0.0-20251024191251-088fa33fb87f` (October 24, 2025)
**Target Version**: Latest (includes October 28, 2025 fix)

**Update Command**:
```bash
cd src
go get go.mau.fi/whatsmeow@latest
go mod tidy
go build -o whatsapp
```

### Expected Behavior After Fix

1. Message sent to multi-device recipient
2. Server returns participant hash mismatch
3. Library automatically clears device cache
4. Fresh device query performed
5. Prekey bundles requested for all devices (including Device 0)
6. Message encrypted and delivered to all devices

### Workaround (Until Fix Deployed)

**Manual Retry**: Sending the message a second time often succeeds because:
- First attempt updates device cache
- Second attempt has fresh device information
- Prekey bundles may be available on retry

## Prevention

### Steps Taken to Prevent Recurrence

1. **Library Update Process**:
   - Monitor whatsmeow repository for releases and important commits
   - Review commit messages for fixes related to device management and encryption
   - Establish regular update schedule (monthly or per critical fix)

2. **Monitoring and Alerting**:
   - Log participant hash mismatches
   - Alert on patterns of incomplete device delivery
   - Track Device 0 encryption failures separately

3. **Retry Logic** (optional enhancement):
   ```go
   func (service *serviceSend) wrapSendMessage(...) error {
       resp, err := whatsapp.GetClient().SendMessage(...)
       if err != nil {
           return err
       }

       // Check for participant hash mismatch
       if resp.ParticipantHash != expectedHash {
           log.Warn("Participant hash mismatch detected, retrying after delay...")
           time.Sleep(3 * time.Second)

           // Retry send
           resp, err = whatsapp.GetClient().SendMessage(...)
           if err != nil || resp.ParticipantHash != expectedHash {
               log.Error("Retry failed, some devices may not have received message")
           }
       }

       return nil
   }
   ```

4. **Testing**:
   - Test with contacts that have multiple devices
   - Verify delivery to all devices, especially Device 0
   - Test with contacts in LID migration state

## Lessons Learned

### What Went Well

1. **Quick Diagnosis**: Root cause identified through detailed log analysis
2. **Upstream Fix Available**: Issue was already fixed in whatsmeow upstream
3. **Clear Pattern**: Participant hash mismatch warning provided clear signal
4. **Documentation**: whatsmeow documentation helped understand multi-device protocol

### What Could Be Improved

1. **Proactive Updates**: Should have updated to latest whatsmeow sooner (4-day lag)
2. **Multi-Device Testing**: Need test suite that validates delivery to all devices
3. **Monitoring**: Should have detected incomplete delivery patterns earlier
4. **User Feedback**: Should surface delivery warnings to senders

### Action Items

- [x] Update whatsmeow to latest version (includes Oct 28 fix)
- [x] Deploy to staging and verify fix
- [x] Test with multiple multi-device recipients
- [x] Monitor participant hash mismatch rates
- [x] Deploy to production
- [ ] Consider implementing retry logic for additional safety
- [ ] Add integration tests for multi-device scenarios
- [ ] Document multi-device behavior for operators

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - Project architecture and development guide
- [Webhook Documentation](../webhook-payload.md) - Message delivery events
- [Troubleshooting](../reference/troubleshooting.md) - Common issues

## External References

- **whatsmeow Repository**: https://github.com/tulir/whatsmeow
- **Fix Commit**: https://github.com/tulir/whatsmeow/commit/ad7a618 - "send: clear device cache for DMs on phash mismatch"
- **Issue #62**: https://github.com/tulir/whatsmeow/issues/62 - "Some devices may not have received the message"
- **Issue #960**: https://github.com/tulir/whatsmeow/issues/960 - Session prefetch regression
- **PR #955**: https://github.com/tulir/whatsmeow/pull/955 - JID format mismatch fix
- **Original Issue**: `docs/issues/ISSUE-003-MULTIDEVICE-ENCRYPTION-FAILURE.md` (archived)

## Additional Context

### Understanding WhatsApp Multi-Device Protocol

WhatsApp's multi-device protocol uses the Signal Protocol for end-to-end encryption. Each device has its own identity and requires:

1. **Device Discovery**: Query server for recipient's devices
2. **Prekey Bundles**: Fetch cryptographic keys for each device
3. **Session Establishment**: Process bundles to create Signal sessions
4. **Message Encryption**: Encrypt separately for each device
5. **Delivery**: Send encrypted payload to all devices

**Key Insight**: If any step fails for a device, that device won't receive the message, but other devices may still receive it (partial delivery).

### LID Migration Background

WhatsApp is migrating from Phone Number (PN) based JIDs to Linked Identity Device (LID) based JIDs:
- **PN Format**: `5521998762522:0@s.whatsapp.net`
- **LID Format**: `151474956939293_1:0@lid.whatsapp.net`

This migration introduces complexity in device addressing and session management, which contributed to this issue.

---

**Postmortem Author**: Development Team
**Last Updated**: 2025-12-05
**Resolution Status**: Resolved and deployed (v7.8.0+)
**Resolution Time**: 2 days (library update + testing + deployment)

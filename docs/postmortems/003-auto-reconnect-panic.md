# Postmortem: Auto-Reconnect Goroutine Panic

**Date**: 2025-10-30
**Severity**: Critical
**Status**: Resolved
**Version Affected**: All versions prior to fix
**Version Fixed**: Same day fix (2025-10-30)

## Incident Summary

The application crashed with a nil pointer dereference panic approximately 5 minutes after startup. The panic occurred in the auto-reconnect checking goroutine when it attempted to call `Connect()` on a stale WhatsApp client instance. The issue was caused by a race condition where the global client was replaced but the background goroutine continued using an old reference with a nil Store field.

This was a critical architectural issue that caused predictable service crashes exactly 5 minutes after any client reconnection event.

## Impact

**Severity: CRITICAL**

- **Service Availability**: Complete service crash requiring restart
- **Predictability**: Occurred exactly 5 minutes after startup or reconnection
- **Recovery**: Required manual restart
- **Scope**: Affected both REST and MCP modes
- **Message Loss**: Potential loss of in-flight messages during crash

### Timeline

- **12:30:13**: Service starts, client initializes successfully
- **12:30:14**: Client authenticated and marked as available
- **12:32:32**: Messages sent and received successfully
- **12:34:21**: More messages processed successfully
- **12:35:10**: PANIC occurs (exactly 5 minutes after startup)
- **12:35:10**: Service crashes and restarts
- **2025-10-30**: Issue diagnosed and fixed same day

## Root Cause

### Technical Details

**Panic Message**:
```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x2 addr=0x0 pc=0x1029902d0]

goroutine 50 [running]:
go.mau.fi/whatsmeow.(*Client).Connect(...)
    /Users/milesibastos/go/pkg/mod/go.mau.fi/whatsmeow@.../client.go:447
github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/helpers.SetAutoReconnectChecking.func1()
    .../src/ui/rest/helpers/common.go:23 +0x40
```

**Location**: `src/ui/rest/helpers/common.go:23`

### Root Cause Analysis

**1. Stale Client Reference Problem**

The original implementation captured a specific client instance at startup:

```go
func SetAutoReconnectChecking(cli *whatsmeow.Client) {
    // Run every 5 minutes to check if the connection is still alive
    go func() {
        for {
            time.Sleep(5 * time.Minute)
            if !cli.IsConnected() {
                _ = cli.Connect()  // PANIC HERE - stale reference
            }
        }
    }()
}
```

**The Problem**:
- Goroutine captures `cli` parameter (e.g., `cli_v1`) at startup
- Global client gets replaced via `UpdateGlobalClient(cli_v2)` during reconnection
- Goroutine still holds reference to old `cli_v1`
- Old client's `Store` field set to `nil` during cleanup
- 5 minutes later: goroutine calls `cli_v1.Connect()`
- `Connect()` tries to access `cli_v1.Store.ID` → **nil pointer dereference**

**2. Race Condition Scenario**

```
T=0s:    Application starts
         SetAutoReconnectChecking(cli_v1) called
         Goroutine starts with reference to cli_v1

T=2s:    SetAutoConnectAfterBooting() triggers reconnection
         UpdateGlobalClient(cli_v2, db_v2) called
         Global cli now points to cli_v2
         Goroutine still has reference to cli_v1

T=5m:    Auto-reconnect check fires
         Goroutine calls cli_v1.IsConnected()
         If false, calls cli_v1.Connect()
         cli_v1.Store is nil (invalidated during session replacement)
         PANIC: Nil pointer dereference at cli_v1.Store.ID
```

**3. whatsmeow Connection Flow**

Per whatsmeow's connection lifecycle:

```
Disconnected → Connecting → CheckSession → Authenticate → Connected
                               ↓
                        (accesses Store.ID)
```

When `Connect()` is called on a client with `nil` Store, the CheckSession step fails catastrophically with a nil pointer panic.

## Resolution

### Fix Applied: Use Global Client Pattern

**Date**: 2025-10-30 (same day as discovery)

Changed the implementation to always use the current global client instead of capturing a specific instance:

**Modified Files**:
1. `src/ui/rest/helpers/common.go` - Auto-reconnect function
2. `src/cmd/rest.go:116` - REST mode caller
3. `src/cmd/mcp.go:33` - MCP mode caller

**Before (Broken)**:
```go
func SetAutoReconnectChecking(cli *whatsmeow.Client) {
    go func() {
        for {
            time.Sleep(5 * time.Minute)
            if !cli.IsConnected() {
                _ = cli.Connect()  // PANIC - stale client
            }
        }
    }()
}
```

**After (Fixed)**:
```go
func SetAutoReconnectChecking() {
    go func() {
        for {
            time.Sleep(5 * time.Minute)
            currentCli := whatsapp.GetClient()
            // Add nil checks to prevent panic
            if currentCli != nil && currentCli.Store != nil && !currentCli.IsConnected() {
                _ = currentCli.Connect()
            }
        }
    }()
}
```

### Benefits of This Fix

1. **No Stale References**: Always uses the current global client instance
2. **Nil-Safe**: Defensive checks prevent panic even if client or Store is nil
3. **Automatic Adaptation**: Works correctly after `UpdateGlobalClient()` calls
4. **Consistent Pattern**: Follows same approach as `GetConnectionStatus()`
5. **No API Breaking Changes**: Internal refactoring only

### Verification

```bash
cd src && go build -o whatsapp
# Build successful ✓
```

## Prevention

### Steps Taken to Prevent Recurrence

1. **Architectural Pattern**:
   - Established global client pattern as standard
   - All background goroutines should use `GetClient()` not captured instances
   - Document this pattern in architecture guide

2. **Defensive Programming**:
   - Added nil checks before accessing client and client.Store
   - Follows the safe pattern already used in `GetConnectionStatus()`
   - Consider panic recovery for all goroutines as additional safety

3. **Code Review Guidelines**:
   - Flag any goroutines that capture whatsmeow client instances
   - Require nil checks for all client access
   - Document safe client access patterns

4. **Testing Strategy**:
   - Test service running for >10 minutes
   - Test reconnection scenarios
   - Verify no panic after client updates

### Related Upstream Issues

**whatsmeow Issue #808**: "Nil dereference in (*Client).Connect"
- Same panic signature
- Race condition during reconnection
- Emphasizes need for nil checking

**whatsmeow Issue #734**: "panic: runtime error: invalid memory address"
- Nil keychain in libsignal-protocol-go
- Shows nil checking is critical in whatsmeow ecosystem

## Lessons Learned

### What Went Well

1. **Fast Diagnosis**: Panic stack trace provided clear location
2. **Predictable Timing**: 5-minute pattern made reproduction easy
3. **Quick Fix**: Root cause obvious once understood
4. **No Breaking Changes**: Internal fix, no API changes needed

### What Could Be Improved

1. **Earlier Detection**: Should have caught in testing before production
2. **Architecture Review**: Global client pattern should have been established earlier
3. **Panic Recovery**: Should have defensive panic recovery in all goroutines
4. **Documentation**: Client lifecycle and safe access patterns needed documentation

### Action Items

- [x] Fix auto-reconnect implementation
- [x] Update both REST and MCP mode callers
- [x] Build verification
- [x] Run service for >10 minutes to verify no panic
- [x] Test reconnection scenarios
- [x] Monitor for 24 hours in production
- [ ] Document global client pattern in architecture guide
- [ ] Add panic recovery to other goroutines as defensive measure

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - Project architecture and development guide
- [Architecture Guide](../developer/architecture.md) - System design patterns
- [Troubleshooting](../reference/troubleshooting.md) - Common issues

## External References

- **whatsmeow Issue #808**: https://github.com/tulir/whatsmeow/issues/808 - Nil dereference in Connect
- **whatsmeow Issue #734**: https://github.com/tulir/whatsmeow/issues/734 - Runtime panic
- **whatsmeow Issue #75**: https://github.com/tulir/whatsmeow/issues/75 - Panic that restarts service
- **Original Issue**: `docs/issues/ISSUE-004-AUTO-RECONNECT-PANIC.md` (archived)

## Additional Context

### Safe Client Access Pattern

The codebase already had a safe pattern in `GetConnectionStatus()`:

```go
func GetConnectionStatus() (isConnected bool, isLoggedIn bool, deviceID string) {
    if cli == nil {
        return false, false, ""
    }

    isConnected = cli.IsConnected()
    isLoggedIn = cli.IsLoggedIn()

    if cli.Store != nil && cli.Store.ID != nil {
        deviceID = cli.Store.ID.String()
    }

    return isConnected, isLoggedIn, deviceID
}
```

This function properly checks for `nil` before accessing `cli` and `cli.Store`. The auto-reconnect function now follows this same safe pattern.

### Future Consideration: Context-Based Cancellation

For more complex scenarios, consider implementing context-based goroutine cancellation:

```go
func SetAutoReconnectChecking(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return  // Clean shutdown
            case <-ticker.C:
                currentCli := GetClient()
                if currentCli != nil && !currentCli.IsConnected() {
                    _ = currentCli.Connect()
                }
            }
        }
    }()
}
```

This would allow clean shutdown of old goroutines when the client is replaced.

---

**Postmortem Author**: Development Team
**Last Updated**: 2025-12-05
**Resolution Time**: Same day (< 8 hours)
**Status**: Resolved and deployed (v7.8.0+)

# ISSUE-004: Panic in Auto-Reconnect Goroutine

**Status**: FIXED
**Severity**: CRITICAL
**Date**: 2025-10-30
**Component**: Auto-Reconnect Mechanism
**File**: `src/ui/rest/helpers/common.go:23`

## Summary

The application panics with "runtime error: invalid memory address or nil pointer dereference" in the auto-reconnect checking goroutine approximately 5 minutes after startup. The panic occurs when the goroutine attempts to call `cli.Connect()` on a stale client instance.

## Evidence from Logs

### Panic Stack Trace
```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x2 addr=0x0 pc=0x1029902d0]

goroutine 50 [running]:
go.mau.fi/whatsmeow.(*Client).Connect(...)
        /Users/milesibastos/go/pkg/mod/go.mau.fi/whatsmeow@v0.0.0-20251028165006-ad7a618ba42f/client.go:447
github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/helpers.SetAutoReconnectChecking.func1()
        /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src/ui/rest/helpers/common.go:23 +0x40
created by github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/helpers.SetAutoReconnectChecking in goroutine 44
        /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src/ui/rest/helpers/common.go:19 +0x5c
```

### Timeline of Events
```
12:30:13 - Service starts, client initializes successfully
12:30:14 - Client authenticated, marked as available
12:32:32 - Messages sent and received successfully
12:34:21 - More messages processed
12:35:10 - PANIC occurs (~5 minutes after startup)
```

The timing (exactly 5 minutes after startup) correlates with the auto-reconnect check interval configured in `SetAutoReconnectChecking`.

## Root Cause Analysis

### 1. Architecture Issue: Stale Client Reference

**Current Implementation** (`src/ui/rest/helpers/common.go:17-27`):
```go
func SetAutoReconnectChecking(cli *whatsmeow.Client) {
	// Run every 5 minutes to check if the connection is still alive, if not, reconnect
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			if !cli.IsConnected() {
				_ = cli.Connect()  // LINE 23 - PANIC OCCURS HERE
			}
		}
	}()
}
```

**Problem**: The goroutine captures a specific client instance at startup but never updates its reference. If the global client is replaced via `UpdateGlobalClient()`, the goroutine continues using the old instance.

### 2. Global Client Management

The application uses a global client pattern (`src/infrastructure/whatsapp/init.go:40-47`):
```go
var (
	cli    *whatsmeow.Client
	db     *sqlstore.Container
	keysDB *sqlstore.Container
	log    waLog.Logger
	// ...
)

func UpdateGlobalClient(newCli *whatsmeow.Client, newDB *sqlstore.Container) {
	cli = newCli
	db = newDB
	log.Infof("Global WhatsApp client updated successfully")
}

func GetClient() *whatsmeow.Client {
	return cli
}
```

**Problem**: When `UpdateGlobalClient()` is called (during reconnection, login, or logout), the global `cli` is replaced, but the auto-reconnect goroutine still holds a reference to the old client.

### 3. Nil Store Reference in whatsmeow

According to DeepWiki research on tulir/whatsmeow:

> A `nil` pointer dereference could occur within `cli.unlockedConnect()` if `cli.Store` is `nil` when `cli.Store.ID` is accessed. At line 477, `cli.Store.ID` is accessed to determine proxy behavior. If `cli.Store` was not initialized, this would lead to a `nil` pointer dereference.

The old client instance may have had its `Store` field set to `nil` during cleanup or session replacement, causing the panic when `Connect()` tries to access `cli.Store.ID`.

### 4. Comparison with Safe Implementation

The codebase has a safe pattern in `GetConnectionStatus()` (`init.go:170-183`):
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

This function properly checks for `nil` before accessing `cli` and `cli.Store`, but `SetAutoReconnectChecking` does not.

## GitHub Research Findings

### Related whatsmeow Issues

**Issue #808**: "Nil dereference in (*Client).Connect"
- Status: Closed as "not planned"
- Cause: Race condition during reconnection after websocket abnormal closure
- Same panic signature: `runtime error: invalid memory address or nil pointer dereference`
- Occurs in handler queue loop when objects aren't properly initialized

**Issue #734**: "panic: runtime error: invalid memory address or nil pointer dereference"
- Cause: Nil keychain in libsignal-protocol-go dependency
- Fix: PR `tulir/libsignal-protocol-go#3` - ensures `LoadSession` returns nil on error
- Relevance: Shows nil checking is critical in whatsmeow ecosystem

**Issue #75**: "Panic error that restarts the service"
- Similar panic after database locking issues
- Highlights importance of defensive programming around client state

## Technical Analysis

### Race Condition Scenario

1. **T=0s**: Application starts
   - `SetAutoReconnectChecking(cli)` called with initial client instance
   - Goroutine starts with reference to `cli_v1`

2. **T=2s**: `SetAutoConnectAfterBooting()` triggers reconnection
   - May call `UpdateGlobalClient(cli_v2, db_v2)`
   - Global `cli` now points to `cli_v2`
   - Goroutine still has reference to `cli_v1`

3. **T=5m**: Auto-reconnect check fires
   - Goroutine calls `cli_v1.IsConnected()`
   - If false, calls `cli_v1.Connect()`
   - `cli_v1.Store` is `nil` (invalidated during session replacement)
   - **PANIC**: Nil pointer dereference at `cli_v1.Store.ID`

### whatsmeow Connection Flow

Per DeepWiki documentation, the connection lifecycle is:

```
Disconnected → Connecting → CheckSession → Authenticate → Connected
                                ↓
                         (accesses Store.ID)
```

When `Connect()` is called on a client with `nil` Store, the CheckSession step fails catastrophically.

## Impact

- **Severity**: CRITICAL - Service completely crashes
- **Frequency**: Predictable - occurs 5 minutes after startup if conditions are met
- **Recovery**: None - requires manual restart
- **Scope**: Affects both REST and MCP modes
- **Message Loss**: Potential - in-flight messages may be lost during crash

## Recommended Fix Options

### Option 1: Use Global Client (Recommended)
Modify `SetAutoReconnectChecking` to always use the current global client:

```go
func SetAutoReconnectChecking() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			currentCli := GetClient()
			if currentCli != nil && !currentCli.IsConnected() {
				_ = currentCli.Connect()
			}
		}
	}()
}
```

**Pros**:
- Always uses current client instance
- Automatically adapts to client updates
- Consistent with global client pattern

**Cons**:
- Requires updating callers to not pass client parameter

### Option 2: Add Nil Checks (Quick Fix)
Add defensive nil checking before calling `Connect()`:

```go
func SetAutoReconnectChecking(cli *whatsmeow.Client) {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			if cli != nil && cli.Store != nil && !cli.IsConnected() {
				_ = cli.Connect()
			}
		}
	}()
}
```

**Pros**:
- Minimal code change
- No API changes

**Cons**:
- Still uses potentially stale client
- Doesn't solve underlying architecture issue

### Option 3: Channel-Based Cancellation
Use context cancellation to stop old goroutines when client is replaced:

```go
func SetAutoReconnectChecking(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
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

**Pros**:
- Clean shutdown of old goroutines
- Prevents goroutine leaks
- Uses current client

**Cons**:
- More complex implementation
- Requires context management

## Testing Strategy

### Reproduction Steps
1. Start service in REST mode
2. Trigger reconnection via `/app/reconnect` endpoint within first 5 minutes
3. Wait for 5-minute auto-reconnect check to fire
4. Observe panic

### Validation Tests
After fix implementation:
1. Run service for >10 minutes without activity
2. Trigger manual reconnection
3. Wait for next auto-reconnect check
4. Verify no panic occurs
5. Check logs show reconnection using updated client

## Related Files

- `src/ui/rest/helpers/common.go:17-27` - Auto-reconnect implementation
- `src/infrastructure/whatsapp/init.go:40-47` - Global client variables
- `src/infrastructure/whatsapp/init.go:150-157` - UpdateGlobalClient function
- `src/infrastructure/whatsapp/init.go:169-183` - GetConnectionStatus (safe pattern)
- `src/cmd/rest.go:116` - SetAutoReconnectChecking caller
- `src/cmd/mcp.go:33` - SetAutoReconnectChecking caller

## References

- whatsmeow Issue #808: https://github.com/tulir/whatsmeow/issues/808
- whatsmeow Issue #734: https://github.com/tulir/whatsmeow/issues/734
- whatsmeow client.go line 447: Connection entry point
- whatsmeow client.go line 477: Store.ID access point
- DeepWiki whatsmeow architecture: https://deepwiki.com/tulir/whatsmeow

## Fix Implementation

**Date Implemented**: 2025-10-30

### Changes Made

**Option 1 (Recommended)** was implemented - Always use the current global client.

#### Modified Files

1. **`src/ui/rest/helpers/common.go`**:
   - Removed `cli *whatsmeow.Client` parameter from `SetAutoReconnectChecking()`
   - Added import: `"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"`
   - Changed to use `whatsapp.GetClient()` to get current global client
   - Added nil checks: `currentCli != nil && currentCli.Store != nil`
   - Updated comments to explain the fix

2. **`src/cmd/rest.go:116`**:
   - Changed call from `helpers.SetAutoReconnectChecking(whatsappCli)` to `helpers.SetAutoReconnectChecking()`
   - Updated comment to clarify it uses global client

3. **`src/cmd/mcp.go:33`**:
   - Changed call from `helpers.SetAutoReconnectChecking(whatsappCli)` to `helpers.SetAutoReconnectChecking()`
   - Updated comment to clarify it uses global client

### Code Changes

```go
// Before
func SetAutoReconnectChecking(cli *whatsmeow.Client) {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			if !cli.IsConnected() {
				_ = cli.Connect()  // PANIC HERE - stale client reference
			}
		}
	}()
}

// After
func SetAutoReconnectChecking() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			currentCli := whatsapp.GetClient()
			// Add nil checks to prevent panic if client or Store is nil
			if currentCli != nil && currentCli.Store != nil && !currentCli.IsConnected() {
				_ = currentCli.Connect()
			}
		}
	}()
}
```

### Benefits of This Fix

✅ **No stale references**: Always uses the current global client instance
✅ **Nil-safe**: Defensive checks prevent panic even if client is nil
✅ **Automatic adaptation**: Works correctly after `UpdateGlobalClient()` calls
✅ **Consistent pattern**: Follows the same approach as `GetConnectionStatus()`
✅ **No API breaking changes**: Internal refactoring only

### Build Verification

```bash
cd src && go build -o whatsapp
# Build successful ✓
```

## Next Steps

1. ✅ Diagnose root cause
2. ✅ Implement Option 1 (use global client)
3. ✅ Update both REST and MCP mode callers
4. ✅ Build verification
5. ⏳ Run service for >10 minutes to verify no panic
6. ⏳ Test reconnection scenarios
7. ⏳ Monitor for 24 hours in production

---

**Diagnosis completed**: 2025-10-30
**Fix implemented**: 2025-10-30
**Analysis tools used**: DeepWiki, GitHub Issues Search, Code Review
**Confidence level**: HIGH - Clear race condition with reproducible timing
**Fix approach**: Global client pattern with nil safety checks

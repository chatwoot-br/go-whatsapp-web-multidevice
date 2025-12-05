# Fix Code Review Issues from Commit f06a56c

**Status**: COMPLETED (v7.10.1)

## Summary

Address critical, high, and medium severity issues found in the code review of commit f06a56c (feat: chatwoot-br custom). This commit introduced the Admin API and audio processing improvements.

## Issues Fixed

| Priority | Issue | File | Status |
|----------|-------|------|--------|
| CRITICAL | Missing timeouts on FFmpeg/FFprobe calls | `src/usecase/send.go` | FIXED |
| CRITICAL | Lock acquisition lacks context/timeout | `src/internal/admin/conf.go` | FIXED |
| HIGH | Temp file leak in getAudioDuration | `src/usecase/send.go` | FIXED |
| MEDIUM | Weak default credentials warning | `src/internal/admin/security.go` | FIXED |
| MEDIUM | Cleanup goroutine improvements | `src/pkg/utils/general.go` | FIXED |
| MEDIUM | processAudioForWhatsApp missing context | `src/usecase/send.go` | FIXED |

## Implementation Details

### Phase 1: Audio Processing Timeouts (CRITICAL) - COMPLETED

**Files:** `src/usecase/send.go`

#### 1.1 Add context to processAudioForWhatsApp - DONE

Function signature updated to accept context:
```go
func (service serviceSend) processAudioForWhatsApp(ctx context.Context, audioBytes []byte, originalMimeType string) (...)
```

FFmpeg call now uses `exec.CommandContext` with 45-second timeout (line 1194).

#### 1.2 Add context to getAudioDuration - DONE

Function signature updated to accept context:
```go
func (service serviceSend) getAudioDuration(ctx context.Context, audioBytes []byte, mimeType string) uint32
```

FFprobe call now uses `exec.CommandContext` with 15-second timeout (line 1269).

#### 1.3 Add timeouts to video processing - DONE

Video processing now uses context-aware execution.

#### 1.4 Update callers in SendAudio - DONE

Callers updated to pass context (lines 791 and 814).

### Phase 2: Fix Temp File Leak (HIGH) - COMPLETED

**File:** `src/usecase/send.go` (lines 1249-1262)

Fixed by registering cleanup before file write:
```go
tempPath := fmt.Sprintf("%s/%s_duration_check", config.PathSendItems, generateUUID)

// Register cleanup first to handle partial writes and ensure cleanup even on early returns
defer func() {
    if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
        logrus.Warnf("Failed to cleanup temp file %s: %v", tempPath, err)
    }
}()

err = os.WriteFile(tempPath, audioBytes, 0644)
if err != nil {
    logrus.Warnf("Failed to save audio for duration check: %v", err)
    return 10
}
```

### Phase 3: Admin Lock Timeout (CRITICAL) - COMPLETED

**File:** `src/internal/admin/conf.go`

#### 3.1 Add AcquireLockWithContext method - DONE

Added `AcquireLockWithContext` method (lines 218-250) with:
- 100ms retry interval
- Non-blocking lock attempts with `LOCK_NB` flag
- Context-based timeout/cancellation support
- `DefaultLockTimeout` constant set to 30 seconds

#### 3.2 Update lifecycle.go to use context - DONE

**File:** `src/internal/admin/lifecycle.go`

Interface `ILifecycleManager` updated with context parameters:
- `CreateInstance(ctx context.Context, port int)`
- `CreateInstanceWithConfig(ctx context.Context, port int, customConfig *InstanceConfig)`
- `UpdateInstanceConfig(ctx context.Context, port int, customConfig *InstanceConfig)`
- `DeleteInstance(ctx context.Context, port int)`
- `ListInstances(ctx context.Context)` (read-only, no locks)
- `GetInstance(ctx context.Context, port int)` (read-only, no locks)

`waitForInstanceState` now uses context-based timeout.

#### 3.3 Update api.go handlers - DONE

**File:** `src/internal/admin/api.go`

Handlers pass `c.UserContext()` from Fiber to lifecycle methods.

### Phase 4: Security Warnings (MEDIUM) - COMPLETED

**File:** `src/internal/admin/security.go`

Created security validation module with:
- `SecurityConfig` struct for security-related configuration
- `SecurityWarning` struct with Level, Code, and Message fields
- `ValidateAndWarn()` function that checks for weak credentials and returns warnings
- `LogDefaultCredentialWarnings()` function for startup logging
- `isWeakCredential()` helper to detect common weak passwords
- `hasRequiredComplexity()` helper for password strength validation
- `isProductionMode()` helper to detect production environment

Security checks implemented:
- CRITICAL warning for weak `GOWA_BASIC_AUTH` (e.g., `admin:admin`)
- HIGH warning for weak `GOWA_WEBHOOK_SECRET` (less than 16 chars or `secret`)
- Password complexity validation (min 8 chars, mixed character types)

### Phase 5: Cleanup Goroutine Improvements (MEDIUM) - COMPLETED

**File:** `src/pkg/utils/general.go`

Added `RemoveFileWithTimeout` function (lines 44-72):
```go
// RemoveFileWithTimeout removes files after a delay with a maximum timeout.
// It logs failures instead of returning errors, suitable for cleanup goroutines.
// The timeout prevents goroutine leaks if file removal hangs.
func RemoveFileWithTimeout(delaySecond int, timeout time.Duration, paths ...string)
```

Features:
- Optional delay before cleanup
- Maximum timeout to prevent goroutine leaks
- Logs warnings for cleanup failures
- Ignores `os.ErrNotExist` for already-deleted files

### Phase 6: Documentation - COMPLETED

**File:** `docs/developer/adr/0001-admin-api.md`

ADR updated with:
- Lock timeout behavior (30 seconds default) documented in "Idempotency & concurrency" section
- Eventual consistency for ListInstances/GetInstance documented
- Security recommendations for production credentials added
- Implementation status section added confirming v7.10.0+ completion

---

## Files Modified

| File | Changes | Status |
|------|---------|--------|
| `src/usecase/send.go` | Add context to audio funcs, timeouts, fix temp file leak | DONE |
| `src/internal/admin/conf.go` | Add AcquireLockWithContext, DefaultLockTimeout | DONE |
| `src/internal/admin/lifecycle.go` | Update interface with context params | DONE |
| `src/internal/admin/api.go` | Pass context from handlers | DONE |
| `src/internal/admin/security.go` | NEW: Security validation module | DONE |
| `src/pkg/utils/general.go` | Add RemoveFileWithTimeout | DONE |
| `src/internal/admin/mocks_test.go` | Update mock interface | DONE |
| `src/internal/admin/security_test.go` | NEW: Security validation tests | DONE |
| `src/internal/admin/lifecycle_test.go` | NEW: Lifecycle tests with context | DONE |
| `docs/developer/adr/0001-admin-api.md` | Documentation updates | DONE |

## Testing Completed

1. Updated existing tests for new function signatures
2. Added timeout tests for lock acquisition (`conf_test.go`)
3. Added context cancellation tests for audio processing (`send_audio_test.go`)
4. Added security validation tests (`security_test.go`)
5. All tests pass with `go test -race ./...`

## Completion Summary

All issues from the code review have been addressed in commit f69a769:
- **CRITICAL issues**: Audio processing timeouts and lock acquisition timeouts implemented
- **HIGH issues**: Temp file leak in `getAudioDuration` fixed
- **MEDIUM issues**: Security warnings, cleanup improvements completed

The fixes are included in version v7.10.1.

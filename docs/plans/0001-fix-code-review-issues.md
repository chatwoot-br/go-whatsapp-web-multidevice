# Fix Code Review Issues from Commit f06a56c

## Summary

Address critical, high, and medium severity issues found in the code review of commit f06a56c (feat: chatwoot-br custom). This commit introduced the Admin API and audio processing improvements.

## Issues to Fix

| Priority | Issue | File |
|----------|-------|------|
| CRITICAL | Missing timeouts on FFmpeg/FFprobe calls | `src/usecase/send.go` |
| CRITICAL | Lock acquisition lacks context/timeout | `src/internal/admin/conf.go` |
| HIGH | Temp file leak in getAudioDuration | `src/usecase/send.go:1227-1233` |
| MEDIUM | Weak default credentials warning | `src/internal/admin/conf.go:43` |
| MEDIUM | Cleanup goroutine improvements | `src/usecase/send.go:786-795` |
| MEDIUM | processAudioForWhatsApp missing context | `src/usecase/send.go:1143` |

## Implementation Plan

### Phase 1: Audio Processing Timeouts (CRITICAL)

**Files:** `src/usecase/send.go`

#### 1.1 Add context to processAudioForWhatsApp

Change signature from:
```go
func (service serviceSend) processAudioForWhatsApp(audioBytes []byte, originalMimeType string) (...)
```
To:
```go
func (service serviceSend) processAudioForWhatsApp(ctx context.Context, audioBytes []byte, originalMimeType string) (...)
```

Update FFmpeg call at line 1178 to use `exec.CommandContext` with 45-second timeout.

#### 1.2 Add context to getAudioDuration

Change signature from:
```go
func (service serviceSend) getAudioDuration(audioBytes []byte, mimeType string) uint32
```
To:
```go
func (service serviceSend) getAudioDuration(ctx context.Context, audioBytes []byte, mimeType string) uint32
```

Update FFprobe call at line 1236 to use `exec.CommandContext` with 15-second timeout.

#### 1.3 Add timeouts to video processing

- Line 470: Video thumbnail - add 30-second timeout
- Line 502: Video compression - add 120-second timeout

#### 1.4 Update callers in SendAudio

Update calls at lines ~780 and ~808 to pass context.

### Phase 2: Fix Temp File Leak (HIGH)

**File:** `src/usecase/send.go:1227-1233`

Current pattern has defer after potential early return:
```go
tempPath := fmt.Sprintf(...)
err = os.WriteFile(tempPath, audioBytes, 0644)
if err != nil {
    return 10  // RETURNS BEFORE DEFER
}
defer os.Remove(tempPath)
```

Fix by registering cleanup first:
```go
tempPath := fmt.Sprintf(...)
defer func() {
    if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
        logrus.Warnf("Failed to cleanup temp file %s: %v", tempPath, err)
    }
}()
err = os.WriteFile(tempPath, audioBytes, 0644)
if err != nil {
    return 10
}
```

### Phase 3: Admin Lock Timeout (CRITICAL)

**File:** `src/internal/admin/conf.go`

#### 3.1 Add AcquireLockWithContext method

Add after line 242:
```go
// AcquireLockWithContext acquires a lock with context timeout support
func (lm *LockManager) AcquireLockWithContext(ctx context.Context, port int) (*os.File, error) {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            return nil, fmt.Errorf("failed to open lock file: %w", err)
        }

        if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
            return lockFile, nil
        } else if err != syscall.EWOULDBLOCK {
            lockFile.Close()
            return nil, fmt.Errorf("failed to acquire lock: %w", err)
        }
        lockFile.Close()

        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("lock acquisition cancelled: %w", ctx.Err())
        case <-ticker.C:
            // Retry
        }
    }
}

const DefaultLockTimeout = 30 * time.Second
```

#### 3.2 Update lifecycle.go to use context

**File:** `src/internal/admin/lifecycle.go`

Update interface and implementations:
- `CreateInstance(ctx context.Context, port int)`
- `CreateInstanceWithConfig(ctx context.Context, port int, customConfig *InstanceConfig)`
- `UpdateInstanceConfig(ctx context.Context, port int, customConfig *InstanceConfig)`
- `DeleteInstance(ctx context.Context, port int)`
- `ListInstances(ctx context.Context)` (no lock change, just context passthrough)
- `GetInstance(ctx context.Context, port int)` (no lock change, just context passthrough)

Update `waitForInstanceState` to use context instead of manual deadline polling.

#### 3.3 Update api.go handlers

**File:** `src/internal/admin/api.go`

Pass `c.UserContext()` from Fiber handlers to lifecycle methods.

### Phase 4: Security Warnings (MEDIUM)

**File:** `src/internal/admin/security.go` (NEW)

Create a security validation module:
```go
package admin

// ValidateAndWarn checks for weak credentials and logs warnings
func ValidateAndWarn(config *SecurityConfig) []SecurityWarning
```

**File:** `src/cmd/admin.go`

Add security validation call at startup after `DefaultInstanceConfig()`:
- Log warning if `GOWA_BASIC_AUTH` uses default `admin:admin`
- Log warning if `GOWA_WEBHOOK_SECRET` uses default `secret`

### Phase 5: Cleanup Goroutine Improvements (MEDIUM)

**File:** `src/pkg/utils/general.go`

Add timeout-aware cleanup function:
```go
func RemoveFileWithTimeout(delaySecond int, timeout time.Duration, paths ...string) error
```

**File:** `src/usecase/send.go`

Update cleanup patterns at lines 319-324, 427-432, and 786-795:
- Use consistent 1-second delay
- Add 10-second timeout
- Log cleanup failures

### Phase 6: Documentation

**File:** `docs/developer/adr/0001-admin-api.md`

Add section documenting:
- Lock timeout behavior (30 seconds default)
- Eventual consistency for ListInstances/GetInstance (no read locks)
- Security recommendations for production credentials

---

## Critical Files to Modify

| File | Changes |
|------|---------|
| `src/usecase/send.go` | Add context to audio funcs, timeouts, fix temp file leak |
| `src/internal/admin/conf.go` | Add AcquireLockWithContext |
| `src/internal/admin/lifecycle.go` | Update interface with context params |
| `src/internal/admin/api.go` | Pass context from handlers |
| `src/internal/admin/security.go` | NEW: Security validation |
| `src/cmd/admin.go` | Add security check at startup |
| `src/pkg/utils/general.go` | Add RemoveFileWithTimeout |
| `src/internal/admin/mocks_test.go` | Update mock interface |

## Testing Requirements

1. Update existing tests for new function signatures
2. Add timeout tests for lock acquisition
3. Add context cancellation tests for audio processing
4. Add security validation tests
5. Run `go test -race ./...` to verify no race conditions

## Implementation Order

1. **Audio timeouts** (most critical, prevents service hangs)
2. **Temp file leak fix** (simple, high impact)
3. **Lock timeout** (prevents admin API hangs)
4. **Security warnings** (important for production)
5. **Cleanup improvements** (polish)
6. **Documentation** (final step)

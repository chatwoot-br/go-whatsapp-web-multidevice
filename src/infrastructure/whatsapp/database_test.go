package whatsapp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

// scriptedOpener is a fake dbOpener: call N returns errs[N] (nil = success);
// once the script is exhausted it returns success. It records the call count so
// tests can assert the retry loop made exactly the expected number of attempts.
type scriptedOpener struct {
	errs  []error
	calls int
}

func (s *scriptedOpener) open(_ context.Context) (*sqlstore.Container, error) {
	i := s.calls
	s.calls++
	if i < len(s.errs) {
		return nil, s.errs[i]
	}
	return nil, nil // past the script => success
}

func TestOpenDBWithRetry(t *testing.T) {
	// A representative transient transport error — the issue's reported case.
	boom := errors.New("dial tcp 10.0.0.1:5432: connect: connection refused")

	t.Run("succeeds on the first attempt", func(t *testing.T) {
		s := &scriptedOpener{}
		if _, err := openDBWithRetry(context.Background(), nil, s.open, 3, 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.calls != 1 {
			t.Errorf("calls = %d, want 1 (no retry on success)", s.calls)
		}
	})

	t.Run("retries a transient error then succeeds", func(t *testing.T) {
		s := &scriptedOpener{errs: []error{boom, boom}} // fail twice, then succeed
		if _, err := openDBWithRetry(context.Background(), nil, s.open, 5, 0); err != nil {
			t.Fatalf("unexpected error after retries: %v", err)
		}
		if s.calls != 3 {
			t.Errorf("calls = %d, want 3", s.calls)
		}
	})

	t.Run("returns wrapped last error after exhausting attempts", func(t *testing.T) {
		s := &scriptedOpener{errs: []error{boom, boom, boom}}
		_, err := openDBWithRetry(context.Background(), nil, s.open, 3, 0)
		if err == nil {
			t.Fatal("expected an error after exhausting attempts")
		}
		if s.calls != 3 {
			t.Errorf("calls = %d, want 3 (must try exactly attempts times)", s.calls)
		}
		if !errors.Is(err, boom) {
			t.Errorf("error should wrap the last attempt error, got %v", err)
		}
		if !strings.Contains(err.Error(), "3 attempt") {
			t.Errorf("error should report the attempt count, got %v", err)
		}
	})

	t.Run("stops immediately when ctx is already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // budget gone before the first backoff
		s := &scriptedOpener{errs: []error{boom, boom, boom}}
		// A 30s backoff means this test would hang for ~30s if the ctx.Done()
		// select arm didn't fire — proving the cancellation path is exercised.
		_, err := openDBWithRetry(ctx, nil, s.open, 3, 30*time.Second)
		if err == nil {
			t.Fatal("expected a cancellation error")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error should wrap context.Canceled, got %v", err)
		}
		if s.calls != 1 {
			t.Errorf("calls = %d, want 1 (must not keep retrying after cancel)", s.calls)
		}
	})

	t.Run("clamps attempts below 1 to a single try", func(t *testing.T) {
		s := &scriptedOpener{errs: []error{boom}}
		_, err := openDBWithRetry(context.Background(), nil, s.open, 0, 0)
		if err == nil {
			t.Fatal("expected an error")
		}
		if s.calls != 1 {
			t.Errorf("calls = %d, want 1", s.calls)
		}
	})
}

func TestInitDatabaseWithRetry_UnknownTypeFailsFast(t *testing.T) {
	// A permanent misconfiguration (unsupported URI) must fail immediately,
	// without entering the retry loop. The 1h backoff makes that observable: if
	// the unknown-type case wrongly retried, this test would hang far past the
	// test timeout instead of returning at once.
	_, err := initDatabaseWithRetry(context.Background(), nil, "mysql://user:pass@host:3306/db", dbInitMaxAttempts, time.Hour)
	if err == nil {
		t.Fatal("expected an error for an unsupported database type")
	}
	if !strings.Contains(err.Error(), "unknown database type") {
		t.Errorf("got %v, want an 'unknown database type' error", err)
	}
}

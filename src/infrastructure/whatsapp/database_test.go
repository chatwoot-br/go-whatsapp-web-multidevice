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
		if _, err := openDBWithRetry(context.Background(), nil, s.open, 3, 0, 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.calls != 1 {
			t.Errorf("calls = %d, want 1 (no retry on success)", s.calls)
		}
	})

	t.Run("retries a transient error then succeeds", func(t *testing.T) {
		s := &scriptedOpener{errs: []error{boom, boom}} // fail twice, then succeed
		if _, err := openDBWithRetry(context.Background(), nil, s.open, 5, 0, 0); err != nil {
			t.Fatalf("unexpected error after retries: %v", err)
		}
		if s.calls != 3 {
			t.Errorf("calls = %d, want 3", s.calls)
		}
	})

	t.Run("returns wrapped last error after exhausting attempts", func(t *testing.T) {
		s := &scriptedOpener{errs: []error{boom, boom, boom}}
		_, err := openDBWithRetry(context.Background(), nil, s.open, 3, 0, 0)
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
		_, err := openDBWithRetry(ctx, nil, s.open, 3, 30*time.Second, 0)
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

	t.Run("bounds each attempt with attemptTimeout", func(t *testing.T) {
		// An opener that hangs until its context is cancelled — it can only return
		// because of the per-attempt deadline. If openDBWithRetry didn't derive a
		// per-attempt timeout, the attempt ctx would have no deadline and this test
		// would hang forever (caught by the test timeout).
		var calls int
		sawDeadline := true
		open := func(ctx context.Context) (*sqlstore.Container, error) {
			calls++
			if _, ok := ctx.Deadline(); !ok {
				sawDeadline = false
			}
			<-ctx.Done()
			return nil, ctx.Err()
		}
		_, err := openDBWithRetry(context.Background(), nil, open, 2, 0, 50*time.Millisecond)
		if err == nil {
			t.Fatal("expected an error from the timed-out attempts")
		}
		if !sawDeadline {
			t.Error("each attempt context should carry a deadline")
		}
		if calls != 2 {
			t.Errorf("calls = %d, want 2", calls)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("error should wrap context.DeadlineExceeded, got %v", err)
		}
	})

	t.Run("clamps attempts below 1 to a single try", func(t *testing.T) {
		s := &scriptedOpener{errs: []error{boom}}
		_, err := openDBWithRetry(context.Background(), nil, s.open, 0, 0, 0)
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
	// test timeout instead of returning at once. The URI carries credentials,
	// which must NOT appear in the error.
	_, err := initDatabaseWithRetry(context.Background(), nil, "mysql://user:s3cr3t@host:3306/db", dbInitMaxAttempts, time.Hour)
	if err == nil {
		t.Fatal("expected an error for an unsupported database type")
	}
	if !strings.Contains(err.Error(), "unknown database type") {
		t.Errorf("got %v, want an 'unknown database type' error", err)
	}
	if strings.Contains(err.Error(), "s3cr3t") {
		t.Errorf("error must not leak DB credentials, got %v", err)
	}
}

func TestRedactDBURI(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		wantContns string // substring that must be present
		wantAbsent string // substring that must NOT be present ("" = skip)
	}{
		{name: "redacts password", in: "mysql://user:s3cr3t@host:3306/db", wantContns: "xxxxx", wantAbsent: "s3cr3t"},
		{name: "no credentials passes through", in: "redis://host:6379", wantContns: "redis://host:6379"},
		{name: "non-url passes through", in: "not a uri", wantContns: "not a uri"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactDBURI(tc.in)
			if !strings.Contains(got, tc.wantContns) {
				t.Errorf("redactDBURI(%q) = %q, want to contain %q", tc.in, got, tc.wantContns)
			}
			if tc.wantAbsent != "" && strings.Contains(got, tc.wantAbsent) {
				t.Errorf("redactDBURI(%q) = %q, must not contain %q", tc.in, got, tc.wantAbsent)
			}
		})
	}
}

package whatsapp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

const (
	// dbInitMaxAttempts bounds how many times the WhatsApp store connection is
	// tried before giving up. A transient transport error (e.g. "connection
	// refused" while the DB pod restarts / fails over) must not crash-loop the
	// process at startup.
	dbInitMaxAttempts = 6
	// dbInitRetryBackoff is the fixed delay between attempts. Total time before
	// giving up is ~40s in the common fast-fail case (connection refused returns
	// immediately: 5 backoffs × 8s), bounded at ~100s worst case if every dial
	// black-holes to the per-attempt timeout (6 × 10s + 5 × 8s).
	dbInitRetryBackoff = 8 * time.Second
	// dbInitAttemptTimeout bounds a single connect attempt so a host that
	// black-holes the dial can't hang startup indefinitely (the caller passes a
	// context without a deadline).
	dbInitAttemptTimeout = 10 * time.Second
)

// dbOpener opens a store container; injected so the retry logic can be tested
// without a live database.
type dbOpener func(ctx context.Context) (*sqlstore.Container, error)

// InitWaDB initializes the WhatsApp store database connection, retrying transient
// connection failures with bounded, context-aware backoff. It returns an error
// (rather than panicking) so the caller can exit cleanly; a permanent
// misconfiguration — an unknown/unsupported DB type — fails fast without
// retrying.
func InitWaDB(ctx context.Context, DBURI string) (*sqlstore.Container, error) {
	log = waLog.Stdout("Main", config.WhatsappLogLevel, true)
	dbLog := waLog.Stdout("Database", config.WhatsappLogLevel, true)

	return initDatabaseWithRetry(ctx, dbLog, DBURI, dbInitMaxAttempts, dbInitRetryBackoff)
}

// initDatabaseWithRetry resolves the driver from DBURI — failing fast on an
// unsupported type, which retrying cannot fix — then opens the store with
// bounded, context-aware retries.
func initDatabaseWithRetry(ctx context.Context, dbLog waLog.Logger, DBURI string, attempts int, backoff time.Duration) (*sqlstore.Container, error) {
	// Strip surrounding quotes that may come from .env file parsing.
	DBURI = strings.Trim(DBURI, `"'`)

	var driver string
	switch {
	case strings.HasPrefix(DBURI, "file:"):
		driver = "sqlite3"
	case strings.HasPrefix(DBURI, "postgres:"):
		driver = "postgres"
	default:
		// Permanent: no amount of retrying fixes an unsupported URI. Redact any
		// credentials so they don't end up in logs via the fatal error.
		return nil, fmt.Errorf("unknown database type: %s. Currently only sqlite3(file:) and postgres are supported", redactDBURI(DBURI))
	}

	open := func(attemptCtx context.Context) (*sqlstore.Container, error) {
		return sqlstore.New(attemptCtx, driver, DBURI, dbLog)
	}

	return openDBWithRetry(ctx, dbLog, open, attempts, backoff, dbInitAttemptTimeout)
}

// openDBWithRetry calls open up to attempts times, sleeping backoff between tries
// and honoring ctx cancellation. Each attempt is given its own deadline derived
// from ctx (attemptTimeout > 0) so a single hung dial can't block startup. It
// returns the first success, or — once attempts are exhausted (or ctx is
// cancelled) — the last error wrapped with context.
func openDBWithRetry(ctx context.Context, dbLog waLog.Logger, open dbOpener, attempts int, backoff, attemptTimeout time.Duration) (*sqlstore.Container, error) {
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		attemptCtx := ctx
		var cancel context.CancelFunc
		if attemptTimeout > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, attemptTimeout)
		}
		container, err := open(attemptCtx)
		if cancel != nil {
			cancel()
		}
		if err == nil {
			return container, nil
		}
		lastErr = err

		if attempt == attempts {
			break
		}
		if dbLog != nil {
			dbLog.Warnf("Database init attempt %d/%d failed, retrying in %s: %v", attempt, attempts, backoff, err)
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, fmt.Errorf("database initialization cancelled after %d attempt(s): %w", attempt, ctx.Err())
		}
	}

	return nil, fmt.Errorf("database initialization failed after %d attempt(s): %w", attempts, lastErr)
}

// redactDBURI hides any credentials embedded in a database URI so they aren't
// leaked through error messages / logs. Non-URL or credential-free values are
// returned unchanged.
func redactDBURI(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.User != nil {
		return u.Redacted()
	}
	return uri
}

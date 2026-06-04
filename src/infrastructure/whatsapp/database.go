package whatsapp

import (
	"context"
	"fmt"
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
	// dbInitRetryBackoff is the fixed delay between attempts; with the attempt
	// count above this gives the ~30-60s retry window requested in the issue.
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
		// Permanent: no amount of retrying fixes an unsupported URI.
		return nil, fmt.Errorf("unknown database type: %s. Currently only sqlite3(file:) and postgres are supported", DBURI)
	}

	open := func(attemptCtx context.Context) (*sqlstore.Container, error) {
		// Bound a single attempt so a black-holed dial can't hang startup.
		attemptCtx, cancel := context.WithTimeout(attemptCtx, dbInitAttemptTimeout)
		defer cancel()
		return sqlstore.New(attemptCtx, driver, DBURI, dbLog)
	}

	return openDBWithRetry(ctx, dbLog, open, attempts, backoff)
}

// openDBWithRetry calls open up to attempts times, sleeping backoff between tries
// and honoring ctx cancellation. It returns the first success, or — once
// attempts are exhausted (or ctx is cancelled) — the last error wrapped with
// context.
func openDBWithRetry(ctx context.Context, dbLog waLog.Logger, open dbOpener, attempts int, backoff time.Duration) (*sqlstore.Container, error) {
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		container, err := open(ctx)
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

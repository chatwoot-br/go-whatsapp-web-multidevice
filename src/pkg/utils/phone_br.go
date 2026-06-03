package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
)

// normalizePhoneBR strips Brazil's "ninth digit" from a mobile phone number when
// it follows the 13-digit pattern `55<area:2><9><subscriber:8>`. Returns the
// 12-digit canonical form `55<area:2><subscriber:8>`. Non-BR numbers, already
// 12-digit numbers, and any other shape pass through unchanged.
//
// This is the deterministic string-level fallback. The authoritative
// normalization is WhatsApp's own IsOnWhatsApp response — see
// ValidateAndNormalizeJID.
//
// Accepts inputs with or without a leading "+"; preserves the prefix on return.
func normalizePhoneBR(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return phone
	}

	hasPlus := strings.HasPrefix(phone, "+")
	digits := phone
	if hasPlus {
		digits = phone[1:]
	}

	// 13-digit BR mobile: 55 + 2-digit area + 9 + 8-digit subscriber.
	if len(digits) == 13 && strings.HasPrefix(digits, "55") && digits[4] == '9' {
		stripped := digits[:4] + digits[5:]
		if hasPlus {
			return "+" + stripped
		}
		return stripped
	}

	return phone
}

// onWhatsAppProber is the minimal whatsmeow surface needed to probe whether a
// phone is registered on WhatsApp. *whatsmeow.Client satisfies it; tests inject
// a fake so the retry/classification logic can be exercised without a session.
type onWhatsAppProber interface {
	IsOnWhatsApp(ctx context.Context, phones []string) ([]types.IsOnWhatsAppResponse, error)
}

// probeOutcome is the classification of an IsOnWhatsApp probe.
type probeOutcome int

const (
	// probeAmbiguous means the probe was inconclusive — a transport error or an
	// empty USync response. This is NOT proof the number is unregistered:
	// WhatsApp's USync is non-deterministic (throttling, post-pairing app-state
	// sync gaps), so an empty/errored probe routinely happens for valid numbers.
	probeAmbiguous probeOutcome = iota
	// probePositive means WhatsApp confirmed the number is registered (IsIn).
	probePositive
	// probeNegative means WhatsApp returned a non-empty result that explicitly
	// reports the number as not registered (IsIn == false) — authoritative.
	probeNegative
)

const (
	// onWhatsAppProbeAttempts bounds how many times an inconclusive probe is
	// retried before giving up and classifying the result as ambiguous.
	onWhatsAppProbeAttempts = 3
	// onWhatsAppProbeErrLimit caps the number of *error* (typically slow,
	// timed-out) attempts so a send isn't blocked for the full attempt budget
	// when the transport is unhealthy. Empty responses return fast and are
	// retried freely up to onWhatsAppProbeAttempts.
	onWhatsAppProbeErrLimit = 2
	// onWhatsAppProbeTimeout is the per-attempt context timeout.
	onWhatsAppProbeTimeout = 8 * time.Second
	// onWhatsAppTotalTimeout caps the *total* wall-clock across all probe
	// attempts. The per-attempt timeout and the errCount cap only bound the
	// error path; a slow-but-not-errored empty USync response is retried freely,
	// so without an overall budget a probe could block a send for roughly
	// onWhatsAppProbeAttempts × onWhatsAppProbeTimeout. Derived from the caller's
	// context so it also honors upstream cancellation.
	onWhatsAppTotalTimeout = 15 * time.Second
	// onWhatsAppRetryBackoff is the delay between retry attempts.
	onWhatsAppRetryBackoff = 750 * time.Millisecond
)

// probeOnWhatsApp runs IsOnWhatsApp for a single phone with bounded retries and
// classifies the result. A single USync probe is unreliable, so an inconclusive
// answer (transport error or empty response) is retried before being treated as
// ambiguous — never as a confirmed negative. The whole attempt loop is bounded
// by one deadline derived from ctx (onWhatsAppTotalTimeout), so even a slow,
// non-errored empty response can't keep a send blocked. backoff is the sleep
// between attempts (tests pass 0). On probePositive the returned JID is
// WhatsApp's canonical JID (may be empty if WhatsApp omitted it).
func probeOnWhatsApp(ctx context.Context, prober onWhatsAppProber, phone string, backoff time.Duration) (types.JID, probeOutcome) {
	deadlineCtx, cancel := context.WithTimeout(ctx, onWhatsAppTotalTimeout)
	defer cancel()

	var errCount int
	for attempt := 0; attempt < onWhatsAppProbeAttempts; attempt++ {
		if attempt > 0 {
			if backoff > 0 {
				select {
				case <-time.After(backoff):
				case <-deadlineCtx.Done():
					return types.JID{}, probeAmbiguous
				}
			}
			if deadlineCtx.Err() != nil {
				break // total budget exhausted
			}
		}

		attemptCtx, attemptCancel := context.WithTimeout(deadlineCtx, onWhatsAppProbeTimeout)
		data, err := prober.IsOnWhatsApp(attemptCtx, []string{phone})
		attemptCancel()

		if err != nil {
			errCount++
			logrus.Warnf("IsOnWhatsApp probe failed for %s (attempt %d/%d): %v", phone, attempt+1, onWhatsAppProbeAttempts, err)
			if errCount >= onWhatsAppProbeErrLimit {
				break // bound the slow/error path
			}
			continue
		}

		// Empty response is inconclusive, not a confirmed negative — retry.
		if len(data) == 0 {
			logrus.Debugf("IsOnWhatsApp returned empty for %s (attempt %d/%d) — inconclusive, retrying", phone, attempt+1, onWhatsAppProbeAttempts)
			continue
		}

		for _, v := range data {
			if v.IsIn {
				return v.JID, probePositive
			}
		}
		// Non-empty response with no registered entry — authoritative negative.
		return types.JID{}, probeNegative
	}

	return types.JID{}, probeAmbiguous
}

// resolveProbeOutcome maps a probe outcome to the final JID/error for the send
// path. originalJID is the caller-supplied JID (used only for messaging);
// normalizedPhone is the BR-stripped, E.164 phone to send to when WhatsApp does
// not hand back a canonical JID. validation reflects WhatsappAccountValidation.
//
// The key behavior: an ambiguous probe NEVER hard-fails — it falls through to
// the deterministically-normalized JID so a transient USync miss doesn't turn a
// valid recipient into a permanent "not on WhatsApp" send failure. Only an
// authoritative negative (with validation on) is rejected.
func resolveProbeOutcome(originalJID, normalizedPhone string, canonicalJID types.JID, outcome probeOutcome, validation bool) (types.JID, error) {
	switch outcome {
	case probePositive:
		if !canonicalJID.IsEmpty() {
			logrus.Debugf("Normalized JID %s to %s", originalJID, canonicalJID.String())
			return canonicalJID, nil
		}
		// Registered, but WhatsApp omitted the canonical JID — use normalized.
		return ParseJID(normalizedPhone)

	case probeNegative:
		if validation {
			return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on WhatsApp", originalJID))
		}
		return ParseJID(normalizedPhone)

	default: // probeAmbiguous
		logrus.Warnf("Could not verify %s on WhatsApp (probe inconclusive); proceeding with normalized JID", originalJID)
		return ParseJID(normalizedPhone)
	}
}

// resolveUserJID handles the user-JID (phone) tail of ValidateAndNormalizeJID:
// BR 9th-digit + E.164 normalization, the bounded WhatsApp probe, and outcome
// mapping. Split out (taking the onWhatsAppProber seam and a ctx) so the
// normalize → probe → resolve glue is unit-testable without a live
// *whatsmeow.Client. jid must already be a "<phone>@s.whatsapp.net" user JID.
func resolveUserJID(ctx context.Context, prober onWhatsAppProber, jid string, validation bool) (types.JID, error) {
	// Extract the phone string from the JID.
	phone := strings.TrimSuffix(jid, "@s.whatsapp.net")
	if phone == "" {
		return types.JID{}, pkgError.InvalidJID("Empty phone number")
	}

	// Apply BR 9th-digit strip first, then upstream E.164 (adds the leading +).
	phone = normalizePhoneBR(phone)
	phone = NormalizePhoneE164(phone)

	// Probe WhatsApp with bounded retries, then classify. An inconclusive probe
	// falls through to the normalized JID rather than hard-failing the send.
	canonicalJID, outcome := probeOnWhatsApp(ctx, prober, phone, onWhatsAppRetryBackoff)
	return resolveProbeOutcome(jid, phone, canonicalJID, outcome, validation)
}

// ValidateAndNormalizeJID queries WhatsApp for the canonical JID, applying
// Brazil 9th-digit normalization for user JIDs. For non-user JIDs (groups,
// newsletters, LID) it returns the parsed JID unchanged.
//
// Thin layer on upstream utils.NormalizePhoneE164 + ParseJID; calls whatsmeow's
// client.IsOnWhatsApp with bounded retries (see probeOnWhatsApp). WhatsApp's
// returned canonical JID is authoritative. An inconclusive probe (empty or
// errored USync) is treated as ambiguous, not as "not on WhatsApp": the send
// falls through to the deterministically-normalized JID rather than failing.
// Only an authoritative negative (IsIn == false) is rejected when
// WhatsappAccountValidation is on.
func ValidateAndNormalizeJID(client *whatsmeow.Client, jid string) (types.JID, error) {
	// LID JIDs route through Slice 3's LID resolution to recover the canonical
	// phone JID, then fall through to the BR normalization pipeline.
	if strings.Contains(jid, "@lid") {
		if client == nil {
			return ParseJID(jid)
		}
		parsed, err := ParseJID(jid)
		if err != nil {
			return types.JID{}, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resolved := ResolveLIDToPhone(ctx, parsed, client)
		// If resolution returned the same @lid JID, no phone mapping exists yet.
		if resolved.Server == "lid" {
			return resolved, nil
		}
		// Re-enter the pipeline with the resolved phone JID for BR normalization.
		jid = resolved.String()
	}

	// For non-user JIDs (groups, newsletters), skip normalization.
	if !strings.Contains(jid, "@s.whatsapp.net") {
		return ParseJID(jid)
	}

	// If no client provided, fall back to simple parsing.
	if client == nil {
		return ParseJID(jid)
	}

	MustLogin(client)

	// The phone-extraction → BR/E.164 normalization → probe → classify tail lives
	// in resolveUserJID (testable via the onWhatsAppProber seam). context.Background
	// for now: threading the request context through the ~40 callers is a separate
	// change; the total probe budget (onWhatsAppTotalTimeout) bounds the wall-clock.
	return resolveUserJID(context.Background(), client, jid, config.WhatsappAccountValidation)
}

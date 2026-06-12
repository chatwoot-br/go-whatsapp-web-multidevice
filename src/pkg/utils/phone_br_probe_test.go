package utils

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// fakeProber replays a scripted sequence of IsOnWhatsApp responses/errors so the
// retry/classification logic can be exercised without a live whatsmeow session.
// Attempt N reads index N from errs/results (missing index = nil/no error).
//
// When byPhone is non-nil the prober is instead INPUT-AWARE: each queried phone is
// looked up (digits-only) and, if present, returned as a response entry with Query
// set to the queried string and IsIn per the map (a registered entry gets the
// queried digits as its canonical JID). Numbers absent from the map are omitted
// from the response, mirroring USync, which need not echo numbers it has no record
// for. This lets candidate routing (BR both-forms probe) be asserted.
type fakeProber struct {
	results [][]types.IsOnWhatsAppResponse
	errs    []error
	calls   int

	byPhone map[string]bool // digits-only phone -> IsIn
}

func (f *fakeProber) IsOnWhatsApp(_ context.Context, phones []string) ([]types.IsOnWhatsAppResponse, error) {
	i := f.calls
	f.calls++
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}
	if f.byPhone != nil {
		var data []types.IsOnWhatsAppResponse
		for _, p := range phones {
			key := CleanPhoneForWhatsApp(p)
			isIn, ok := f.byPhone[key]
			if !ok {
				continue
			}
			var j types.JID
			if isIn {
				j = types.NewJID(key, types.DefaultUserServer)
			}
			data = append(data, types.IsOnWhatsAppResponse{Query: p, JID: j, IsIn: isIn})
		}
		return data, err
	}
	var data []types.IsOnWhatsAppResponse
	if i < len(f.results) {
		data = f.results[i]
	}
	return data, err
}

// oneResp builds a single-entry USync response. An empty jidUser yields a
// response whose JID is empty (the "registered but no canonical JID" case).
func oneResp(jidUser string, isIn bool) []types.IsOnWhatsAppResponse {
	var j types.JID
	if jidUser != "" {
		j = types.NewJID(jidUser, types.DefaultUserServer)
	}
	return []types.IsOnWhatsAppResponse{{JID: j, IsIn: isIn}}
}

func TestProbeOnWhatsApp(t *testing.T) {
	transient := errors.New("usync timeout")

	cases := []struct {
		name        string
		results     [][]types.IsOnWhatsAppResponse
		errs        []error
		wantOutcome probeOutcome
		wantCalls   int
		wantJIDUser string // checked only for probePositive
	}{
		{
			name:        "immediate positive returns canonical JID, no retry",
			results:     [][]types.IsOnWhatsAppResponse{oneResp("556696679626", true)},
			wantOutcome: probePositive,
			wantCalls:   1,
			wantJIDUser: "556696679626",
		},
		{
			name:        "confirmed negative is authoritative, no retry",
			results:     [][]types.IsOnWhatsAppResponse{oneResp("", false)},
			wantOutcome: probeNegative,
			wantCalls:   1,
		},
		{
			name:        "empty response is retried then resolves positive",
			results:     [][]types.IsOnWhatsAppResponse{nil, oneResp("556696679626", true)},
			wantOutcome: probePositive,
			wantCalls:   2,
			wantJIDUser: "556696679626",
		},
		{
			name:        "all-empty exhausts retries and is ambiguous (not negative)",
			results:     [][]types.IsOnWhatsAppResponse{nil, nil, nil},
			wantOutcome: probeAmbiguous,
			wantCalls:   onWhatsAppProbeAttempts,
		},
		{
			name:        "transport error then positive",
			errs:        []error{transient},
			results:     [][]types.IsOnWhatsAppResponse{nil, oneResp("556696679626", true)},
			wantOutcome: probePositive,
			wantCalls:   2,
			wantJIDUser: "556696679626",
		},
		{
			name:        "repeated errors hit the error cap and are ambiguous",
			errs:        []error{transient, transient, transient},
			wantOutcome: probeAmbiguous,
			wantCalls:   onWhatsAppProbeErrLimit,
		},
		{
			name:        "error then empty then positive uses full attempt budget",
			errs:        []error{transient},
			results:     [][]types.IsOnWhatsAppResponse{nil, nil, oneResp("556696679626", true)},
			wantOutcome: probePositive,
			wantCalls:   3,
			wantJIDUser: "556696679626",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeProber{results: tc.results, errs: tc.errs}
			jid, outcome := probeOnWhatsApp(context.Background(), f, []string{"+556696679626"}, 0)

			if outcome != tc.wantOutcome {
				t.Fatalf("outcome = %d, want %d", outcome, tc.wantOutcome)
			}
			if f.calls != tc.wantCalls {
				t.Errorf("calls = %d, want %d", f.calls, tc.wantCalls)
			}
			if tc.wantOutcome == probePositive && tc.wantJIDUser != "" && jid.User != tc.wantJIDUser {
				t.Errorf("positive JID user = %q, want %q", jid.User, tc.wantJIDUser)
			}
		})
	}
}

func TestResolveProbeOutcome(t *testing.T) {
	canonical := types.NewJID("556696679626", types.DefaultUserServer)
	const originalJID = "5566996679626@s.whatsapp.net" // 13-digit BR (un-stripped)
	const normalizedPhone = "+556696679626"            // BR 9th-digit stripped, E.164

	cases := []struct {
		name       string
		canonical  types.JID
		outcome    probeOutcome
		validation bool
		wantErr    bool
		wantUser   string
		wantServer string
	}{
		{
			name:       "positive returns WhatsApp canonical JID",
			canonical:  canonical,
			outcome:    probePositive,
			validation: true,
			wantUser:   "556696679626",
			wantServer: "s.whatsapp.net",
		},
		{
			name:       "positive without canonical JID falls back to normalized phone",
			canonical:  types.JID{},
			outcome:    probePositive,
			validation: true,
			wantUser:   "556696679626",
			wantServer: "s.whatsapp.net",
		},
		{
			name:       "confirmed negative with validation on is rejected",
			outcome:    probeNegative,
			validation: true,
			wantErr:    true,
		},
		{
			name:       "confirmed negative with validation off falls through to normalized",
			outcome:    probeNegative,
			validation: false,
			wantUser:   "556696679626",
			wantServer: "s.whatsapp.net",
		},
		{
			// The core regression fix: an inconclusive probe must NOT hard-fail
			// even with validation on, and the fall-through JID must honor the BR
			// 9th-digit strip (12-digit user, not the original 13-digit).
			name:       "ambiguous with validation on falls open to BR-normalized JID",
			outcome:    probeAmbiguous,
			validation: true,
			wantUser:   "556696679626",
			wantServer: "s.whatsapp.net",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveProbeOutcome(originalJID, normalizedPhone, tc.canonical, tc.outcome, tc.validation)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (jid=%v)", got)
				}
				if msg := err.Error(); !strings.Contains(msg, "is not on WhatsApp") {
					t.Errorf("error %q should mention 'is not on WhatsApp'", msg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.User != tc.wantUser || got.Server != tc.wantServer {
				t.Errorf("got %s@%s, want %s@%s", got.User, got.Server, tc.wantUser, tc.wantServer)
			}
		})
	}
}

// TestProbeOnWhatsApp_Deadline verifies the total-budget deadline cuts the
// retry loop short instead of running every attempt — the guarantee the
// per-attempt timeout and the errCount cap do NOT provide for slow/looping
// empty USync responses. A pre-cancelled ctx stands in for "budget exhausted".
func TestProbeOnWhatsApp_Deadline(t *testing.T) {
	t.Run("expired ctx breaks the loop before exhausting attempts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // budget already gone
		// Always-empty prober would loop onWhatsAppProbeAttempts times if unbounded.
		f := &fakeProber{}
		_, outcome := probeOnWhatsApp(ctx, f, []string{"+556696679626"}, 0)
		if outcome != probeAmbiguous {
			t.Fatalf("outcome = %d, want probeAmbiguous", outcome)
		}
		if f.calls >= onWhatsAppProbeAttempts {
			t.Errorf("calls = %d, want < %d (deadline must short-circuit, not the attempt cap)", f.calls, onWhatsAppProbeAttempts)
		}
	})

	t.Run("expired ctx wins the backoff select instead of sleeping", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f := &fakeProber{}
		// A 30s backoff forces the inter-attempt select; if the Done() arm didn't
		// fire, this test would hang ~30s. It returns immediately.
		_, outcome := probeOnWhatsApp(ctx, f, []string{"+556696679626"}, 30*time.Second)
		if outcome != probeAmbiguous {
			t.Fatalf("outcome = %d, want probeAmbiguous", outcome)
		}
		if f.calls >= onWhatsAppProbeAttempts {
			t.Errorf("calls = %d, want < %d", f.calls, onWhatsAppProbeAttempts)
		}
	})
}

// TestIsOnWhatsApp covers the glue in IsOnWhatsapp: routing (non-user / empty
// phone) and the honest probe-outcome → bool mapping (only positive is true).
func TestIsOnWhatsApp(t *testing.T) {
	cases := []struct {
		name   string
		jid    string
		prober *fakeProber
		want   bool
	}{
		{name: "non-user JID skips probe and returns true", jid: "120363@g.us", prober: &fakeProber{}, want: true},
		{name: "empty phone returns false", jid: "@s.whatsapp.net", prober: &fakeProber{}, want: false},
		{name: "confirmed positive returns true", jid: "556696679626@s.whatsapp.net", prober: &fakeProber{results: [][]types.IsOnWhatsAppResponse{oneResp("556696679626", true)}}, want: true},
		{name: "confirmed negative returns false", jid: "556696679626@s.whatsapp.net", prober: &fakeProber{results: [][]types.IsOnWhatsAppResponse{oneResp("", false)}}, want: false},
		// Ambiguous (all-empty probe) must stay honest here — false, not fall-open.
		{name: "ambiguous returns false (honest, unlike the send path)", jid: "556696679626@s.whatsapp.net", prober: &fakeProber{}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isOnWhatsApp(context.Background(), tc.prober, tc.jid); got != tc.want {
				t.Errorf("isOnWhatsApp(%q) = %v, want %v", tc.jid, got, tc.want)
			}
		})
	}
}

// TestResolveUserJID covers the ValidateAndNormalizeJID tail end-to-end with a
// fake prober: empty-phone guard, positive → canonical, negative → reject, and
// the key fall-open case where an ambiguous probe sends to the BR-stripped JID.
func TestResolveUserJID(t *testing.T) {
	t.Run("empty phone is rejected", func(t *testing.T) {
		if _, err := resolveUserJID(context.Background(), &fakeProber{}, "@s.whatsapp.net", true); err == nil {
			t.Fatal("expected error for empty phone")
		}
	})

	t.Run("positive returns canonical JID", func(t *testing.T) {
		p := &fakeProber{results: [][]types.IsOnWhatsAppResponse{oneResp("556696679626", true)}}
		got, err := resolveUserJID(context.Background(), p, "556696679626@s.whatsapp.net", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.User != "556696679626" {
			t.Errorf("got user %q, want 556696679626", got.User)
		}
	})

	t.Run("ambiguous BR mobile falls open to the 12-digit normalized JID", func(t *testing.T) {
		// 13-digit BR (with 9th digit) + ambiguous probe + validation ON must NOT
		// error; it must send to the BR-stripped 12-digit number — the core fix,
		// verified here through the full extract → normalize → probe → resolve glue.
		got, err := resolveUserJID(context.Background(), &fakeProber{}, "5566996679626@s.whatsapp.net", true)
		if err != nil {
			t.Fatalf("ambiguous probe must not hard-fail: %v", err)
		}
		if got.User != "556696679626" || got.Server != "s.whatsapp.net" {
			t.Errorf("got %s@%s, want 556696679626@s.whatsapp.net", got.User, got.Server)
		}
	})

	t.Run("confirmed negative with validation is rejected", func(t *testing.T) {
		// Complete negative: WhatsApp answers for BOTH probed forms, neither
		// registered. (556696679626's local part starts 9, so it is a mobile and
		// expands to a 2-candidate probe; a single-entry response would be a partial
		// → ambiguous, not an authoritative negative.)
		p := &fakeProber{byPhone: map[string]bool{"556696679626": false, "5566996679626": false}}
		if _, err := resolveUserJID(context.Background(), p, "556696679626@s.whatsapp.net", true); err == nil {
			t.Fatal("expected rejection for confirmed negative with validation on")
		}
	})
}

// TestResolveUserJID_BR9thDigit is the regression for the production incident: a
// BR contact whose WhatsApp account is registered ONLY under one ninth-digit form
// must resolve regardless of which form the caller dialed. Before the both-forms
// probe, the send path stripped the 9 and probed only the 12-digit form
// (IsIn=false) → rejected with validation on. The sibling is generated only for
// MOBILE local parts (6-9); see the landline sub-test for the safety property.
func TestResolveUserJID_BR9thDigit(t *testing.T) {
	// Mobile contact (local part starts 6) registered ONLY as the 13-digit form.
	registered := map[string]bool{
		"5511966665555": true,  // with the mobile 9
		"551166665555":  false, // without the 9
	}

	t.Run("dialed 13-digit (with 9), only 13-digit registered, validation on", func(t *testing.T) {
		p := &fakeProber{byPhone: registered}
		got, err := resolveUserJID(context.Background(), p, "5511966665555@s.whatsapp.net", true)
		if err != nil {
			t.Fatalf("must resolve, not reject: %v", err)
		}
		if got.User != "5511966665555" || got.Server != "s.whatsapp.net" {
			t.Errorf("got %s@%s, want 5511966665555@s.whatsapp.net", got.User, got.Server)
		}
	})

	t.Run("inverse: dialed 12-digit (no 9) mobile, only 13-digit registered", func(t *testing.T) {
		p := &fakeProber{byPhone: registered}
		got, err := resolveUserJID(context.Background(), p, "551166665555@s.whatsapp.net", true)
		if err != nil {
			t.Fatalf("must resolve via the 9-inserted sibling, not reject: %v", err)
		}
		if got.User != "5511966665555" || got.Server != "s.whatsapp.net" {
			t.Errorf("got %s@%s, want 5511966665555@s.whatsapp.net (canonical with-9 form)", got.User, got.Server)
		}
	})

	t.Run("isOnWhatsApp is true for the 12-digit mobile when only 13-digit is registered", func(t *testing.T) {
		p := &fakeProber{byPhone: registered}
		if !isOnWhatsApp(context.Background(), p, "551166665555@s.whatsapp.net") {
			t.Error("isOnWhatsApp should report true via the 9-inserted sibling")
		}
	})

	// The reported incident number's local part starts 4 (non-mobile range), so
	// its 13-digit-dialed form still resolves (no sibling needed), but the inverse
	// 12-digit-dialed form is intentionally NOT auto-resolved (gated) — see below.
	t.Run("reported number: 13-digit-dialed still resolves despite the gate", func(t *testing.T) {
		p := &fakeProber{byPhone: map[string]bool{"5511945590462": true, "551145590462": false}}
		got, err := resolveUserJID(context.Background(), p, "5511945590462@s.whatsapp.net", true)
		if err != nil {
			t.Fatalf("13-digit-dialed must resolve: %v", err)
		}
		if got.User != "5511945590462" {
			t.Errorf("got %s, want 5511945590462", got.User)
		}
	})

	// SAFETY (the misroute guard): a 12-digit LANDLINE whose "+9" sibling is a
	// DIFFERENT subscriber's registered mobile must NOT route to that stranger.
	// The mobile-local gate drops the sibling, so only the (unregistered) landline
	// is probed → rejected. Without the gate this returned the stranger's JID.
	t.Run("landline does not misroute to its stranger +9 sibling", func(t *testing.T) {
		p := &fakeProber{byPhone: map[string]bool{
			"551133334444":  false, // the dialed landline — not on WhatsApp
			"5511933334444": true,  // a DIFFERENT subscriber's mobile (must never be reached)
		}}
		got, err := resolveUserJID(context.Background(), p, "551133334444@s.whatsapp.net", true)
		if err == nil {
			t.Fatalf("landline must be rejected, not routed to a stranger; got %s", got.User)
		}
		if got.User == "5511933334444" {
			t.Fatal("MISROUTE: resolved to a different subscriber's mobile")
		}
	})

	t.Run("genuinely-not-on-WhatsApp BR mobile is still rejected with validation on", func(t *testing.T) {
		// Neither ninth-digit form is registered (both echoed IsIn=false).
		p := &fakeProber{byPhone: map[string]bool{"5511966665555": false, "551166665555": false}}
		if _, err := resolveUserJID(context.Background(), p, "5511966665555@s.whatsapp.net", true); err == nil {
			t.Fatal("expected rejection when neither form is on WhatsApp")
		}
	})
}

// TestProbeOnWhatsApp_PrefersAsDialed verifies that when both ninth-digit forms
// come back registered, the as-dialed candidate (phones[0]) wins over the sibling
// — the ghost-number hardening. The sibling is listed FIRST in the response
// (USync does not guarantee query order), so a plain first-positive fallback
// would return the sibling; only the as-dialed preference returns the dialed
// form. This makes the test fail if the preference branch is removed.
func TestProbeOnWhatsApp_PrefersAsDialed(t *testing.T) {
	resp := []types.IsOnWhatsAppResponse{
		{Query: "+5511945590462", JID: types.NewJID("5511945590462", types.DefaultUserServer), IsIn: true},
		{Query: "+551145590462", JID: types.NewJID("551145590462", types.DefaultUserServer), IsIn: true},
	}
	f := &fakeProber{results: [][]types.IsOnWhatsAppResponse{resp}}
	jid, outcome := probeOnWhatsApp(context.Background(), f, []string{"+551145590462", "+5511945590462"}, 0)
	if outcome != probePositive {
		t.Fatalf("outcome = %d, want probePositive", outcome)
	}
	if jid.User != "551145590462" {
		t.Errorf("as-dialed should win: got %q, want 551145590462", jid.User)
	}
}

// TestProbeOnWhatsApp_PartialResponseIsAmbiguous: when >1 candidate is probed and
// USync echoes only SOME of them (e.g. the registered form is omitted while the
// unregistered sibling comes back IsIn=false), the result is inconclusive, not an
// authoritative negative — it must retry and end ambiguous (so the send falls open
// instead of hard-failing a valid recipient). Pre-fix this returned probeNegative.
func TestProbeOnWhatsApp_PartialResponseIsAmbiguous(t *testing.T) {
	// 2 candidates queried; response carries only the (unregistered) sibling and
	// omits the as-dialed form. Repeated across every attempt.
	partial := []types.IsOnWhatsAppResponse{
		{Query: "+551166665555", JID: types.JID{}, IsIn: false},
	}
	f := &fakeProber{results: [][]types.IsOnWhatsAppResponse{partial, partial, partial}}
	_, outcome := probeOnWhatsApp(context.Background(), f, []string{"+5511966665555", "+551166665555"}, 0)
	if outcome != probeAmbiguous {
		t.Fatalf("partial response must be ambiguous (not negative), got %d", outcome)
	}
	if f.calls != onWhatsAppProbeAttempts {
		t.Errorf("partial response should retry to the attempt cap: calls=%d, want %d", f.calls, onWhatsAppProbeAttempts)
	}
}

// TestProbeOnWhatsApp_CompleteNegativeStillRejects guards the non-regression: when
// the response covers EVERY queried candidate and none is registered, it is still
// an authoritative negative.
func TestProbeOnWhatsApp_CompleteNegativeStillRejects(t *testing.T) {
	complete := []types.IsOnWhatsAppResponse{
		{Query: "+5511966665555", JID: types.JID{}, IsIn: false},
		{Query: "+551166665555", JID: types.JID{}, IsIn: false},
	}
	f := &fakeProber{results: [][]types.IsOnWhatsAppResponse{complete}}
	_, outcome := probeOnWhatsApp(context.Background(), f, []string{"+5511966665555", "+551166665555"}, 0)
	if outcome != probeNegative {
		t.Fatalf("complete all-negative response must be probeNegative, got %d", outcome)
	}
	if f.calls != 1 {
		t.Errorf("authoritative negative should not retry: calls=%d, want 1", f.calls)
	}
}

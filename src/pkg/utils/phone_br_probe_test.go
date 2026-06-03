package utils

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

// fakeProber replays a scripted sequence of IsOnWhatsApp responses/errors so the
// retry/classification logic can be exercised without a live whatsmeow session.
// Attempt N reads index N from errs/results (missing index = nil/no error).
type fakeProber struct {
	results [][]types.IsOnWhatsAppResponse
	errs    []error
	calls   int
}

func (f *fakeProber) IsOnWhatsApp(_ context.Context, _ []string) ([]types.IsOnWhatsAppResponse, error) {
	i := f.calls
	f.calls++
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
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
			jid, outcome := probeOnWhatsApp(f, "+556696679626", 0)

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

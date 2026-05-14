package utils

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestNormalizePhoneBR(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "13-digit BR mobile strips 9th digit",
			in:   "5566996679626",
			out:  "556696679626",
		},
		{
			name: "13-digit BR mobile with + prefix strips 9th digit",
			in:   "+5566996679626",
			out:  "+556696679626",
		},
		{
			name: "12-digit BR (already normalized) passthrough",
			in:   "556696679626",
			out:  "556696679626",
		},
		{
			name: "13-digit non-BR passthrough (not prefixed 55)",
			in:   "1199912345678",
			out:  "1199912345678",
		},
		{
			name: "13-digit BR but no 9 at position 4 passthrough",
			in:   "5566896679626",
			out:  "5566896679626",
		},
		{
			name: "US 11-digit passthrough",
			in:   "14155552671",
			out:  "14155552671",
		},
		{
			name: "empty passthrough",
			in:   "",
			out:  "",
		},
		{
			name: "whitespace trimmed",
			in:   "  5566996679626  ",
			out:  "556696679626",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizePhoneBR(tc.in)
			if got != tc.out {
				t.Errorf("normalizePhoneBR(%q) = %q, want %q", tc.in, got, tc.out)
			}
		})
	}
}

func TestValidateAndNormalizeJID_GroupJIDPassthrough(t *testing.T) {
	// Group JIDs pass through without normalization.
	jid := "120363123456789012@g.us"
	result, err := ValidateAndNormalizeJID(nil, jid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := types.JID{User: "120363123456789012", Server: "g.us"}
	if result.User != expected.User || result.Server != expected.Server {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestValidateAndNormalizeJID_NonUserJIDPassthrough(t *testing.T) {
	// Newsletter and other non-user JIDs pass through.
	jid := "120363123456789012@newsletter"
	result, err := ValidateAndNormalizeJID(nil, jid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Server != "newsletter" {
		t.Errorf("got server %s, want newsletter", result.Server)
	}
}

func TestValidateAndNormalizeJID_LIDPassthrough(t *testing.T) {
	// LID JIDs pass through without normalization (Slice 3 owns LID resolution).
	jid := "215946727821336@lid"
	result, err := ValidateAndNormalizeJID(nil, jid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Server != "lid" {
		t.Errorf("got server %s, want lid", result.Server)
	}
}

// TestNormalizePhoneBR_MalformedInputs locks down the passthrough behavior
// for edge cases the deterministic fallback must not corrupt.
func TestNormalizePhoneBR_MalformedInputs(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		// 14-digit number — too long for the BR pattern. Must passthrough.
		{name: "14-digit-passthrough", in: "55669966796260", out: "55669966796260"},
		// 11-digit — shorter than the BR shape. Must passthrough.
		{name: "11-digit-passthrough", in: "55669966796", out: "55669966796"},
		// Starts with 55 but position-4 char is not '9' (e.g., landline).
		{name: "landline-area-13", in: "5566123456789", out: "5566123456789"},
		// Non-numeric characters mixed in: must not panic; not 13 digits so passthrough.
		{name: "non-numeric", in: "55-66-9966-79626", out: "55-66-9966-79626"},
		// Leading "+ " variant — whitespace inside is preserved (passthrough).
		{name: "spaces-inside", in: "+55 66 99667 9626", out: "+55 66 99667 9626"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizePhoneBR(tc.in)
			if got != tc.out {
				t.Errorf("normalizePhoneBR(%q) = %q, want %q", tc.in, got, tc.out)
			}
		})
	}
}

// TestValidateAndNormalizeJID_EmptyJID locks the empty-input behavior — does
// NOT panic; falls through to ParseJID which returns an empty user JID on the
// default server. (Validation that the caller must check for empty User
// elsewhere lives in the send path.)
func TestValidateAndNormalizeJID_EmptyJID(t *testing.T) {
	result, err := ValidateAndNormalizeJID(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User != "" {
		t.Errorf("expected empty user, got %q", result.User)
	}
}

// TestValidateAndNormalizeJID_BRLikeUserJID_NilClient confirms the no-client
// fallback parses without applying any normalization (Slice 2 invariant — BR
// strip only applies when a live client confirms via IsOnWhatsApp).
func TestValidateAndNormalizeJID_BRLikeUserJID_NilClient(t *testing.T) {
	jid := "5566996679626@s.whatsapp.net"
	result, err := ValidateAndNormalizeJID(nil, jid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Nil client falls back to ParseJID — user part stays 13-digit.
	if result.User != "5566996679626" {
		t.Errorf("nil client must not normalize: got user=%q", result.User)
	}
}

func TestValidateAndNormalizeJID_UserJIDWithNilClient(t *testing.T) {
	// User JID with nil client falls back to ParseJID.
	jid := "5511999999999@s.whatsapp.net"
	result, err := ValidateAndNormalizeJID(nil, jid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.User != "5511999999999" || result.Server != "s.whatsapp.net" {
		t.Errorf("got %v, want 5511999999999@s.whatsapp.net", result)
	}
}

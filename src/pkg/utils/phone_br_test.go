package utils

import (
	"reflect"
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

func TestBrPhoneCandidates(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		// Mobile local part (starts 6-9) → both ninth-digit forms.
		{"13-digit BR mobile adds 12-digit sibling", "5566996679626", []string{"+5566996679626", "+556696679626"}},
		{"12-digit BR mobile adds 13-digit (9-inserted) sibling", "551166665555", []string{"+551166665555", "+5511966665555"}},
		{"+ prefix preserved, mobile both forms", "+5566996679626", []string{"+5566996679626", "+556696679626"}},
		{"jid suffix stripped, mobile both forms", "5566996679626@s.whatsapp.net", []string{"+5566996679626", "+556696679626"}},
		// Non-mobile local part (starts 2-5) → gated to the as-dialed form only,
		// so a landline never yields a stranger's "+9" mobile sibling.
		{"12-digit BR landline (local starts 3) is gated to single", "551133334444", []string{"+551133334444"}},
		{"13-digit BR with non-mobile local (post-9 = 4) is gated to single", "5511945590462", []string{"+5511945590462"}},
		{"12-digit BR with non-mobile local (starts 4) is gated to single", "551145590462", []string{"+551145590462"}},
		{"non-BR 13-digit is single candidate", "1199912345678", []string{"+1199912345678"}},
		{"13-digit BR without 9 at position 4 is single candidate", "5566896679626", []string{"+5566896679626"}},
		{"US 11-digit is single candidate", "14155552671", []string{"+14155552671"}},
		{"12-digit non-BR is single candidate", "115512345678", []string{"+115512345678"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := brPhoneCandidates(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("brPhoneCandidates(%q) = %v, want %v", tc.in, got, tc.want)
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

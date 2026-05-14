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

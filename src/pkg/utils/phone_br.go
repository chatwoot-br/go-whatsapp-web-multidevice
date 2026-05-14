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

// ValidateAndNormalizeJID queries WhatsApp for the canonical JID, applying
// Brazil 9th-digit normalization for user JIDs. For non-user JIDs (groups,
// newsletters, LID) it returns the parsed JID unchanged.
//
// Thin layer on upstream utils.NormalizePhoneE164 + ParseJID; calls
// whatsmeow's client.IsOnWhatsApp under a 10s context timeout. WhatsApp's
// returned canonical JID is authoritative; the string-level BR strip in
// normalizePhoneBR is the deterministic fallback when IsOnWhatsApp is
// unavailable.
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

	// Extract the phone string from the JID.
	phone := strings.TrimSuffix(jid, "@s.whatsapp.net")
	if phone == "" {
		return types.JID{}, pkgError.InvalidJID("Empty phone number")
	}

	// Apply BR 9th-digit strip first, then upstream E.164 (adds the leading +).
	phone = normalizePhoneBR(phone)
	phone = NormalizePhoneE164(phone)

	// Query WhatsApp for the canonical JID.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := client.IsOnWhatsApp(ctx, []string{phone})
	if err != nil {
		logrus.Warnf("Failed to query WhatsApp for %s: %v", jid, err)
		if config.WhatsappAccountValidation {
			return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Failed to validate phone %s: %v", jid, err))
		}
		return ParseJID(jid)
	}

	// Empty response means number not found.
	if len(data) == 0 {
		if config.WhatsappAccountValidation {
			return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on WhatsApp", jid))
		}
		return ParseJID(jid)
	}

	// Check results and return WhatsApp's canonical JID when present.
	for _, v := range data {
		if !v.IsIn {
			if config.WhatsappAccountValidation {
				return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on WhatsApp", jid))
			}
			return ParseJID(jid)
		}

		if !v.JID.IsEmpty() {
			logrus.Debugf("Normalized JID %s to %s", jid, v.JID.String())
			return v.JID, nil
		}
	}

	// Fallback to original parse.
	return ParseJID(jid)
}

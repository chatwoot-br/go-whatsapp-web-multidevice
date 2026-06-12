package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// TestGetMessageDigestOrSignature_KnownVector verifies HMAC-SHA256 with a
// fixed key/body pair against the value an independent reference computation
// produces, locking the cipher choice (HMAC-SHA256, hex-encoded) against
// future regressions. The webhook contract documents
// `X-Hub-Signature-256: sha256=<hex>` so the encoding must remain hex.
func TestGetMessageDigestOrSignature_KnownVector(t *testing.T) {
	body := []byte(`{"event":"history_sync_complete","device_id":"x","payload":{}}`)
	key := []byte("secret-shared-with-webhook-receiver")

	// Reference: compute HMAC-SHA256 the canonical way and compare.
	mac := hmac.New(sha256.New, key)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	got, err := GetMessageDigestOrSignature(body, key)
	if err != nil {
		t.Fatalf("GetMessageDigestOrSignature: %v", err)
	}
	if got != expected {
		t.Fatalf("got %s, want %s (HMAC-SHA256 mismatch)", got, expected)
	}
	// Sanity: hex output is 64 chars (SHA256 = 32 bytes).
	if len(got) != 64 {
		t.Fatalf("HMAC-SHA256 hex must be 64 chars, got %d", len(got))
	}
}

// TestGetMessageDigestOrSignature_NotPlainSHA256 guards against an accidental
// downgrade from HMAC-SHA256 to plain SHA256 — the two produce different
// outputs for the same body+key. The webhook receivers verify by HMAC, so a
// downgrade would silently break every consumer.
func TestGetMessageDigestOrSignature_NotPlainSHA256(t *testing.T) {
	body := []byte("payload")
	key := []byte("k")

	hmacOut, err := GetMessageDigestOrSignature(body, key)
	if err != nil {
		t.Fatalf("GetMessageDigestOrSignature: %v", err)
	}
	plain := sha256.Sum256(body)
	plainHex := hex.EncodeToString(plain[:])
	if hmacOut == plainHex {
		t.Fatalf("function is plain SHA256, not HMAC: %s", hmacOut)
	}
}

// TestGetMessageDigestOrSignature_KeySensitive ensures changing the key changes
// the output — guards against the function ignoring the secret entirely.
func TestGetMessageDigestOrSignature_KeySensitive(t *testing.T) {
	body := []byte("payload")
	a, err := GetMessageDigestOrSignature(body, []byte("k1"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := GetMessageDigestOrSignature(body, []byte("k2"))
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("different keys must produce different signatures (a=%s, b=%s)", a, b)
	}
}

func TestDetermineMediaExtension(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		mimeType   string
		wantSuffix string
	}{
		{
			name:       "DocxFromFilename",
			filename:   "report.docx",
			mimeType:   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			wantSuffix: ".docx",
		},
		{
			name:       "XlsxFromMime",
			filename:   "",
			mimeType:   "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			wantSuffix: ".xlsx",
		},
		{
			name:       "PptxFromMime",
			filename:   "",
			mimeType:   "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			wantSuffix: ".pptx",
		},
		{
			name:       "ZipFallback",
			filename:   "",
			mimeType:   "application/zip",
			wantSuffix: ".zip",
		},
		{
			name:       "AudioOgaWithCodecsParam",
			filename:   "",
			mimeType:   "audio/ogg; codecs=opus",
			wantSuffix: ".oga",
		},
		{
			name:       "ExeFromFilename",
			filename:   "installer.exe",
			mimeType:   "application/octet-stream",
			wantSuffix: ".exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineMediaExtension(tt.filename, tt.mimeType)
			if got != tt.wantSuffix {
				t.Fatalf("determineMediaExtension() = %q, want %q", got, tt.wantSuffix)
			}
		})
	}
}

func TestExtractPhoneFromVCard(t *testing.T) {
	tests := []struct {
		name  string
		vcard string
		want  string
	}{
		{
			name:  "LFEndings",
			vcard: "BEGIN:VCARD\nVERSION:3.0\nFN:Alice\nTEL;type=Mobile:+62 812 3456 7890\nEND:VCARD",
			want:  "+62 812 3456 7890",
		},
		{
			name:  "CRLFEndings",
			vcard: "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Bob\r\nTEL:+1 555 0100\r\nEND:VCARD",
			want:  "+1 555 0100",
		},
		{
			name:  "FoldedLine",
			vcard: "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Julio\r\nTEL;type=CELL;waid=5511998913283:\r\n +5511998913283\r\nEND:VCARD",
			want:  "+5511998913283",
		},
		{
			name:  "NoTelLine",
			vcard: "BEGIN:VCARD\nVERSION:3.0\nFN:Carol\nEND:VCARD",
			want:  "",
		},
		{
			name:  "Empty",
			vcard: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPhoneFromVCard(tt.vcard)
			if got != tt.want {
				t.Fatalf("ExtractPhoneFromVCard() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatContactSummary(t *testing.T) {
	tests := []struct {
		name   string
		dName  string
		phone  string
		plural bool
		want   string
	}{
		{name: "SingleNameAndPhone", dName: "Alice", phone: "+62 812", plural: false, want: "Contact: Alice (+62 812)"},
		{name: "SingleNameOnly", dName: "Alice", phone: "", plural: false, want: "Contact: Alice"},
		{name: "SinglePhoneOnly", dName: "", phone: "+62 812", plural: false, want: "Contact: +62 812"},
		{name: "SingleEmpty", dName: "", phone: "", plural: false, want: "Contact shared"},
		{name: "SingleWhitespaceOnly", dName: " ", phone: " ", plural: false, want: "Contact shared"},
		{name: "PluralNameAndPhone", dName: "Alice", phone: "+62 812", plural: true, want: "Contacts: Alice (+62 812)"},
		{name: "PluralEmpty", dName: "", phone: "", plural: true, want: "Contacts shared"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatContactSummary(tt.dName, tt.phone, tt.plural)
			if got != tt.want {
				t.Fatalf("FormatContactSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractMessageTextFromProtoContactMessage(t *testing.T) {
	phoneVCard := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Alice\r\nTEL;type=Mobile:\r\n +62 812 3456 7890\r\nEND:VCARD"

	tests := []struct {
		name string
		msg  *waE2E.Message
		want string
	}{
		{
			name: "NameAndPhone",
			msg: &waE2E.Message{
				ContactMessage: &waE2E.ContactMessage{
					DisplayName: strPtr("Alice"),
					Vcard:       strPtr(phoneVCard),
				},
			},
			want: "Contact: Alice (+62 812 3456 7890)",
		},
		{
			name: "NameOnly",
			msg: &waE2E.Message{
				ContactMessage: &waE2E.ContactMessage{
					DisplayName: strPtr("Alice"),
					Vcard:       strPtr(""),
				},
			},
			want: "Contact: Alice",
		},
		{
			name: "PhoneOnly",
			msg: &waE2E.Message{
				ContactMessage: &waE2E.ContactMessage{
					DisplayName: strPtr(""),
					Vcard:       strPtr(phoneVCard),
				},
			},
			want: "Contact: +62 812 3456 7890",
		},
		{
			name: "Neither",
			msg: &waE2E.Message{
				ContactMessage: &waE2E.ContactMessage{
					DisplayName: strPtr(""),
					Vcard:       strPtr(""),
				},
			},
			want: "Contact shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMessageTextFromProto(tt.msg)
			if got != tt.want {
				t.Fatalf("ExtractMessageTextFromProto() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractMessageTextFromProtoContactsArrayMessage(t *testing.T) {
	bob := "Bob"
	bobVcard := "BEGIN:VCARD\nVERSION:3.0\nFN:Bob\nTEL:+1 555 0100\nEND:VCARD"
	carol := "Carol"
	carolVcard := "BEGIN:VCARD\nVERSION:3.0\nFN:Carol\nTEL:+1 555 0200\nEND:VCARD"

	tests := []struct {
		name string
		msg  *waE2E.Message
		want string
	}{
		{
			name: "FirstContactWithNameAndPhone",
			msg: &waE2E.Message{
				ContactsArrayMessage: &waE2E.ContactsArrayMessage{
					Contacts: []*waE2E.ContactMessage{
						{DisplayName: &bob, Vcard: &bobVcard},
						{DisplayName: &carol, Vcard: &carolVcard},
					},
				},
			},
			want: "Contacts: Bob (+1 555 0100)",
		},
		{
			name: "EmptyContactsArray",
			msg: &waE2E.Message{
				ContactsArrayMessage: &waE2E.ContactsArrayMessage{Contacts: nil},
			},
			want: "Contacts shared",
		},
		{
			name: "FirstContactEmpty",
			msg: &waE2E.Message{
				ContactsArrayMessage: &waE2E.ContactsArrayMessage{
					Contacts: []*waE2E.ContactMessage{{}},
				},
			},
			want: "Contacts shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMessageTextFromProto(tt.msg)
			if got != tt.want {
				t.Fatalf("ExtractMessageTextFromProto() = %q, want %q", got, tt.want)
			}
		})
	}
}

func strPtr(value string) *string {
	return &value
}

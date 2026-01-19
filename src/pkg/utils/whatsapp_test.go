package utils

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

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

func TestValidateAndNormalizeJID_GroupJIDPassthrough(t *testing.T) {
	// Group JIDs should pass through without modification
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
	// Newsletter and other non-user JIDs should pass through
	jid := "120363123456789012@newsletter"
	result, err := ValidateAndNormalizeJID(nil, jid)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Server != "newsletter" {
		t.Errorf("got server %s, want newsletter", result.Server)
	}
}

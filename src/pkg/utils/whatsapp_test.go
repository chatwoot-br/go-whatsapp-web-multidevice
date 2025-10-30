package utils

import (
	"strings"
	"testing"
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

// TestDetermineMediaExtension_StripsMimeParameters tests that MIME type parameters
// are properly stripped before determining file extensions. This prevents issues like
// files being saved as "file.ogg; codecs=opus" instead of "file.ogg"
// See: docs/issues/ISSUE-002-MEDIA-FILENAME-MIME-POLLUTION.md
func TestDetermineMediaExtension_StripsMimeParameters(t *testing.T) {
	tests := []struct {
		name         string
		originalFile string
		mimeType     string
		expected     string
	}{
		{
			name:         "Audio with codec parameter (bug case)",
			originalFile: "",
			mimeType:     "audio/ogg; codecs=opus",
			expected:     ".oga", // Go's mime package returns .oga for audio/ogg
		},
		{
			name:         "Video with complex codec parameters",
			originalFile: "",
			mimeType:     "video/mp4; codecs=\"avc1.42E01E, mp4a.40.2\"",
			expected:     ".m4v", // Go's mime package returns .m4v for video/mp4
		},
		{
			name:         "Audio with multiple spaces around semicolon",
			originalFile: "",
			mimeType:     "audio/ogg ; codecs=opus",
			expected:     ".oga", // Go's mime package returns .oga for audio/ogg
		},
		{
			name:         "Image with charset parameter",
			originalFile: "",
			mimeType:     "image/webp; charset=binary",
			expected:     ".webp",
		},
		{
			name:         "Application with charset",
			originalFile: "",
			mimeType:     "application/pdf; charset=UTF-8",
			expected:     ".pdf",
		},
		{
			name:         "Simple MIME type without parameters (regression test)",
			originalFile: "",
			mimeType:     "image/jpeg",
			expected:     ".jpe", // Go's mime package returns .jpe for image/jpeg (first extension)
		},
		{
			name:         "Simple audio MIME without parameters",
			originalFile: "",
			mimeType:     "audio/mp3",
			expected:     ".mp3",
		},
		{
			name:         "Original filename takes precedence over MIME with parameters",
			originalFile: "document.pdf",
			mimeType:     "application/octet-stream; charset=UTF-8",
			expected:     ".pdf",
		},
		{
			name:         "M4A audio with codec",
			originalFile: "",
			mimeType:     "audio/mp4; codecs=mp4a.40.2",
			expected:     ".m4a", // Go's mime package returns .m4a for audio/mp4
		},
		{
			name:         "WebM with codec",
			originalFile: "",
			mimeType:     "video/webm; codecs=vp8",
			expected:     ".webm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineMediaExtension(tt.originalFile, tt.mimeType)
			if result != tt.expected {
				t.Errorf("determineMediaExtension(%q, %q) = %q, want %q",
					tt.originalFile, tt.mimeType, result, tt.expected)
			}
		})
	}
}

// TestDetermineMediaExtension_NoSemicolonInResult ensures that no MIME type,
// regardless of parameters, produces a file extension containing a semicolon.
// This is a critical safety check to prevent filename pollution.
func TestDetermineMediaExtension_NoSemicolonInResult(t *testing.T) {
	problematicMimeTypes := []string{
		"audio/ogg; codecs=opus",
		"audio/ogg; codecs=vorbis",
		"audio/mp4; codecs=mp4a.40.2",
		"video/mp4; codecs=\"avc1.42E01E\"",
		"video/mp4; codecs=\"avc1.42E01E, mp4a.40.2\"",
		"video/webm; codecs=vp8",
		"video/webm; codecs=\"vp8, vorbis\"",
		"application/pdf; charset=UTF-8",
		"image/webp; charset=binary",
		"text/plain; charset=utf-8",
		"audio/mpeg; rate=44100",
	}

	for _, mimeType := range problematicMimeTypes {
		t.Run(mimeType, func(t *testing.T) {
			ext := determineMediaExtension("", mimeType)
			if strings.Contains(ext, ";") {
				t.Errorf("Extension contains semicolon: %q (from MIME: %q)", ext, mimeType)
			}
			if ext != "" && !strings.HasPrefix(ext, ".") {
				t.Errorf("Extension doesn't start with dot: %q (from MIME: %q)", ext, mimeType)
			}
		})
	}
}

// TestDetermineMediaExtension_EmptyAndInvalidInputs tests edge cases
func TestDetermineMediaExtension_EmptyAndInvalidInputs(t *testing.T) {
	tests := []struct {
		name         string
		originalFile string
		mimeType     string
		expected     string
	}{
		{
			name:         "Empty MIME type",
			originalFile: "",
			mimeType:     "",
			expected:     "",
		},
		{
			name:         "Only semicolon",
			originalFile: "",
			mimeType:     ";",
			expected:     "",
		},
		{
			name:         "Only parameter no type",
			originalFile: "",
			mimeType:     "; codecs=opus",
			expected:     "",
		},
		{
			name:         "Invalid MIME type (no slash)",
			originalFile: "",
			mimeType:     "notavalidmimetype",
			expected:     "", // No slash means empty extension
		},
		{
			name:         "Original filename with no extension",
			originalFile: "noextension",
			mimeType:     "application/octet-stream",
			expected:     ".bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineMediaExtension(tt.originalFile, tt.mimeType)
			if result != tt.expected {
				t.Errorf("determineMediaExtension(%q, %q) = %q, want %q",
					tt.originalFile, tt.mimeType, result, tt.expected)
			}
		})
	}
}

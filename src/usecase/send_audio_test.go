package usecase

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/validations"
	"github.com/stretchr/testify/assert"
)

func TestProcessAudioForWhatsApp(t *testing.T) {
	// Setup test environment
	config.PathSendItems = os.TempDir()

	service := serviceSend{}

	t.Run("should return original audio when FFmpeg not available", func(t *testing.T) {
		// Test data
		originalBytes := []byte("fake audio data")
		originalMimeType := "audio/wav"

		// Mock FFmpeg not available by temporarily renaming it
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			// FFmpeg is available, so we'll test the path where it would convert
			t.Skip("FFmpeg is available, skipping test for missing FFmpeg scenario")
		}

		processedBytes, finalMimeType, deletedItems, err := service.processAudioForWhatsApp(originalBytes, originalMimeType)

		assert.NoError(t, err)
		assert.Equal(t, originalBytes, processedBytes)
		assert.Equal(t, originalMimeType, finalMimeType)
		assert.Empty(t, deletedItems)
	})

	t.Run("should return original audio for already optimal format", func(t *testing.T) {
		// Test AAC audio (already optimal format)
		audioBytes := []byte("fake AAC audio data")
		processedBytes, finalMimeType, deletedItems, err := service.processAudioForWhatsApp(audioBytes, "audio/aac")

		assert.NoError(t, err)
		assert.Equal(t, audioBytes, processedBytes)
		assert.Equal(t, "audio/aac", finalMimeType)

		// Clean up any created temp files
		for _, item := range deletedItems {
			os.Remove(item)
		}
	})

	t.Run("should handle conversion when FFmpeg is available", func(t *testing.T) {
		// Check if FFmpeg is available
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			t.Skip("FFmpeg not available, skipping conversion test")
		}

		// Create a minimal valid WAV file for testing
		wavData := createMinimalWAVData()
		originalMimeType := "audio/wav"

		processedBytes, finalMimeType, deletedItems, err := service.processAudioForWhatsApp(wavData, originalMimeType)

		// Should not error even if conversion fails (it falls back to original)
		assert.NoError(t, err)
		assert.NotEmpty(t, processedBytes)

		// If conversion succeeded, format should be AAC, otherwise original
		if finalMimeType == "audio/aac" {
			assert.Equal(t, "audio/aac", finalMimeType)
			assert.NotEqual(t, wavData, processedBytes) // Should be different after conversion
		} else {
			// Conversion failed, should fallback to original
			assert.Equal(t, originalMimeType, finalMimeType)
			assert.Equal(t, wavData, processedBytes)
		}

		// Should have temporary files to cleanup
		assert.NotEmpty(t, deletedItems)

		// Cleanup temporary files
		for _, file := range deletedItems {
			if _, err := os.Stat(file); err == nil {
				os.Remove(file)
			}
		}
	})
}

// createMinimalWAVData creates a minimal valid WAV file for testing
func createMinimalWAVData() []byte {
	// This creates a minimal WAV header for a very short audio file
	// It's not a complete audio file but enough for basic testing
	wav := []byte{
		// RIFF header
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x24, 0x00, 0x00, 0x00, // File size - 8
		0x57, 0x41, 0x56, 0x45, // "WAVE"

		// fmt subchunk
		0x66, 0x6d, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Subchunk size
		0x01, 0x00, // Audio format (PCM)
		0x01, 0x00, // Number of channels
		0x44, 0xac, 0x00, 0x00, // Sample rate (44100)
		0x88, 0x58, 0x01, 0x00, // Byte rate
		0x02, 0x00, // Block align
		0x10, 0x00, // Bits per sample

		// data subchunk
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x00, 0x00, 0x00, // Data size (0 for empty)
	}
	return wav
}

func TestAudioValidationWithSize(t *testing.T) {
	ctx := context.Background()
	audioURL := "https://example.com/audio.mp3"
	request := domainSend.AudioRequest{
		BaseRequest: domainSend.BaseRequest{
			Phone: "5521999999999",
		},
		AudioURL: &audioURL,
	}

	err := validations.ValidateSendAudio(ctx, request)
	assert.NoError(t, err, "Audio validation should pass for valid phone and audio URL")
}

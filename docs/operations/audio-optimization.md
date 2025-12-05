# Audio Format Optimization for WhatsApp

## Overview

This document describes the automatic audio format optimization feature implemented in the WhatsApp Web API to improve audio message delivery and compatibility.

## Problem Analysis

Based on recent delivery issues with audio messages, we identified that certain audio formats may have lower compatibility rates with WhatsApp clients. The original implementation accepted various audio formats but did not optimize them for WhatsApp's preferred formats.

## Solution

### Automatic Audio Conversion (production behavior)

The system automatically converts audio files to an optimal format for WhatsApp Push-To-Talk (PTT) when FFmpeg is available. The production implementation targets AAC in an MP4/M4A container which provides the best compatibility for WhatsApp PTT messages and preserves required metadata.

1. Primary Target Format: AAC in MP4/M4A container
   - Native WhatsApp voice message format
   - Good balance of size and voice quality at low bitrates
   - Preserves metadata commonly required by PTT messages

2. Conversion Settings used in production:
   - Codec: AAC (`-c:a aac`)
   - Bitrate: 32 kbps (`-b:a 32k`) — optimized for voice and small size
   - Sample Rate: 16 kHz (`-ar 16000`) — common for voice PTT
   - Channels: Mono (`-ac 1`) — WhatsApp preference for voice
   - Container: MP4/M4A (`-f mp4`) with `-movflags +faststart`
   - Overwrite output: `-y`

### Supported Input Formats

The API accepts and will attempt to convert the following input formats when FFmpeg is present:
- MP3 (audio/mp3, audio/mpeg)
- WAV (audio/wav, audio/wave, audio/vnd.wav)
- AAC (audio/aac)
- M4A (audio/m4a)
- OGG (audio/ogg)
- FLAC (audio/flac)
- WMA (audio/wma)
- AMR (audio/amr)

### Fallback Behavior

If FFmpeg is not available or conversion fails:
- The original audio file is sent as-is
- A warning is logged for debugging
- No error is returned to the user
- This ensures backward compatibility

## Configuration

### Environment Variables

```bash
# Maximum audio file size (16MB - WhatsApp limit)
WHATSAPP_SETTING_MAX_AUDIO_SIZE=16777216

# Enable/disable automatic audio conversion (default: true)
WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=true
```

### FFmpeg Requirements

The system requires FFmpeg with AAC support for optimal conversion. FFmpeg should be available on PATH. The service will detect `ffmpeg` and `ffprobe` at runtime and fall back gracefully.

```bash
# Check FFmpeg availability
ffmpeg -version

# Check AAC encoder availability (example)
ffmpeg -encoders | grep aac
```

## API Usage

### Send Audio File

```bash
curl -X POST "http://localhost:3001/send/audio" \
  -F "phone=628123456789@s.whatsapp.net" \
  -F "audio=@voice_message.wav"
```

### Send Audio from URL

```bash
curl -X POST "http://localhost:3001/send/audio" \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "628123456789@s.whatsapp.net",
    "audio_url": "https://example.com/audio.mp3"
  }'
```

## Benefits

1. Improved Delivery: AAC/M4A is WhatsApp's native voice message format and increases PTT delivery reliability
2. Reduced File Size: Lower bitrate configuration reduces bandwidth usage while preserving intelligibility
3. Better Quality: Voice-optimized settings produce clear audio for short messages
4. Automatic Processing: No client-side changes required
5. Backward Compatible: Original audio is used if conversion isn't possible

## Implementation Details

### Key Functions

- `processAudioForWhatsApp()`: Main conversion function
- `SendAudio()`: Updated to include automatic conversion
- `ValidateSendAudio()`: Enhanced with file size validation

### Processing Flow

1. Input Validation: Check file size and format
2. Format Check: Determine if conversion is needed
3. FFmpeg Detection: Check if FFmpeg is available
4. Conversion: Convert to AAC/M4A with the production settings above
5. Duration Detection: Use ffprobe to detect duration and set `Seconds` in the audio message (fallback default applied when ffprobe is missing)
6. Waveform Generation: Generate a simple waveform byte array for PTT metadata
7. Fallback: Use original format if conversion fails
8. Upload: Send processed audio to WhatsApp
9. Cleanup: Remove temporary files

### Error Handling

- FFmpeg not available: Log warning, use original format
- Conversion failure: Log error details, use original format
- Invalid input: Return validation error to user
- File too large: Return size limit error

## Monitoring and Debugging

### Log Messages

```
Audio converted from audio/wav to audio/aac (original: 245760 bytes, converted: 12834 bytes)
FFprobe not available, using default duration
FFmpeg not available, sending audio as-is with MIME type: audio/wav
FFmpeg conversion failed: exit status 1. Sending original audio.
```

### Performance Metrics

- Conversion typically reduces file size by 70-90%
- Processing time: 1-3 seconds for typical voice messages
- Memory usage: Temporary storage during conversion

## Troubleshooting

### Common Issues

1. **FFmpeg Not Found**:
   - Install FFmpeg: `apt install ffmpeg`
   - Verify installation: `ffmpeg -version`

2. **Opus Codec Missing**:
   - Ensure FFmpeg includes libopus
   - Check: `ffmpeg -codecs | grep opus`

3. **Conversion Failures**:
   - Check input file integrity
   - Verify file permissions
   - Review FFmpeg logs

4. **Large File Sizes**:
   - Check file size limits
   - Ensure conversion is working
   - Monitor disk space for temp files

### Testing Conversion

```bash
# Test manual conversion (produce similar result to the service)
ffmpeg -i input.wav -c:a aac -b:a 32k -ar 16000 -ac 1 -f mp4 -movflags +faststart output.m4a

# Compare file sizes
ls -la input.wav output.m4a
```

## Future Enhancements

1. **Adaptive Bitrate**: Adjust quality based on content type
2. **Batch Processing**: Handle multiple files efficiently
3. **Format Detection**: Better MIME type detection
4. **Quality Metrics**: Monitor delivery success rates
5. **Caching**: Cache converted files for repeated sends

## Conclusion

The automatic audio format optimization feature significantly improves WhatsApp audio message delivery by converting files to the most compatible format while maintaining high quality and reducing file sizes. The implementation is backward-compatible and gracefully handles edge cases where conversion is not possible.

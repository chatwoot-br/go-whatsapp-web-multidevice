# Media Handling Guide

Complete guide for handling media files (images, videos, audio, documents, stickers) in the WhatsApp Web API.

## Table of Contents

- [Overview](#overview)
- [Supported Formats](#supported-formats)
- [File Size Limits](#file-size-limits)
- [FFmpeg Configuration](#ffmpeg-configuration)
- [Image Handling](#image-handling)
- [Video Handling](#video-handling)
- [Audio Handling](#audio-handling)
- [Document Handling](#document-handling)
- [Sticker Handling](#sticker-handling)
- [Media Storage](#media-storage)
- [Troubleshooting](#troubleshooting)
- [Configuration](#configuration)
- [Examples](#examples)

## Overview

The WhatsApp Web API supports sending and receiving various media types with automatic processing, compression, and format conversion.

**Key Features:**
- Automatic image compression
- Video compression and optimization
- Audio format conversion
- Sticker generation from images
- Document file support
- Base64 and URL support
- View-once media support

**Processing Flow:**
```
User Upload → Validation → FFmpeg Processing → Compression → WhatsApp Upload → Delivery
```

## Supported Formats

### Images

| Format | Extension | Supported | Compression | Notes |
|--------|-----------|-----------|-------------|-------|
| JPEG | .jpg, .jpeg | ✅ | Yes | Most common format |
| PNG | .png | ✅ | Yes | Transparency preserved |
| WebP | .webp | ✅ | Yes | Modern format |
| GIF | .gif | ✅ | No | Animations preserved |
| BMP | .bmp | ⚠️ | Yes | Converted to JPEG |
| TIFF | .tiff, .tif | ⚠️ | Yes | Converted to JPEG |

**Recommended:** JPEG for photos, PNG for graphics with transparency

### Videos

| Format | Extension | Supported | Compression | Notes |
|--------|-----------|-----------|-------------|-------|
| MP4 | .mp4 | ✅ | Yes | H.264 codec recommended |
| AVI | .avi | ✅ | Yes | Converted to MP4 |
| MOV | .mov | ✅ | Yes | Converted to MP4 |
| MKV | .mkv | ✅ | Yes | Converted to MP4 |
| WebM | .webm | ✅ | Yes | Converted to MP4 |
| 3GP | .3gp | ✅ | Yes | Converted to MP4 |

**Recommended:** MP4 with H.264 video codec and AAC audio codec

### Audio

| Format | Extension | Supported | Conversion | Notes |
|--------|-----------|-----------|------------|-------|
| MP3 | .mp3 | ✅ | To Opus | Most common |
| AAC | .aac, .m4a | ✅ | To Opus | Apple format |
| OGG | .ogg | ✅ | To Opus | Opus codec ideal |
| WAV | .wav | ✅ | To Opus | Uncompressed |
| FLAC | .flac | ✅ | To Opus | Lossless |
| Opus | .opus | ✅ | No | Native WhatsApp format |

**Recommended:** Opus in OGG container (automatic conversion enabled by default)

### Documents

| Category | Extensions | Supported | Max Size |
|----------|------------|-----------|----------|
| PDF | .pdf | ✅ | 50MB |
| Office | .doc, .docx, .xls, .xlsx, .ppt, .pptx | ✅ | 50MB |
| Text | .txt, .csv, .rtf | ✅ | 50MB |
| Archive | .zip, .rar, .7z, .tar, .gz | ✅ | 50MB |
| Code | .html, .xml, .json, .js, .py, .java | ✅ | 50MB |
| Other | All other formats | ✅ | 50MB |

**Note:** Document MIME type is automatically detected and extension preserved.

### Stickers

| Format | Supported | Auto-Convert | Output |
|--------|-----------|--------------|--------|
| JPG/JPEG | ✅ | Yes | WebP 512x512 |
| PNG | ✅ | Yes | WebP 512x512 with transparency |
| WebP | ✅ | Resize only | WebP 512x512 |
| GIF | ✅ | First frame | WebP 512x512 |

**Requirements:**
- Automatically resized to 512x512 pixels
- Transparency preserved for PNG
- Background color for non-transparent images

## File Size Limits

### Default Limits

| Media Type | Default Limit | WhatsApp Limit | Configurable |
|------------|---------------|----------------|--------------|
| Images | 20MB | 20MB | Yes |
| Videos | 100MB | 100MB | Yes |
| Audio | 16MB | 16MB (WhatsApp) | Yes |
| Documents | 50MB | 100MB | Yes |
| Downloads | 500MB | - | Yes |

### Configure Limits

Set custom limits via environment variables:

```bash
# .env file
WHATSAPP_SETTING_MAX_IMAGE_SIZE=20971520      # 20MB (bytes)
WHATSAPP_SETTING_MAX_VIDEO_SIZE=104857600     # 100MB
WHATSAPP_SETTING_MAX_FILE_SIZE=52428800       # 50MB
WHATSAPP_SETTING_MAX_AUDIO_SIZE=16777216      # 16MB
WHATSAPP_SETTING_MAX_DOWNLOAD_SIZE=524288000  # 500MB
```

**Convert MB to Bytes:**
```bash
# 1 MB = 1,048,576 bytes
# 10 MB = 10,485,760 bytes
# 20 MB = 20,971,520 bytes
# 50 MB = 52,428,800 bytes
# 100 MB = 104,857,600 bytes
```

### Size Validation

Files exceeding limits are rejected with error:

```json
{
  "code": "ERROR",
  "message": "File size exceeds maximum limit of 20MB"
}
```

## FFmpeg Configuration

FFmpeg is required for media processing (compression, conversion, thumbnails).

### Install FFmpeg

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install ffmpeg

# Verify installation
ffmpeg -version
```

**macOS:**
```bash
brew install ffmpeg

# Verify installation
ffmpeg -version
```

**Windows:**
1. Download from [ffmpeg.org](https://ffmpeg.org/download.html#build-windows)
2. Extract to `C:\ffmpeg`
3. Add `C:\ffmpeg\bin` to PATH environment variable
4. Verify: `ffmpeg -version` in CMD

**Docker:**
FFmpeg is pre-installed in the official Docker images.

### Verify FFmpeg

```bash
# Check FFmpeg installation
ffmpeg -version

# Check codecs
ffmpeg -codecs | grep h264
ffmpeg -codecs | grep opus

# Check formats
ffmpeg -formats | grep mp4
ffmpeg -formats | grep ogg
```

### FFmpeg Not Found Error

**Problem:** Application fails with "FFmpeg not found" error

**Solution:**
```bash
# Check if FFmpeg is in PATH
which ffmpeg

# Add to PATH (Linux/macOS)
export PATH=$PATH:/usr/local/bin

# Or install FFmpeg
sudo apt install ffmpeg  # Ubuntu/Debian
brew install ffmpeg      # macOS
```

## Image Handling

### Send Image from URL

**Request:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/image.jpg",
    "caption": "Check out this image!",
    "compress": true,
    "view_once": false
  }'
```

**Parameters:**
- `phone` (required): Recipient phone number
- `image` (required): Image URL or base64 data
- `caption` (optional): Image caption text
- `compress` (optional): Enable compression (default: true)
- `view_once` (optional): View once mode (default: false)

### Send Image from File

**Request:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "caption=Beautiful photo!" \
  -F "image=@/path/to/image.jpg"
```

### Send Base64 Image

**Request:**
```bash
# Encode image to base64
base64_image=$(base64 -i image.jpg)

curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d "{
    \"phone\": \"5511999998888\",
    \"image\": \"data:image/jpeg;base64,$base64_image\",
    \"caption\": \"Base64 encoded image\"
  }"
```

### Image Compression

**Compression Settings:**
- **Quality:** 75% (balance between quality and size)
- **Max Width:** 1920px
- **Max Height:** 1920px
- **Format:** JPEG for photos, PNG for graphics

**Disable Compression:**
```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/image.jpg",
    "compress": false
  }'
```

**Compression Process:**
```
Original Image → FFmpeg → Resize (if needed) → Compress (75% quality) → Output
```

### View Once Images

Send images that can only be viewed once:

```bash
curl -X POST http://localhost:3000/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/secret.jpg",
    "caption": "View once only!",
    "view_once": true
  }'
```

**Notes:**
- Recipient can only view once
- Screenshot prevention (depends on device)
- Cannot be forwarded
- Not saved in chat history

### Image Best Practices

**✅ DO:**
- Use JPEG for photographs (smaller file size)
- Use PNG for graphics, logos, screenshots (transparency)
- Compress images before sending large batches
- Use appropriate resolution (1920x1920 max)
- Optimize images with tools like ImageOptim, TinyPNG

**❌ DON'T:**
- Send extremely large images (> 20MB)
- Send raw camera photos without compression
- Use BMP or TIFF formats (unnecessary large size)
- Disable compression for high-volume use cases

## Video Handling

### Send Video from URL

**Request:**
```bash
curl -X POST http://localhost:3000/send/video \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "video": "https://example.com/video.mp4",
    "caption": "Check out this video!",
    "compress": true,
    "view_once": false
  }'
```

### Send Video from File

**Request:**
```bash
curl -X POST http://localhost:3000/send/video \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "caption=Amazing video!" \
  -F "video=@/path/to/video.mp4"
```

### Video Compression

**Compression Settings:**
- **Video Codec:** H.264 (x264)
- **Audio Codec:** AAC
- **Container:** MP4
- **Video Bitrate:** 1000k (adjusts based on resolution)
- **Audio Bitrate:** 128k
- **Frame Rate:** 30fps max

**Compression Process:**
```
Original Video → FFmpeg → Transcode (H.264/AAC) → Compress → Generate Thumbnail → Output
```

**Disable Compression:**
```bash
curl -X POST http://localhost:3000/send/video \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "video": "https://example.com/video.mp4",
    "compress": false
  }'
```

### View Once Videos

```bash
curl -X POST http://localhost:3000/send/video \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "video": "https://example.com/secret.mp4",
    "view_once": true
  }'
```

### Video Best Practices

**✅ DO:**
- Use MP4 format with H.264 codec
- Compress videos before sending (reduce bitrate)
- Keep videos under 100MB
- Use 720p or 1080p resolution max
- Encode at 30fps or less

**❌ DON'T:**
- Send 4K videos (unnecessary large)
- Use uncommon codecs (HEVC, VP9)
- Send videos without compression for high volume
- Exceed 100MB file size

## Audio Handling

### Send Audio from URL

**Request:**
```bash
curl -X POST http://localhost:3000/send/audio \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "audio": "https://example.com/audio.mp3"
  }'
```

### Send Audio from File

**Request:**
```bash
curl -X POST http://localhost:3000/send/audio \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "audio=@/path/to/audio.mp3"
```

### Audio Conversion

**Auto-Convert to WhatsApp Format:**

By default, all audio is converted to Opus codec in OGG container (WhatsApp native format):

```bash
# Enable auto-conversion (default)
export WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=true
```

**Conversion Settings:**
- **Target Codec:** Opus
- **Container:** OGG
- **Bitrate:** 64k (optimized for voice)
- **Sample Rate:** 48kHz
- **Channels:** 1 (mono, voice) or 2 (stereo, music)

**Conversion Process:**
```
MP3/AAC/WAV → FFmpeg → Opus Encoding (64k) → OGG Container → Output
```

**Disable Auto-Conversion:**
```bash
export WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=false
```

### Voice Messages vs Audio Files

**Voice Message:**
- Appears as waveform in chat
- Plays inline
- Optimized for voice (mono, 64k)

**Audio File:**
- Appears as file attachment
- Downloads to play
- Can be stereo, higher bitrate

The API automatically handles this based on file characteristics.

### Audio Best Practices

**✅ DO:**
- Use Opus format for best compatibility
- Keep audio under 16MB (WhatsApp limit)
- Use 64k bitrate for voice, 128k for music
- Convert to Opus before sending large batches

**❌ DON'T:**
- Send uncompressed WAV files
- Use very high bitrates (320k unnecessary)
- Exceed 16MB file size
- Disable auto-conversion for production

## Document Handling

### Send Document from URL

**Request:**
```bash
curl -X POST http://localhost:3000/send/file \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "file": "https://example.com/document.pdf",
    "filename": "report.pdf"
  }'
```

### Send Document from File

**Request:**
```bash
curl -X POST http://localhost:3000/send/file \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "file=@/path/to/document.pdf"
```

### MIME Type Detection

The API automatically detects MIME types and preserves file extensions:

**Supported MIME Types:**
- **PDF:** `application/pdf`
- **Word:** `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- **Excel:** `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- **PowerPoint:** `application/vnd.openxmlformats-officedocument.presentationml.presentation`
- **Text:** `text/plain`, `text/csv`, `text/html`
- **Archive:** `application/zip`, `application/x-rar-compressed`
- **Generic:** `application/octet-stream` (for unknown types)

**Extension Preservation:**
Original file extensions are preserved to ensure proper handling on recipient's device.

### Document Best Practices

**✅ DO:**
- Use standard formats (PDF, DOCX, XLSX)
- Keep files under 50MB recommended
- Use descriptive filenames
- Compress archives before sending

**❌ DON'T:**
- Send executables (.exe, .bat, .sh) unless necessary
- Exceed 100MB (WhatsApp limit)
- Use obscure file formats without recipient's approval

## Sticker Handling

### Send Sticker

The API automatically converts images to WebP sticker format:

**Request:**
```bash
curl -X POST http://localhost:3000/send/sticker \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "sticker": "https://example.com/sticker.png"
  }'
```

### Send Sticker from File

**Request:**
```bash
curl -X POST http://localhost:3000/send/sticker \
  -H "Content-Type: multipart/form-data" \
  -F "phone=5511999998888" \
  -F "sticker=@/path/to/image.png"
```

### Sticker Conversion Process

**Automatic Processing:**
1. Load source image (JPG, PNG, GIF, WebP)
2. Resize to 512x512 pixels (maintaining aspect ratio)
3. Add background color for non-transparent images
4. Preserve transparency for PNG images
5. Convert to WebP format
6. Optimize for sticker use

**Supported Input Formats:**
- JPG/JPEG
- PNG (transparency preserved)
- WebP
- GIF (first frame only)

**Output:**
- Format: WebP
- Size: 512x512 pixels
- Transparent background (if source has transparency)
- White background (if source has no transparency)

### Sticker Best Practices

**✅ DO:**
- Use PNG with transparent background for best results
- Use images with clear, simple designs
- Pre-resize to 512x512 if possible
- Use high-contrast colors

**❌ DON'T:**
- Send photos as stickers (poor quality)
- Use very detailed images (hard to see at small size)
- Send animated GIFs expecting animation (only first frame used)

## Media Storage

### Storage Paths

**Local Storage Locations:**
```
/app/storages/                     # Database files
/app/statics/media/                # Uploaded media cache
/app/statics/images/qrcode/        # QR codes
```

**Docker Volumes:**
```yaml
volumes:
  - whatsapp-data:/app/storages     # Persistent data
  - whatsapp-media:/app/statics     # Media cache
```

### Media Caching

**Caching Behavior:**
- Downloaded media is cached temporarily
- Media is automatically cleaned up after sending
- Cache prevents re-downloading for retries

**Cache Configuration:**
```bash
# Media cache directory (default)
/app/statics/media/

# Set custom cache directory (not configurable currently)
```

### Media Cleanup

**Automatic Cleanup:**
- Temporary files cleaned after successful send
- Failed uploads cleaned after retries exhausted
- QR codes cleaned after login complete

**Manual Cleanup:**
```bash
# Clean temporary media
rm -rf /app/statics/media/*

# Clean old QR codes
find /app/statics/images/qrcode/ -mtime +1 -delete

# Docker cleanup
docker exec whatsapp-api rm -rf /app/statics/media/*
```

### Disk Space Management

**Monitor Disk Usage:**
```bash
# Check storage usage
du -sh /app/storages/
du -sh /app/statics/

# Docker volume usage
docker system df -v
```

**Free Up Space:**
```bash
# Disable chat storage if not needed
export WHATSAPP_CHAT_STORAGE=false

# Clear media cache
rm -rf /app/statics/media/*

# Vacuum SQLite database
sqlite3 /app/storages/whatsapp.db "VACUUM;"
```

## Troubleshooting

### Media Not Sending

**Problem:** Media files fail to send

**Common Causes and Solutions:**

**1. FFmpeg Not Installed:**
```bash
# Check FFmpeg
ffmpeg -version

# Install if missing
sudo apt install ffmpeg  # Ubuntu/Debian
brew install ffmpeg      # macOS
```

**2. File Size Too Large:**
```bash
# Check file size
ls -lh /path/to/file.mp4

# Compress before sending
ffmpeg -i input.mp4 -vcodec h264 -acodec aac -b:v 1000k output.mp4
```

**3. Unsupported Format:**
```bash
# Convert to supported format
ffmpeg -i input.avi -vcodec h264 -acodec aac output.mp4
```

**4. URL Not Accessible:**
```bash
# Test URL accessibility
curl -I https://example.com/image.jpg

# Use public URL, not localhost
```

**5. Permission Issues:**
```bash
# Check directory permissions
ls -la /app/statics/media/

# Fix permissions
sudo chmod 755 /app/statics/media/
sudo chown -R whatsapp:whatsapp /app/statics/
```

### Image Compression Issues

**Problem:** Images not being compressed properly

**Solutions:**

**1. Check FFmpeg:**
```bash
# Verify FFmpeg supports JPEG
ffmpeg -codecs | grep mjpeg
```

**2. Check Compression Setting:**
```bash
# Enable compression in request
curl -X POST .../send/image -d '{"compress": true, ...}'
```

**3. Manual Pre-Compression:**
```bash
# Compress with ImageMagick
convert input.jpg -quality 75 -resize 1920x1920\> output.jpg

# Compress with FFmpeg
ffmpeg -i input.jpg -q:v 5 output.jpg
```

### Video Processing Errors

**Problem:** Video fails to process or send

**Solutions:**

**1. Check Video Codec:**
```bash
# Check video details
ffprobe video.mp4

# Re-encode to H.264
ffmpeg -i input.mp4 -vcodec h264 -acodec aac output.mp4
```

**2. Reduce File Size:**
```bash
# Compress video
ffmpeg -i input.mp4 \
  -vcodec h264 -crf 28 \
  -acodec aac -b:a 128k \
  -vf scale=1280:-2 \
  output.mp4
```

**3. Check Duration:**
```bash
# Very long videos may timeout
# Split into smaller parts
ffmpeg -i long.mp4 -ss 00:00:00 -t 00:05:00 part1.mp4
ffmpeg -i long.mp4 -ss 00:05:00 -t 00:05:00 part2.mp4
```

### Audio Conversion Errors

**Problem:** Audio files fail to convert

**Solutions:**

**1. Check FFmpeg Opus Support:**
```bash
# Verify Opus codec
ffmpeg -codecs | grep opus

# Install libopus if missing
sudo apt install libopus-dev
```

**2. Manual Conversion:**
```bash
# Convert to Opus
ffmpeg -i input.mp3 -c:a libopus -b:a 64k output.ogg
```

**3. Disable Auto-Conversion:**
```bash
# If issues persist
export WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=false
```

### Sticker Conversion Issues

**Problem:** Stickers not displaying correctly

**Solutions:**

**1. Check Source Image:**
```bash
# Verify image format
file image.png

# Should be supported format (PNG, JPG, WebP, GIF)
```

**2. Pre-Resize Image:**
```bash
# Resize to 512x512
convert input.png -resize 512x512 -background white -flatten output.png
```

**3. Add Transparency:**
```bash
# Convert white background to transparent
convert input.png -transparent white output.png
```

### Memory Issues

**Problem:** Out of memory during media processing

**Solutions:**

**1. Increase Memory Limits:**
```bash
# Docker memory limit
docker run --memory="1g" ...

# Systemd resource limits
[Service]
MemoryLimit=1G
```

**2. Process Smaller Files:**
```bash
# Pre-compress media
ffmpeg -i large-video.mp4 -vcodec h264 -crf 28 compressed.mp4
```

**3. Disable Chat Storage:**
```bash
# Reduce memory usage
export WHATSAPP_CHAT_STORAGE=false
```

## Configuration

### Media Configuration Variables

```bash
# File Size Limits (bytes)
WHATSAPP_SETTING_MAX_IMAGE_SIZE=20971520      # 20MB
WHATSAPP_SETTING_MAX_VIDEO_SIZE=104857600     # 100MB
WHATSAPP_SETTING_MAX_FILE_SIZE=52428800       # 50MB
WHATSAPP_SETTING_MAX_AUDIO_SIZE=16777216      # 16MB
WHATSAPP_SETTING_MAX_DOWNLOAD_SIZE=524288000  # 500MB

# Audio Settings
WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=true      # Auto-convert to Opus
```

### Complete Media Configuration Example

```bash
# .env file
# File Size Limits
WHATSAPP_SETTING_MAX_IMAGE_SIZE=20971520
WHATSAPP_SETTING_MAX_VIDEO_SIZE=104857600
WHATSAPP_SETTING_MAX_FILE_SIZE=52428800
WHATSAPP_SETTING_MAX_AUDIO_SIZE=16777216
WHATSAPP_SETTING_MAX_DOWNLOAD_SIZE=524288000

# Audio Settings
WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=true

# Storage Paths (default, not configurable)
# /app/storages/
# /app/statics/media/
```

## Examples

### Complete Image Example

```bash
#!/bin/bash
# send-image.sh

# Send compressed JPEG
curl -X POST "http://localhost:3000/send/image" \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "image": "https://example.com/photo.jpg",
    "caption": "Beautiful sunset!",
    "compress": true,
    "view_once": false
  }'
```

### Complete Video Example

```bash
#!/bin/bash
# send-video.sh

# Send compressed MP4 video
curl -X POST "http://localhost:3000/send/video" \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "video": "https://example.com/video.mp4",
    "caption": "Check this out!",
    "compress": true,
    "view_once": false
  }'
```

### Batch Media Upload

```bash
#!/bin/bash
# batch-upload.sh

PHONE="5511999998888"
IMAGES_DIR="/path/to/images"

# Send all images in directory
for image in "$IMAGES_DIR"/*.jpg; do
    echo "Sending: $image"

    curl -X POST "http://localhost:3000/send/image" \
      -H "Content-Type: multipart/form-data" \
      -F "phone=$PHONE" \
      -F "image=@$image" \
      -F "compress=true"

    # Add delay to avoid rate limiting
    sleep 2
done
```

## Related Documentation

- **[Configuration Reference](../reference/configuration.md)** - Media configuration options
- **[First Message Guide](../getting-started/first-message.md)** - Media sending examples
- **[API Documentation](../openapi.yaml)** - Complete API reference
- **[Troubleshooting Guide](../getting-started/quick-start.md)** - Common issues

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-11-14

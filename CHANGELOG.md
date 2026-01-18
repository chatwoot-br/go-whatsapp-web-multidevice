# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v8.1.0+5] - 2026-01-17

### Fixed
- fix(utils): derive image extension from Content-Type for S3 URLs

## [v8.1.0+4] - 2026-01-15

### Added
- feat(whatsapp): enable full history sync and ON_DEMAND capability
- feat(whatsapp): handle unavailable messages from linked devices
- feat(whatsapp): process ON_DEMAND history sync responses
- feat: add logs directory to .gitignore and create .keep file

### Fixed
- fix(whatsapp): normalize chat_id from LID to phone number in webhook

### Changed
- chore: update dependencies for go.mau.fi/whatsmeow and golang.org/x packages

## [v8.1.0+3] - 2026-01-13

### Added
- feat(proxy): add SOCKS5/HTTP/HTTPS proxy support for WhatsApp connections
- feat(proxy): display external proxy IP in device card UI

### Fixed
- fix(webhook): include caption in payload when auto-downloading media (image, video, video_note)

## [v8.1.0+2] - 2026-01-08

### Added
- feat(cache): add short-term caching for info requests
- feat(webhook): update events list to include history_sync_complete and improve documentation

### Fixed
- fix(cache): cache error responses to prevent repeated API calls
- fix(send): use LID for message delivery with targeted approach
- Various CI workflow fixes for tag patterns and multi-arch builds

### Changed
- refactor(workflow): trigger Helm chart release on version tags only

## [v8.1.0+1] - 2025-01-07

### Added
- feat(helm): add gowa Helm chart for Kubernetes deployment
- feat(webhook): add chat_name to outgoing message payload
- feat(chat): add sender_name field for group message contacts
- feat(whatsapp): add history sync webhook notification
- feat(audio): add OGG Opus conversion for PTT voice notes
- feat(whatsapp): include is_from_me in webhook payload
- feat: add multi-device support guide documentation
- feat: add waveform generation for audio messages
- feat: enhance audio handling with MIME type resolution and duration retrieval

### Fixed
- fix(whatsapp): debounce history sync webhook to wait for all events
- fix(login): use background context for QR channel
- fix(device.go): Fix DeviceMiddleware to allow if APP_BASE_PATH is changed

### Changed
- Updated GitHub Actions workflows to support fork versioning (v8.1.0+1 format)
- Added chart-releaser workflow for Helm chart releases


# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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


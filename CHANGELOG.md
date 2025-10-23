# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v7.8.0] - 2025-10-23

### Added
- **Admin API**: Multi-instance management with Supervisor integration
  - Create and manage multiple WhatsApp instances dynamically
  - REST API for instance lifecycle management (start, stop, restart, delete)
  - Health check and status monitoring endpoints
  - Swagger UI documentation at `/admin/docs`
  - Supervisor integration for process management
  - Lock file management to prevent duplicate instances
  - Configurable instance directories and configuration
  - Support for per-instance settings (auth, webhooks, debug mode)

### Fixed
- **Webhook Forwarding**: Hardened webhook forwarding against partial failures (#434)
  - Improved error handling when multiple webhook URLs are configured
  - Better retry logic for individual webhook failures

### Changed
- Updated whatsmeow library to latest version
- Improved documentation across the project
  - Enhanced release process guide
  - Complete deployment guide with Docker, Kubernetes, and bare metal instructions
  - Comprehensive webhook payload documentation with integration examples
  - Updated CLAUDE.md with better versioning and development instructions
  - Added OpenAPI documentation for better API understanding
- Updated dependencies for improved stability and security

## [v7.7.1] - 2025-10-07

### Fixed
- **Critical**: Fixed service panic on profile picture fetch due to unsupported PrivacyToken payload type
  - Service would crash when attempting to fetch profile pictures from WhatsApp
  - Caused message loss in downstream systems (like Chatwoot) during service restart
  - Issue documented in `docs/issues/ISSUE-001-PROFILE-PICTURE-PANIC.md`

### Changed
- Updated `go.mau.fi/whatsmeow` from `v0.0.0-20251003111114-4479f300784e` to `v0.0.0-20251005083110-4fe97da162dc`
  - Includes fix for PrivacyToken support in GetProfilePictureInfo (PR#950)
  - Adds proper handling for privacy tokens in profile picture requests
- Updated `go.mau.fi/libsignal` from `v0.2.0` to `v0.2.1-0.20251004173110-6e0a3f2435ed`

### Documentation
- Added comprehensive issue documentation for profile picture panic
- Created release process documentation (`docs/RELEASE-PROCESS.md`)
- Created CHANGELOG.md to track version history
- Updated CLAUDE.md with release instructions

## [v7.7.0] - 2025-10-03

### Added
- **Sticker Support**: Automatic conversion of images to WebP sticker format
  - Supports JPG, JPEG, PNG, WebP, and GIF formats
  - Automatic resizing to 512x512 pixels
  - Preserves transparency for PNG images
- **Trusted Proxy Support**: Added support for proxy deployments
- **Document MIME Detection**: Improved document type detection and extension preservation

### Changed
- **Reconnect Error Handling**: Better session guard and error handling for LoginWithCode

### Fixed
- Improved error messages for reconnection failures
- Fixed issue with document uploads losing file extensions

## [v7.6.0] - 2025-09-15

### Added
- Enhanced webhook payload documentation
- Additional error handling for media uploads

### Changed
- Improved performance for large media files
- Updated dependencies for security patches

## [v7.5.1] - 2025-08-25

### Fixed
- Memory leak in group participant operations
- Issue with group invite link generation

## [v7.5.0] - 2025-08-20

### Added
- **Group Participants**: List and CSV export functionality
- **Media Download API**: On-demand media download with UI support
- **Group Invite Links**: API endpoint for generating/resetting invite links

### Changed
- **Memory Optimization**: Improved memory allocation in group operations
- Enhanced error messages for failed operations

## [v7.4.1] - 2025-07-30

### Fixed
- Issue with message timestamps in chat history
- Webhook retry logic not honoring backoff delays

## [v7.4.0] - 2025-07-15

### Added
- Chat history query endpoints
- Message search functionality
- Enhanced webhook event types

### Changed
- Improved database query performance
- Updated OpenAPI documentation

## [v7.3.1] - 2025-06-25

### Fixed
- Race condition in WebSocket connections
- Issue with multi-device synchronization

## [v7.3.0] - 2025-06-10

### Added
- Newsletter support (subscribe, unsubscribe, list)
- Business profile API endpoints
- Enhanced presence subscription

### Changed
- Refactored message handling for better reliability
- Updated webhook payload structure (backward compatible)

## [v7.2.1] - 2025-05-20

### Fixed
- Connection stability issues
- Memory leak in event handlers

## [v7.2.0] - 2025-05-05

### Added
- Auto-mark read functionality for incoming messages
- Enhanced media compression settings
- Configurable webhook retry logic

### Changed
- Improved reconnection logic
- Better error reporting for failed messages

## [v7.0.0] - 2025-04-01

### Added
- **MCP (Model Context Protocol) Server Support**: Complete integration for AI agents
  - SSE (Server-Sent Events) transport
  - Comprehensive tool suite for WhatsApp operations
  - Resource capabilities for data access
- **Admin API**: Multi-instance management with Supervisor integration
- Swagger UI for Admin API documentation

### Changed
- **Breaking**: Command structure changed to support REST and MCP modes
  - REST mode: `./whatsapp rest` (previously `./whatsapp`)
  - MCP mode: `./whatsapp mcp`
- **Breaking**: Minimum Go version: 1.24+
- Switched to goreleaser for binary builds
- Improved CLI help and documentation

### Migration Guide (v6 → v7)
If upgrading from v6.x:
1. Update command to run REST mode: `./whatsapp rest` instead of `./whatsapp`
2. For MCP mode: `./whatsapp mcp`
3. Update any startup scripts or systemd services
4. Verify Go version is 1.24 or higher

---

## Version History

- **v7.7.x**: Bug fixes and dependency updates
- **v7.6.x**: Media handling improvements
- **v7.5.x**: Group management features
- **v7.4.x**: Chat history and search
- **v7.3.x**: Newsletter and business features
- **v7.2.x**: Connection reliability improvements
- **v7.1.x**: Admin API enhancements
- **v7.0.x**: MCP support and architecture changes

## Links

- [Release Process](docs/RELEASE-PROCESS.md) - How to create a new release
- [GitHub Releases](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases) - Download releases
- [Issues](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues) - Report bugs
- [Deployment Guide](docs/deployment-guide.md) - Deployment instructions

---

**Note**: For a complete list of changes, see the [commit history](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/commits/main).

# Changelog

All notable changes to this project will be documented in this file.

This project follows [Semantic Versioning](https://semver.org/) with fork revision tracking (`vX.Y.Z+N`).

## [v7.11.0+3] - 2025-12-21

### Fork Changes
- feat(api): add LID fields to all chat-related endpoints for contact deduplication
  - Add `lid` field to ChatInfo, MessageInfo, PinChatResponse, SetDisappearingTimerResponse
  - Add `lid` field to MyListContactsResponseData and GetGroupRequestParticipantsResponse
  - Add LIDResolver calls in ListChats, GetChatMessages, PinChat, SetDisappearingTimer usecases
  - Add LID resolution to receipt, group, and delete webhook event handlers
  - MCP tools inherit LID fields automatically via shared usecases
- feat(ui): add LID columns to chat list, messages, and contacts screens
  - Add LID column to Chat List with search support
  - Add chat LID in header and sender LID per message in Chat Messages
  - Add JID and LID columns to Contacts with updated CSV export
- docs: add implementation plan for LID/JID fields in chat endpoints

### Versions
- Git tag: `v7.11.0+3`
- Docker tag: `v7.11.0-3` (+ converted to - for Docker compatibility)
- Helm chart: `7.11.3` (X.Y from upstream + N from fork rev)

---

## [v7.11.0+2] - 2025-12-19

### Fork Changes
- fix(chatstorage): resolve LID-JID chat split issue ([#484](https://github.com/aldinokemal/go-whatsapp-web-multidevice/issues/484))
  - Add LIDResolver service for bidirectional LID ↔ PN resolution
  - Use `@lid` as canonical JID format for internal chat storage
  - Auto-migrate existing `@s.whatsapp.net` chats to `@lid` format
  - Include both formats in webhook payload (`from_lid`, `chat_lid`) for backward compatibility
  - Add proactive LID resolution via GetUserInfo for new contacts
- fix(lid_resolver): prevent memory leak with background cache cleanup
- fix(chatstorage): prevent race condition with migration deduplication lock
- docs: add issue investigation documentation for LID-JID chat split

### Versions
- Git tag: `v7.11.0+2`
- Docker tag: `v7.11.0-2` (+ converted to - for Docker compatibility)
- Helm chart: `7.11.2` (X.Y from upstream + N from fork rev)

---

## [v7.11.0+1] - 2025-12-11 (Synced with upstream v7.11.0)

### Upstream Changes
- feat: add support for Push-To-Video (PTV) messages
- refactor: normalize JID handling and clean up unused code

### Fork Changes
- feat: add disappearing messages management for chats via new API endpoint and UI component
- feat: enhance group request participants with phone number and display name
- chore: bump API version to 6.13.1 and add phone number and display name fields

### Versions
- Git tag: `v7.11.0+1`
- Docker tag: `v7.11.0-1` (+ converted to - for Docker compatibility)
- Helm chart: `7.11.1` (X.Y from upstream + N from fork rev)

---

## [v7.10.1+3] - 2025-12-11

### Fork Changes
- fix(helm): use admin API endpoint for cleanup CronJob to avoid Multi-Attach errors
- feat(api): add media_path to chat messages API for external consumers

### Versions
- Git tag: `v7.10.1+3`
- Docker tag: `v7.10.1-3` (+ converted to - for Docker compatibility)
- Helm chart: `7.10.3` (bumped due to chart changes)

---

## [v7.10.1+2] - 2025-12-08

### Fork Changes
- fix(admin): clean up storage and logs when deleting instance
- feat(helm): add local k8s development values and fix deployment issues
- refactor(docker): align docker-compose with Helm chart architecture
- fix(whatsapp): eliminate duplicate media downloads

### Versions
- Git tag: `v7.10.1+2`
- Docker tag: `v7.10.1-2` (+ converted to - for Docker compatibility)
- Helm chart: `7.10.2` (bumped due to chart changes)

---

## [v7.10.1+1] - 2025-12-06 (Based on upstream v7.10.1)

First fork release with fork versioning scheme.

### Upstream Changes
- refactor: normalize JID handling and clean up unused code

### Fork Changes
- feat: add fork versioning strategy with upstream tracking
- feat: add /release slash command for version releases
- feat(helm): add cleanup CronJob for old file removal
- fix(helm): security hardening and architecture improvements
- fix(go.mod, go.sum): update dependencies and remove unused ones
- fix(admin): address ADR-0001 code review issues
- feat: add developer agent documentation with core responsibilities
- docs: comprehensive documentation updates

### Docker Tags
- Git tag: `v7.10.1+1`
- Docker tag: `v7.10.1-1` (+ converted to - for Docker compatibility)

---

## Previous Releases

For releases before fork versioning, see the [upstream repository](https://github.com/aldinokemal/go-whatsapp-web-multidevice/releases).

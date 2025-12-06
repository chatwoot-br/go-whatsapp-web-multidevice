# Changelog

All notable changes to this project will be documented in this file.

This project follows [Semantic Versioning](https://semver.org/) with fork revision tracking (`vX.Y.Z+N`).

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

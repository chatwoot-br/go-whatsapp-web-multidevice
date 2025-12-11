# Changelog

All notable changes to this project will be documented in this file.

This project follows [Semantic Versioning](https://semver.org/) with fork revision tracking (`vX.Y.Z+N`).

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

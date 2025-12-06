# Fork Versioning Strategy Plan

## Overview

Update the release process documentation to support fork-specific versioning that:
- Preserves upstream version visibility (e.g., `v7.10.1`)
- Adds fork revision tracking (e.g., `+1`, `+2`)
- Resets fork revision when syncing with upstream

## Chosen Version Format

```
v{MAJOR}.{MINOR}.{PATCH}+{FORK_REV}
```

**Examples:**
- `v7.10.1+1` - First fork release based on upstream v7.10.1
- `v7.10.1+2` - Second fork release (internal changes)
- `v7.11.0+1` - First fork release after syncing to upstream v7.11.0 (reset)

## Version Precedence

Per SemVer 2.0, build metadata (`+...`) is ignored for precedence:
- `v7.10.1` == `v7.10.1+1` == `v7.10.1+2` (for dependency resolution)
- This is acceptable since we control our own deployments

## Docker Tag Conversion

Docker tags don't support `+`, so GitHub Actions automatically converts:
- Git tag: `v7.10.1+1` → Docker tag: `v7.10.1-1`

## Files Modified

### 1. `docs/developer/release-process.md`

- New "Fork Versioning" section explaining the format
- Updated version examples throughout
- New section for "Upstream Sync Process"
- Modified release steps to include fork revision
- Fixed incorrect line numbers (18/24 → 8/11)

### 2. `.claude/commands/release.md`

- Modified version determination logic
- Support for both upstream sync releases and fork-only releases
- Updated file locations for version updates

### 3. `.github/workflows/build-docker-image.yaml`

- Added tag pattern for fork versions (`v7.10.1+1`)
- Auto-converts `+` to `-` for Docker tags

### 4. `.github/workflows/chart-releaser.yaml`

- Added tag pattern for fork versions

## Version Files (correct line numbers)

The following files contain version numbers:
- `src/config/settings.go` line 8: `AppVersion = "v7.10.1+1"`
- `charts/gowa/Chart.yaml` line 8: `version: 7.10.1`
- `charts/gowa/Chart.yaml` line 11: `appVersion: "v7.10.1+1"`

**Note on Helm Chart:** The `version` field follows chart versioning (keep as upstream version for chart compatibility). The `appVersion` field uses the full fork version format `v7.10.1+1`.

## Release Type Decision Tree

```
Is this release syncing with upstream?
├── YES: Upstream Sync Release
│   ├── Get upstream version (e.g., v7.11.0)
│   ├── Set fork revision to 1
│   └── Result: v7.11.0+1
│
└── NO: Fork-only Release
    ├── Keep current upstream version (e.g., v7.10.1)
    ├── Increment fork revision (+1 → +2)
    └── Result: v7.10.1+2
```

## CHANGELOG Format

```markdown
## [v7.10.1+2] - 2025-12-06

### Fork Changes
- Description of fork-specific change

---

## [v7.10.1+1] - 2025-12-05 (Synced with upstream v7.10.1)

### Upstream Changes
- Changes from upstream release

### Fork Changes
- Fork-specific additions
```

---

**Created**: 2025-12-06
**Status**: Implemented

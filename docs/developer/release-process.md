# Release Process Guide

This document describes the release process for go-whatsapp-web-multidevice.

## Version Numbering

We follow [Semantic Versioning](https://semver.org/) (MAJOR.MINOR.PATCH):

- **MAJOR** (v8.0.0): Breaking changes that require user intervention
- **MINOR** (v7.8.0): New features, backward compatible changes
- **PATCH** (v7.7.1): Bug fixes, security patches, backward compatible

### When to Bump Each Version

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Breaking API changes | MAJOR | Removing endpoints, changing request/response format |
| New features | MINOR | New API endpoints, new functionality |
| Bug fixes | PATCH | Fixing crashes, correcting behavior |
| Security fixes | PATCH | Vulnerability patches |
| Dependency updates (breaking) | MAJOR | New Go version requirement |
| Dependency updates (non-breaking) | PATCH | Library updates without API changes |

## Fork Versioning

This repository is a fork of [aldinokemal/go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice). We use a versioning scheme that tracks both upstream versions and our fork-specific changes.

### Version Format

```
v{MAJOR}.{MINOR}.{PATCH}+{FORK_REV}
```

Where:
- `{MAJOR}.{MINOR}.{PATCH}` - The upstream version we're based on
- `+{FORK_REV}` - Our fork revision number (1, 2, 3, ...)

### Examples

| Version | Meaning |
|---------|---------|
| `v7.10.1+1` | First fork release based on upstream v7.10.1 |
| `v7.10.1+2` | Second fork release (our changes on top of v7.10.1) |
| `v7.11.0+1` | First fork release after syncing to upstream v7.11.0 |

### Release Types

**Fork-only Release** (increment fork revision):
- Making our own bug fixes, features, or improvements
- `v7.10.1+1` → `v7.10.1+2`

**Upstream Sync Release** (reset fork revision to 1):
- Merging changes from upstream repository
- `v7.10.1+2` → `v7.11.0+1` (after syncing to upstream v7.11.0)

### Version Precedence

Per [SemVer 2.0](https://semver.org/), build metadata (`+...`) is ignored for precedence comparison. This means `v7.10.1` and `v7.10.1+1` are considered equal for dependency resolution. This is acceptable since we control our own deployments.

### Docker Tag Conversion

Docker tags don't support the `+` character, so GitHub Actions automatically converts `+` to `-`:

| Git Tag | Docker Tag |
|---------|------------|
| `v7.10.1+1` | `v7.10.1-1` |
| `v7.10.1+2` | `v7.10.1-2` |

When pulling Docker images, use the `-` format:
```bash
docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.10.1-1
```

## Files to Update

When releasing a new version, you must update these files:

### 1. Application Version (`src/config/settings.go`)

```go
var (
    AppVersion = "v7.10.1+1"  // Update this line
    // ... rest of config
)
```

**Location**: Line 8

### 2. Helm Chart Version (`charts/gowa/Chart.yaml`)

```yaml
# Chart version (upstream version, without 'v' prefix)
version: 7.10.1

# Application version (full fork version with 'v' prefix)
appVersion: "v7.10.1+1"
```

**Locations**: Lines 8 and 11

**Note**: The `version` field uses the upstream version (for Helm chart compatibility). The `appVersion` field uses the full fork version format.

### 3. Changelog (`CHANGELOG.md`)

Add a new section at the top following the format:

```markdown
## [v7.10.1+1] - 2025-12-06 (Based on upstream v7.10.1)

### Fork Changes
- Description of fork-specific changes

### Upstream Changes
- Changes inherited from upstream (if syncing)
```

For fork-only releases:

```markdown
## [v7.10.1+2] - 2025-12-07

### Fork Changes
- Description of fork-specific changes
```

## Release Steps

### Prerequisites

- Clean working directory (commit all changes)
- All tests passing: `cd src && go test ./...`
- Updated dependencies: `cd src && go mod tidy`

### Step-by-Step Process

#### 1. Determine Version Number

First, identify what type of release this is:

```bash
# Check current version
git describe --tags --abbrev=0

# Review changes since last release
git log $(git describe --tags --abbrev=0)..HEAD --oneline

# Check if upstream has new releases
git fetch upstream
git log upstream/main --oneline -5
```

**Determine release type:**
- **Upstream sync**: If merging upstream changes → use upstream version + reset fork rev to 1
- **Fork-only**: If only our changes → keep upstream version + increment fork rev

**Examples:**
- Current: `v7.10.1+1`, upstream sync to v7.11.0 → New: `v7.11.0+1`
- Current: `v7.10.1+1`, our own changes → New: `v7.10.1+2`

#### 2. Update Version Files

```bash
# Update src/config/settings.go
# Change line 8: AppVersion = "v7.10.1+1"

# Update charts/gowa/Chart.yaml
# Change line 8: version: 7.10.1 (upstream version only)
# Change line 11: appVersion: "v7.10.1+1" (full fork version)

# Update CHANGELOG.md
# Add new version section at the top
```

#### 3. Commit Version Bump

```bash
# Stage the version files
git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md

# Commit with conventional commit message
# For fork-only release:
git commit -m "chore: bump version to v7.10.1+2"

# For upstream sync release:
git commit -m "chore: sync upstream v7.11.0 and bump to v7.11.0+1"
```

#### 4. Create and Push Git Tag

```bash
# Create annotated tag
git tag -a v7.10.1+1 -m "Release v7.10.1+1"

# Push commits and tags
git push origin main
git push origin v7.10.1+1
```

#### 5. Verify Automated Builds

After pushing the tag, GitHub Actions will automatically:

1. **Build Docker Images** (`.github/workflows/build-docker-image.yaml`):
   - Monitor: https://github.com/chatwoot-br/go-whatsapp-web-multidevice/actions
   - Creates AMD64 and ARM64 images
   - Pushes to `ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.10.1-1` (note: `+` converted to `-`)
   - Updates `latest` tag

2. **Release Helm Chart** (`.github/workflows/chart-releaser.yaml`):
   - Monitor: https://github.com/chatwoot-br/go-whatsapp-web-multidevice/actions
   - Packages Helm chart
   - Creates GitHub release

Wait for both workflows to complete successfully (green checkmarks).

#### 6. Verify Release

```bash
# Check Docker image (note: + becomes - in Docker tags)
docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.10.1-1

# Verify version
docker run --rm ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.10.1-1 rest --version

# Check GitHub release
# Visit: https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases
```

#### 7. Update Release Notes (Optional)

Visit the GitHub release page and enhance the auto-generated release notes:

1. Go to https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases
2. Find your release
3. Click "Edit release"
4. Add detailed notes from CHANGELOG.md
5. Highlight breaking changes (if any)
6. Add migration instructions (if needed)

## Quick Reference Commands

```bash
# One-command release (after updating files) - fork-only release
git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md && \
git commit -m "chore: bump version to v7.10.1+2" && \
git tag -a v7.10.1+2 -m "Release v7.10.1+2" && \
git push origin main && \
git push origin v7.10.1+2

# One-command release - upstream sync release
git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md && \
git commit -m "chore: sync upstream v7.11.0 and bump to v7.11.0+1" && \
git tag -a v7.11.0+1 -m "Release v7.11.0+1" && \
git push origin main && \
git push origin v7.11.0+1
```

## Automated Release Script

For convenience, use the release script (if available):

```bash
# Make script executable
chmod +x scripts/release.sh

# Run release script
./scripts/release.sh v7.10.1+1
```

## Troubleshooting

### Build Failures

If GitHub Actions workflows fail:

1. **Check workflow logs**: Click on the failed workflow in Actions tab
2. **Common issues**:
   - Missing secrets (HCLOUD_TOKEN, PERSONAL_ACCESS_TOKEN)
   - Syntax errors in Chart.yaml
   - Docker build failures
3. **Fix and retry**:
   ```bash
   # Delete the tag locally and remotely
   git tag -d v7.10.1+1
   git push origin :refs/tags/v7.10.1+1

   # Fix the issue
   # Commit the fix

   # Recreate the tag
   git tag -a v7.10.1+1 -m "Release v7.10.1+1"
   git push origin main
   git push origin v7.10.1+1
   ```

### Version Mismatch

If version numbers don't match across files:

```bash
# Search for version references
grep -r "v7.10.1" src/config/settings.go charts/gowa/Chart.yaml

# Ensure consistency
# All three locations must have the same version number
```

### Helm Chart Issues

If Helm chart release fails:

```bash
# Validate Chart.yaml
helm lint charts/gowa

# Check for syntax errors
yamllint charts/gowa/Chart.yaml
```

## Release Checklist

Before releasing, ensure:

- [ ] All code changes are committed
- [ ] Tests pass: `cd src && go test ./...`
- [ ] Dependencies updated: `cd src && go mod tidy`
- [ ] Version updated in `src/config/settings.go`
- [ ] Version updated in `charts/gowa/Chart.yaml` (both `version` and `appVersion`)
- [ ] CHANGELOG.md updated with changes
- [ ] Commit message follows format: `chore: bump version to vX.Y.Z+N`
- [ ] Git tag created and pushed
- [ ] GitHub Actions workflows completed successfully
- [ ] Docker image pulled and tested
- [ ] GitHub release notes updated (optional)

## Rolling Back a Release

If you need to rollback a release:

```bash
# Delete the tag
git tag -d v7.10.1+1
git push origin :refs/tags/v7.10.1+1

# Delete the GitHub release
# Visit: https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases
# Click "Delete" on the release

# Revert the version bump commit
git revert HEAD
git push origin main
```

Note: Docker images and Helm charts cannot be automatically deleted. You'll need to manually deprecate them if necessary.

## Upstream Sync Process

When the upstream repository releases a new version:

### 1. Fetch and Review Upstream Changes

```bash
# Add upstream remote if not already added
git remote add upstream https://github.com/aldinokemal/go-whatsapp-web-multidevice.git

# Fetch upstream changes
git fetch upstream

# Review upstream changes
git log upstream/main --oneline -20

# Check upstream tags
git tag -l --sort=-v:refname | head -10
```

### 2. Merge Upstream Changes

```bash
# Create a sync branch
git checkout -b sync/upstream-v7.11.0

# Merge upstream main or specific tag
git merge upstream/main
# OR merge specific tag:
git merge v7.11.0

# Resolve any conflicts
# Test the merged changes
cd src && go test ./...
```

### 3. Update Version for Sync Release

Reset fork revision to 1 with the new upstream version:

- `src/config/settings.go`: `AppVersion = "v7.11.0+1"`
- `charts/gowa/Chart.yaml` line 8: `version: 7.11.0`
- `charts/gowa/Chart.yaml` line 11: `appVersion: "v7.11.0+1"`

### 4. Create Release

```bash
# Commit and tag
git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md
git commit -m "chore: sync upstream v7.11.0 and bump to v7.11.0+1"
git tag -a v7.11.0+1 -m "Release v7.11.0+1"

# Push
git push origin sync/upstream-v7.11.0
git push origin v7.11.0+1

# Create PR to merge sync branch to main (recommended)
```

## Hotfix Process

For urgent fixes that need immediate release:

1. **Create hotfix from latest tag**:
   ```bash
   git checkout -b hotfix/v7.10.1+2 v7.10.1+1
   ```

2. **Make the fix and commit**

3. **Follow normal release process**:
   - Increment fork revision: `v7.10.1+1` → `v7.10.1+2`
   - Update CHANGELOG.md
   - Commit, tag, and push

4. **Merge back to main**:
   ```bash
   git checkout main
   git merge hotfix/v7.10.1+2
   git push origin main
   ```

## Release Schedule

Recommended release schedule:

- **Patch releases**: As needed (bug fixes, security)
- **Minor releases**: Monthly (new features)
- **Major releases**: Quarterly or as needed (breaking changes)

## Related Documentation

- [CHANGELOG.md](../CHANGELOG.md) - Version history
- [GitHub Actions](.github/workflows/) - Automated build workflows
- [Helm Chart](charts/gowa/) - Kubernetes deployment chart
- [Deployment Guide](docs/deployment-guide.md) - Deployment instructions

---

**Last Updated**: 2025-12-06
**Upstream Version**: v7.10.1
**Current Fork Version**: v7.10.1+1
**Version Format**: `v{MAJOR}.{MINOR}.{PATCH}+{FORK_REV}`

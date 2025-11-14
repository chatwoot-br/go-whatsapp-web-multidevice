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

## Files to Update

When releasing a new version, you must update these files:

### 1. Application Version (`src/config/settings.go`)

```go
var (
    AppVersion = "v7.7.1"  // Update this line
    // ... rest of config
)
```

**Location**: Line 8

### 2. Helm Chart Version (`charts/gowa/Chart.yaml`)

```yaml
# Increment chart version (matches app version, without 'v' prefix)
version: 7.7.1

# Increment app version (with 'v' prefix)
appVersion: "v7.7.1"
```

**Locations**: Lines 18 and 24

### 3. Changelog (`CHANGELOG.md`)

Add a new section at the top following the format:

```markdown
## [v7.7.1] - 2025-10-07

### Fixed
- Brief description of bug fix

### Changed
- Brief description of changes

### Added
- Brief description of new features

### Security
- Brief description of security fixes
```

## Release Steps

### Prerequisites

- Clean working directory (commit all changes)
- All tests passing: `cd src && go test ./...`
- Updated dependencies: `cd src && go mod tidy`

### Step-by-Step Process

#### 1. Determine Version Number

Based on the changes since the last release:

```bash
# Check current version
git describe --tags --abbrev=0

# Review changes since last release
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

#### 2. Update Version Files

```bash
# Update src/config/settings.go
# Change line 8: AppVersion = "v7.7.1"

# Update charts/gowa/Chart.yaml
# Change line 18: version: 7.7.1
# Change line 24: appVersion: "v7.7.1"

# Update CHANGELOG.md
# Add new version section at the top
```

#### 3. Commit Version Bump

```bash
# Stage the version files
git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md

# Commit with conventional commit message
git commit -m "chore: bump version to v7.7.1"
```

#### 4. Create and Push Git Tag

```bash
# Create annotated tag
git tag -a v7.7.1 -m "Release v7.7.1"

# Push commits and tags
git push origin main
git push origin v7.7.1
```

#### 5. Verify Automated Builds

After pushing the tag, GitHub Actions will automatically:

1. **Build Docker Images** (`.github/workflows/build-docker-image.yaml`):
   - Monitor: https://github.com/chatwoot-br/go-whatsapp-web-multidevice/actions
   - Creates AMD64 and ARM64 images
   - Pushes to `ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.7.1`
   - Updates `latest` tag

2. **Release Helm Chart** (`.github/workflows/chart-releaser.yaml`):
   - Monitor: https://github.com/chatwoot-br/go-whatsapp-web-multidevice/actions
   - Packages Helm chart
   - Creates GitHub release

Wait for both workflows to complete successfully (green checkmarks).

#### 6. Verify Release

```bash
# Check Docker image
docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.7.1

# Verify version
docker run --rm ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v7.7.1 rest --version

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
# One-command release (after updating files)
git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md && \
git commit -m "chore: bump version to v7.7.1" && \
git tag -a v7.7.1 -m "Release v7.7.1" && \
git push origin main && \
git push origin v7.7.1
```

## Automated Release Script

For convenience, use the release script (if available):

```bash
# Make script executable
chmod +x scripts/release.sh

# Run release script
./scripts/release.sh v7.7.1
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
   git tag -d v7.7.1
   git push origin :refs/tags/v7.7.1

   # Fix the issue
   # Commit the fix

   # Recreate the tag
   git tag -a v7.7.1 -m "Release v7.7.1"
   git push origin main
   git push origin v7.7.1
   ```

### Version Mismatch

If version numbers don't match across files:

```bash
# Search for version references
grep -r "v7.7.0" src/config/settings.go charts/gowa/Chart.yaml

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
- [ ] Commit message follows format: `chore: bump version to vX.Y.Z`
- [ ] Git tag created and pushed
- [ ] GitHub Actions workflows completed successfully
- [ ] Docker image pulled and tested
- [ ] GitHub release notes updated (optional)

## Rolling Back a Release

If you need to rollback a release:

```bash
# Delete the tag
git tag -d v7.7.1
git push origin :refs/tags/v7.7.1

# Delete the GitHub release
# Visit: https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases
# Click "Delete" on the release

# Revert the version bump commit
git revert HEAD
git push origin main
```

Note: Docker images and Helm charts cannot be automatically deleted. You'll need to manually deprecate them if necessary.

## Hotfix Process

For urgent fixes that need immediate release:

1. **Create hotfix from latest tag**:
   ```bash
   git checkout -b hotfix/v7.7.1 v7.7.0
   ```

2. **Make the fix and commit**

3. **Follow normal release process**:
   - Update version to v7.7.1 (patch bump)
   - Update CHANGELOG.md
   - Commit, tag, and push

4. **Merge back to main**:
   ```bash
   git checkout main
   git merge hotfix/v7.7.1
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

**Last Updated**: 2025-10-07
**Current Version**: v7.7.0
**Next Version**: v7.7.1

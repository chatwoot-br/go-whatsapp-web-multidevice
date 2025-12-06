Create a new release for this project. This is a fork, so we use the version format `vX.Y.Z+N` where X.Y.Z is the upstream version and N is our fork revision.

Follow these steps:

1. **Determine release type** by reviewing changes:
   - Run `git describe --tags --abbrev=0` to get current version
   - Run `git log $(git describe --tags --abbrev=0)..HEAD --oneline` to see changes
   - Check upstream for new releases: `git fetch upstream && git log upstream/main --oneline -5`
   - Ask me: **Upstream sync** (syncing with upstream) or **Fork-only** (our own changes)?

   **Version calculation:**
   - **Upstream sync**: Use new upstream version + reset fork rev to 1 (e.g., `v7.11.0+1`)
   - **Fork-only**: Keep upstream version + increment fork rev (e.g., `v7.10.1+1` → `v7.10.1+2`)

2. **Run pre-release checks**:
   - Run tests: `cd src && go test ./...`
   - Run `cd src && go mod tidy` to ensure dependencies are clean
   - Verify working directory is clean with `git status`

3. **Update version in these files** (use the new version number):
   - `src/config/settings.go` line 8: `AppVersion = "vX.Y.Z+N"`
   - `charts/gowa/Chart.yaml` line 8: `version: X.Y.Z` (upstream version, without 'v' or fork rev)
   - `charts/gowa/Chart.yaml` line 11: `appVersion: "vX.Y.Z+N"` (full fork version)

4. **Update CHANGELOG.md** at the top with the new version section:

   For upstream sync:
   ```markdown
   ## [vX.Y.Z+1] - YYYY-MM-DD (Synced with upstream vX.Y.Z)

   ### Upstream Changes
   - List changes from upstream

   ### Fork Changes
   - Our additions (if any)
   ```

   For fork-only:
   ```markdown
   ## [vX.Y.Z+N] - YYYY-MM-DD

   ### Fork Changes
   - Description of changes
   ```

5. **Commit and tag**:
   ```bash
   git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md
   # For fork-only:
   git commit -m "chore: bump version to vX.Y.Z+N"
   # For upstream sync:
   git commit -m "chore: sync upstream vX.Y.Z and bump to vX.Y.Z+1"
   git tag -a vX.Y.Z+N -m "Release vX.Y.Z+N"
   ```

6. **Show me the commands to push** (do not push automatically):
   ```bash
   git push origin <current-branch>
   git push origin vX.Y.Z+N
   ```

Reference: docs/developer/release-process.md

Create a new release for this project. Follow these steps:

1. **Determine version type** by reviewing changes since last tag:
   - Run `git describe --tags --abbrev=0` to get current version
   - Run `git log $(git describe --tags --abbrev=0)..HEAD --oneline` to see changes
   - Ask me what type of release: MAJOR (breaking), MINOR (features), or PATCH (fixes)

2. **Run pre-release checks**:
   - Run tests: `cd src && go test ./...`
   - Run `cd src && go mod tidy` to ensure dependencies are clean
   - Verify working directory is clean with `git status`

3. **Update version in these files** (use the new version number):
   - `src/config/settings.go` line 8: `AppVersion = "vX.Y.Z"`
   - `charts/gowa/Chart.yaml` line 8: `version: X.Y.Z` (without 'v')
   - `charts/gowa/Chart.yaml` line 11: `appVersion: "vX.Y.Z"`

4. **Update CHANGELOG.md** at the top with the new version section following the existing format

5. **Commit and tag**:
   ```bash
   git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md
   git commit -m "chore: bump version to vX.Y.Z"
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   ```

6. **Show me the commands to push** (do not push automatically):
   ```bash
   git push origin <current-branch>
   git push origin vX.Y.Z
   ```

Reference: docs/developer/release-process.md

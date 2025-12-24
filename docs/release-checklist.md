# Release Checklist

This document describes the release process for secretctl.

## Pre-Release Checklist

### 1. Code Quality

- [ ] All CI jobs passing (GitHub Actions)
- [ ] E2E tests passing (Playwright)
- [ ] Test coverage meets targets (crypto: 90%+, vault: 80%+)
- [ ] No critical security vulnerabilities (`gosec`, `govulncheck`)

```bash
# Run all checks
go test ./...
golangci-lint run
gosec ./...
govulncheck ./...
```

### 2. Documentation

- [ ] CHANGELOG.md updated with version entry
- [ ] README.md reflects new features
- [ ] Website documentation updated (if applicable)
- [ ] CLI help text accurate (`secretctl --help`)

### 3. Version Bump

- [ ] Version updated in `internal/mcp/server.go`
- [ ] CHANGELOG.md `[Unreleased]` moved to version section
- [ ] Git tag matches version number

```bash
# Check version consistency
grep -r "Version" internal/mcp/server.go
head -20 CHANGELOG.md
```

## Release Process

### 1. Prepare Release Branch

```bash
# Ensure main is up to date
git checkout main
git pull origin main

# Create release branch (optional for minor releases)
git checkout -b release/v0.x.0
```

### 2. Update Version

```bash
# Update version in code
# Edit internal/mcp/server.go: Version: "0.x.0"

# Update CHANGELOG.md
# Move [Unreleased] content to [0.x.0] - YYYY-MM-DD
```

### 3. Create Release Commit

```bash
git add -A
git commit -m "chore: prepare release v0.x.0"
git push origin main  # or release branch
```

### 4. Create Tag

```bash
git tag -a v0.x.0 -m "Release v0.x.0"
git push origin v0.x.0
```

### 5. GitHub Release

GoReleaser automatically creates the release when tag is pushed.

Manual steps if needed:
1. Go to GitHub Releases
2. Click "Draft a new release"
3. Select the tag
4. Copy CHANGELOG entry to release notes
5. Upload artifacts (if not automated)
6. Publish release

## Release Artifacts

### CLI Binaries (5 platforms)

| OS | Arch | Filename |
|----|------|----------|
| Linux | amd64 | `secretctl-linux-amd64` |
| Linux | arm64 | `secretctl-linux-arm64` |
| macOS | amd64 | `secretctl-darwin-amd64` |
| macOS | arm64 | `secretctl-darwin-arm64` |
| Windows | amd64 | `secretctl-windows-amd64.exe` |

### Desktop App (3 platforms)

| OS | Filename | Notes |
|----|----------|-------|
| macOS | `secretctl-desktop-macos.zip` | Universal binary |
| Windows | `secretctl-desktop-windows.exe` | Installer |
| Linux | `secretctl-desktop-linux.AppImage` | Portable |

### Checksums

```bash
# Generate checksums
sha256sum secretctl-* > checksums.txt

# Verify downloads
sha256sum -c checksums.txt
```

## Post-Release Verification

### Smoke Tests

#### CLI (each platform)

```bash
# Basic workflow
./secretctl init
echo "test-value" | ./secretctl set TEST_KEY
./secretctl get TEST_KEY
./secretctl list
./secretctl delete TEST_KEY

# New features (v0.6.0)
echo "KEY=value" > test.env
./secretctl import test.env --dry-run
./secretctl backup -o backup.enc
./secretctl restore backup.enc --verify-only
```

#### Desktop App (each platform)

1. Launch application
2. Create new vault OR unlock existing
3. View secret list
4. Add a secret
5. View/copy secret
6. Delete secret
7. Lock vault
8. Verify auto-lock timeout works

### Rollback Procedure

If critical issues are found:

1. **Remove Release** (if not downloaded yet)
   ```bash
   # Delete GitHub release (draft it first)
   gh release delete v0.x.0 --yes
   git push --delete origin v0.x.0
   git tag -d v0.x.0
   ```

2. **Patch Release** (if already distributed)
   ```bash
   # Fix the issue
   git checkout main
   # ... make fixes ...
   git commit -m "fix: critical issue"
   git tag -a v0.x.1 -m "Hotfix release"
   git push origin v0.x.1
   ```

3. **User Communication**
   - Create GitHub Issue describing the problem
   - Update release notes with known issues
   - Post in Discussions if applicable

## Version History

| Version | Date | Highlights |
|---------|------|------------|
| v0.6.0 | TBD | backup/restore, import command |
| v0.5.0 | 2025-12-08 | Audit Log Viewer |
| v0.4.1 | 2025-12-08 | Desktop App Secret CRUD |
| v0.4.0 | 2025-12-06 | Desktop App |
| v0.3.0 | 2025-12-05 | MCP Server |
| v0.2.0 | 2025-12-03 | run, export, generate commands |
| v0.1.0 | 2025-12-01 | Initial release |

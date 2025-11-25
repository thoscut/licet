# Release Guide

This document explains how to create releases for the Licet project using GitHub Actions.

## Automated Release Process

The project uses GitHub Actions to automatically build and release binaries for multiple platforms when you push a version tag.

### Supported Platforms

The release workflow builds for the following platforms:

| Platform | Architecture | Binary Name |
|----------|-------------|-------------|
| Linux | AMD64 | `licet-linux-amd64` |
| Linux | ARM64 | `licet-linux-arm64` |
| Linux | ARMv7 | `licet-linux-armv7` |
| macOS | AMD64 (Intel) | `licet-darwin-amd64` |
| macOS | ARM64 (M1/M2/M3) | `licet-darwin-arm64` |
| Windows | AMD64 | `licet-windows-amd64.exe` |

## Creating a New Release

### 1. Prepare the Release

Before creating a release, ensure:

```bash
# Make sure all changes are committed
git status

# Run tests locally
make test

# Build locally to verify
make build

# Test the build
./build/licet --help
```

### 2. Update Version Information

If you have version information in code (e.g., in `main.go`), update it:

```go
// cmd/server/main.go
var Version = "1.0.0"  // Update this
```

### 3. Create and Push a Version Tag

The release workflow triggers on tags matching the pattern `v*.*.*` (semantic versioning).

```bash
# Create a version tag (example: v1.0.0)
git tag -a v1.0.0 -m "Release version 1.0.0

- Feature 1
- Feature 2
- Bug fixes
"

# Push the tag to GitHub
git push origin v1.0.0
```

### 4. Monitor the Build

1. Go to your repository on GitHub
2. Navigate to **Actions** tab
3. Watch the "Build and Release" workflow
4. The workflow will:
   - Build binaries for all platforms
   - Create archives (.tar.gz for Unix, .zip for Windows)
   - Generate SHA256 checksums
   - Create a GitHub release
   - Upload all assets to the release

### 5. Verify the Release

Once the workflow completes:

1. Go to the **Releases** section of your repository
2. Verify all binaries are present
3. Test download and execution of at least one binary

## Release Workflow Details

### Workflow Triggers

The release workflow (`.github/workflows/release.yml`) runs when:
- A tag matching `v*.*.*` is pushed (e.g., `v1.0.0`, `v2.1.3`)

### Build Process

For each platform, the workflow:

1. **Checks out** the code
2. **Sets up Go** 1.21
3. **Downloads** dependencies
4. **Builds** the binary with:
   - Version information embedded via `-ldflags`
   - Stripped symbols (`-s -w`) for smaller binaries
5. **Creates archives**:
   - `.tar.gz` for Linux/macOS
   - `.zip` for Windows
6. **Generates SHA256** checksums for verification
7. **Uploads** to GitHub release

### Build Flags

Binaries are built with these optimizations:

```bash
go build -ldflags="-s -w -X main.Version=${VERSION}" \
  -o build/licet-platform-arch \
  ./cmd/server
```

- `-s`: Strip debug symbols
- `-w`: Strip DWARF debugging information
- `-X main.Version=${VERSION}`: Embed version string

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (1.0.0 → 2.0.0): Incompatible API changes
- **MINOR** version (1.0.0 → 1.1.0): New functionality (backwards compatible)
- **PATCH** version (1.0.0 → 1.0.1): Bug fixes (backwards compatible)

### Examples

```bash
# First stable release
git tag -a v1.0.0 -m "First stable release"

# Bug fix release
git tag -a v1.0.1 -m "Fix license parsing bug"

# New feature release
git tag -a v1.1.0 -m "Add RLM parser support"

# Breaking change release
git tag -a v2.0.0 -m "New API design (breaking changes)"
```

## Pre-releases

For beta or release candidate versions:

```bash
# Beta release
git tag -a v1.1.0-beta.1 -m "Beta release for testing"

# Release candidate
git tag -a v2.0.0-rc.1 -m "Release candidate"
```

The workflow will mark these as "pre-release" in GitHub.

## Continuous Integration

The project also has a continuous integration workflow (`.github/workflows/build.yml`) that:

- **Runs on every push** to main and claude/** branches
- **Runs on every pull request** to main
- **Tests** the code with `go test`
- **Checks formatting** with `go fmt`
- **Runs linting** with `go vet`
- **Builds** for all platforms to catch build errors early
- **Tests Docker** image builds

## Manual Release (Without GitHub Actions)

If you need to create a release manually:

```bash
# Use the Makefile
make build-all

# This creates binaries in build/:
# - licet-linux-amd64
# - licet-linux-arm64
# - licet-darwin-amd64
# - licet-darwin-arm64
# - licet-windows-amd64.exe

# Create archives
cd build
tar -czf licet-linux-amd64.tar.gz licet-linux-amd64
tar -czf licet-darwin-amd64.tar.gz licet-darwin-amd64
# ... etc

# Generate checksums
sha256sum *.tar.gz > checksums.txt
```

## Rollback a Release

If you need to remove a release:

1. Delete the release in GitHub UI
2. Delete the tag:
   ```bash
   # Delete local tag
   git tag -d v1.0.0

   # Delete remote tag
   git push origin :refs/tags/v1.0.0
   ```

## Troubleshooting

### Workflow Fails to Build

1. Check the Actions tab for error logs
2. Test build locally: `make build-all`
3. Ensure `go.mod` is up to date: `go mod tidy`

### Release Not Created

1. Verify tag format matches `v*.*.*`
2. Check workflow permissions in Settings → Actions
3. Ensure GITHUB_TOKEN has `contents: write` permission

### Binary Won't Run

**Linux/macOS:**
```bash
# Make executable
chmod +x licet-linux-amd64

# Check file type
file licet-linux-amd64

# Run
./licet-linux-amd64
```

**Windows:**
- Ensure Windows Defender isn't blocking it
- Run from PowerShell/Command Prompt

### Checksum Verification Fails

```bash
# Download both the binary archive and .sha256 file
# Verify on Linux/macOS:
sha256sum -c licet-linux-amd64.tar.gz.sha256

# Verify on Windows (PowerShell):
(Get-FileHash licet-windows-amd64.exe.zip).Hash -eq (Get-Content licet-windows-amd64.exe.zip.sha256).Split()[0]
```

## Best Practices

1. **Test before tagging**: Always run `make test` and `make build` locally
2. **Semantic versioning**: Follow semver for version numbers
3. **Changelog**: Maintain a CHANGELOG.md with release notes
4. **Tag messages**: Write descriptive tag messages
5. **Pre-releases**: Use beta/rc tags for testing before stable releases
6. **Security**: Review dependencies before major releases
7. **Documentation**: Update README.md with new features

## Example Release Workflow

Complete example of creating a release:

```bash
# 1. Ensure you're on main branch with latest changes
git checkout main
git pull origin main

# 2. Run tests
make test

# 3. Update version in code if needed
vim cmd/server/main.go  # Update Version variable

# 4. Commit version change
git add cmd/server/main.go
git commit -m "Bump version to 1.0.0"
git push origin main

# 5. Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0

New Features:
- FlexLM parser fully implemented
- RLM parser added
- Web dashboard with Bootstrap UI
- RESTful API with 7 endpoints

Improvements:
- 5x faster than PHP version
- Single binary deployment
- Docker support

Bug Fixes:
- Fixed license expiration parsing
- Corrected user checkout detection
"

git push origin v1.0.0

# 6. Monitor GitHub Actions
# Visit: https://github.com/thoscut/licet/actions

# 7. Verify release
# Visit: https://github.com/thoscut/licet/releases
```

## Resources

- [Semantic Versioning](https://semver.org/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Build Modes](https://golang.org/cmd/go/#hdr-Build_modes)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)

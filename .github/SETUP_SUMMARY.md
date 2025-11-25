# GitHub Actions CI/CD Setup Summary

This repository is now configured with automated building, testing, and releasing using GitHub Actions.

## 🚀 What's Been Set Up

### 1. **Continuous Integration** (`.github/workflows/build.yml`)

Runs on every push and pull request to ensure code quality:

- ✅ **Automated testing** with race detection
- ✅ **Code formatting** checks (`go fmt`)
- ✅ **Static analysis** (`go vet`)
- ✅ **Multi-platform builds** (Linux, macOS, Windows)
- ✅ **Docker image** build verification
- ✅ **Coverage reporting** (optional Codecov integration)

**Triggers:**
- Push to `main` branch
- Push to any `claude/**` branch
- Pull requests to `main`

### 2. **Automated Releases** (`.github/workflows/release.yml`)

Automatically builds and publishes releases when you create a version tag:

- 🏗️ **Builds for 6 platforms:**
  - Linux (AMD64, ARM64, ARMv7)
  - macOS (Intel, Apple Silicon)
  - Windows (AMD64)

- 📦 **Creates release artifacts:**
  - Compressed archives (.tar.gz, .zip)
  - SHA256 checksums for verification
  - Automatic release notes

- 🎯 **Publishes to GitHub Releases** automatically

**Trigger:**
- Push a tag matching `v*.*.*` (e.g., `v1.0.0`)

## 📋 Quick Start

### Create Your First Release

```bash
# 1. Make sure all changes are committed
git add .
git commit -m "Ready for v1.0.0"
git push

# 2. Create and push a version tag
git tag -a v1.0.0 -m "First release"
git push origin v1.0.0

# 3. Watch the magic happen!
# Go to: https://github.com/thoscut/licet/actions
```

Within a few minutes, you'll have:
- ✅ 6 platform-specific binaries
- ✅ A GitHub release with all downloads
- ✅ SHA256 checksums for security
- ✅ Automatic release notes

## 📖 Documentation

- **[RELEASE_GUIDE.md](.github/workflows/RELEASE_GUIDE.md)** - Complete release process documentation
- **[build.yml](.github/workflows/build.yml)** - CI workflow configuration
- **[release.yml](.github/workflows/release.yml)** - Release workflow configuration

## 🔍 What Happens on Each Push

When you push code or create a PR:

1. **Tests run** across all platforms
2. **Code is checked** for formatting issues
3. **Builds are verified** for Linux, macOS, Windows
4. **Docker image** builds are tested
5. You get **instant feedback** on any issues

## 🎁 What Happens on Tag Push

When you push a version tag (e.g., `v1.0.0`):

1. **Code is checked out** at that tag
2. **6 binaries are built** in parallel
3. **Archives are created** (.tar.gz for Unix, .zip for Windows)
4. **Checksums are generated** (SHA256)
5. **GitHub Release is created** with:
   - Release notes
   - Download links
   - Installation instructions
   - All binary artifacts

## 🛠️ Customization

### Add More Platforms

Edit `.github/workflows/release.yml` and add to the matrix:

```yaml
- goos: linux
  goarch: 386
  output_name: licet-linux-386
```

### Change Go Version

Update in both workflow files:

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.22'  # Change this
```

### Add Pre-release Support

Tags like `v1.0.0-beta.1` will automatically be marked as pre-releases.

## 📊 Monitoring

### View Build Status

Badge for README.md:
```markdown
![Build Status](https://github.com/thoscut/licet/actions/workflows/build.yml/badge.svg)
![Release Status](https://github.com/thoscut/licet/actions/workflows/release.yml/badge.svg)
```

### Check Workflow Runs

Visit: `https://github.com/thoscut/licet/actions`

## 🔒 Security

- **Checksums included** for all downloads
- **Binaries stripped** of debug symbols
- **No credentials** stored in workflows (uses GITHUB_TOKEN)
- **Minimal permissions** (only `contents: write`)

## 📝 Best Practices

1. **Test locally first**: Run `make test` and `make build` before tagging
2. **Follow semver**: Use semantic versioning (v1.0.0, v1.1.0, v2.0.0)
3. **Write release notes**: Use descriptive tag messages
4. **Verify downloads**: Test at least one binary from each release
5. **Keep changelog**: Maintain CHANGELOG.md for users

## 🆘 Troubleshooting

### Build Fails

1. Check Actions tab for error logs
2. Test locally: `make build-all`
3. Ensure dependencies are up to date: `go mod tidy`

### Release Not Created

1. Verify tag format: must be `v*.*.*`
2. Check Actions permissions in repo settings
3. Review workflow logs for errors

### Binary Issues

```bash
# Make executable (Linux/macOS)
chmod +x licet-linux-amd64

# Verify checksum
sha256sum -c licet-linux-amd64.tar.gz.sha256
```

## 🎯 Next Steps

1. **Push this setup** to your repository
2. **Create your first release** with `git tag v1.0.0`
3. **Add status badges** to README.md
4. **Tell users** where to download releases

## 📚 Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)
- [Go Release Best Practices](https://goreleaser.com/intro/)

---

**Status**: ✅ Ready to use!
**Estimated build time**: 3-5 minutes per release
**Supported platforms**: 6 (Linux x3, macOS x2, Windows x1)

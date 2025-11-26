# Build Configuration and Platform Support

## Database Driver Support by Build Type

This document explains how SQLite and PostgreSQL support varies across different build configurations.

### Overview

The project uses build tags to conditionally include SQLite support based on whether CGO is enabled. This allows for:
- **Full database support** (SQLite + PostgreSQL) in native and Docker builds
- **PostgreSQL-only** support in cross-compiled release binaries
- **Successful compilation** across all platforms without complex cross-compiler setups

## Build Configurations

### 1. Docker Builds (Full Support)

**File:** `Dockerfile`

```dockerfile
RUN CGO_ENABLED=1 go build ...
```

**Database Support:**
- ✅ SQLite (via CGO and go-sqlite3)
- ✅ PostgreSQL (pure Go driver)

**Platform:** Linux AMD64 (typical Docker host)

**Use Case:** Production deployments via Docker/Kubernetes

### 2. Local Development Builds (Full Support)

**Command:** `make build` or `go build ./cmd/server`

**CGO Default:** Enabled when building for native platform

**Database Support:**
- ✅ SQLite (if building on Linux/macOS)
- ✅ PostgreSQL

**Use Case:** Local development and testing

### 3. Cross-Compiled Release Binaries (PostgreSQL Only)

**Workflow:** `.github/workflows/release.yml`

**CGO Default:** Disabled for cross-compilation (automatic)

**Database Support:**
- ❌ SQLite (requires CGO, complex cross-compilation)
- ✅ PostgreSQL (pure Go, works everywhere)

**Platforms:**
- Linux: AMD64, ARM64, ARMv7
- macOS: Intel, Apple Silicon
- Windows: AMD64

**Use Case:** Downloadable release binaries for all platforms

## Why This Approach?

### SQLite Requires CGO

The `github.com/mattn/go-sqlite3` driver is a CGO binding to the C SQLite library. This means:

1. **Native Builds:** CGO is enabled by default ✅
   - SQLite works out of the box
   - Docker builds work (with gcc + musl-dev)

2. **Cross-Compilation:** CGO is disabled by default ❌
   - Prevents SQLite driver from compiling
   - Requires platform-specific C cross-compilers (complex)

### PostgreSQL is Pure Go

The `github.com/lib/pq` driver is written in pure Go:
- Works with or without CGO
- Cross-compiles to any platform easily
- No C dependencies required

## Build Tag Implementation

### File: `internal/database/sqlite.go`

```go
//go:build cgo
// +build cgo

package database

import (
	_ "github.com/mattn/go-sqlite3"
)
```

**How it works:**
- Only included when CGO is enabled
- Allows builds to succeed without SQLite
- No code changes needed in main database logic

### File: `internal/database/database.go`

```go
import (
	_ "github.com/lib/pq"  // Always available
	// SQLite imported via sqlite.go when cgo is available
)
```

## Platform-Specific Guidance

### Docker (Recommended for Production)

```bash
docker build -t licet .
docker run -e DATABASE_TYPE=sqlite licet
# OR
docker run -e DATABASE_TYPE=postgres licet
```

**Supports:** Both SQLite and PostgreSQL

### Linux (Native Build)

```bash
make build
./build/licet
```

**Config:** Use `type: sqlite` or `type: postgres`

**Supports:** Both databases

### macOS (Native Build)

```bash
make build
./build/licet
```

**Supports:** Both databases (CGO works natively)

### Downloaded Release Binaries

**For all platforms:**

```yaml
# config.yaml
database:
  type: postgres  # Must use PostgreSQL
  host: localhost
  port: 5432
  username: licet
  password: changeme
  database: licet
```

**Supports:** PostgreSQL only

**Why?** Cross-compiled binaries don't include SQLite driver

## Testing Builds

### Test Native Build with CGO

```bash
CGO_ENABLED=1 go build -o licet-cgo ./cmd/server
./licet-cgo  # SQLite works
```

### Test Cross-Compile Without CGO

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o licet-nocgo ./cmd/server
# Build succeeds, but SQLite won't work at runtime
```

### Test Docker Build

```bash
docker build -t licet:test .
docker run --rm licet:test ./licet --help
```

## Troubleshooting

### Error: "Binary was compiled with 'CGO_ENABLED=0'"

**Symptom:** Downloaded release binary fails when `config.yaml` has `type: sqlite`

**Solution:** Change config to use PostgreSQL:
```yaml
database:
  type: postgres
```

**Why:** Release binaries are cross-compiled without CGO

### Error: "gcc: command not found" (Docker)

**Symptom:** Docker build fails during `go build`

**Solution:** Ensure Dockerfile has:
```dockerfile
RUN apk add --no-cache gcc musl-dev
```

**Why:** SQLite compilation requires C compiler

### Error: "failed to connect to database: unknown driver 'sqlite3'"

**Symptom:** Binary built without CGO trying to use SQLite

**Solutions:**
1. Use PostgreSQL instead, OR
2. Rebuild with CGO enabled: `CGO_ENABLED=1 go build ...`

## Release Binary Support Matrix

| Platform | Architecture | SQLite | PostgreSQL | Notes |
|----------|-------------|--------|------------|-------|
| Linux | AMD64 | ❌ | ✅ | Use Docker for SQLite |
| Linux | ARM64 | ❌ | ✅ | Use Docker for SQLite |
| Linux | ARMv7 | ❌ | ✅ | Use Docker for SQLite |
| macOS | Intel | ❌ | ✅ | Use Docker for SQLite |
| macOS | Apple Silicon | ❌ | ✅ | Use Docker for SQLite |
| Windows | AMD64 | ❌ | ✅ | Use Docker for SQLite |
| Docker | Linux AMD64 | ✅ | ✅ | **Recommended** |

## Recommendations

### For Production

**Use Docker with PostgreSQL:**
```yaml
database:
  type: postgres
  host: postgres-server
  port: 5432
```

**Why:**
- Production-grade database
- Better performance at scale
- Easier backups and replication

### For Development/Testing

**Option 1: Docker with SQLite (easiest):**
```bash
docker run -v $(pwd)/data:/app/data licet
```

**Option 2: Native build with SQLite:**
```bash
make build  # CGO enabled automatically
./build/licet
```

### For Single-Server Deployments

**Use Docker with SQLite:**
```yaml
database:
  type: sqlite
  database: /app/data/licet.db
```

**Why:**
- No external database required
- Simple setup
- Sufficient for monitoring 10-50 license servers

## Future Improvements

Potential enhancements to build system:

1. **Conditional SQLite in releases:**
   - Set up cross-compilation toolchains in GitHub Actions
   - Build platform-specific binaries with CGO
   - More complex, but enables SQLite everywhere

2. **Alternative pure-Go SQLite:**
   - Use `modernc.org/sqlite` (pure Go implementation)
   - Slower than C version, but no CGO needed
   - Enables SQLite in all builds

3. **Build variants:**
   - Provide two versions of each release
   - `licet-linux-amd64-full` (with SQLite)
   - `licet-linux-amd64-minimal` (PostgreSQL only)

## Summary

**Current approach:**
- ✅ Simple and maintainable
- ✅ Works for all users (with appropriate database choice)
- ✅ Docker provides full functionality
- ✅ No complex cross-compilation setup needed

**Trade-off:**
- Release binaries don't include SQLite
- Users must run Docker or use PostgreSQL for downloaded binaries

**Recommendation:**
- Production: Use Docker with PostgreSQL
- Development: Native build or Docker with SQLite
- Downloaded binaries: Use with PostgreSQL

This design prioritizes reliability and ease of maintenance while providing full functionality through Docker.

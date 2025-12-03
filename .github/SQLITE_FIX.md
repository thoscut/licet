# SQLite Driver Fix - Quick Reference

## The Issue

Error when running:
```
level=fatal msg="Failed to initialize database: sql: unknown driver \"sqlite3\" (forgotten import?)"
```

## Root Cause

The SQLite driver is conditionally imported via build tags in `internal/database/sqlite.go`:
```go
//go:build cgo
// +build cgo
```

This means SQLite is only included when **CGO is enabled**. If you built without CGO or with an old binary, SQLite won't work.

## Solution: Rebuild with CGO Enabled

### Quick Fix
```bash
# Clean old builds
rm -rf build/

# Build with CGO explicitly enabled
CGO_ENABLED=1 go build -o build/licet ./cmd/server

# Run the application
./build/licet
```

### Or Use the Makefile
```bash
make build
./build/licet
```

The Makefile already sets CGO_ENABLED=1 by default.

## Verification

After rebuilding, you should see:
```
time="..." level=info msg="Starting Licet (Licet)"
time="..." level=info msg="Server listening on :8080"
```

✅ **No more "unknown driver" error!**

## For Different Use Cases

### 1. Local Development (SQLite)
```bash
# Build with CGO (includes SQLite)
CGO_ENABLED=1 go build -o licet ./cmd/server

# Use SQLite in config
# config.yaml:
database:
  type: sqlite
  database: licet.db
```

### 2. Production (Docker with SQLite)
```bash
docker build -t licet .
docker run -p 8080:8080 -v $(pwd)/data:/app/data licet
```
Docker builds automatically include SQLite (gcc/musl-dev installed).

### 3. Production (PostgreSQL)
```bash
# Build without CGO works fine for PostgreSQL
CGO_ENABLED=0 go build -o licet ./cmd/server

# Use PostgreSQL in config
# config.yaml:
database:
  type: postgres
  host: localhost
  port: 5432
  username: licet
  password: changeme
  database: licet
```

### 4. Cross-Platform Releases
GitHub Actions automatically builds without CGO for portability.
**Use PostgreSQL** with downloaded release binaries.

## Why This Happens

**SQLite requires CGO** because it's a C library. The Go driver `mattn/go-sqlite3` uses cgo to call into the C code.

**PostgreSQL is pure Go** so it works with or without CGO.

We use build tags to make SQLite optional:
- ✅ **CGO enabled**: SQLite + PostgreSQL both work
- ✅ **CGO disabled**: PostgreSQL only (useful for cross-compilation)

## Quick Checks

### Check if CGO is enabled:
```bash
go env CGO_ENABLED
# 1 = enabled
# 0 = disabled
```

### Test if binary includes SQLite:
```bash
# If this works, SQLite is included:
./licet  # Should start without "unknown driver" error
```

### Rebuild if needed:
```bash
make clean
make build
```

## Summary

**Problem**: Old binary or CGO-disabled build
**Solution**: Rebuild with `CGO_ENABLED=1 go build`
**Quick fix**: Run `make build`

✅ **Application now works with SQLite!**

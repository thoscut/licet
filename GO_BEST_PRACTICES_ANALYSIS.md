# Go Best Practices, KISS & Idioms Analysis: Licet

## Executive Summary

Licet is a well-structured Go application with a clear separation of concerns, solid testing on parsers, and good use of key libraries (Chi, sqlx, Viper, logrus). However, there are meaningful violations of KISS, several non-idiomatic Go patterns, a few bugs, and some areas where the codebase has grown organically without refactoring.

---

## 1. What the Codebase Does Well

### Project Layout
- Follows the standard `cmd/` + `internal/` layout convention. All domain code is unexportable.
- Embedded assets via `//go:embed` for templates and static files (`web/templates.go`) -- clean, single-binary deployment.

### Error Handling
- Errors are consistently wrapped with context: `fmt.Errorf("failed to X: %w", err)` throughout `database.go`, `storage.go`, `config_writer.go`.
- `database.RunMigrations` properly handles the `migrate.ErrNilVersion` and `migrate.ErrNoChange` sentinel errors.

### Concurrency
- Graceful shutdown with `signal.Notify` + `context.WithTimeout` in `main.go`.
- Worker pool pattern in `CollectorService.CollectAll()` with bounded goroutines.
- WebSocket hub uses proper context-based cancellation (`ctx, cancel`).

### Security
- All database queries use prepared statements or parameterized queries via sqlx -- no SQL injection.
- Constant-time password comparison (`subtle.ConstantTimeCompare`) in `auth.go:254`.
- API key hashing with SHA-256 before storage/lookup in `auth.go:160-163`.
- Atomic config file writes (write to temp, rename) in `config_writer.go:55-97`.
- Input validation at system boundaries (`util/validation.go`).

### Design Patterns
- **Interface-based parser design**: `Parser` interface in `parser.go` with factory pattern (`ParserFactory`).
- **Database dialect abstraction**: `Dialect` interface in `dialect.go` cleanly handles SQL differences across SQLite/PostgreSQL/MySQL.
- **Closure-based HTTP handlers**: `ListServers(query)` returns `http.HandlerFunc` -- standard Chi pattern for dependency injection.

---

## 2. KISS Violations

### 2.1 Massive Route Duplication in `setupRouter` (Critical)

**File:** `cmd/server/main.go:287-339`

The entire API route registration block (~50 lines of routes) is duplicated verbatim for cached vs. non-cached modes. The fix is a single `r.Group` with conditional middleware:

```go
r.Group(func(r chi.Router) {
    if cache != nil {
        r.Use(appmiddleware.CacheMiddleware(cache, ...))
    }
    r.Get("/servers", handlers.ListServers(query))
    // ... all routes once
})
```

### 2.2 Config Type Duplication

**Files:** `config/config.go` vs `middleware/auth.go`

`middleware.AuthConfig`, `middleware.APIKey`, `middleware.BasicUser` are near-identical copies of `config.AuthConfig`, `config.APIKeyConfig`, `config.BasicUserConfig`. The manual field-by-field mapping in `setupRouter` (main.go:186-204) is unnecessary boilerplate. Either use config types directly in middleware, or define a shared type.

### 2.3 Duplicated Analytics Services

**Files:** `services/analytics.go` vs `services/enhanced_analytics.go`

`EnhancedAnalyticsService.getUtilizationStats()` is nearly identical to `AnalyticsService.GetUtilizationStats()`. The linear regression + trend calculation pattern is repeated across 3 methods in `enhanced_analytics.go`. `EnhancedAnalyticsService` should compose `AnalyticsService` rather than duplicate it.

### 2.4 Hardcoded Fake Analytics

**File:** `services/enhanced_analytics.go:110-113`

```go
peakHour := 14     // Default to afternoon
peakDayOfWeek := 2 // Default to Tuesday
weekdayAvg := avgUsage
weekendAvg := avgUsage * 0.3 // Estimate
```

These are fabricated constants, not computed from data. Either compute from actual hourly/daily data or remove them.

---

## 3. Go Idiom Violations

### 3.1 Error-in-Struct Anti-pattern

**Files:** `models/models.go:113`, `parsers/parser.go:11`

```go
type ServerQueryResult struct {
    Error error  // anti-pattern: store error in struct
}
type Parser interface {
    Query(hostname string) ServerQueryResult  // no error return
}
```

Go convention is `(result, error)` returns. The idiomatic signature: `Query(hostname string) (ServerQueryResult, error)`

### 3.2 Regex Compilation Inside Hot Functions

**Files:** `util/validation.go`, `parsers/flexlm.go:40-74`, `parsers/rlm.go:40-49`

All regexes are compiled on every function call. These should be package-level variables compiled once at init:

```go
var ipv4Regex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
```

This is both a performance issue and a Go style convention violation.

### 3.3 `map[string]interface{}` Overuse

Extensively used for JSON responses and template data across all handlers instead of typed response structs. This prevents compile-time type checking and is non-idiomatic for Go.

### 3.4 Missing `context.Context` Propagation

The `Parser.Query()` method doesn't accept `context.Context`, so long-running license server queries can't be cancelled. `QueryService.QueryServer()` creates a background context internally instead of accepting one from the caller -- HTTP request context should flow through.

### 3.5 Three Context Values Instead of One Struct

**File:** `middleware/auth.go:394-396`

Three separate `context.WithValue` calls to store username, role, and method. Idiomatic Go stores a single `AuthInfo` struct under one key.

### 3.6 Dead Code

- `sortFeaturesByName` in `handlers/web.go:104-108` -- defined but never called.
- `fileExists` in `util/binpath.go:60-66` -- defined but never used.

---

## 4. Bugs

### 4.1 `PostgresDialect.Placeholder` Only Works for Indices 1-9

**File:** `database/dialect.go:68-70`

```go
func (d *PostgresDialect) Placeholder(index int) string {
    return "$" + string(rune('0'+index))
}
```

For `index >= 10`, `rune('0'+10)` produces `:` (ASCII 58), resulting in `$:` instead of `$10`. Should use `strconv.Itoa(index)`. Currently latent (method isn't called) but will break when used.

### 4.2 `database.New` Missing MySQL Support

**File:** `database/database.go:27-37`

MySQL is missing from `New()` but is handled in `RunMigrations()` and the `Dialect` interface. A user configuring `type: mysql` gets "unsupported database type" at startup.

### 4.3 Unsafe JSON Construction

**File:** `cmd/server/main.go:383-384`

```go
w.Write([]byte(fmt.Sprintf(
    `{"authenticated":%t,"username":"%s","role":"%s","method":"%s"}`,
    authInfo.Authenticated, authInfo.Username, authInfo.Role, authInfo.Method)))
```

If any field contains `"` or `\`, this produces invalid/exploitable JSON. Should use `json.NewEncoder(w).Encode(authInfo)`.

---

## 5. Goroutine Lifecycle Issues

### 5.1 Unstoppable Cleanup Goroutines

**Files:** `middleware/auth.go:143`, `middleware/cache.go:60`, `middleware/ratelimit.go:76`

All three launch goroutines that loop forever with `for range ticker.C` but have no cancellation mechanism. Contrast with `WebSocketHub.statusBroadcaster()` which properly uses `h.ctx.Done()`.

### 5.2 WebSocket Upgrader Global Mutation

**File:** `handlers/websocket.go:272-273`

```go
upgrader.ReadBufferSize = h.config.ReadBufferSize
upgrader.WriteBufferSize = h.config.WriteBufferSize
```

The package-level `upgrader` is mutated on every WebSocket connection -- data race if multiple connections arrive concurrently.

---

## 6. Test Coverage Gaps

| Package | Has Tests | Coverage Quality |
|---|---|---|
| `parsers` | Yes | Thorough (46KB of tests) |
| `config` | Yes | Good |
| `database` | Yes | Good |
| `services` | Partial | `license_test.go`, `config_writer_test.go`, `dbstats_test.go` |
| `util` | Yes | `binpath_test.go`, `validation_test.go` |
| `handlers` | Minimal | `api_test.go` is 809 bytes |
| **middleware** | **No** | Auth, cache, rate-limit, pagination untested |
| **models** | **No** | Method logic (`AvailableLicenses`, `DaysToExpiration`) untested |
| **scheduler** | **No** | No tests |

The middleware package is security-critical (authentication, authorization, rate limiting) and has zero test coverage.

---

## 7. Module Name

**File:** `go.mod:1` -- `module licet`

Using a bare `licet` module name prevents external importability and doesn't match Go conventions for modules hosted on GitHub. Should be `github.com/thoscut/licet`.

---

## 8. Dependency Notes

- **logrus**: In maintenance mode. Author recommends migrating to `log/slog` (standard library since Go 1.21). Not urgent.
- **gorilla/websocket**: Archived since December 2022, directly imported in `handlers/websocket.go`. Consider `nhooyr.io/websocket` or `coder/websocket`.

---

## 9. Summary by Priority

### High Priority
1. Fix unsafe JSON construction in `main.go:383-384` (security)
2. Add MySQL to `database.New()` (completeness bug)
3. Fix `PostgresDialect.Placeholder()` for indices >= 10
4. Move regex compilation to package-level vars in parsers and validation
5. Eliminate route duplication in `setupRouter`

### Medium Priority
6. Refactor `Parser.Query()` to return `(ServerQueryResult, error)`
7. Add `context.Context` to `Parser.Query()` interface
8. Add cancellation to middleware cleanup goroutines
9. Fix WebSocket upgrader global mutation (data race)
10. Add tests for middleware package (security-critical)
11. Remove dead code (`sortFeaturesByName`, `fileExists`)

### Low Priority
12. Replace `map[string]interface{}` with typed response structs
13. Consolidate `AnalyticsService` and `EnhancedAnalyticsService`
14. Remove or properly implement hardcoded analytics values
15. Consolidate config/middleware type duplication
16. Consider migrating from logrus to `log/slog`
17. Consider replacing archived `gorilla/websocket`

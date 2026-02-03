# Licet - AI Assistant Guide

## Project Overview

Licet is a web application for monitoring software license servers. It was originally forked from [phplicensewatcher](https://github.com/proche-rainmaker/phplicensewatcher) and completely rewritten for improved performance, security, and maintainability.

**Current Version:** 1.9.2+

**License:** GNU General Public License v3.0

**Repository:** https://github.com/thoscut/licet

### Key Features

- **Multi-Server Support** - Monitor FlexLM, RLM, and other license servers
- **Real-time Monitoring** - Web dashboard showing license server status, usage, and users
- **Historical Tracking** - Store and visualize license usage over time
- **Expiration Alerts** - Email notifications for expiring licenses
- **RESTful API** - JSON API for integration with other systems
- **Modern Web UI** - Responsive interface built with Bootstrap
- **Background Workers** - Automated data collection via cron-like scheduler
- **Multiple Databases** - Support for SQLite, PostgreSQL, and MySQL
- **Secure** - No SQL injection, proper input validation, prepared statements throughout

## Supported License Server Types

Currently implemented license server types:

1. **FlexLM** ✅ - Fully implemented (Flexera License Manager)
2. **RLM** ✅ - Fully implemented (Reprise License Manager)
3. **SPM** 🚧 - Planned (Sentinel Protection Manager)
4. **SESI** 🚧 - Planned (Side Effects Software)
5. **RVL** 🚧 - Planned (RE:Vision Effects)
6. **Tweak** 🚧 - Planned (Tweak Software)
7. **Pixar** 🚧 - Planned (Pixar licensing)

## Architecture and File Structure

### Directory Layout

```
licet/
├── cmd/
│   └── server/              # Application entry point
│       └── main.go          # Main executable (4735 bytes)
├── internal/
│   ├── config/              # Configuration management
│   │   └── config.go        # YAML config loading with Viper
│   ├── database/            # Database layer
│   │   ├── database.go      # Database abstraction with sqlx
│   │   └── sqlite.go        # SQLite-specific code
│   ├── handlers/            # HTTP handlers
│   │   ├── api.go           # REST API endpoints
│   │   ├── web.go           # Web UI handlers
│   │   └── settings.go      # Settings page handler
│   ├── models/              # Data models
│   │   └── models.go        # Structs for servers, features, users, etc.
│   ├── parsers/             # License server parsers
│   │   ├── parser.go        # Parser interface and factory
│   │   ├── flexlm.go        # FlexLM implementation
│   │   ├── flexlm_test.go   # FlexLM tests
│   │   ├── rlm.go           # RLM implementation
│   │   └── rlm_test.go      # RLM tests
│   ├── scheduler/           # Background job scheduler
│   │   └── scheduler.go     # Cron-like task scheduling
│   ├── services/            # Business logic
│   │   ├── license.go       # License operations
│   │   ├── alert.go         # Alert management
│   │   ├── collector.go     # Data collection
│   │   ├── binutils.go      # Binary path utilities
│   │   ├── config_writer.go # Config file writing
│   │   └── utility.go       # Utility functions
│   └── util/                # Shared utilities
│       ├── binpath.go       # Binary path detection
│       └── binpath_test.go  # Binary path tests
├── web/
│   ├── static/              # CSS, JS, fonts, images
│   │   ├── css/
│   │   ├── js/
│   │   └── fonts/
│   ├── templates/           # HTML templates
│   └── templates.go         # Template embedding
├── config.yaml              # User configuration (not in git)
├── config.example.yaml      # Example configuration
├── go.mod                   # Go module dependencies
├── go.sum                   # Dependency checksums
├── Makefile                 # Build and development tasks
├── Dockerfile               # Container image definition
└── README.md                # Main documentation
```

### Core Components

#### 1. Application Entry Point
- **cmd/server/main.go** - Initializes config, database, router, scheduler, and starts server

#### 2. Configuration Management
- **internal/config/config.go** - Uses Viper for YAML config and environment variables
- **config.yaml** - User configuration (servers, database, email, alerts)
- **config.example.yaml** - Template configuration file

#### 3. Database Layer
- **internal/database/database.go** - Database abstraction using sqlx
- **internal/database/sqlite.go** - SQLite-specific initialization
- Supports: SQLite, PostgreSQL, MySQL
- Auto-migrations on startup

#### 4. Parsers
- **internal/parsers/parser.go** - Parser interface and factory pattern
- **internal/parsers/flexlm.go** - FlexLM query and parsing
- **internal/parsers/rlm.go** - RLM query and parsing
- Each parser returns: status, features, users, expiration data

#### 5. Services (Business Logic)
- **internal/services/license.go** - License server operations
- **internal/services/collector.go** - Data collection workers
- **internal/services/alert.go** - Alert generation and email sending
- **internal/services/binutils.go** - Binary detection and execution
- **internal/services/config_writer.go** - Configuration persistence

#### 6. HTTP Handlers
- **internal/handlers/web.go** - Web UI routes (/, /details, /expiration)
- **internal/handlers/api.go** - REST API routes (/api/v1/*)
- **internal/handlers/settings.go** - Settings page for server management

#### 7. Background Scheduler
- **internal/scheduler/scheduler.go** - Cron-like job scheduling
- Handles periodic data collection and alerts

#### 8. Models
- **internal/models/models.go** - Data structures for:
  - Server configuration
  - Features and usage
  - Users and checkouts
  - Alerts and events
  - License expiration

## Configuration Guide

### Configuration File (config.yaml)

```yaml
# Server settings
server:
  port: 8080
  host: 0.0.0.0
  settings_enabled: true  # Enable/disable settings page

# Database configuration
database:
  type: sqlite  # sqlite, postgres, mysql
  database: licet.db

  # For PostgreSQL/MySQL:
  # host: localhost
  # port: 5432  # 5432 for postgres, 3306 for mysql
  # username: licet
  # password: changeme
  # sslmode: disable

# Logging
logging:
  level: info  # debug, info, warn, error
  format: text  # text or json

# License servers to monitor
servers:
  - hostname: "27000@flexlm.example.com"
    description: "Production FlexLM Server"
    type: "flexlm"
    cacti_id: ""
    webui: ""

  - hostname: "5053@rlm.example.com"
    description: "RLM License Server"
    type: "rlm"
    webui: "http://rlm.example.com:4000"

# Email settings
email:
  enabled: false
  from: "licensing@example.com"
  to:
    - "admin@example.com"
  alerts:
    - "alerts@example.com"
  smtp_host: "smtp.example.com"
  smtp_port: 587
  username: ""
  password: ""

# Alert configuration
alerts:
  enabled: true
  lead_time_days: 10  # Warn this many days before expiration
  resend_interval_min: 60  # Minutes between duplicate alerts

# RRD graphing (optional)
rrd:
  enabled: false
  directory: "./rrd"
  collection_interval: 5  # Minutes
```

### Environment Variables

Configuration can be overridden with environment variables:

```bash
LICET_SERVER_PORT=8080
LICET_DATABASE_TYPE=postgres
LICET_DATABASE_HOST=localhost
LICET_DATABASE_DATABASE=licet
LICET_LOGGING_LEVEL=debug
```

## Database Schema

### Tables

1. **servers** - Configured license servers (managed via UI/config)
   - hostname, description, type, cacti_id, webui, enabled

2. **features** - Current license features and usage
   - server_hostname, feature_name, total_licenses, licenses_used, vendor_daemon, version, etc.

3. **feature_usage** - Historical usage data
   - server_hostname, feature_name, timestamp, users_count
   - Used for utilization graphs and trends

4. **license_events** - Checkout/denial events
   - event_date, event_time, event_type, feature, username, reason

5. **alerts** - Generated alerts
   - server_hostname, feature_name, alert_type, message, created_at, resolved_at

6. **alert_events** - Alert throttling/deduplication
   - Prevents duplicate alerts within resend interval

## Development Workflows

### Building and Running

```bash
# Install dependencies
go mod download

# Build
go build -o licet ./cmd/server

# Run
./licet

# Run with specific config
./licet -config /path/to/config.yaml

# Build for production
make build

# Run tests
go test ./...
make test

# Run with hot reload (requires air)
air
```

### Adding a New License Server Type

1. Create parser in `internal/parsers/`:
   - Implement `Parser` interface (Query, ParseStatus, ParseFeatures, ParseUsers, ParseExpiration)
   - Follow pattern of `flexlm.go` or `rlm.go`

2. Register in `internal/parsers/parser.go`:
   - Add case to `NewParser()` factory function

3. Add binary detection in `internal/util/binpath.go`

4. Add tests in `internal/parsers/<type>_test.go`

5. Update `config.example.yaml` with example server

### Modifying the Web UI

1. **Templates** - Located in `web/templates/`
   - Uses Go `html/template` package
   - Bootstrap 5 framework
   - Embedded at compile time via `web/templates.go`

2. **Static Assets** - Located in `web/static/`
   - CSS: `web/static/css/`
   - JavaScript: `web/static/js/`
   - Fonts: `web/static/fonts/`

3. **Handlers** - `internal/handlers/web.go`
   - Add new routes to Chi router
   - Pass data to templates

### Adding API Endpoints

1. Define handler in `internal/handlers/api.go`
2. Register route in Chi router
3. Return JSON responses
4. Add error handling
5. Update API documentation

## API Reference

### REST API Endpoints

#### Server Operations
- `GET /api/v1/servers` - List all configured servers
- `GET /api/v1/servers/{server}/status?type={type}` - Get server status
- `GET /api/v1/servers/{server}/features?type={type}` - List features
- `GET /api/v1/servers/{server}/users?type={type}` - List current users
- `GET /api/v1/servers/{server}/expiration?type={type}` - Get expiration dates

#### Feature Operations
- `GET /api/v1/features/{feature}/usage` - Get usage history for feature

#### Alert Operations
- `GET /api/v1/alerts` - List active alerts

#### System Operations
- `GET /api/v1/health` - Health check endpoint

### Web UI Routes

- `GET /` - Dashboard (server status overview)
- `GET /details/{server}` - Detailed server view with features and users
- `GET /expiration/{server}` - License expiration dates
- `GET /utilization` - Usage graphs
- `GET /alerts` - Active alerts
- `GET /settings` - Server configuration (if enabled)

## Key Functions and Code Locations

### Parsers

#### FlexLM Parser (internal/parsers/flexlm.go)
- `Query(hostname string) (ServerStatus, error)` - Query FlexLM server
- `ParseStatus(output string) ServiceStatus` - Parse server status
- `ParseFeatures(output string) []Feature` - Parse license features
- `ParseUsers(output string) []User` - Parse current users
- `ParseExpiration(output string) []Expiration` - Parse expiration dates

#### RLM Parser (internal/parsers/rlm.go)
- Same interface as FlexLM
- Different regex patterns for RLM output format
- Recent fixes for feature header parsing and utility name handling

### Services

#### License Service (internal/services/license.go)
- `QueryServer(server models.Server) (*models.ServerStatus, error)` - Query any server type
- `StoreFeatures(features []models.Feature) error` - Save features to database
- `StoreUsage(usage []models.FeatureUsage) error` - Save usage history

#### Alert Service (internal/services/alert.go)
- `CheckExpirations() error` - Check for expiring licenses
- `SendAlert(alert models.Alert) error` - Send email alert
- `MuffleAlert(server, feature string) bool` - Check alert throttling

#### Collector Service (internal/services/collector.go)
- `CollectAll() error` - Query all servers and store data
- `CollectServer(server models.Server) error` - Query single server

### Handlers

#### Web Handlers (internal/handlers/web.go)
- `HandleIndex(w http.ResponseWriter, r *http.Request)` - Dashboard
- `HandleDetails(w http.ResponseWriter, r *http.Request)` - Server details
- `HandleExpiration(w http.ResponseWriter, r *http.Request)` - Expiration view

#### API Handlers (internal/handlers/api.go)
- `HandleServers(w http.ResponseWriter, r *http.Request)` - List servers API
- `HandleServerStatus(w http.ResponseWriter, r *http.Request)` - Server status API
- `HandleFeatures(w http.ResponseWriter, r *http.Request)` - Features API

## Dependencies

### Go Modules (go.mod)

```go
module github.com/thoscut/licet

go 1.21

require (
    github.com/go-chi/chi/v5 v5.0.11          // HTTP router
    github.com/go-chi/cors v1.2.1             // CORS middleware
    github.com/jmoiron/sqlx v1.3.5            // Database extensions
    github.com/lib/pq v1.10.9                 // PostgreSQL driver
    github.com/mattn/go-sqlite3 v1.14.19      // SQLite driver
    github.com/robfig/cron/v3 v3.0.1          // Cron scheduler
    github.com/sirupsen/logrus v1.9.3         // Structured logging
    github.com/spf13/viper v1.18.2            // Configuration
    gopkg.in/gomail.v2 v2.0.0-20160411212932  // Email sending
)
```

### External Binaries

- **lmutil** - FlexLM utilities (from Flexera)
- **rlmutil** - RLM utilities (from Reprise)
- **spmstat** - SPM utilities (planned)
- **sesictrl** - SESI utilities (planned)
- **rvlstatus** - RVL utilities (planned)
- **tlm_server** - Tweak utilities (planned)

These binaries must be installed and accessible in the system PATH or configured via binary paths.

## Installation and Deployment

### Quick Start

```bash
# Clone repository
git clone https://github.com/thoscut/licet.git
cd licet

# Copy example config
cp config.example.yaml config.yaml

# Edit configuration
vim config.yaml

# Build
go build -o licet ./cmd/server

# Run
./licet
```

### Production Deployment

#### Using systemd

```ini
[Unit]
Description=Licet License Server Monitor
After=network.target

[Service]
Type=simple
User=licet
WorkingDirectory=/opt/licet
ExecStart=/opt/licet/licet
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

#### Using Docker

```bash
# Build image
docker build -t licet:latest .

# Run container
docker run -d \
  -p 8080:8080 \
  -v /path/to/config.yaml:/app/config.yaml \
  -v /path/to/data:/app/data \
  --name licet \
  licet:latest
```

#### Behind nginx (Reverse Proxy)

```nginx
server {
    listen 80;
    server_name licet.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Testing and Debugging

### ⚠️ CRITICAL: All Tests Must Pass

**All tests must always pass before committing any code changes.** This is a non-negotiable requirement.

- Run `go test ./...` before every commit
- If a test fails, fix the code or the test before proceeding
- Never commit code with failing tests
- When adding new functionality, add corresponding tests
- When fixing bugs, add tests to prevent regression

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/parsers

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

### Debug Logging

Enable debug logging in `config.yaml`:

```yaml
logging:
  level: debug
  format: text
```

Debug output includes:
- Commands executed (e.g., `lmutil lmstat -a -c 27000@server`)
- Raw output from license servers
- Database operations
- Query results and parsing details

### Manual Testing

1. **Test license binary directly:**
   ```bash
   lmutil lmstat -a -c 27000@server.example.com
   rlmutil rlmstat -a -c 5053@server.example.com
   ```

2. **Test database connection:**
   ```bash
   # SQLite
   sqlite3 licet.db "SELECT * FROM servers;"

   # PostgreSQL
   psql -h localhost -U licet -d licet -c "SELECT * FROM servers;"
   ```

3. **Test API endpoints:**
   ```bash
   curl http://localhost:8080/api/v1/health
   curl http://localhost:8080/api/v1/servers
   ```

### Common Issues

1. **License server connection failures**
   - Check binary is in PATH: `which lmutil`
   - Test connectivity: `telnet server.example.com 27000`
   - Check firewall rules
   - Verify hostname format: `port@hostname`

2. **Database errors**
   - Check file permissions (SQLite)
   - Verify database credentials (PostgreSQL/MySQL)
   - Review migration logs on startup

3. **Email alerts not sending**
   - Verify `email.enabled: true` in config
   - Check SMTP settings
   - Review logs for SMTP errors
   - Test with `alerts.enabled: true`

## Recent Changes and Fixes

Based on recent commit history:

1. **FlexLM Parser Checkout Fixes** (branch claude/fix-flexlm-checkout-parsing-rrHnQ)
   - Fixed checkout parsing for versions without 'v' prefix: `(2023.1)` vs `(v2023.1)`
   - Fixed checkout date parsing to handle year in date: `Mon 1/2/24` or `Mon 1/2/2024`
   - Fixed template to match checkouts by feature name only (client version often differs from license version)

2. **RLM Parser Improvements** (commits e0220a0, 51cad50)
   - Fixed feature header parsing to match actual RLM output format
   - Fixed parser excluding utility names from features
   - Improved license checkout display

3. **UI Enhancements** (commits e503a8b, 8b8bc7d)
   - Added version display to server details page
   - Compact layout improvements
   - Checkout filter functionality
   - Checkout time display with sorting

4. **Settings Page** (commit e2b201a)
   - Fixed settings page enable configuration
   - Proper mapstructure tags for config binding

5. **License Expiration Page** (commit 88ca47d)
   - Fixed display issues on expiration page

6. **.gitignore Updates** (commit d05b9ea)
   - Added licet binary to .gitignore

## Code Conventions and Patterns

### Testing Requirements
- **ALL TESTS MUST PASS** - Run `go test ./...` before every commit
- Never commit code with failing tests
- Add tests for new functionality
- Add regression tests when fixing bugs
- Aim for high test coverage on critical code paths

### Go Style
- Follow standard Go formatting: `go fmt`
- Use `gofmt` and `go vet` before committing
- Descriptive variable names
- Error handling: always check and handle errors
- Context-based cancellation for long-running operations

### Naming Conventions
- **Packages**: lowercase, single word (parsers, handlers, services)
- **Types**: PascalCase (Server, Feature, User)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase
- **Constants**: PascalCase or UPPER_SNAKE_CASE

### Data Structure Patterns

#### Parser Interface
```go
type Parser interface {
    Query(hostname string) (ServerStatus, error)
    ParseStatus(output string) ServiceStatus
    ParseFeatures(output string) []Feature
    ParseUsers(output string) []User
    ParseExpiration(output string) []Expiration
}
```

#### Server Status Response
```go
type ServerStatus struct {
    Service     ServiceStatus
    Features    []Feature
    Users       []User
    Expirations []Expiration
}
```

### Error Handling
- Return errors, don't panic
- Wrap errors with context: `fmt.Errorf("failed to query server: %w", err)`
- Log errors with appropriate level
- Return meaningful HTTP status codes

## Security Features

### Built-in Security

1. **SQL Injection Protection** ✅
   - All database queries use prepared statements (sqlx)
   - No direct string interpolation in queries

2. **Command Injection Protection** ✅
   - Proper command execution with argument separation
   - No shell interpretation of user input

3. **XSS Protection** ✅
   - Automatic output escaping in html/template
   - Context-aware template rendering

4. **Authentication** ⚠️
   - Recommended: Run behind VPN or add reverse proxy auth
   - Settings page can be disabled: `server.settings_enabled: false`

### Security Best Practices

1. Use reverse proxy (nginx) with HTTPS
2. Implement authentication layer (Basic Auth, OAuth)
3. Run with minimal privileges (dedicated user)
4. Keep dependencies updated: `go get -u`
5. Review logs regularly
6. Restrict database access
7. Validate all configuration inputs

## Performance Characteristics

Licet is designed for high performance and low resource usage:

- **Startup Time**: < 1 second
- **Memory Usage**: ~20MB baseline
- **Concurrent Requests**: 1000s/sec
- **License Queries**: Parallel execution for multiple servers
- **Binary Size**: ~20MB single self-contained binary

## Git Workflow

### Branch Strategy
- Main branch: production-ready code
- Feature branches: `claude/*` pattern for AI-assisted development
- Pull requests required for merging

### Commit Message Format
- Clear, descriptive messages
- Reference issue numbers when applicable
- Use conventional commits style when possible

## Future Enhancement Considerations

### Short Term
1. ✅ Complete FlexLM parser (DONE)
2. ✅ Complete RLM parser (DONE)
3. 🚧 Add remaining license types (SPM, SESI, RVL, Tweak, Pixar)
4. 🚧 Complete web UI templates
5. 🚧 Add RRD graphing support

### Medium Term
1. Add authentication/authorization
2. Multi-tenancy support
3. Dashboard customization
4. GraphQL API
5. WebSocket real-time updates
6. Prometheus metrics export

### Long Term
1. Kubernetes operator
2. High availability / clustering
3. Advanced analytics and forecasting
4. Mobile app
5. Plugin system for custom parsers

## Additional Resources

- **Main Documentation**: README.md
- **Go Implementation Details**: GO_IMPLEMENTATION.md
- **Example Configuration**: config.example.yaml
- **Original Project (PHP)**: https://github.com/proche-rainmaker/phplicensewatcher
- **FlexLM Documentation**: http://www.globetrotter.com/flexlm/
- **RLM Documentation**: https://www.reprisesoftware.com/

## Notes for AI Assistants

1. **⚠️ ALL TESTS MUST PASS** - Before committing any changes, run `go test ./...` and ensure ALL tests pass. This is MANDATORY. Never commit code with failing tests.
2. **This is modern Go code** - Follow Go best practices and idioms
3. **Strong typing** - Leverage Go's type system for safety
4. **Testing** - Write tests for new functionality; add regression tests for bug fixes
5. **Parser pattern** - All license types implement Parser interface
6. **Configuration** - Use Viper for config management
7. **Database** - Use sqlx for all database operations
8. **HTTP** - Chi router for all routing
9. **Templates** - Go html/template with Bootstrap
10. **Logging** - Structured logging with logrus
11. **Error handling** - Always handle errors, never panic in HTTP handlers
12. **Context** - Use context.Context for cancellation and timeouts
13. **Concurrency** - Use goroutines carefully, avoid race conditions
14. **Recent focus** - RLM parser fixes and UI improvements
15. **No PHP** - Project completely migrated from PHP to Go

## Version History

**Current Version: 1.9.2+**

Recent changes (from git log):
- Fixed license expiration page display issues
- Fixed RLM feature header parsing
- Fixed RLM parser excluding utility names
- Improved server details page with version display
- Added checkout time display and filtering
- Fixed settings page configuration
- Added licet binary to .gitignore
- Multiple UI and parser improvements

---

*For historical development information, see git history.*

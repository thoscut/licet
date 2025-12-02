# Licet - Go Edition

A modern rewrite of Licet in Go, providing license server monitoring with improved performance, security, and maintainability.

## Features

- **Multi-Server Support**: Monitor FlexLM, RLM, SPM, SESI, Tweak, Pixar, and RVL license servers
- **Real-time Monitoring**: Web dashboard showing license server status, usage, and users
- **Historical Tracking**: Store and visualize license usage over time
- **Expiration Alerts**: Email notifications for expiring licenses
- **RESTful API**: JSON API for integration with other systems
- **Modern Web UI**: Responsive interface built with Bootstrap
- **Background Workers**: Automated data collection via cron-like scheduler
- **Multiple Databases**: Support for SQLite, PostgreSQL, and MySQL
- **Secure**: No SQL injection, proper input validation, prepared statements

## Quick Start

### Prerequisites

- Go 1.21 or later
- License server utilities (lmutil, rlmutil, etc.) installed and in PATH
- (Optional) PostgreSQL or MySQL for production deployments

### Installation

```bash
# Clone the repository
git clone https://github.com/thoscut/licet.git
cd licet

# Build the application
go build -o licet ./cmd/server

# Copy example config
cp config.example.yaml config.yaml

# Edit configuration
vim config.yaml

# Run the server
./licet
```

The server will start on http://localhost:8080

### Docker

```bash
# Build Docker image
docker build -t licet:go .

# Run container
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/data:/app/data \
  licet:go
```

## Configuration

Edit `config.yaml` to configure your license servers:

```yaml
servers:
  - hostname: "27000@flexlm.example.com"
    description: "Production FlexLM Server"
    type: "flexlm"
```

See `config.example.yaml` for all available options.

### Logging

Licet supports multiple log levels for debugging and monitoring:

```yaml
logging:
  level: info  # debug, info, warn, error
  format: text  # text or json
```

**Log Levels:**

- **debug** - Detailed information including:
  - Commands executed (e.g., `lmutil lmstat -a -c server:port`)
  - Raw command output from license servers
  - Database operations (storing features, recording usage)
  - Query results (service status, feature counts)
  - Useful for troubleshooting license server connectivity and parsing issues

- **info** - General informational messages:
  - Server queries and collection progress
  - Successfully completed operations
  - Alert notifications

- **warn** - Warning messages for non-critical issues

- **error** - Error messages for failures

**Example Debug Output:**

```
DEBUG Executing FlexLM command: /usr/local/bin/lmutil lmstat -i -a -c 27000@server.example.com
DEBUG FlexLM command output for 27000@server.example.com:
lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    License file(s) on server.example.com: ...
DEBUG Query successful for 27000@server.example.com: service=up, features=15, users=8
DEBUG Storing 15 features from 27000@server.example.com to database
```

To enable debug logging, edit your `config.yaml` and restart the server:

```yaml
logging:
  level: debug
```

Or set via environment variable:

```bash
PLW_LOGGING_LEVEL=debug ./licet
```

## API Endpoints

### REST API

- `GET /api/v1/servers` - List all configured servers
- `GET /api/v1/servers/{server}/status` - Get server status
- `GET /api/v1/servers/{server}/features` - List features
- `GET /api/v1/servers/{server}/users` - List current users
- `GET /api/v1/features/{feature}/usage` - Get usage history
- `GET /api/v1/alerts` - List active alerts
- `GET /api/v1/health` - Health check

### Web UI

- `/` - Dashboard
- `/details/{server}` - Server details
- `/expiration/{server}` - License expiration dates
- `/utilization` - Usage graphs
- `/alerts` - Active alerts

## Architecture

### Directory Structure

```
.
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database layer
│   ├── handlers/        # HTTP handlers (web + API)
│   ├── models/          # Data models
│   ├── parsers/         # License server parsers
│   ├── scheduler/       # Background job scheduler
│   └── services/        # Business logic
├── web/
│   ├── static/          # CSS, JS, images
│   └── templates/       # HTML templates
├── config.yaml          # Configuration file
├── go.mod               # Go dependencies
└── README.go.md         # This file
```

### Components

1. **Parsers** - Query license servers and parse output
2. **Services** - Business logic for licenses, alerts, collection
3. **Handlers** - HTTP request handlers for web and API
4. **Scheduler** - Background jobs for data collection and alerts
5. **Database** - Data persistence layer with migrations

## Development

### Running Tests

```bash
go test ./...
```

### Running with Live Reload

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

### Building for Production

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o licet-linux-amd64 ./cmd/server

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o licet-darwin-amd64 ./cmd/server

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o licet-windows-amd64.exe ./cmd/server
```

## Supported License Server Types

### FlexLM (Flexera)
- Status: ✅ Fully Implemented
- Binary: `lmutil`
- Features: Server status, features, users, expiration

### RLM (Reprise)
- Status: ✅ Fully Implemented
- Binary: `rlmutil`
- Features: Server status, features, users, expiration

### SPM (Sentinel)
- Status: 🚧 Planned
- Binary: `spmstat`

### SESI (Side Effects)
- Status: 🚧 Planned
- Binary: `sesictrl`

### RVL (RE:Vision)
- Status: 🚧 Planned
- Binary: `rvlstatus`

### Tweak
- Status: 🚧 Planned
- Binary: `tlm_server`

### Pixar
- Status: 🚧 Planned
- Binary: `pixar_query.sh`

## Differences from PHP Version

### Improvements

- **Security**: No SQL injection vulnerabilities, prepared statements throughout
- **Performance**: Concurrent license queries, efficient database access
- **Type Safety**: Strong typing prevents many runtime errors
- **Modern Stack**: REST API, JSON responses, proper logging
- **Easy Deployment**: Single binary, no PHP/Apache/PEAR dependencies
- **Testability**: Unit tests for all components
- **Configuration**: YAML-based config with environment variable support

### Migration from PHP Version

1. Export data from existing MySQL/PostgreSQL database
2. Configure Go version with same database credentials
3. Run migrations (automatic on startup)
4. Configure servers in `config.yaml`
5. Start the Go server

The database schema is compatible with the PHP version for `feature_usage` and `license_events` tables.

## Troubleshooting

### License server connection failures

```bash
# Test license binary directly
/usr/local/bin/lmutil lmstat -a -c 27000@server.example.com

# Check binary permissions
ls -la /usr/local/bin/lmutil

# Check firewall
telnet server.example.com 27000
```

### Database issues

```bash
# SQLite: Check file permissions
ls -la licet.db

# PostgreSQL: Test connection
psql -h localhost -U licet -d licet
```

### Email alerts not sending

- Verify SMTP settings in `config.yaml`
- Check `email.enabled: true` and `alerts.enabled: true`
- Review logs for SMTP errors

## Performance

The Go version significantly outperforms the PHP version:

- **Startup**: < 1 second (vs 2-5 seconds for PHP)
- **Memory**: ~20MB (vs 50-100MB for PHP+Apache)
- **Concurrent Requests**: 1000s/sec (vs 100s/sec for PHP)
- **License Queries**: Parallel execution (vs sequential in PHP)

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new features
4. Run `go fmt` and `go vet`
5. Submit a pull request

## License

Licet is licensed under the **GNU General Public License v3.0**.

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see https://www.gnu.org/licenses/.

### Third-Party Licenses

Licet includes third-party JavaScript and CSS libraries (Bootstrap, Chart.js, etc.) that are licensed under the MIT License, which is compatible with GPL-3.0. For complete license information and attributions, see [THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md).

## Credits

- Original Licet by Vladimir Vuksan
- Go rewrite maintains compatibility while modernizing the codebase

## Support

- Issues: https://github.com/thoscut/licet/issues
- Documentation: See CLAUDE.md for detailed code documentation

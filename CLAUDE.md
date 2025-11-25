# Licet - AI Assistant Guide

## Project Overview

Licet is a PHP-based web application for monitoring software license servers. It provides:
- Web UI for viewing license server status
- License usage reporting (RRD or MySQL)
- Alerting for license server downtime and expiration
- Historical usage tracking and graphs
- Integration with Cacti for graphing

**Current Version:** 1.9.2

**License:** GNU General Public License

## Supported License Server Types

The application supports seven different license server types:
1. **FlexLM** - Most common, well-tested (see tools.php:453)
2. **RLM** - Reprise License Manager (tools.php:1059)
3. **SESI** - Side Effects Software (tools.php:872)
4. **Tweak** - Tweak Software (tools.php:977)
5. **Pixar** - Pixar licensing (tools.php:1170)
6. **SPM** - Sentinel Protection Manager (tools.php:601)
7. **RVL** - RE:Vision Effects (tools.php:729)

## Architecture and File Structure

### Core Files

#### Entry Points
- **index.php** - Main dashboard showing all license servers
- **details.php** - Detailed license usage and expiration for a specific server
- **admin.php** - Administrative interface
- **utilization.php** - License utilization graphs
- **monitor.php** - License utilization trends
- **denials.php** - FlexLM denial reports
- **checkouts.php** - FlexLM checkout reports

#### Core Libraries
- **common.php** - Configuration loader and header generator
- **tools.php** - Main business logic (1260+ lines)
  - License server query functions (get_flexlm, get_rlm, etc.)
  - RRD/database operations
  - Server filtering and data processing
  - Email alerting
  - Time calculation utilities

#### Configuration
- **config.php** - Main configuration file
  - Server definitions
  - Database credentials
  - Email settings
  - Binary paths for license utilities
  - Cacti integration settings

#### Automation Scripts (Cron)
- **license_util.php** - Collects current usage (run every 5-15 min)
- **license_cache.php** - Stores daily license totals (run daily)
- **license_alert.php** - Email alerts for expiring licenses (run daily)

#### Utilities
- **version.php** - Version footer display
- **lmremove.php** - License removal functionality
- **check_installation.php** - Installation checker

### Frontend Assets
- **css/bootstrap.min.css** - Bootstrap 3 framework
- **js/bootstrap.min.js** - Bootstrap JavaScript
- **fonts/** - Glyphicons font files
- **style.css** - Custom styles

### Third-Party Components
- **cdiagram-0.39/** - PHP diagram class for graphing
- **HTML/Table.php** - PEAR HTML_Table for rendering tables

## Database Schema

Located in: **licet.sql**

### Tables

1. **flexlm_events** - License checkout/denial events
   - Primary Key: (flmevent_date, flmevent_time, flmevent_feature, flmevent_user)
   - Stores: date, time, type, feature, user, reason

2. **license_usage** - Historical usage data
   - Primary Key: (flmusage_product, flmusage_server, flmusage_date, flmusage_time)
   - Stores: server, product, date, time, number of users
   - Used for utilization graphs

3. **licenses_available** - Daily license counts
   - Primary Key: (flmavailable_date, flmavailable_server, flmavailable_product, flmavailable_num_licenses)
   - Stores: date, server, product, total licenses
   - Updated daily to track license pool changes

4. **alert_events** (referenced but not in SQL file) - Alert throttling
   - Prevents duplicate alerts within notify_resend window

## Configuration Guide

### Essential config.php Settings

```php
// Server definitions (array of license servers)
$server[] = array(
    "hostname" => "port@server.example.com",
    "desc" => "Description",
    "type" => "flexlm",  // or rlm, sesi, tweak, pixar, spm, rvl
    "cacti" => "0000",   // optional: Cacti graph ID
    "webui" => "http://..." // optional: RLM web UI
);

// Database configuration
$db_type = "mysql";
$db_hostname = "localhost";
$db_username = "phplic";
$db_password = "phplic";
$db_database = "licet";

// Email alerting
$notify_from = "licensing@example.com";
$notify_to = "licensing@example.com";
$notify_alerts = "alert@example.com";
$notify_resend = "60"; // minutes between alerts

// License monitoring
$lead_time = 10; // days before expiration to warn
$collection_interval = 5; // minutes between samples

// Binary paths (must be executable)
$lmutil = "/usr/local/bin/lmutil";
$rlmstat = "/usr/local/bin/rlmutil rlmstat";
$spmstat = "/usr/local/bin/spmstat";
// ... etc for other license types
```

### Monitor Specific Features

```php
$monitor_license[] = array(
    "feature" => "feature_name",
    "description" => "Feature Description"
);
```

## Development Workflows

### Adding a New License Server Type

1. Add query function to **tools.php** (follow pattern of get_flexlm, get_rlm)
2. Function must return array with keys: status, licenses, expiration, users
3. Add case to getDetails() switch statement (tools.php:317)
4. Add binary path to config.php
5. Test thoroughly with actual license server

### Modifying the UI

1. All pages use Bootstrap 3 framework
2. Common header via print_header() in common.php
3. Use HTML_Table PEAR class for tabular data
4. Status colors defined via CSS classes: up, down, warning

### Database Queries

- Use PEAR DB library (DB::connect, DB::query)
- Always check for errors: `if (DB::isError($result))`
- Use prepared statements or proper escaping (SECURITY NOTE: current code vulnerable to SQL injection)

### RRD Graph Generation

- RRD files stored in: $rrd_dir
- Auto-created via create_rrd() if missing (tools.php:194)
- Update via insert_into_rrd() (tools.php:157)
- Collection interval from config.php affects RRD step size

## Cron Job Setup

### Typical Crontab Configuration

```bash
# License utilization collection (every 15 minutes)
0,15,30,45 * * * * wget -O - http://server/licet/license_util.php >> /dev/null

# Daily license cache (runs at 12:15 AM)
15 0 * * * wget -O - http://server/licet/license_cache.php >> /dev/null

# License expiration alerts (runs at 2 AM)
0 2 * * * wget -O - http://server/licet/license_alert.php >> /dev/null
```

Alternative using PHP CLI:
```bash
0,15,30,45 * * * * php /var/www/html/licet/license_util.php >> /dev/null
```

## Key Functions Reference

### tools.php Core Functions

#### Server Querying
- **get_flexlm($server, $pos)** - Query FlexLM server (line 453)
- **get_rlm($server, $pos)** - Query RLM server (line 1059)
- **get_spm($server, $pos)** - Query SPM server (line 601)
- **get_sesi($server, $pos)** - Query SESI server (line 872)
- **get_tweak($server, $pos)** - Query Tweak server (line 977)
- **get_rvl($server, $pos)** - Query RVL server (line 729)
- **get_pixar($server, $pos)** - Query Pixar server (line 1170)
- **getDetails($server)** - Dispatcher to specific query function (line 317)

#### Data Management
- **writeLicense_Usage($usage)** - Write usage to DB and RRD (line 220)
- **findServers($needle, $key, $needle2, $key2)** - Filter server array (line 293)
- **cleanHostname($name)** - Normalize hostname for RRD files (line 270)

#### RRD Operations
- **create_rrd($filename)** - Create RRD file if missing (line 194)
- **insert_into_rrd($name, $payload, $date)** - Update RRD (line 157)
- **bulk_rrd($name, $payload)** - Bulk RRD update (line 183)

#### Alerting
- **emailAlerts($host, $statusMsg)** - Send email alert (line 378)
- **muffleAlerts($host)** - Check alert throttling (line 400)

#### UI Helpers
- **get_server_detail_link($pos)** - Generate details link (line 439)
- **get_server_expiration_link($pos)** - Generate expiration link (line 446)
- **AppendStatusMsg($statusMsg, $msg)** - Concatenate status messages (line 370)

#### Utilities
- **timespan** class - Calculate time differences (line 41)
- **getTime()** - Microtime for page execution timing (line 362)
- **generate_error_image($str)** - Generate error image (line 140)

## Common Development Tasks

### Adding a New Server

Edit config.php:
```php
$server[] = array(
    "hostname" => "27000@newserver.example.com",
    "desc" => "Production FlexLM Server",
    "type" => "flexlm"
);
```

### Debugging License Queries

Add `?debug=1` to URL or check these:
1. Verify binary paths in config.php are correct
2. Test binary manually: `/usr/local/bin/lmutil lmstat -a -c port@server`
3. Check server connectivity from web server host
4. Review regex patterns in get_*() functions for output parsing

### Modifying Email Alerts

1. Edit email settings in config.php
2. Test with: `http://server/licet/license_alert.php?nomail=1`
3. Modify license_alert.php for custom logic
4. Alert throttling in muffleAlerts() (tools.php:400)

### Customizing Graphs

1. Graph sizes: $smallgraph, $largegraph in config.php
2. Colors: $colors variable (comma-separated)
3. RRD parameters in create_rrd() (tools.php:194)
4. Collection interval affects graph granularity

## Code Conventions and Patterns

### PHP Style
- Opening PHP tags: `<?php` (no short tags)
- File headers include SVN $Id$ tags
- Error handling: check file_exists, is_readable before includes
- Database: PEAR DB abstraction layer

### Naming Conventions
- Functions: lowercase with underscores (get_flexlm, write_license_usage)
- Arrays: descriptive names with _array suffix
- Global variables: $server, $config values from config.php
- CSS classes match status: "up", "down", "warning"

### Data Structure Patterns

All get_*() functions return this structure:
```php
array(
    "status" => array(
        "service" => "up|down|warning",
        "clients" => "link or message",
        "listing" => "link or message",
        "version" => "version string",
        "master" => "master server name",
        "msg" => "error message or empty"
    ),
    "licenses" => array(
        "feature_name" => array(
            array(
                "num_licenses" => int,
                "licenses_used" => int,
                "extra" => "optional metadata"
            )
        )
    ),
    "expiration" => array(
        "feature_name" => array(
            array(
                "vendor_daemon" => "string",
                "version" => "string",
                "expiration_date" => "date",
                "num_licenses" => int,
                "days_to_expiration" => int,
                "type" => "string"
            )
        )
    ),
    "users" => array(
        "feature_name" => array(
            array(
                "line" => "full output line",
                "time_checkedout" => unix_timestamp
            )
        )
    )
)
```

### Regular Expression Patterns

License server output parsing uses preg_match():
- FlexLM: `/(users of) (.*)(\(total of) (\d+)/` (line 524)
- RLM: `/rlm status on ([^\s]+)/` (line 1082)
- Server down detection: `/Cannot connect to license server/`

## Security Considerations

### Known Vulnerabilities (IMPORTANT)

1. **SQL Injection** - Direct variable interpolation in SQL queries
   - Example: tools.php:241 `$sql = "INSERT ... VALUES ('$usage[0]','$usage[1]'..."`
   - FIX: Use prepared statements or proper escaping

2. **Command Injection** - User input in popen() calls
   - Server hostnames passed to shell commands
   - FIX: Validate and sanitize all inputs

3. **XSS** - Some output not properly escaped
   - Use htmlspecialchars() for all user-controlled data

4. **Authentication** - No built-in authentication
   - README warns: "do not run on publicly available Internet server"
   - FIX: Add authentication layer (Apache .htaccess, PHP sessions)

### Recommended Security Measures

1. Run behind VPN or internal network only
2. Implement authentication (Apache Basic Auth minimum)
3. Validate all config.php server definitions
4. Restrict file permissions on config.php (contains DB credentials)
5. Use prepared statements for all DB queries
6. Sanitize inputs before shell execution

## Testing and Debugging

### Debug Mode

Add to URLs: `?debug=1` to enable SQL query printing and verbose output

### Manual Testing

1. Test license server connectivity:
   ```bash
   /usr/local/bin/lmutil lmstat -a -c port@server
   ```

2. Test database connection:
   ```bash
   mysql -u phplic -p licet
   ```

3. Test RRD creation:
   ```bash
   ls -la /path/to/rrd/
   ```

### Common Issues

1. **"Cannot connect to license server"**
   - Check firewall rules
   - Verify binary paths in config.php
   - Test connectivity from web server host

2. **No graphs appearing**
   - Check RRD directory permissions
   - Verify rrdtool binary path
   - Ensure data collection is running

3. **Email alerts not sending**
   - Check PHP mail configuration
   - Review notify_* settings in config.php
   - Check alert_events table for throttling

4. **Database errors**
   - Verify DB credentials in config.php
   - Check table schema matches licet.sql
   - Ensure DB user has INSERT/SELECT permissions

## Dependencies

### Required PHP Extensions
- PHP (tested on 5.x+)
- PEAR (PHP Extension and Application Repository)
- PEAR::DB - Database abstraction
- PEAR::HTML_Table - Table generation
- GD extension (for image generation)

### External Binaries
- **lmutil/lmstat** - FlexLM utilities
- **rlmutil** - RLM utilities
- **spmstat** - SPM utilities
- **sesictrl** - SESI utilities
- **rvlstatus** - RVL utilities
- **tlm_server** - Tweak utilities
- **rrdtool** - RRD graphing (optional)

### Web Server
- Apache with mod_php (recommended)
- PHP-FPM with nginx (alternative)

### Database
- MySQL/MariaDB (primary support)
- PostgreSQL (with modifications)

## Installation Quick Reference

### Basic Installation
1. Extract to web directory
2. Edit config.php with server definitions
3. Point browser to index.php
4. No database needed for basic viewing

### Extended Installation (Graphs/Alerts)
1. Create MySQL database: `mysqladmin create licenses`
2. Import schema: `mysql -f licenses < licet.sql`
3. Create DB user and grant permissions
4. Configure database settings in config.php
5. Set up cron jobs for data collection
6. Configure email settings for alerts

## File Permissions

```bash
# Web server readable
chmod 644 *.php css/* js/* fonts/*

# Configuration (protect credentials)
chmod 640 config.php
chown www-data:www-data config.php

# RRD directory (web server writable)
mkdir rrd
chmod 755 rrd
chown www-data:www-data rrd
```

## Git Workflow

- Main branch not specified in current config
- Feature branches: `claude/claude-md-*` pattern
- Commit messages should be clear and descriptive
- Current version in version.php (1.9.2)

## Useful Code Locations

| Feature | File | Line(s) |
|---------|------|---------|
| Server status table | index.php | 90-131 |
| FlexLM query | tools.php | 453-599 |
| RLM query | tools.php | 1059-1168 |
| Database write | tools.php | 220-268 |
| Email alerts | tools.php | 378-396 |
| Alert throttling | tools.php | 400-436 |
| RRD creation | tools.php | 194-218 |
| Server filtering | tools.php | 293-315 |
| Timespan calculation | tools.php | 41-137 |
| License expiration | license_alert.php | full file |
| Usage collection | license_util.php | full file |
| Detail view | details.php | full file |

## Future Enhancement Considerations

1. **Modern PHP** - Update to PHP 7.4+ with type hints
2. **Security** - Add authentication, prepared statements
3. **Framework** - Consider Laravel/Symfony for better structure
4. **API** - REST API for external integrations
5. **Real-time** - WebSocket updates instead of page refresh
6. **Docker** - Containerization for easier deployment
7. **Tests** - PHPUnit test coverage
8. **Modern JS** - Replace inline JavaScript with Vue/React
9. **Bootstrap 5** - Update from Bootstrap 3
10. **Composer** - Modern dependency management

## Additional Resources

- Original project: http://freshmeat.net/projects/licet/
- FlexLM: http://www.globetrotter.com/flexlm/
- Cacti integration: See config.php $cactiurl settings
- PHP PEAR: http://pear.php.net/

## Notes for AI Assistants

1. This is legacy PHP code (circa 2011-2013) - modern PHP practices differ
2. Security is a major concern - always validate/sanitize inputs
3. Database schema is simple but effective
4. The core pattern (query → parse → store → display) is consistent
5. Each license type has unique output format requiring specific regex
6. RRD and MySQL storage are independent - can use either or both
7. Cron-based architecture means no real-time updates
8. Bootstrap 3 styling throughout - maintain consistency
9. PEAR dependencies are old but functional
10. No test coverage - manual testing required

## Version History Reference

Current version: 1.9.2 (version.php:3)
Recent changes visible in git log:
- Cleanup version footer
- Add bootstrap Restore remove functionality
- Fix server name handling

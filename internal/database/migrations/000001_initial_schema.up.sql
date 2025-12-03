-- Initial database schema

CREATE TABLE IF NOT EXISTS servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hostname TEXT NOT NULL UNIQUE,
    description TEXT,
    type TEXT NOT NULL,
    cacti_id TEXT,
    webui TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_hostname TEXT NOT NULL,
    name TEXT NOT NULL,
    version TEXT,
    vendor_daemon TEXT,
    total_licenses INTEGER NOT NULL,
    used_licenses INTEGER NOT NULL,
    expiration_date TIMESTAMP,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_hostname, name, version, expiration_date)
);

CREATE TABLE IF NOT EXISTS feature_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_hostname TEXT NOT NULL,
    feature_name TEXT NOT NULL,
    date DATE NOT NULL,
    time TIME NOT NULL,
    users_count INTEGER NOT NULL,
    UNIQUE(server_hostname, feature_name, date, time)
);

CREATE TABLE IF NOT EXISTS license_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_date DATE NOT NULL,
    event_time TIME NOT NULL,
    event_type TEXT NOT NULL,
    feature_name TEXT NOT NULL,
    username TEXT NOT NULL,
    reason TEXT,
    UNIQUE(event_date, event_time, feature_name, username)
);

CREATE TABLE IF NOT EXISTS alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_hostname TEXT NOT NULL,
    feature_name TEXT,
    alert_type TEXT NOT NULL,
    message TEXT NOT NULL,
    severity TEXT NOT NULL,
    sent BOOLEAN DEFAULT FALSE,
    sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS alert_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    datetime TIMESTAMP NOT NULL,
    type TEXT NOT NULL,
    hostname TEXT NOT NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_features_server ON features(server_hostname);
CREATE INDEX IF NOT EXISTS idx_features_expiration ON features(expiration_date);
CREATE INDEX IF NOT EXISTS idx_usage_server_feature ON feature_usage(server_hostname, feature_name);
CREATE INDEX IF NOT EXISTS idx_usage_date ON feature_usage(date);
CREATE INDEX IF NOT EXISTS idx_alerts_sent ON alerts(sent);
CREATE INDEX IF NOT EXISTS idx_events_date ON license_events(event_date);

-- Fix features table UNIQUE constraint
-- This migration rebuilds the features table with the correct constraint

-- Create backup
DROP TABLE IF EXISTS features_backup;
CREATE TABLE features_backup AS SELECT * FROM features;

-- Rebuild features table with correct constraint
DROP TABLE features;
CREATE TABLE features (
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

-- Restore data
INSERT INTO features (id, server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated)
SELECT id, server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated
FROM features_backup;

-- Clean up backup
DROP TABLE features_backup;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_features_server ON features(server_hostname);
CREATE INDEX IF NOT EXISTS idx_features_expiration ON features(expiration_date);

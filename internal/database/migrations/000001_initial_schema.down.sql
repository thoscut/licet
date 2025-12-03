-- Rollback initial schema

DROP INDEX IF EXISTS idx_events_date;
DROP INDEX IF EXISTS idx_alerts_sent;
DROP INDEX IF EXISTS idx_usage_date;
DROP INDEX IF EXISTS idx_usage_server_feature;
DROP INDEX IF EXISTS idx_features_expiration;
DROP INDEX IF EXISTS idx_features_server;

DROP TABLE IF EXISTS alert_events;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS license_events;
DROP TABLE IF EXISTS feature_usage;
DROP TABLE IF EXISTS features;
DROP TABLE IF EXISTS servers;

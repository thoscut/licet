-- Add is_active column to features table
-- This column tracks whether a feature is currently active on the license server.
-- When licenses are replaced on a server, old features are marked as inactive.

ALTER TABLE features ADD COLUMN is_active BOOLEAN DEFAULT TRUE;

-- Set all existing features to active by default
UPDATE features SET is_active = TRUE WHERE is_active IS NULL;

-- Create index for filtering active features
CREATE INDEX IF NOT EXISTS idx_features_active ON features(is_active);

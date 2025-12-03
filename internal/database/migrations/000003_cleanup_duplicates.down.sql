-- Rollback duplicate cleanup
-- NOTE: This is a no-op migration as we cannot restore deleted duplicate records.
-- The cleanup was a data correction operation, not a schema change.

SELECT 'Migration 000003 rollback is a no-op (cannot restore deleted duplicates)' AS message;

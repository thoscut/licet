-- Rollback features constraint fix
-- NOTE: This is a no-op migration as rolling back the constraint change
-- would require recreating the old problematic constraint, which is not desirable.
-- If you need to rollback to the initial schema, use migration 000001.

SELECT 'Migration 000002 rollback is a no-op' AS message;

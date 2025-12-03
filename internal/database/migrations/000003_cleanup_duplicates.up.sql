-- Clean up duplicate permanent license records
-- This removes duplicate records that were created before the parser fix.
-- For each (server_hostname, name, version) combination, keep only the latest record.

DELETE FROM features
WHERE id NOT IN (
    SELECT MAX(id)
    FROM features
    WHERE expiration_date IS NOT NULL
    GROUP BY server_hostname, name, version
)
AND expiration_date IS NOT NULL;

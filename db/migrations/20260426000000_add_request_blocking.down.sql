-- Rollback request blocking fields
DROP INDEX IF EXISTS idx_requests_is_blocked;
DROP INDEX IF EXISTS idx_requests_replaced_by;

ALTER TABLE requests
DROP CONSTRAINT IF EXISTS fk_requests_replaced_by;

ALTER TABLE requests
DROP COLUMN IF EXISTS is_blocked,
DROP COLUMN IF EXISTS replaced_by_request_id;

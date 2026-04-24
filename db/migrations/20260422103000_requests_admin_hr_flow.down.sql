DROP TABLE IF EXISTS "public"."request_target_dzos";
DROP TABLE IF EXISTS "public"."request_participants";

DROP INDEX IF EXISTS "public"."request_target_dzo_id";
DROP INDEX IF EXISTS "public"."request_assigned_hr_id";
DROP INDEX IF EXISTS "public"."request_request_type";
DROP INDEX IF EXISTS "public"."request_parent_request_id";

ALTER TABLE "public"."requests"
  DROP CONSTRAINT IF EXISTS "requests_users_responsible_admin",
  DROP CONSTRAINT IF EXISTS "requests_users_assigned_hr",
  DROP CONSTRAINT IF EXISTS "requests_requests_parent_request";

ALTER TABLE "public"."requests"
  DROP COLUMN IF EXISTS "updated_at",
  DROP COLUMN IF EXISTS "cost_mode",
  DROP COLUMN IF EXISTS "cost_amount",
  DROP COLUMN IF EXISTS "deadline_at",
  DROP COLUMN IF EXISTS "training_date",
  DROP COLUMN IF EXISTS "responsible_admin_id",
  DROP COLUMN IF EXISTS "format",
  DROP COLUMN IF EXISTS "category",
  DROP COLUMN IF EXISTS "title",
  DROP COLUMN IF EXISTS "target_dzo_id",
  DROP COLUMN IF EXISTS "assigned_hr_id",
  DROP COLUMN IF EXISTS "request_type",
  DROP COLUMN IF EXISTS "parent_request_id";

ALTER TABLE "public"."requests"
  ALTER COLUMN "status" SET DEFAULT 'PENDING';

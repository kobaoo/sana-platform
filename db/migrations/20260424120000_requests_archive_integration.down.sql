DROP INDEX IF EXISTS "public"."request_dzo_contract_request_id_dzo_id_key";
DROP INDEX IF EXISTS "public"."request_dzo_contract_dzo_id";
DROP INDEX IF EXISTS "public"."request_dzo_contract_request_id";
DROP TABLE IF EXISTS "public"."request_dzo_contracts";

DROP INDEX IF EXISTS "public"."request_kind";

ALTER TABLE "public"."requests"
  DROP COLUMN IF EXISTS "completed_at",
  DROP COLUMN IF EXISTS "kind";

-- align existing "requests" table shape
ALTER TABLE "public"."requests"
  ALTER COLUMN "step" SET DEFAULT 0,
  ALTER COLUMN "status" SET DEFAULT 'PENDING';
-- create index "request_entity_id" to table: "requests"
CREATE INDEX IF NOT EXISTS "request_entity_id" ON "public"."requests" ("entity_id");
-- create index "request_initiator_id" to table: "requests"
CREATE INDEX IF NOT EXISTS "request_initiator_id" ON "public"."requests" ("initiator_id");
-- create index "request_status" to table: "requests"
CREATE INDEX IF NOT EXISTS "request_status" ON "public"."requests" ("status");
-- create index "request_step" to table: "requests"
CREATE INDEX IF NOT EXISTS "request_step" ON "public"."requests" ("step");

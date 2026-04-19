-- create "requests" table
CREATE TABLE "public"."requests" (
  "id" uuid NOT NULL,
  "initiator_id" uuid NOT NULL,
  "entity_id" uuid NOT NULL,
  "entity_type" character varying NOT NULL,
  "step" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL,
  "status" character varying NOT NULL DEFAULT 'PENDING',
  PRIMARY KEY ("id")
);
-- create index "request_entity_id" to table: "requests"
CREATE INDEX "request_entity_id" ON "public"."requests" ("entity_id");
-- create index "request_initiator_id" to table: "requests"
CREATE INDEX "request_initiator_id" ON "public"."requests" ("initiator_id");
-- create index "request_status" to table: "requests"
CREATE INDEX "request_status" ON "public"."requests" ("status");
-- create index "request_step" to table: "requests"
CREATE INDEX "request_step" ON "public"."requests" ("step");

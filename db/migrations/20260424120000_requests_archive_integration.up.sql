ALTER TABLE "public"."requests"
  ADD COLUMN IF NOT EXISTS "kind" character varying(30) NOT NULL DEFAULT 'REGULAR',
  ADD COLUMN IF NOT EXISTS "completed_at" timestamptz NULL;

UPDATE "public"."requests"
SET "kind" = COALESCE(NULLIF("kind", ''), 'REGULAR')
WHERE "kind" IS DISTINCT FROM 'REGULAR';

CREATE INDEX IF NOT EXISTS "request_kind" ON "public"."requests" ("kind");

CREATE TABLE IF NOT EXISTS "public"."request_dzo_contracts" (
  "id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "dzo_id" uuid NOT NULL,
  "file_name" character varying(255) NOT NULL,
  "file_url" character varying(1024) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("id"),
  CONSTRAINT "request_dzo_contracts_requests_request"
    FOREIGN KEY ("request_id") REFERENCES "public"."requests" ("id")
    ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "request_dzo_contracts_dzo_organizations_dzo"
    FOREIGN KEY ("dzo_id") REFERENCES "public"."dzo_organizations" ("id")
    ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "request_dzo_contract_request_id" ON "public"."request_dzo_contracts" ("request_id");
CREATE INDEX IF NOT EXISTS "request_dzo_contract_dzo_id" ON "public"."request_dzo_contracts" ("dzo_id");
CREATE UNIQUE INDEX IF NOT EXISTS "request_dzo_contract_request_id_dzo_id_key"
  ON "public"."request_dzo_contracts" ("request_id", "dzo_id");

ALTER TABLE "public"."requests"
  ADD COLUMN IF NOT EXISTS "parent_request_id" uuid NULL,
  ADD COLUMN IF NOT EXISTS "request_type" character varying(30) NOT NULL DEFAULT 'MAIN',
  ADD COLUMN IF NOT EXISTS "assigned_hr_id" uuid NULL,
  ADD COLUMN IF NOT EXISTS "target_dzo_id" uuid NULL,
  ADD COLUMN IF NOT EXISTS "title" character varying(255) NULL,
  ADD COLUMN IF NOT EXISTS "category" character varying(100) NULL,
  ADD COLUMN IF NOT EXISTS "format" character varying(50) NULL,
  ADD COLUMN IF NOT EXISTS "responsible_admin_id" uuid NULL,
  ADD COLUMN IF NOT EXISTS "training_date" timestamptz NULL,
  ADD COLUMN IF NOT EXISTS "deadline_at" timestamptz NULL,
  ADD COLUMN IF NOT EXISTS "cost_amount" double precision NULL,
  ADD COLUMN IF NOT EXISTS "cost_mode" character varying(30) NULL,
  ADD COLUMN IF NOT EXISTS "updated_at" timestamptz NOT NULL DEFAULT NOW();

UPDATE "public"."requests" r
SET
  "title" = COALESCE(r."title", te."title"),
  "format" = COALESCE(r."format", te."location_type"),
  "training_date" = COALESCE(r."training_date", te."start_date"),
  "responsible_admin_id" = COALESCE(r."responsible_admin_id", r."initiator_id"),
  "updated_at" = NOW(),
  "status" = CASE UPPER(r."status")
    WHEN 'DRAFT' THEN 'DRAFT'
    WHEN 'SUBMITTED' THEN 'IN_PROGRESS'
    WHEN 'APPROVED' THEN 'APPROVED'
    WHEN 'REJECTED' THEN 'REJECTED'
    WHEN 'CANCELLED' THEN 'REJECTED'
    WHEN 'PENDING' THEN 'PENDING'
    ELSE 'DRAFT'
  END
FROM "public"."training_events" te
WHERE r."entity_type" IN ('training_event', 'TRAINING_EVENT')
  AND r."entity_id" = te."id";

UPDATE "public"."requests"
SET
  "title" = COALESCE("title", 'Request ' || LEFT("id"::text, 8)),
  "responsible_admin_id" = COALESCE("responsible_admin_id", "initiator_id"),
  "updated_at" = NOW()
WHERE "title" IS NULL OR "responsible_admin_id" IS NULL;

ALTER TABLE "public"."requests"
  ALTER COLUMN "title" SET NOT NULL,
  ALTER COLUMN "status" SET DEFAULT 'DRAFT';

ALTER TABLE "public"."requests"
  ADD CONSTRAINT "requests_requests_parent_request"
    FOREIGN KEY ("parent_request_id") REFERENCES "public"."requests" ("id")
    ON UPDATE NO ACTION ON DELETE CASCADE,
  ADD CONSTRAINT "requests_users_assigned_hr"
    FOREIGN KEY ("assigned_hr_id") REFERENCES "public"."users" ("id")
    ON UPDATE NO ACTION ON DELETE SET NULL,
  ADD CONSTRAINT "requests_users_responsible_admin"
    FOREIGN KEY ("responsible_admin_id") REFERENCES "public"."users" ("id")
    ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS "request_parent_request_id" ON "public"."requests" ("parent_request_id");
CREATE INDEX IF NOT EXISTS "request_request_type" ON "public"."requests" ("request_type");
CREATE INDEX IF NOT EXISTS "request_assigned_hr_id" ON "public"."requests" ("assigned_hr_id");
CREATE INDEX IF NOT EXISTS "request_target_dzo_id" ON "public"."requests" ("target_dzo_id");

CREATE TABLE IF NOT EXISTS "public"."request_participants" (
  "id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("id"),
  CONSTRAINT "request_participants_requests_request"
    FOREIGN KEY ("request_id") REFERENCES "public"."requests" ("id")
    ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "request_participants_employees_employee"
    FOREIGN KEY ("employee_id") REFERENCES "public"."employees" ("id")
    ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "request_participant_request_id" ON "public"."request_participants" ("request_id");
CREATE INDEX IF NOT EXISTS "request_participant_employee_id" ON "public"."request_participants" ("employee_id");
CREATE UNIQUE INDEX IF NOT EXISTS "request_participant_request_id_employee_id_key"
  ON "public"."request_participants" ("request_id", "employee_id");

CREATE TABLE IF NOT EXISTS "public"."request_target_dzos" (
  "id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "dzo_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("id"),
  CONSTRAINT "request_target_dzos_requests_request"
    FOREIGN KEY ("request_id") REFERENCES "public"."requests" ("id")
    ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "request_target_dzos_dzo_organizations_dzo"
    FOREIGN KEY ("dzo_id") REFERENCES "public"."dzo_organizations" ("id")
    ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "request_target_dzo_request_id" ON "public"."request_target_dzos" ("request_id");
CREATE INDEX IF NOT EXISTS "request_target_dzo_dzo_id" ON "public"."request_target_dzos" ("dzo_id");
CREATE UNIQUE INDEX IF NOT EXISTS "request_target_dzo_request_id_dzo_id_key"
  ON "public"."request_target_dzos" ("request_id", "dzo_id");

ALTER TABLE "public"."requests"
  ADD COLUMN "kind" character varying(30) NOT NULL DEFAULT 'REGULAR',
  ADD COLUMN "title" character varying(255) NULL,
  ADD COLUMN "category" character varying(100) NULL,
  ADD COLUMN "updated_at" timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN "completed_at" timestamptz NULL;

UPDATE "public"."requests"
SET "updated_at" = "created_at"
WHERE "updated_at" IS DISTINCT FROM "created_at";

CREATE INDEX "request_kind" ON "public"."requests" ("kind");

CREATE TABLE "public"."request_employees" (
  "id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "request_employees_request_id_fkey" FOREIGN KEY ("request_id") REFERENCES "public"."requests" ("id") ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT "request_employees_employee_id_fkey" FOREIGN KEY ("employee_id") REFERENCES "public"."employees" ("id") ON DELETE NO ACTION ON UPDATE NO ACTION
);

CREATE INDEX "requestemployee_request_id" ON "public"."request_employees" ("request_id");
CREATE INDEX "requestemployee_employee_id" ON "public"."request_employees" ("employee_id");

CREATE TABLE "public"."request_dzo_contracts" (
  "id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "dzo_id" uuid NOT NULL,
  "file_name" character varying(255) NOT NULL,
  "file_url" character varying(1024) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "request_dzo_contracts_request_id_fkey" FOREIGN KEY ("request_id") REFERENCES "public"."requests" ("id") ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT "request_dzo_contracts_dzo_id_fkey" FOREIGN KEY ("dzo_id") REFERENCES "public"."dzo_organizations" ("id") ON DELETE NO ACTION ON UPDATE NO ACTION
);

CREATE INDEX "requestdzocontract_request_id" ON "public"."request_dzo_contracts" ("request_id");
CREATE INDEX "requestdzocontract_dzo_id" ON "public"."request_dzo_contracts" ("dzo_id");

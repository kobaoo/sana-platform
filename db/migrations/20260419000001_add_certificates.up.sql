CREATE TABLE "certificates" (
  "id" uuid NOT NULL,
  "employee_id" bigint NOT NULL,
  "dzo_id" bigint NULL,
  "title" varchar(255) NOT NULL,
  "file_url" text NOT NULL,
  "issue_date" timestamptz NOT NULL,
  "expiration_date" timestamptz NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
CREATE INDEX "certificates_employee_id" ON "certificates" ("employee_id");
CREATE INDEX "certificates_dzo_id" ON "certificates" ("dzo_id");
CREATE INDEX "certificates_is_active" ON "certificates" ("is_active");
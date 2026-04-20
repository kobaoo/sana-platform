CREATE TABLE "certificates" (
    "id" uuid NOT NULL,
    "employee_id" uuid NOT NULL,
    "type" varchar(50) NOT NULL,
    "title" varchar(300) NOT NULL,
    "file_url" text NULL,
    "issued_date" date NOT NULL,
    "expiry_date" date NULL,
    "uploaded_by" uuid NULL,
    "event_id" uuid NULL,
    "scorm_course_id" uuid NULL,
    "entity_type" varchar(50) NOT NULL,
    "entity_id" uuid NOT NULL,
    "is_active" boolean NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT NOW(),
    "updated_at" timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY ("id")
);

CREATE INDEX "certificates_employee_id" ON "certificates" ("employee_id");
CREATE INDEX "certificates_is_active" ON "certificates" ("is_active");
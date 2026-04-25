-- create "scorm_courses" table
CREATE TABLE "public"."scorm_courses" (
  "id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "title" character varying NOT NULL,
  "category_ids" jsonb NOT NULL,
  "description" text NULL,
  "lecturer" character varying NULL,
  "scorm_url" text NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);
-- create "scorm_progresses" table
CREATE TABLE "public"."scorm_progresses" (
  "id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "status" character varying NOT NULL DEFAULT 'NOT_STARTED',
  "score" bigint NULL,
  "completed_at" timestamptz NULL,
  "suspend_data" text NULL,
  "course_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "scorm_progresses_scorm_courses_course_progress" FOREIGN KEY ("course_id") REFERENCES "public"."scorm_courses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

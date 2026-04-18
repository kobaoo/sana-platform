-- create "courses" table
CREATE TABLE "public"."courses" (
  "id" uuid NOT NULL,
  "title" character varying(255) NOT NULL,
  "description" text NULL,
  "format" character varying(50) NULL,
  "category" character varying(100) NULL,
  "is_external" boolean NOT NULL DEFAULT true,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  "updated_at" timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("id")
);
-- create index "course_title" to table: "courses"
CREATE INDEX "course_title" ON "public"."courses" ("title");
-- create index "course_is_active" to table: "courses"
CREATE INDEX "course_is_active" ON "public"."courses" ("is_active");

-- create "course_modules" table
CREATE TABLE "public"."course_modules" (
  "id" uuid NOT NULL,
  "course_id" uuid NOT NULL,
  "title" character varying(255) NOT NULL,
  "description" text NULL,
  "sort_order" integer NOT NULL DEFAULT 0,
  "duration_minutes" integer NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  "updated_at" timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("id"),
  CONSTRAINT "course_modules_courses_course" FOREIGN KEY ("course_id") REFERENCES "public"."courses" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "course_module_course_id" to table: "course_modules"
CREATE INDEX "course_module_course_id" ON "public"."course_modules" ("course_id");
-- create index "course_module_course_id_sort_order" to table: "course_modules"
CREATE INDEX "course_module_course_id_sort_order" ON "public"."course_modules" ("course_id", "sort_order");

-- create "applications" table
CREATE TABLE "public"."applications" (
  "id" uuid NOT NULL,
  "kind" character varying(20) NOT NULL DEFAULT 'regular',
  "status" character varying(20) NOT NULL DEFAULT 'draft',
  "dzo_id" uuid NULL,
  "created_by_user_id" uuid NULL,
  "course_id" uuid NULL,
  "requested_course_name" character varying(255) NOT NULL,
  "expense_category" character varying(100) NULL,
  "comment" text NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  "updated_at" timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("id"),
  CONSTRAINT "applications_organizations_dzo" FOREIGN KEY ("dzo_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "applications_users_creator" FOREIGN KEY ("created_by_user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "applications_courses_course" FOREIGN KEY ("course_id") REFERENCES "public"."courses" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "application_kind" to table: "applications"
CREATE INDEX "application_kind" ON "public"."applications" ("kind");
-- create index "application_status" to table: "applications"
CREATE INDEX "application_status" ON "public"."applications" ("status");
-- create index "application_dzo_id" to table: "applications"
CREATE INDEX "application_dzo_id" ON "public"."applications" ("dzo_id");
-- create index "application_created_by_user_id" to table: "applications"
CREATE INDEX "application_created_by_user_id" ON "public"."applications" ("created_by_user_id");
-- create index "application_course_id" to table: "applications"
CREATE INDEX "application_course_id" ON "public"."applications" ("course_id");

-- create "application_participants" table
CREATE TABLE "public"."application_participants" (
  "id" uuid NOT NULL,
  "application_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY ("id"),
  CONSTRAINT "application_participants_applications_application" FOREIGN KEY ("application_id") REFERENCES "public"."applications" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "application_participants_users_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "application_participant_application_id" to table: "application_participants"
CREATE INDEX "application_participant_application_id" ON "public"."application_participants" ("application_id");
-- create index "application_participant_user_id" to table: "application_participants"
CREATE INDEX "application_participant_user_id" ON "public"."application_participants" ("user_id");
-- create unique index "application_participant_application_id_user_id_key" to table: "application_participants"
CREATE UNIQUE INDEX "application_participant_application_id_user_id_key" ON "public"."application_participants" ("application_id", "user_id");

-- create "employees" table
CREATE TABLE "public"."employees" (
  "id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- create "organizations" table
CREATE TABLE "public"."organizations" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "code" character varying NOT NULL,
  "type" character varying NOT NULL DEFAULT 'subsidiary',
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "parent_id" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "organizations_organizations_children" FOREIGN KEY ("parent_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "organization_code" to table: "organizations"
CREATE INDEX "organization_code" ON "public"."organizations" ("code");
-- create index "organization_parent_id" to table: "organizations"
CREATE INDEX "organization_parent_id" ON "public"."organizations" ("parent_id");
-- create index "organizations_code_key" to table: "organizations"
CREATE UNIQUE INDEX "organizations_code_key" ON "public"."organizations" ("code");
-- create "clients" table
CREATE TABLE "public"."clients" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "domain" character varying NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- create "users" table
CREATE TABLE "public"."users" (
  "id" uuid NOT NULL,
  "keycloak_user_id" character varying NOT NULL,
  "email" character varying NOT NULL,
  "role" character varying NOT NULL DEFAULT 'EMP',
  "dzo_id" uuid NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "is_onboarded" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "user_dzo_id" to table: "users"
CREATE INDEX "user_dzo_id" ON "public"."users" ("dzo_id");
-- create index "user_email" to table: "users"
CREATE INDEX "user_email" ON "public"."users" ("email");
-- create index "users_keycloak_user_id_key" to table: "users"
CREATE UNIQUE INDEX "users_keycloak_user_id_key" ON "public"."users" ("keycloak_user_id");
-- create "events" table
CREATE TABLE "public"."events" (
  "id" uuid NOT NULL,
  "title" character varying NOT NULL,
  "description" character varying NULL,
  "zoom_link" character varying NULL,
  "event_date" timestamptz NOT NULL,
  "materials_url" character varying NULL,
  "status" character varying NOT NULL DEFAULT 'DRAFT',
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "client_id" uuid NOT NULL,
  "host_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "events_clients_events" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "events_users_hosted_events" FOREIGN KEY ("host_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create "event_participants" table
CREATE TABLE "public"."event_participants" (
  "id" uuid NOT NULL,
  "joined_at" timestamptz NULL,
  "attendance_status" character varying NOT NULL DEFAULT 'PENDING',
  "reviewed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "employee_id" uuid NOT NULL,
  "event_id" uuid NOT NULL,
  "reviewed_by" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "event_participants_employees_event_participations" FOREIGN KEY ("employee_id") REFERENCES "public"."employees" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "event_participants_events_participants" FOREIGN KEY ("event_id") REFERENCES "public"."events" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "event_participants_users_reviewed_participations" FOREIGN KEY ("reviewed_by") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

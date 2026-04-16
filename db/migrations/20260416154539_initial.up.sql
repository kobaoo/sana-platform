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
<<<<<<<< HEAD:db/migrations/20260416162209_initial.sql
-- Create "training_events" table
CREATE TABLE "public"."training_events" (
  "id" uuid NOT NULL,
  "title" character varying NOT NULL,
  "start_date" timestamptz NOT NULL,
  "end_date" timestamptz NOT NULL,
  "location_type" character varying NOT NULL,
  "location_city" character varying NULL,
  "category_id" uuid NOT NULL,
  "direction" character varying NULL,
  "dzo_id" uuid NOT NULL,
  "dzo_contract_id" uuid NULL,
  "participants_count" bigint NOT NULL,
  "cost_per_person_vat" double precision NULL,
  "cost_group_vat" double precision NULL,
  "kyu_hourly_rate" double precision NULL,
  "supplier_id" uuid NULL,
  "supplier_contract_id" uuid NULL,
  "supplier_cost_vat" double precision NULL,
  "supplier_cost_currency" double precision NULL,
  "supplier_currency" character varying NULL,
  "local_content_pct" double precision NULL,
  PRIMARY KEY ("id")
);
-- Create "training_participants" table
CREATE TABLE "public"."training_participants" (
  "id" uuid NOT NULL,
  "event_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "status" character varying NOT NULL,
  "certificate_id" uuid NULL,
  PRIMARY KEY ("id")
);
-- Create "users" table
========
-- create "users" table
>>>>>>>> origin/dev:db/migrations/20260416154539_initial.up.sql
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
-- Create "requests" table
CREATE TABLE "public"."requests" (
  "id" uuid NOT NULL,
  "entity_id" uuid NOT NULL,
  "entity_type" character varying NOT NULL,
  "step" bigint NOT NULL,
  "created_at" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "initiator_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "requests_users_requests" FOREIGN KEY ("initiator_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

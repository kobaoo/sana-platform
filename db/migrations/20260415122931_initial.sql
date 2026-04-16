-- Create "organizations" table
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
-- Create index "organization_code" to table: "organizations"
CREATE INDEX "organization_code" ON "public"."organizations" ("code");
-- Create index "organization_parent_id" to table: "organizations"
CREATE INDEX "organization_parent_id" ON "public"."organizations" ("parent_id");
-- Create index "organizations_code_key" to table: "organizations"
CREATE UNIQUE INDEX "organizations_code_key" ON "public"."organizations" ("code");
-- Create "users" table
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
-- Create index "user_dzo_id" to table: "users"
CREATE INDEX "user_dzo_id" ON "public"."users" ("dzo_id");
-- Create index "user_email" to table: "users"
CREATE INDEX "user_email" ON "public"."users" ("email");
-- Create index "users_keycloak_user_id_key" to table: "users"
CREATE UNIQUE INDEX "users_keycloak_user_id_key" ON "public"."users" ("keycloak_user_id");

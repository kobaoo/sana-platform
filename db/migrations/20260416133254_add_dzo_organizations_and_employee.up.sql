-- Create "dzo_organizations" table
CREATE TABLE "public"."dzo_organizations" (
  "id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "short_name" character varying NULL,
  "bin" character varying NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);
-- Create "employees" table
CREATE TABLE "public"."employees" (
  "id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "position" character varying NULL,
  "full_name" character varying NOT NULL,
  "short_name" character varying NULL,
  "department" character varying NULL,
  "direction" character varying NULL,
  "email" character varying NOT NULL,
  "internal_phone" character varying NULL,
  "birth_date" date NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "user_id" uuid NULL,
  "dzo_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "employees_dzo_organizations_employees" FOREIGN KEY ("dzo_id") REFERENCES "public"."dzo_organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

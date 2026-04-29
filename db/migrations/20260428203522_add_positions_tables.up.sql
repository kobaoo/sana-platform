-- create "general_positions" table
CREATE TABLE "public"."general_positions" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "description" text NULL,
  "is_deleted" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- create "dzo_position_titles" table
CREATE TABLE "public"."dzo_position_titles" (
  "id" uuid NOT NULL,
  "local_title" character varying NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "is_deleted" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "dzo_id" uuid NOT NULL,
  "general_position_id" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "dzo_position_titles_dzo_organizations_position_titles" FOREIGN KEY ("dzo_id") REFERENCES "public"."dzo_organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "dzo_position_titles_general_positions_dzo_position_titles" FOREIGN KEY ("general_position_id") REFERENCES "public"."general_positions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "dzopositiontitle_dzo_id_local_title" to table: "dzo_position_titles"
CREATE UNIQUE INDEX "dzopositiontitle_dzo_id_local_title" ON "public"."dzo_position_titles" ("dzo_id", "local_title");
-- modify "employees" table
ALTER TABLE "public"."employees" DROP COLUMN "position", ADD COLUMN "dzo_position_id" uuid NULL, ADD CONSTRAINT "employees_dzo_position_titles_employees" FOREIGN KEY ("dzo_position_id") REFERENCES "public"."dzo_position_titles" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;

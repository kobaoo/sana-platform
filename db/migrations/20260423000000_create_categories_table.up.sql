-- Create "categories" table
CREATE TABLE "public"."categories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "description" text NULL,
  PRIMARY KEY ("id")
);

-- Create index on name for faster lookups
CREATE INDEX "categories_name_idx" ON "public"."categories" ("name");
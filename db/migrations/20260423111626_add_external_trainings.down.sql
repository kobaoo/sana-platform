-- reverse: create "external_training_events" table
DROP TABLE "public"."external_training_events";
-- reverse: modify "categories" table
ALTER TABLE "public"."categories" ALTER COLUMN "description" TYPE text, ALTER COLUMN "name" TYPE character varying(100), ALTER COLUMN "id" SET DEFAULT gen_random_uuid();
-- reverse: drop index "categories_name_idx" from table: "categories"
CREATE INDEX "categories_name_idx" ON "public"."categories" ("name");

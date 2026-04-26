-- modify "scorm_courses" table
ALTER TABLE "public"."scorm_courses" ADD COLUMN "image_url" character varying NULL;
-- create "categories" table
CREATE TABLE "public"."categories" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "description" text NULL,
  PRIMARY KEY ("id")
);

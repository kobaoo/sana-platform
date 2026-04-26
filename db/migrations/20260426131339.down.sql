-- reverse: create "categories" table
DROP TABLE "public"."categories";
-- reverse: modify "scorm_courses" table
ALTER TABLE "public"."scorm_courses" DROP COLUMN "image_url";

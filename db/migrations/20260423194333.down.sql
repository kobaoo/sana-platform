-- reverse: modify "scorm_progresses" table
ALTER TABLE "public"."scorm_progresses" DROP COLUMN "suspend_data";
-- reverse: modify "scorm_courses" table
ALTER TABLE "public"."scorm_courses" DROP COLUMN "category_ids", ADD COLUMN "category_id" uuid NOT NULL;

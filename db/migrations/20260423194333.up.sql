-- modify "scorm_courses" table
ALTER TABLE "public"."scorm_courses" DROP COLUMN "category_id", ADD COLUMN "category_ids" jsonb NOT NULL;
-- modify "scorm_progresses" table
ALTER TABLE "public"."scorm_progresses" ADD COLUMN "suspend_data" text NULL;

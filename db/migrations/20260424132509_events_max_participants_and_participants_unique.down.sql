-- drop unique index on event_participants
DROP INDEX IF EXISTS "public"."eventparticipant_event_id_employee_id";

-- revert events changes
ALTER TABLE "public"."events"
  ALTER COLUMN "status" SET DEFAULT 'DRAFT',
  ALTER COLUMN "zoom_link" DROP NOT NULL;

ALTER TABLE "public"."events"
  DROP COLUMN "max_participants";

-- backfill: DRAFT -> ACTIVE (DRAFT is being removed from the events.status enum)
UPDATE "public"."events" SET "status" = 'ACTIVE' WHERE "status" = 'DRAFT';

-- backfill: zoom_link NULL -> '' (column becomes NOT NULL)
UPDATE "public"."events" SET "zoom_link" = '' WHERE "zoom_link" IS NULL;

-- add required max_participants, make zoom_link required, flip default status
ALTER TABLE "public"."events"
  ADD COLUMN "max_participants" bigint NOT NULL DEFAULT 0;

ALTER TABLE "public"."events"
  ALTER COLUMN "zoom_link" SET NOT NULL,
  ALTER COLUMN "status" SET DEFAULT 'ACTIVE';

-- prevent duplicate enrollment of the same employee into the same event
CREATE UNIQUE INDEX "eventparticipant_event_id_employee_id"
  ON "public"."event_participants" ("event_id", "employee_id");

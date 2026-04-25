-- modify "external_training_events" table
ALTER TABLE "public"."external_training_events" ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;

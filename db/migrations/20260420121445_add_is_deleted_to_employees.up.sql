-- modify "employees" table
ALTER TABLE "public"."employees" ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;

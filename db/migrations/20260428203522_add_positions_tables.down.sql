-- reverse: modify "employees" table
ALTER TABLE "public"."employees" DROP CONSTRAINT "employees_dzo_position_titles_employees", DROP COLUMN "dzo_position_id", ADD COLUMN "position" character varying NULL;
-- reverse: create index "dzopositiontitle_dzo_id_local_title" to table: "dzo_position_titles"
DROP INDEX "public"."dzopositiontitle_dzo_id_local_title";
-- reverse: create "dzo_position_titles" table
DROP TABLE "public"."dzo_position_titles";
-- reverse: create "general_positions" table
DROP TABLE "public"."general_positions";

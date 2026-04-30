-- reverse: create index "dzopositiontitle_dzo_id_local_title" to table: "dzo_position_titles"
DROP INDEX "public"."dzopositiontitle_dzo_id_local_title";
-- reverse: drop index "dzopositiontitle_dzo_id_local_title" from table: "dzo_position_titles"
CREATE UNIQUE INDEX "dzopositiontitle_dzo_id_local_title" ON "public"."dzo_position_titles" ("dzo_id", "local_title");

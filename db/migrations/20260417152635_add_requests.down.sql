-- reverse: create index "request_step" to table: "requests"
DROP INDEX "public"."request_step";
-- reverse: create index "request_status" to table: "requests"
DROP INDEX "public"."request_status";
-- reverse: create index "request_initiator_id" to table: "requests"
DROP INDEX "public"."request_initiator_id";
-- reverse: create index "request_entity_id" to table: "requests"
DROP INDEX "public"."request_entity_id";
-- reverse: create "requests" table
DROP TABLE "public"."requests";

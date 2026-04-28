-- reverse: create index "eventparticipant_event_id_employee_id" to table: "event_participants"
DROP INDEX "public"."eventparticipant_event_id_employee_id";
-- reverse: create "event_participants" table
DROP TABLE "public"."event_participants";
-- reverse: create "events" table
DROP TABLE "public"."events";
-- reverse: create index "notification_user_id_type_entity_type_entity_id" to table: "notifications"
DROP INDEX "public"."notification_user_id_type_entity_type_entity_id";
-- reverse: create index "notification_user_id" to table: "notifications"
DROP INDEX "public"."notification_user_id";
-- reverse: create index "notification_status" to table: "notifications"
DROP INDEX "public"."notification_status";
-- reverse: create "notifications" table
DROP TABLE "public"."notifications";
-- reverse: create index "certificate_is_active" to table: "certificates"
DROP INDEX "public"."certificate_is_active";
-- reverse: create index "certificate_expiry_date" to table: "certificates"
DROP INDEX "public"."certificate_expiry_date";
-- reverse: create index "certificate_employee_id" to table: "certificates"
DROP INDEX "public"."certificate_employee_id";
-- reverse: create "certificates" table
DROP TABLE "public"."certificates";

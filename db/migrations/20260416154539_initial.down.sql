-- reverse: create "requests" table
DROP TABLE "public"."requests";
-- reverse: create index "users_keycloak_user_id_key" to table: "users"
DROP INDEX "public"."users_keycloak_user_id_key";
-- reverse: create index "user_email" to table: "users"
DROP INDEX "public"."user_email";
-- reverse: create index "user_dzo_id" to table: "users"
DROP INDEX "public"."user_dzo_id";
-- reverse: create "users" table
DROP TABLE "public"."users";
-- reverse: create "training_participants" table
DROP TABLE "public"."training_participants";
-- reverse: create "training_events" table
DROP TABLE "public"."training_events";
-- reverse: create index "organizations_code_key" to table: "organizations"
DROP INDEX "public"."organizations_code_key";
-- reverse: create index "organization_parent_id" to table: "organizations"
DROP INDEX "public"."organization_parent_id";
-- reverse: create index "organization_code" to table: "organizations"
DROP INDEX "public"."organization_code";
-- reverse: create "organizations" table
DROP TABLE "public"."organizations";

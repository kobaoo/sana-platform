-- reverse: create index "request_target_dzo_id" to table: "requests"
DROP INDEX "public"."request_target_dzo_id";
-- reverse: create index "request_step" to table: "requests"
DROP INDEX "public"."request_step";
-- reverse: create index "request_status" to table: "requests"
DROP INDEX "public"."request_status";
-- reverse: create index "request_request_type" to table: "requests"
DROP INDEX "public"."request_request_type";
-- reverse: create index "request_parent_request_id" to table: "requests"
DROP INDEX "public"."request_parent_request_id";
-- reverse: create index "request_kind" to table: "requests"
DROP INDEX "public"."request_kind";
-- reverse: create index "request_initiator_id" to table: "requests"
DROP INDEX "public"."request_initiator_id";
-- reverse: create index "request_entity_id" to table: "requests"
DROP INDEX "public"."request_entity_id";
-- reverse: create index "request_assigned_hr_id" to table: "requests"
DROP INDEX "public"."request_assigned_hr_id";
-- reverse: create "requests" table
DROP TABLE "public"."requests";
-- reverse: create "external_training_events" table
DROP TABLE "public"."external_training_events";
-- reverse: create index "users_keycloak_user_id_key" to table: "users"
DROP INDEX "public"."users_keycloak_user_id_key";
-- reverse: create index "user_email" to table: "users"
DROP INDEX "public"."user_email";
-- reverse: create index "user_dzo_id" to table: "users"
DROP INDEX "public"."user_dzo_id";
-- reverse: create "users" table
DROP TABLE "public"."users";
-- reverse: create "clients" table
DROP TABLE "public"."clients";
-- reverse: create index "suppliers_bin_or_iin_key" to table: "suppliers"
DROP INDEX "public"."suppliers_bin_or_iin_key";
-- reverse: create index "supplier_type" to table: "suppliers"
DROP INDEX "public"."supplier_type";
-- reverse: create index "supplier_is_active" to table: "suppliers"
DROP INDEX "public"."supplier_is_active";
-- reverse: create index "supplier_client_id" to table: "suppliers"
DROP INDEX "public"."supplier_client_id";
-- reverse: create "suppliers" table
DROP TABLE "public"."suppliers";
-- reverse: create index "contract_supplier_supplier_id" to table: "contract_suppliers"
DROP INDEX "public"."contract_supplier_supplier_id";
-- reverse: create index "contract_supplier_contract_number" to table: "contract_suppliers"
DROP INDEX "public"."contract_supplier_contract_number";
-- reverse: create "contract_suppliers" table
DROP TABLE "public"."contract_suppliers";
-- reverse: create "categories" table
DROP TABLE "public"."categories";
-- reverse: create "employees" table
DROP TABLE "public"."employees";
-- reverse: create index "organizations_code_key" to table: "organizations"
DROP INDEX "public"."organizations_code_key";
-- reverse: create index "organization_parent_id" to table: "organizations"
DROP INDEX "public"."organization_parent_id";
-- reverse: create index "organization_code" to table: "organizations"
DROP INDEX "public"."organization_code";
-- reverse: create "organizations" table
DROP TABLE "public"."organizations";
-- reverse: create index "requesttargetdzo_request_id_dzo_id" to table: "request_target_dzos"
DROP INDEX "public"."requesttargetdzo_request_id_dzo_id";
-- reverse: create index "requesttargetdzo_request_id" to table: "request_target_dzos"
DROP INDEX "public"."requesttargetdzo_request_id";
-- reverse: create index "requesttargetdzo_dzo_id" to table: "request_target_dzos"
DROP INDEX "public"."requesttargetdzo_dzo_id";
-- reverse: create "request_target_dzos" table
DROP TABLE "public"."request_target_dzos";
-- reverse: create "training_events" table
DROP TABLE "public"."training_events";
-- reverse: create "dzo_organizations" table
DROP TABLE "public"."dzo_organizations";
-- reverse: create "training_participants" table
DROP TABLE "public"."training_participants";
-- reverse: create index "contract_supplier_history_contract_id" to table: "contract_supplier_histories"
DROP INDEX "public"."contract_supplier_history_contract_id";
-- reverse: create index "contract_supplier_history_changed_at" to table: "contract_supplier_histories"
DROP INDEX "public"."contract_supplier_history_changed_at";
-- reverse: create index "contract_supplier_histories_history_id_key" to table: "contract_supplier_histories"
DROP INDEX "public"."contract_supplier_histories_history_id_key";
-- reverse: create "contract_supplier_histories" table
DROP TABLE "public"."contract_supplier_histories";
-- reverse: create index "requestparticipant_request_id_employee_id" to table: "request_participants"
DROP INDEX "public"."requestparticipant_request_id_employee_id";
-- reverse: create index "requestparticipant_request_id" to table: "request_participants"
DROP INDEX "public"."requestparticipant_request_id";
-- reverse: create index "requestparticipant_employee_id" to table: "request_participants"
DROP INDEX "public"."requestparticipant_employee_id";
-- reverse: create "request_participants" table
DROP TABLE "public"."request_participants";
-- reverse: create index "requestdzocontract_request_id_dzo_id" to table: "request_dzo_contracts"
DROP INDEX "public"."requestdzocontract_request_id_dzo_id";
-- reverse: create index "requestdzocontract_request_id" to table: "request_dzo_contracts"
DROP INDEX "public"."requestdzocontract_request_id";
-- reverse: create index "requestdzocontract_dzo_id" to table: "request_dzo_contracts"
DROP INDEX "public"."requestdzocontract_dzo_id";
-- reverse: create "request_dzo_contracts" table
DROP TABLE "public"."request_dzo_contracts";

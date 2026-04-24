-- reverse: create index "contract_supplier_supplier_id" to table: "contract_suppliers"
DROP INDEX "public"."contract_supplier_supplier_id";
-- reverse: create index "contract_supplier_contract_number" to table: "contract_suppliers"
DROP INDEX "public"."contract_supplier_contract_number";
-- reverse: create "contract_suppliers" table
DROP TABLE "public"."contract_suppliers";
-- reverse: create index "contract_supplier_history_contract_id" to table: "contract_supplier_histories"
DROP INDEX "public"."contract_supplier_history_contract_id";
-- reverse: create index "contract_supplier_history_changed_at" to table: "contract_supplier_histories"
DROP INDEX "public"."contract_supplier_history_changed_at";
-- reverse: create index "contract_supplier_histories_history_id_key" to table: "contract_supplier_histories"
DROP INDEX "public"."contract_supplier_histories_history_id_key";
-- reverse: create "contract_supplier_histories" table
DROP TABLE "public"."contract_supplier_histories";

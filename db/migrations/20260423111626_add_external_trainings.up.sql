-- drop index "categories_name_idx" from table: "categories"
DROP INDEX "public"."categories_name_idx";
-- modify "categories" table
ALTER TABLE "public"."categories" ALTER COLUMN "id" DROP DEFAULT, ALTER COLUMN "name" TYPE character varying, ALTER COLUMN "description" TYPE character varying;
-- create "external_training_events" table
CREATE TABLE "public"."external_training_events" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "format" character varying NULL,
  "capacity" bigint NULL,
  "supplier_cost_vat" double precision NULL,
  "start_date" timestamptz NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  "category_id" uuid NULL,
  "contract_id" uuid NOT NULL,
  "supplier_id" uuid NOT NULL,
  "responsible_user_id" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "external_training_events_categories_external_training_events" FOREIGN KEY ("category_id") REFERENCES "public"."categories" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "external_training_events_contract_suppliers_external_training_e" FOREIGN KEY ("contract_id") REFERENCES "public"."contract_suppliers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "external_training_events_suppliers_external_training_events" FOREIGN KEY ("supplier_id") REFERENCES "public"."suppliers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "external_training_events_users_responsible_external_training_ev" FOREIGN KEY ("responsible_user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

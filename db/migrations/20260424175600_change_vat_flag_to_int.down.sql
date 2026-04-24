-- reverse: modify "contract_suppliers" table
ALTER TABLE "public"."contract_suppliers" ALTER COLUMN "vat_flag" TYPE boolean, ALTER COLUMN "vat_flag" SET DEFAULT false;

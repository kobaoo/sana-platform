-- modify "contract_suppliers" table
ALTER TABLE "public"."contract_suppliers" ALTER COLUMN "vat_flag" DROP DEFAULT;
ALTER TABLE "public"."contract_suppliers" ALTER COLUMN "vat_flag" TYPE bigint USING (CASE WHEN vat_flag = true THEN 1 ELSE 0 END);
ALTER TABLE "public"."contract_suppliers" ALTER COLUMN "vat_flag" SET DEFAULT 0;

-- modify "contract_suppliers" table
ALTER TABLE "public"."contract_suppliers" ADD COLUMN "file_key" character varying NULL, ADD COLUMN "file_name" character varying NULL, ADD COLUMN "file_size" bigint NULL, ADD COLUMN "file_mime_type" character varying NULL;

-- reverse: modify "contract_suppliers" table
ALTER TABLE "public"."contract_suppliers" DROP COLUMN "file_mime_type", DROP COLUMN "file_size", DROP COLUMN "file_name", DROP COLUMN "file_key";

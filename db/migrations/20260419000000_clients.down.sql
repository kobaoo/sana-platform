-- reverse clients up migration
ALTER TABLE "users" DROP CONSTRAINT "users_client_id_fkey";
ALTER TABLE "users" DROP COLUMN "client_id";
DROP TABLE "clients";

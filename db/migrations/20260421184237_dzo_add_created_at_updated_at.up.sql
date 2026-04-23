-- modify "clients" table
ALTER TABLE "public"."clients" ALTER COLUMN "name" TYPE character varying, ALTER COLUMN "domain" TYPE character varying, ALTER COLUMN "language" TYPE character varying, ALTER COLUMN "user_limit" TYPE bigint;
-- modify "dzo_organizations" table
ALTER TABLE "public"."dzo_organizations" ADD COLUMN "created_at" timestamptz NOT NULL, ADD COLUMN "updated_at" timestamptz NOT NULL;
-- modify "users" table
ALTER TABLE "public"."users" DROP CONSTRAINT "users_client_id_fkey", ADD CONSTRAINT "users_clients_users" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;

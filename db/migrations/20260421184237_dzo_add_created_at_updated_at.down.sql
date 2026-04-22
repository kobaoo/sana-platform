-- reverse: modify "users" table
ALTER TABLE "public"."users" DROP CONSTRAINT "users_clients_users", ADD CONSTRAINT "users_client_id_fkey" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "dzo_organizations" table
ALTER TABLE "public"."dzo_organizations" DROP COLUMN "updated_at", DROP COLUMN "created_at";
-- reverse: modify "clients" table
ALTER TABLE "public"."clients" ALTER COLUMN "user_limit" TYPE integer, ALTER COLUMN "language" TYPE character varying(10), ALTER COLUMN "domain" TYPE character varying(100), ALTER COLUMN "name" TYPE character varying(255);

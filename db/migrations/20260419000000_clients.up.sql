-- create "clients" table
CREATE TABLE "clients" (
  "id" uuid NOT NULL,
  "name" character varying(255) NOT NULL,
  "domain" character varying(100) NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);

-- add client_id to users and foreign key
-- Users agreed database reset is ok so we can safely add NOT NULL if table is empty. But just in case, we do it safely:
ALTER TABLE "users" ADD COLUMN "client_id" uuid;
ALTER TABLE "users" ADD CONSTRAINT "users_client_id_fkey" FOREIGN KEY ("client_id") REFERENCES "clients" ("id") ON DELETE NO ACTION ON UPDATE NO ACTION;

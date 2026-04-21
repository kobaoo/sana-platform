-- Create "suppliers" table
CREATE TABLE "public"."suppliers" (
  "id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "type" character varying NOT NULL,
  "name" character varying NOT NULL,
  "bin_or_iin" character varying NULL,
  "local_content_pct" numeric(5,2) NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);
-- Create index "supplier_client_id" to table: "suppliers"
CREATE INDEX "supplier_client_id" ON "public"."suppliers" ("client_id");
-- Create index "supplier_is_active" to table: "suppliers"
CREATE INDEX "supplier_is_active" ON "public"."suppliers" ("is_active");
-- Create index "supplier_type" to table: "suppliers"
CREATE INDEX "supplier_type" ON "public"."suppliers" ("type");

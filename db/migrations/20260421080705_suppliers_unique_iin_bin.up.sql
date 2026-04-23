-- Create index "suppliers_bin_or_iin_key" to table: "suppliers"
CREATE UNIQUE INDEX "suppliers_bin_or_iin_key" ON "public"."suppliers" ("bin_or_iin");

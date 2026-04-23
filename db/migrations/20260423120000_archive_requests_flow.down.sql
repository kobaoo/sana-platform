DROP INDEX "public"."requestdzocontract_dzo_id";
DROP INDEX "public"."requestdzocontract_request_id";
DROP TABLE "public"."request_dzo_contracts";

DROP INDEX "public"."requestemployee_employee_id";
DROP INDEX "public"."requestemployee_request_id";
DROP TABLE "public"."request_employees";

DROP INDEX "public"."request_kind";

ALTER TABLE "public"."requests"
  DROP COLUMN "completed_at",
  DROP COLUMN "updated_at",
  DROP COLUMN "category",
  DROP COLUMN "title",
  DROP COLUMN "kind";

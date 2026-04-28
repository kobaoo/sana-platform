-- create "certificates" table
CREATE TABLE "public"."certificates" (
  "id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "type" character varying NOT NULL,
  "title" character varying NOT NULL,
  "issued_date" date NOT NULL,
  "expiry_date" date NULL,
  "file_url" text NULL,
  "uploaded_by" uuid NULL,
  "entity_type" character varying NOT NULL,
  "entity_id" uuid NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "certificate_employee_id" to table: "certificates"
CREATE INDEX "certificate_employee_id" ON "public"."certificates" ("employee_id");
-- create index "certificate_expiry_date" to table: "certificates"
CREATE INDEX "certificate_expiry_date" ON "public"."certificates" ("expiry_date");
-- create index "certificate_is_active" to table: "certificates"
CREATE INDEX "certificate_is_active" ON "public"."certificates" ("is_active");
-- create "notifications" table
CREATE TABLE "public"."notifications" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "type" character varying NOT NULL,
  "entity_type" character varying NOT NULL,
  "entity_id" uuid NOT NULL,
  "status" character varying NOT NULL DEFAULT 'PENDING',
  "sent_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "notification_status" to table: "notifications"
CREATE INDEX "notification_status" ON "public"."notifications" ("status");
-- create index "notification_user_id" to table: "notifications"
CREATE INDEX "notification_user_id" ON "public"."notifications" ("user_id");
-- create index "notification_user_id_type_entity_type_entity_id" to table: "notifications"
CREATE UNIQUE INDEX "notification_user_id_type_entity_type_entity_id" ON "public"."notifications" ("user_id", "type", "entity_type", "entity_id");
-- create "events" table
CREATE TABLE "public"."events" (
  "id" uuid NOT NULL,
  "title" character varying NOT NULL,
  "description" character varying NULL,
  "zoom_link" character varying NOT NULL,
  "event_date" timestamptz NOT NULL,
  "max_participants" bigint NOT NULL,
  "materials_url" character varying NULL,
  "status" character varying NOT NULL DEFAULT 'ACTIVE',
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "client_id" uuid NOT NULL,
  "host_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "events_clients_events" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "events_users_hosted_events" FOREIGN KEY ("host_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create "event_participants" table
CREATE TABLE "public"."event_participants" (
  "id" uuid NOT NULL,
  "joined_at" timestamptz NULL,
  "attendance_status" character varying NOT NULL DEFAULT 'PENDING',
  "reviewed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "employee_id" uuid NOT NULL,
  "event_id" uuid NOT NULL,
  "reviewed_by" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "event_participants_employees_event_participations" FOREIGN KEY ("employee_id") REFERENCES "public"."employees" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "event_participants_events_participants" FOREIGN KEY ("event_id") REFERENCES "public"."events" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "event_participants_users_reviewed_participations" FOREIGN KEY ("reviewed_by") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "eventparticipant_event_id_employee_id" to table: "event_participants"
CREATE UNIQUE INDEX "eventparticipant_event_id_employee_id" ON "public"."event_participants" ("event_id", "employee_id");

-- create "events" table
CREATE TABLE "public"."events" (
  "id" uuid NOT NULL,
  "title" character varying NOT NULL,
  "description" character varying NULL,
  "zoom_link" character varying NULL,
  "event_date" timestamptz NOT NULL,
  "materials_url" character varying NULL,
  "status" character varying NOT NULL DEFAULT 'DRAFT',
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

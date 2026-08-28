-- +goose Up
CREATE TABLE "companies" (
  "id" uuid,
  "code" varchar(20) NOT NULL,
  "name" varchar(160) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  "deleted_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_companies_deleted_at" ON "companies" ("deleted_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_companies_code" ON "companies" ("code");

-- Enum role berubah dari user/admin menjadi client/admin/inspector/cs, dan
-- user klien kini terikat pada perusahaan pemesan (asumsi A-03).
ALTER TABLE "users" ALTER COLUMN "role" SET DEFAULT 'client';
ALTER TABLE "users" ADD COLUMN "company_id" uuid;
CREATE INDEX IF NOT EXISTS "idx_users_company_id" ON "users" ("company_id");
ALTER TABLE "users" ADD CONSTRAINT "fk_users_company" FOREIGN KEY ("company_id") REFERENCES "companies"("id");

CREATE TABLE "inspection_types" (
  "id" uuid,
  "code" varchar(40) NOT NULL,
  "name" varchar(120) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_inspection_types_code" ON "inspection_types" ("code");

CREATE TABLE "job_orders" (
  "id" uuid,
  "reference_number" varchar(30) NOT NULL,
  "company_id" uuid NOT NULL,
  "created_by_id" uuid NOT NULL,
  "inspection_type_id" uuid NOT NULL,
  "object_description" varchar(255) NOT NULL,
  "location_name" varchar(160) NOT NULL,
  "location_address" text NOT NULL,
  "city" varchar(80) NOT NULL,
  "scheduled_start_at" timestamptz NOT NULL,
  "scheduled_end_at" timestamptz NOT NULL,
  "inspector_id" uuid,
  "current_status" varchar(20) NOT NULL DEFAULT 'requested',
  "status_changed_at" timestamptz NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  "deleted_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_job_orders_deleted_at" ON "job_orders" ("deleted_at");
CREATE INDEX IF NOT EXISTS "idx_job_orders_status_changed" ON "job_orders" ("current_status","status_changed_at");
CREATE INDEX IF NOT EXISTS "idx_job_orders_inspector_status" ON "job_orders" ("inspector_id","current_status");
CREATE INDEX IF NOT EXISTS "idx_job_orders_inspection_type_id" ON "job_orders" ("inspection_type_id");
CREATE INDEX IF NOT EXISTS "idx_job_orders_created_by_id" ON "job_orders" ("created_by_id");
CREATE INDEX IF NOT EXISTS "idx_job_orders_company_created" ON "job_orders" ("company_id","created_at" desc);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_job_orders_reference_number" ON "job_orders" ("reference_number");

CREATE TABLE "job_status_events" (
  "id" uuid,
  "seq" bigserial,
  "job_order_id" uuid NOT NULL,
  "from_status" varchar(20),
  "to_status" varchar(20) NOT NULL,
  "actor_id" uuid,
  "actor_role" varchar(20) NOT NULL,
  "occurred_at" timestamptz NOT NULL,
  "received_at" timestamptz NOT NULL,
  "occurred_at_adjusted" boolean NOT NULL DEFAULT false,
  "client_event_id" varchar(64),
  -- Tanpa DEFAULT dengan sengaja: GORM membuang nilai zero dari INSERT untuk
  -- kolom yang punya default, sehingga Accepted=false akan tersimpan sebagai
  -- true dan mematikan keputusan B-07.
  "accepted" boolean NOT NULL,
  "rejection_reason" varchar(40),
  "is_correction" boolean NOT NULL DEFAULT false,
  "reason" text,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_job_status_events_actor_id" ON "job_status_events" ("actor_id");
-- Penegak idempotency (keputusan B-03). client_event_id NULL tidak pernah
-- bertabrakan di unique index Postgres, sehingga event buatan sistem bebas berulang.
CREATE UNIQUE INDEX IF NOT EXISTS "idx_events_idempotency" ON "job_status_events" ("job_order_id","client_event_id");
CREATE INDEX IF NOT EXISTS "idx_events_order_occurred" ON "job_status_events" ("job_order_id","occurred_at");
-- Kursor monotonik untuk menentukan status terkini dan memutar ulang event yang
-- terlewat saat klien menyambung kembali (keputusan B-01).
CREATE UNIQUE INDEX IF NOT EXISTS "idx_job_status_events_seq" ON "job_status_events" ("seq");

CREATE TABLE "cancellation_requests" (
  "id" uuid,
  "job_order_id" uuid NOT NULL,
  "requested_by_id" uuid NOT NULL,
  "reason" text NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "decided_by_id" uuid,
  "decided_at" timestamptz,
  "decision_note" text,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_cancel_order_status" ON "cancellation_requests" ("job_order_id","status");

CREATE TABLE "job_order_alerts" (
  "id" uuid,
  "job_order_id" uuid NOT NULL,
  "type" varchar(30) NOT NULL,
  "source_event_id" uuid,
  "message" text NOT NULL,
  "resolved_at" timestamptz,
  "resolved_by_id" uuid,
  "created_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_job_order_alerts_resolved_at" ON "job_order_alerts" ("resolved_at");
CREATE INDEX IF NOT EXISTS "idx_job_order_alerts_job_order_id" ON "job_order_alerts" ("job_order_id");

CREATE TABLE "reference_counters" (
  "scope" varchar(20),
  "year" bigint,
  "last_number" bigint NOT NULL DEFAULT 0,
  "updated_at" timestamptz,
  PRIMARY KEY ("scope","year")
);

ALTER TABLE "job_orders" ADD CONSTRAINT "fk_job_orders_company" FOREIGN KEY ("company_id") REFERENCES "companies"("id");
ALTER TABLE "job_orders" ADD CONSTRAINT "fk_job_orders_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id");
ALTER TABLE "job_orders" ADD CONSTRAINT "fk_job_orders_inspection_type" FOREIGN KEY ("inspection_type_id") REFERENCES "inspection_types"("id");
ALTER TABLE "job_orders" ADD CONSTRAINT "fk_job_orders_inspector" FOREIGN KEY ("inspector_id") REFERENCES "users"("id");
ALTER TABLE "job_status_events" ADD CONSTRAINT "fk_job_status_events_actor" FOREIGN KEY ("actor_id") REFERENCES "users"("id");
ALTER TABLE "job_status_events" ADD CONSTRAINT "fk_job_orders_status_events" FOREIGN KEY ("job_order_id") REFERENCES "job_orders"("id") ON DELETE CASCADE;
ALTER TABLE "cancellation_requests" ADD CONSTRAINT "fk_cancellation_requests_decided_by" FOREIGN KEY ("decided_by_id") REFERENCES "users"("id");
ALTER TABLE "cancellation_requests" ADD CONSTRAINT "fk_cancellation_requests_job_order" FOREIGN KEY ("job_order_id") REFERENCES "job_orders"("id") ON DELETE CASCADE;
ALTER TABLE "cancellation_requests" ADD CONSTRAINT "fk_cancellation_requests_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id");
ALTER TABLE "job_order_alerts" ADD CONSTRAINT "fk_job_order_alerts_job_order" FOREIGN KEY ("job_order_id") REFERENCES "job_orders"("id") ON DELETE CASCADE;
ALTER TABLE "job_order_alerts" ADD CONSTRAINT "fk_job_order_alerts_source_event" FOREIGN KEY ("source_event_id") REFERENCES "job_status_events"("id") ON DELETE CASCADE;

-- +goose Down
DROP TABLE IF EXISTS "reference_counters";
DROP TABLE IF EXISTS "job_order_alerts";
DROP TABLE IF EXISTS "cancellation_requests";
DROP TABLE IF EXISTS "job_status_events";
DROP TABLE IF EXISTS "job_orders";
DROP TABLE IF EXISTS "inspection_types";

ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "fk_users_company";
DROP INDEX IF EXISTS "idx_users_company_id";
ALTER TABLE "users" DROP COLUMN IF EXISTS "company_id";
ALTER TABLE "users" ALTER COLUMN "role" SET DEFAULT 'user';

DROP TABLE IF EXISTS "companies";

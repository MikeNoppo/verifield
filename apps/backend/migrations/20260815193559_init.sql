-- +goose Up
CREATE TABLE "users" (
  "id" uuid,
  "name" varchar(120) NOT NULL,
  "email" varchar(160) NOT NULL,
  "password" varchar(255) NOT NULL,
  "role" varchar(20) NOT NULL DEFAULT 'user',
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  "deleted_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_users_deleted_at" ON "users" ("deleted_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_users_email" ON "users" ("email");

-- +goose Down
DROP TABLE IF EXISTS "users";

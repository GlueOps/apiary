-- +goose Up
-- create "api_keys" table
CREATE TABLE "api_keys" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "org_id" text NULL,
  "scope" text NOT NULL DEFAULT '',
  "purpose" text NULL,
  "is_ephemeral" boolean NULL,
  "name" text NOT NULL DEFAULT '',
  "key_hash" text NOT NULL,
  "secret_hash" text NULL,
  "user_id" text NULL,
  "expires_at" timestamptz NULL,
  "revoked" boolean NOT NULL DEFAULT false,
  "prefix" text NULL,
  "last_used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_api_keys_key_hash" to table: "api_keys"
CREATE UNIQUE INDEX "idx_api_keys_key_hash" ON "api_keys" ("key_hash");

-- +goose Down
-- reverse: create index "idx_api_keys_key_hash" to table: "api_keys"
DROP INDEX "idx_api_keys_key_hash";
-- reverse: create "api_keys" table
DROP TABLE "api_keys";

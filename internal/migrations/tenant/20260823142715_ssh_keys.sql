-- +goose Up
-- create "ssh_keys" table
CREATE TABLE "ssh_keys" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "public_key" text NOT NULL,
  "encrypted_private_key" text NOT NULL,
  "private_iv" text NOT NULL,
  "private_tag" text NOT NULL,
  "private_key_id" uuid NULL,
  "fingerprint" text NOT NULL,
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_ssh_keys_fingerprint" to table: "ssh_keys"
CREATE INDEX "idx_ssh_keys_fingerprint" ON "ssh_keys" ("fingerprint");

-- +goose Down
-- reverse: create index "idx_ssh_keys_fingerprint" to table: "ssh_keys"
DROP INDEX "idx_ssh_keys_fingerprint";
-- reverse: create "ssh_keys" table
DROP TABLE "ssh_keys";

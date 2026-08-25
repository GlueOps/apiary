-- +goose Up
-- create "credentials" table
CREATE TABLE "credentials" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL DEFAULT '',
  "provider" character varying(50) NOT NULL,
  "kind" character varying(20) NOT NULL DEFAULT 'static_secret',
  "fingerprint" character varying(80) NOT NULL DEFAULT '',
  "key_id" uuid NULL,
  "ciphertext" text NOT NULL DEFAULT '',
  "iv" text NOT NULL DEFAULT '',
  "tag" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create "estates" table
CREATE TABLE "estates" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL DEFAULT '',
  "provider" character varying(50) NOT NULL,
  "scope" jsonb NOT NULL DEFAULT '{}',
  "scope_version" bigint NOT NULL DEFAULT 1,
  "credential_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_estates_credential" FOREIGN KEY ("credential_id") REFERENCES "credentials" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);
-- create index "idx_estates_scope" to table: "estates"
CREATE INDEX "idx_estates_scope" ON "estates" USING GIN ("scope");
-- create index "uniq_estates_credential" to table: "estates"
CREATE UNIQUE INDEX "uniq_estates_credential" ON "estates" ("credential_id");
-- create index "uniq_estates_provider_scope" to table: "estates"
CREATE UNIQUE INDEX "uniq_estates_provider_scope" ON "estates" ("provider", "scope");

-- +goose Down
-- reverse: create index "uniq_estates_provider_scope" to table: "estates"
DROP INDEX "uniq_estates_provider_scope";
-- reverse: create index "uniq_estates_credential" to table: "estates"
DROP INDEX "uniq_estates_credential";
-- reverse: create index "idx_estates_scope" to table: "estates"
DROP INDEX "idx_estates_scope";
-- reverse: create "estates" table
DROP TABLE "estates";
-- reverse: create "credentials" table
DROP TABLE "credentials";

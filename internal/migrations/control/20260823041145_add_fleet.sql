-- +goose Up
-- create "hives" table
CREATE TABLE "hives" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "provider" character varying(32) NOT NULL DEFAULT 'proxmox',
  "description" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_hives_name" to table: "hives"
CREATE UNIQUE INDEX "idx_hives_name" ON "hives" ("name");
-- create "bee_enrolment_tokens" table
CREATE TABLE "bee_enrolment_tokens" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NOT NULL,
  "token_hash" character varying(64) NOT NULL,
  "uses" bigint NOT NULL DEFAULT 0,
  "max_uses" bigint NOT NULL DEFAULT 0,
  "expires_at" timestamptz NOT NULL,
  "revoked_at" timestamptz NULL,
  "created_by_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_bee_enrolment_tokens_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_bee_enrolment_tokens_expires_at" to table: "bee_enrolment_tokens"
CREATE INDEX "idx_bee_enrolment_tokens_expires_at" ON "bee_enrolment_tokens" ("expires_at");
-- create index "idx_bee_enrolment_tokens_hive_id" to table: "bee_enrolment_tokens"
CREATE INDEX "idx_bee_enrolment_tokens_hive_id" ON "bee_enrolment_tokens" ("hive_id");
-- create index "idx_bee_enrolment_tokens_token_hash" to table: "bee_enrolment_tokens"
CREATE UNIQUE INDEX "idx_bee_enrolment_tokens_token_hash" ON "bee_enrolment_tokens" ("token_hash");
-- create "bees" table
CREATE TABLE "bees" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NOT NULL,
  "hostname" character varying(253) NOT NULL,
  "machine_id" character varying(64) NOT NULL,
  "credential_hash" character varying(64) NOT NULL,
  "status" character varying(16) NOT NULL DEFAULT 'offline',
  "agent_version" character varying(64) NULL,
  "last_seen_at" timestamptz NULL,
  "unavailable_until" timestamptz NULL,
  "unavailable_reason" character varying(200) NULL,
  "enrolled_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_bees_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_bees_credential_hash" to table: "bees"
CREATE UNIQUE INDEX "idx_bees_credential_hash" ON "bees" ("credential_hash");
-- create index "idx_bees_hive_hostname" to table: "bees"
CREATE INDEX "idx_bees_hive_hostname" ON "bees" ("hive_id", "hostname");
-- create index "idx_bees_last_seen_at" to table: "bees"
CREATE INDEX "idx_bees_last_seen_at" ON "bees" ("last_seen_at");
-- create index "idx_bees_machine_id" to table: "bees"
CREATE UNIQUE INDEX "idx_bees_machine_id" ON "bees" ("machine_id");
-- create "fleet_tasks" table
CREATE TABLE "fleet_tasks" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "bee_id" uuid NOT NULL,
  "kind" character varying(64) NOT NULL,
  "spec" jsonb NOT NULL DEFAULT '{}',
  "status" character varying(16) NOT NULL DEFAULT 'pending',
  "phase" character varying(64) NULL,
  "lease_expires_at" timestamptz NULL,
  "attempt" bigint NOT NULL DEFAULT 0,
  "result" jsonb NULL,
  "error" text NULL,
  "created_by_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "claimed_at" timestamptz NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_fleet_tasks_bee" FOREIGN KEY ("bee_id") REFERENCES "bees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_fleet_tasks_bee_status" to table: "fleet_tasks"
CREATE INDEX "idx_fleet_tasks_bee_status" ON "fleet_tasks" ("bee_id", "status");
-- create "fleet_task_logs" table
CREATE TABLE "fleet_task_logs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "task_id" uuid NOT NULL,
  "seq" bigint NOT NULL,
  "stream" character varying(8) NOT NULL DEFAULT 'stdout',
  "message" text NOT NULL,
  "logged_at" timestamptz NOT NULL,
  "received_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_fleet_task_logs_task" FOREIGN KEY ("task_id") REFERENCES "fleet_tasks" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "uniq_fleet_task_logs_task_seq" to table: "fleet_task_logs"
CREATE UNIQUE INDEX "uniq_fleet_task_logs_task_seq" ON "fleet_task_logs" ("task_id", "seq");

-- +goose Down
-- reverse: create index "uniq_fleet_task_logs_task_seq" to table: "fleet_task_logs"
DROP INDEX "uniq_fleet_task_logs_task_seq";
-- reverse: create "fleet_task_logs" table
DROP TABLE "fleet_task_logs";
-- reverse: create index "idx_fleet_tasks_bee_status" to table: "fleet_tasks"
DROP INDEX "idx_fleet_tasks_bee_status";
-- reverse: create "fleet_tasks" table
DROP TABLE "fleet_tasks";
-- reverse: create index "idx_bees_machine_id" to table: "bees"
DROP INDEX "idx_bees_machine_id";
-- reverse: create index "idx_bees_last_seen_at" to table: "bees"
DROP INDEX "idx_bees_last_seen_at";
-- reverse: create index "idx_bees_hive_hostname" to table: "bees"
DROP INDEX "idx_bees_hive_hostname";
-- reverse: create index "idx_bees_credential_hash" to table: "bees"
DROP INDEX "idx_bees_credential_hash";
-- reverse: create "bees" table
DROP TABLE "bees";
-- reverse: create index "idx_bee_enrolment_tokens_token_hash" to table: "bee_enrolment_tokens"
DROP INDEX "idx_bee_enrolment_tokens_token_hash";
-- reverse: create index "idx_bee_enrolment_tokens_hive_id" to table: "bee_enrolment_tokens"
DROP INDEX "idx_bee_enrolment_tokens_hive_id";
-- reverse: create index "idx_bee_enrolment_tokens_expires_at" to table: "bee_enrolment_tokens"
DROP INDEX "idx_bee_enrolment_tokens_expires_at";
-- reverse: create "bee_enrolment_tokens" table
DROP TABLE "bee_enrolment_tokens";
-- reverse: create index "idx_hives_name" to table: "hives"
DROP INDEX "idx_hives_name";
-- reverse: create "hives" table
DROP TABLE "hives";

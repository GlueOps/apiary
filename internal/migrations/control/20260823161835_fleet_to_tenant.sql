-- +goose Up
-- modify "bee_enrolment_tokens" table
ALTER TABLE "bee_enrolment_tokens" DROP COLUMN "hive_id", ADD COLUMN "organization_id" uuid NOT NULL, ADD CONSTRAINT "fk_bee_enrolment_tokens_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- create index "idx_bee_enrolment_tokens_organization_id" to table: "bee_enrolment_tokens"
CREATE INDEX "idx_bee_enrolment_tokens_organization_id" ON "bee_enrolment_tokens" ("organization_id");
-- drop "fleet_task_logs" table
DROP TABLE "fleet_task_logs";
-- drop "fleet_tasks" table
DROP TABLE "fleet_tasks";
-- drop "bees" table
DROP TABLE "bees";
-- drop "hives" table
DROP TABLE "hives";

-- +goose Down
-- reverse: drop "hives" table
CREATE TABLE "hives" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "provider" character varying(32) NOT NULL DEFAULT 'proxmox',
  "description" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_hives_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE UNIQUE INDEX "idx_hives_org_name" ON "hives" ("organization_id", "name");
-- reverse: drop "bees" table
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
CREATE UNIQUE INDEX "idx_bees_credential_hash" ON "bees" ("credential_hash");
CREATE INDEX "idx_bees_hive_hostname" ON "bees" ("hive_id", "hostname");
CREATE INDEX "idx_bees_last_seen_at" ON "bees" ("last_seen_at");
CREATE UNIQUE INDEX "idx_bees_machine_id" ON "bees" ("machine_id");
-- reverse: drop "fleet_tasks" table
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
CREATE INDEX "idx_fleet_tasks_bee_status" ON "fleet_tasks" ("bee_id", "status");
-- reverse: drop "fleet_task_logs" table
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
CREATE UNIQUE INDEX "uniq_fleet_task_logs_task_seq" ON "fleet_task_logs" ("task_id", "seq");
-- reverse: create index "idx_bee_enrolment_tokens_organization_id" to table: "bee_enrolment_tokens"
DROP INDEX "idx_bee_enrolment_tokens_organization_id";
-- reverse: modify "bee_enrolment_tokens" table
ALTER TABLE "bee_enrolment_tokens" DROP CONSTRAINT "fk_bee_enrolment_tokens_organization", DROP COLUMN "organization_id", ADD COLUMN "hive_id" uuid NOT NULL;

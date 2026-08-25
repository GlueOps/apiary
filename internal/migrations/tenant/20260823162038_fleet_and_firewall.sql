-- +goose Up
-- create "hives" table
CREATE TABLE "hives" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "provider" character varying(32) NOT NULL DEFAULT 'proxmox',
  "description" text NULL,
  "firewall_revision" bigint NOT NULL DEFAULT 0,
  "firewall_digest" character varying(71) NOT NULL DEFAULT '',
  "firewall_published_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_hives_name" to table: "hives"
CREATE UNIQUE INDEX "idx_hives_name" ON "hives" ("name");
-- create "bees" table
CREATE TABLE "bees" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NULL,
  "hostname" character varying(253) NOT NULL,
  "machine_id" character varying(64) NOT NULL,
  "api_key_id" uuid NOT NULL,
  "status" character varying(16) NOT NULL DEFAULT 'offline',
  "agent_version" character varying(64) NULL,
  "last_seen_at" timestamptz NULL,
  "unavailable_until" timestamptz NULL,
  "unavailable_reason" character varying(200) NULL,
  "applied_firewall_revision" bigint NOT NULL DEFAULT 0,
  "applied_firewall_digest" character varying(71) NOT NULL DEFAULT '',
  "applied_firewall_at" timestamptz NULL,
  "enrolled_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_bees_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "idx_bees_api_key_id" to table: "bees"
CREATE UNIQUE INDEX "idx_bees_api_key_id" ON "bees" ("api_key_id");
-- create index "idx_bees_hive_hostname" to table: "bees"
CREATE INDEX "idx_bees_hive_hostname" ON "bees" ("hive_id", "hostname");
-- create index "idx_bees_last_seen_at" to table: "bees"
CREATE INDEX "idx_bees_last_seen_at" ON "bees" ("last_seen_at");
-- create index "idx_bees_machine_id" to table: "bees"
CREATE UNIQUE INDEX "idx_bees_machine_id" ON "bees" ("machine_id");
-- create "firewall_rulesets" table
CREATE TABLE "firewall_rulesets" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_firewall_rulesets_name" to table: "firewall_rulesets"
CREATE UNIQUE INDEX "idx_firewall_rulesets_name" ON "firewall_rulesets" ("name");
-- create "firewall_bindings" table
CREATE TABLE "firewall_bindings" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NULL,
  "ruleset_id" uuid NOT NULL,
  "priority" bigint NOT NULL DEFAULT 100,
  "enabled" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_firewall_bindings_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "fk_firewall_bindings_ruleset" FOREIGN KEY ("ruleset_id") REFERENCES "firewall_rulesets" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_firewall_bindings_hive_id" to table: "firewall_bindings"
CREATE INDEX "idx_firewall_bindings_hive_id" ON "firewall_bindings" ("hive_id");
-- create index "idx_firewall_bindings_hive_ruleset" to table: "firewall_bindings"
CREATE UNIQUE INDEX "idx_firewall_bindings_hive_ruleset" ON "firewall_bindings" ("hive_id", "ruleset_id") WHERE (hive_id IS NOT NULL);
-- create index "idx_firewall_bindings_org_ruleset" to table: "firewall_bindings"
CREATE UNIQUE INDEX "idx_firewall_bindings_org_ruleset" ON "firewall_bindings" ("ruleset_id") WHERE (hive_id IS NULL);
-- create "firewall_rules" table
CREATE TABLE "firewall_rules" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "ruleset_id" uuid NOT NULL,
  "name" text NOT NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  "family" text NOT NULL DEFAULT 'inet',
  "table_name" text NOT NULL DEFAULT 'filter',
  "chain_name" text NOT NULL DEFAULT 'input',
  "position" bigint NOT NULL DEFAULT 100,
  "src_cidr" text NOT NULL DEFAULT '0.0.0.0/0',
  "dst_cidr" text NOT NULL DEFAULT '0.0.0.0/0',
  "protocol" text NOT NULL DEFAULT 'tcp',
  "src_port" bigint NOT NULL DEFAULT 0,
  "src_port_end" bigint NOT NULL DEFAULT 0,
  "dst_port" bigint NOT NULL DEFAULT 0,
  "dst_port_end" bigint NOT NULL DEFAULT 0,
  "in_iface" text NOT NULL DEFAULT '',
  "out_iface" text NOT NULL DEFAULT '',
  "action" text NOT NULL DEFAULT 'accept',
  "comment" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_firewall_rules_ruleset" FOREIGN KEY ("ruleset_id") REFERENCES "firewall_rulesets" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_firewall_rules_order" to table: "firewall_rules"
CREATE INDEX "idx_firewall_rules_order" ON "firewall_rules" ("ruleset_id", "family", "table_name", "chain_name", "position");
-- create "firewall_snapshots" table
CREATE TABLE "firewall_snapshots" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NOT NULL,
  "revision" bigint NOT NULL,
  "content" text NOT NULL,
  "digest" text NOT NULL,
  "revert_of" bigint NULL,
  "created_by_id" uuid NULL,
  "reason" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_firewall_snapshots_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_firewall_snapshots_digest" to table: "firewall_snapshots"
CREATE INDEX "idx_firewall_snapshots_digest" ON "firewall_snapshots" ("digest");
-- create index "idx_firewall_snapshots_hive_rev" to table: "firewall_snapshots"
CREATE UNIQUE INDEX "idx_firewall_snapshots_hive_rev" ON "firewall_snapshots" ("hive_id", "revision");
-- create "fleet_tasks" table
CREATE TABLE "fleet_tasks" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "bee_id" uuid NOT NULL,
  "hive_id" uuid NULL,
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
-- create index "idx_fleet_tasks_hive_id" to table: "fleet_tasks"
CREATE INDEX "idx_fleet_tasks_hive_id" ON "fleet_tasks" ("hive_id");
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
-- reverse: create index "idx_fleet_tasks_hive_id" to table: "fleet_tasks"
DROP INDEX "idx_fleet_tasks_hive_id";
-- reverse: create index "idx_fleet_tasks_bee_status" to table: "fleet_tasks"
DROP INDEX "idx_fleet_tasks_bee_status";
-- reverse: create "fleet_tasks" table
DROP TABLE "fleet_tasks";
-- reverse: create index "idx_firewall_snapshots_hive_rev" to table: "firewall_snapshots"
DROP INDEX "idx_firewall_snapshots_hive_rev";
-- reverse: create index "idx_firewall_snapshots_digest" to table: "firewall_snapshots"
DROP INDEX "idx_firewall_snapshots_digest";
-- reverse: create "firewall_snapshots" table
DROP TABLE "firewall_snapshots";
-- reverse: create index "idx_firewall_rules_order" to table: "firewall_rules"
DROP INDEX "idx_firewall_rules_order";
-- reverse: create "firewall_rules" table
DROP TABLE "firewall_rules";
-- reverse: create index "idx_firewall_bindings_org_ruleset" to table: "firewall_bindings"
DROP INDEX "idx_firewall_bindings_org_ruleset";
-- reverse: create index "idx_firewall_bindings_hive_ruleset" to table: "firewall_bindings"
DROP INDEX "idx_firewall_bindings_hive_ruleset";
-- reverse: create index "idx_firewall_bindings_hive_id" to table: "firewall_bindings"
DROP INDEX "idx_firewall_bindings_hive_id";
-- reverse: create "firewall_bindings" table
DROP TABLE "firewall_bindings";
-- reverse: create index "idx_firewall_rulesets_name" to table: "firewall_rulesets"
DROP INDEX "idx_firewall_rulesets_name";
-- reverse: create "firewall_rulesets" table
DROP TABLE "firewall_rulesets";
-- reverse: create index "idx_bees_machine_id" to table: "bees"
DROP INDEX "idx_bees_machine_id";
-- reverse: create index "idx_bees_last_seen_at" to table: "bees"
DROP INDEX "idx_bees_last_seen_at";
-- reverse: create index "idx_bees_hive_hostname" to table: "bees"
DROP INDEX "idx_bees_hive_hostname";
-- reverse: create index "idx_bees_api_key_id" to table: "bees"
DROP INDEX "idx_bees_api_key_id";
-- reverse: create "bees" table
DROP TABLE "bees";
-- reverse: create index "idx_hives_name" to table: "hives"
DROP INDEX "idx_hives_name";
-- reverse: create "hives" table
DROP TABLE "hives";

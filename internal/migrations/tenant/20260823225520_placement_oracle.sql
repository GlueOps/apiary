-- +goose Up
-- modify "hives" table
ALTER TABLE "hives" ADD COLUMN "api_url" character varying(512) NOT NULL DEFAULT '', ADD COLUMN "credential_id" uuid NULL, ADD COLUMN "insecure_skip_verify" boolean NOT NULL DEFAULT false, ADD COLUMN "cpu_overcommit_ratio" numeric(5,2) NOT NULL DEFAULT 1, ADD CONSTRAINT "fk_hives_credential" FOREIGN KEY ("credential_id") REFERENCES "credentials" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create "hypervisors" table
CREATE TABLE "hypervisors" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NOT NULL,
  "name" character varying(255) NOT NULL,
  "bee_id" uuid NULL,
  "observed_cpu_total" bigint NOT NULL DEFAULT 0,
  "observed_ram_gb_total" bigint NOT NULL DEFAULT 0,
  "observed_disk_gb_total" bigint NOT NULL DEFAULT 0,
  "observed_cpu_used" bigint NOT NULL DEFAULT 0,
  "observed_ram_gb_used" bigint NOT NULL DEFAULT 0,
  "observed_disk_gb_used" bigint NOT NULL DEFAULT 0,
  "last_synced_at" timestamptz NULL,
  "cpu_reserved" bigint NOT NULL DEFAULT 0,
  "ram_gb_reserved" bigint NOT NULL DEFAULT 0,
  "disk_gb_reserved" bigint NOT NULL DEFAULT 0,
  "cpu_overcommit_ratio" numeric(5,2) NOT NULL DEFAULT 0,
  "schedulable" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_hypervisors_bee" FOREIGN KEY ("bee_id") REFERENCES "bees" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "fk_hypervisors_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "ux_hypervisor_bee" to table: "hypervisors"
CREATE UNIQUE INDEX "ux_hypervisor_bee" ON "hypervisors" ("bee_id") WHERE (bee_id IS NOT NULL);
-- create index "ux_hypervisor_hive_name" to table: "hypervisors"
CREATE UNIQUE INDEX "ux_hypervisor_hive_name" ON "hypervisors" ("hive_id", "name");
-- create "slots" table
CREATE TABLE "slots" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(255) NOT NULL,
  "vcpu" bigint NOT NULL,
  "ram_gb" bigint NOT NULL,
  "disk_gb" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_slots_name" to table: "slots"
CREATE UNIQUE INDEX "idx_slots_name" ON "slots" ("name");
-- create "pools" table
CREATE TABLE "pools" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NOT NULL,
  "slot_id" uuid NOT NULL,
  "name" character varying(255) NOT NULL,
  "desired_count" bigint NOT NULL DEFAULT 0,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_pools_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "fk_pools_slot" FOREIGN KEY ("slot_id") REFERENCES "slots" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);
-- create index "idx_pools_slot_id" to table: "pools"
CREATE INDEX "idx_pools_slot_id" ON "pools" ("slot_id");
-- create index "ux_pools_hive_name" to table: "pools"
CREATE UNIQUE INDEX "ux_pools_hive_name" ON "pools" ("hive_id", "name");
-- create "placements" table
CREATE TABLE "placements" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "pool_id" uuid NOT NULL,
  "hypervisor_id" uuid NOT NULL,
  "state" character varying(16) NOT NULL DEFAULT 'pending',
  "hive_id" uuid NOT NULL,
  "vmid" bigint NULL,
  "reason" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_placements_hypervisor" FOREIGN KEY ("hypervisor_id") REFERENCES "hypervisors" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "fk_placements_pool" FOREIGN KEY ("pool_id") REFERENCES "pools" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_placements_hypervisor_state" to table: "placements"
CREATE INDEX "idx_placements_hypervisor_state" ON "placements" ("hypervisor_id", "state");
-- create index "idx_placements_pool_id" to table: "placements"
CREATE INDEX "idx_placements_pool_id" ON "placements" ("pool_id");
-- create index "ux_placements_hive_vmid" to table: "placements"
CREATE UNIQUE INDEX "ux_placements_hive_vmid" ON "placements" ("hive_id", "vmid") WHERE (vmid IS NOT NULL);

-- +goose Down
-- reverse: create index "ux_placements_hive_vmid" to table: "placements"
DROP INDEX "ux_placements_hive_vmid";
-- reverse: create index "idx_placements_pool_id" to table: "placements"
DROP INDEX "idx_placements_pool_id";
-- reverse: create index "idx_placements_hypervisor_state" to table: "placements"
DROP INDEX "idx_placements_hypervisor_state";
-- reverse: create "placements" table
DROP TABLE "placements";
-- reverse: create index "ux_pools_hive_name" to table: "pools"
DROP INDEX "ux_pools_hive_name";
-- reverse: create index "idx_pools_slot_id" to table: "pools"
DROP INDEX "idx_pools_slot_id";
-- reverse: create "pools" table
DROP TABLE "pools";
-- reverse: create index "idx_slots_name" to table: "slots"
DROP INDEX "idx_slots_name";
-- reverse: create "slots" table
DROP TABLE "slots";
-- reverse: create index "ux_hypervisor_hive_name" to table: "hypervisors"
DROP INDEX "ux_hypervisor_hive_name";
-- reverse: create index "ux_hypervisor_bee" to table: "hypervisors"
DROP INDEX "ux_hypervisor_bee";
-- reverse: create "hypervisors" table
DROP TABLE "hypervisors";
-- reverse: modify "hives" table
ALTER TABLE "hives" DROP CONSTRAINT "fk_hives_credential", DROP COLUMN "cpu_overcommit_ratio", DROP COLUMN "insecure_skip_verify", DROP COLUMN "credential_id", DROP COLUMN "api_url";

-- +goose Up
-- modify "hypervisors" table
ALTER TABLE "hypervisors" ADD COLUMN "maintenance_acknowledged_at" timestamptz NULL;
-- modify "maintenance_runs" table
ALTER TABLE "maintenance_runs" ADD CONSTRAINT "ck_maintenance_max_nodes" CHECK ((max_nodes_per_run >= 0) AND (max_nodes_per_run <= 1000)), ADD CONSTRAINT "ck_maintenance_pct" CHECK ((maintenance_pct >= 0) AND (maintenance_pct <= 50)), ADD COLUMN "max_nodes_per_run" bigint NOT NULL DEFAULT 0, ADD COLUMN "maintenance_pct" bigint NOT NULL DEFAULT 0;
-- modify "maintenance_schedules" table
ALTER TABLE "maintenance_schedules" ADD CONSTRAINT "ck_maintenance_max_nodes" CHECK ((max_nodes_per_run >= 0) AND (max_nodes_per_run <= 1000)), ADD CONSTRAINT "ck_maintenance_pct" CHECK ((maintenance_pct >= 0) AND (maintenance_pct <= 50)), ADD COLUMN "max_nodes_per_run" bigint NOT NULL DEFAULT 0, ADD COLUMN "maintenance_pct" bigint NOT NULL DEFAULT 0;

-- +goose Down
-- Constraints before columns: dropping a column drops its check constraints
-- with it, and the DROP CONSTRAINT that Atlas generated after it then fails.
-- reverse: modify "maintenance_schedules" table
ALTER TABLE "maintenance_schedules" DROP CONSTRAINT "ck_maintenance_pct", DROP CONSTRAINT "ck_maintenance_max_nodes", DROP COLUMN "maintenance_pct", DROP COLUMN "max_nodes_per_run";
-- reverse: modify "maintenance_runs" table
ALTER TABLE "maintenance_runs" DROP CONSTRAINT "ck_maintenance_pct", DROP CONSTRAINT "ck_maintenance_max_nodes", DROP COLUMN "maintenance_pct", DROP COLUMN "max_nodes_per_run";
-- reverse: modify "hypervisors" table
ALTER TABLE "hypervisors" DROP COLUMN "maintenance_acknowledged_at";

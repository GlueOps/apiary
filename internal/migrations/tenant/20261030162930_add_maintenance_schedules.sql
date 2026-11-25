-- +goose Up
-- create "maintenance_schedules" table
CREATE TABLE "maintenance_schedules" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NULL,
  "standalone" boolean NOT NULL,
  "name" character varying(100) NOT NULL,
  "kind" character varying(16) NOT NULL,
  "weekdays" bigint NOT NULL,
  "start_time" character varying(5) NOT NULL,
  "timezone" character varying(64) NOT NULL,
  "once_at" timestamptz NULL,
  "duration_minutes" bigint NOT NULL,
  "reboot" character varying(16) NOT NULL,
  "enabled" boolean NOT NULL,
  "last_window_at" timestamptz NULL,
  "created_by_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_maintenance_schedules_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "ck_maintenance_schedules_kind" CHECK ((kind)::text = ANY ((ARRAY['recurring'::character varying, 'once'::character varying])::text[])),
  CONSTRAINT "ck_maintenance_schedules_scope" CHECK ((hive_id IS NULL) = standalone)
);
-- create index "idx_maintenance_schedules_hive_id" to table: "maintenance_schedules"
CREATE INDEX "idx_maintenance_schedules_hive_id" ON "maintenance_schedules" ("hive_id");
-- create "maintenance_runs" table
CREATE TABLE "maintenance_runs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "schedule_id" uuid NULL,
  "hive_id" uuid NULL,
  "standalone" boolean NOT NULL,
  "status" character varying(16) NOT NULL,
  "reboot" character varying(16) NOT NULL,
  "error" text NOT NULL,
  "stop_requested" boolean NOT NULL,
  "window_start" timestamptz NOT NULL,
  "window_end" timestamptz NOT NULL,
  "started_at" timestamptz NOT NULL,
  "finished_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_maintenance_runs_hive" FOREIGN KEY ("hive_id") REFERENCES "hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "fk_maintenance_runs_schedule" FOREIGN KEY ("schedule_id") REFERENCES "maintenance_schedules" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "idx_maintenance_runs_hive_id" to table: "maintenance_runs"
CREATE INDEX "idx_maintenance_runs_hive_id" ON "maintenance_runs" ("hive_id");
-- create index "idx_maintenance_runs_schedule_id" to table: "maintenance_runs"
CREATE INDEX "idx_maintenance_runs_schedule_id" ON "maintenance_runs" ("schedule_id");
-- create index "idx_maintenance_runs_status" to table: "maintenance_runs"
CREATE INDEX "idx_maintenance_runs_status" ON "maintenance_runs" ("status");
-- create "maintenance_run_nodes" table
CREATE TABLE "maintenance_run_nodes" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "run_id" uuid NOT NULL,
  "hypervisor_id" uuid NOT NULL,
  "name" character varying(255) NOT NULL,
  "position" bigint NOT NULL,
  "status" character varying(16) NOT NULL,
  "reason" text NOT NULL,
  "task_id" uuid NULL,
  "cordoned_at" timestamptz NULL,
  "restored" boolean NOT NULL,
  "started_at" timestamptz NULL,
  "deadline" timestamptz NULL,
  "finished_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_maintenance_runs_nodes" FOREIGN KEY ("run_id") REFERENCES "maintenance_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_maintenance_run_nodes_hypervisor_id" to table: "maintenance_run_nodes"
CREATE INDEX "idx_maintenance_run_nodes_hypervisor_id" ON "maintenance_run_nodes" ("hypervisor_id");
-- create index "idx_maintenance_run_nodes_run_id" to table: "maintenance_run_nodes"
CREATE INDEX "idx_maintenance_run_nodes_run_id" ON "maintenance_run_nodes" ("run_id");

-- +goose Down
-- reverse: create index "idx_maintenance_run_nodes_run_id" to table: "maintenance_run_nodes"
DROP INDEX "idx_maintenance_run_nodes_run_id";
-- reverse: create index "idx_maintenance_run_nodes_hypervisor_id" to table: "maintenance_run_nodes"
DROP INDEX "idx_maintenance_run_nodes_hypervisor_id";
-- reverse: create "maintenance_run_nodes" table
DROP TABLE "maintenance_run_nodes";
-- reverse: create index "idx_maintenance_runs_status" to table: "maintenance_runs"
DROP INDEX "idx_maintenance_runs_status";
-- reverse: create index "idx_maintenance_runs_schedule_id" to table: "maintenance_runs"
DROP INDEX "idx_maintenance_runs_schedule_id";
-- reverse: create index "idx_maintenance_runs_hive_id" to table: "maintenance_runs"
DROP INDEX "idx_maintenance_runs_hive_id";
-- reverse: create "maintenance_runs" table
DROP TABLE "maintenance_runs";
-- reverse: create index "idx_maintenance_schedules_hive_id" to table: "maintenance_schedules"
DROP INDEX "idx_maintenance_schedules_hive_id";
-- reverse: create "maintenance_schedules" table
DROP TABLE "maintenance_schedules";

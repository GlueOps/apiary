-- +goose Up
-- modify "hypervisors" table
ALTER TABLE "public"."hypervisors" ADD COLUMN "observed_foreign_cpu_used" bigint NOT NULL DEFAULT 0, ADD COLUMN "observed_foreign_ram_gb_used" bigint NOT NULL DEFAULT 0, ADD COLUMN "observed_foreign_disk_gb_used" bigint NOT NULL DEFAULT 0;

-- +goose Down
-- reverse: modify "hypervisors" table
ALTER TABLE "public"."hypervisors" DROP COLUMN "observed_foreign_disk_gb_used", DROP COLUMN "observed_foreign_ram_gb_used", DROP COLUMN "observed_foreign_cpu_used";

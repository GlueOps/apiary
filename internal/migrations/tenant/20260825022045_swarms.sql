-- +goose Up
-- create "swarms" table
CREATE TABLE "public"."swarms" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NOT NULL,
  "name" character varying(255) NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_swarms_hive" FOREIGN KEY ("hive_id") REFERENCES "public"."hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "ux_swarms_hive_id" to table: "swarms"
CREATE UNIQUE INDEX "ux_swarms_hive_id" ON "public"."swarms" ("hive_id", "id");
-- create index "ux_swarms_hive_name" to table: "swarms"
CREATE UNIQUE INDEX "ux_swarms_hive_name" ON "public"."swarms" ("hive_id", "name");
-- modify "hypervisors" table
ALTER TABLE "public"."hypervisors" ADD COLUMN "swarm_id" uuid NULL, ADD CONSTRAINT "fk_hypervisors_swarm" FOREIGN KEY ("hive_id", "swarm_id") REFERENCES "public"."swarms" ("hive_id", "id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "idx_hypervisors_swarm_id" to table: "hypervisors"
CREATE INDEX "idx_hypervisors_swarm_id" ON "public"."hypervisors" ("swarm_id");
-- modify "pools" table
ALTER TABLE "public"."pools" ADD COLUMN "swarm_id" uuid NOT NULL, ADD CONSTRAINT "fk_pools_swarm" FOREIGN KEY ("hive_id", "swarm_id") REFERENCES "public"."swarms" ("hive_id", "id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "idx_pools_swarm_id" to table: "pools"
CREATE INDEX "idx_pools_swarm_id" ON "public"."pools" ("swarm_id");

-- +goose Down
-- reverse: create index "idx_pools_swarm_id" to table: "pools"
DROP INDEX "public"."idx_pools_swarm_id";
-- reverse: modify "pools" table
ALTER TABLE "public"."pools" DROP CONSTRAINT "fk_pools_swarm", DROP COLUMN "swarm_id";
-- reverse: create index "idx_hypervisors_swarm_id" to table: "hypervisors"
DROP INDEX "public"."idx_hypervisors_swarm_id";
-- reverse: modify "hypervisors" table
ALTER TABLE "public"."hypervisors" DROP CONSTRAINT "fk_hypervisors_swarm", DROP COLUMN "swarm_id";
-- reverse: create index "ux_swarms_hive_name" to table: "swarms"
DROP INDEX "public"."ux_swarms_hive_name";
-- reverse: create index "ux_swarms_hive_id" to table: "swarms"
DROP INDEX "public"."ux_swarms_hive_id";
-- reverse: create "swarms" table
DROP TABLE "public"."swarms";

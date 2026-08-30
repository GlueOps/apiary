-- +goose Up
-- create "membership_snapshots" table
CREATE TABLE "membership_snapshots" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "hive_id" uuid NOT NULL,
  "revision" bigint NOT NULL,
  "content" jsonb NOT NULL,
  "digest" character varying(71) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "ux_membership_snapshot_rev" to table: "membership_snapshots"
CREATE UNIQUE INDEX "ux_membership_snapshot_rev" ON "membership_snapshots" ("hive_id", "revision");

-- +goose Down
-- reverse: create index "ux_membership_snapshot_rev" to table: "membership_snapshots"
DROP INDEX "ux_membership_snapshot_rev";
-- reverse: create "membership_snapshots" table
DROP TABLE "membership_snapshots";

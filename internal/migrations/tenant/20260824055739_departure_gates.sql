-- +goose Up
-- modify "hives" table
ALTER TABLE "public"."hives" ADD COLUMN "minimum_nodes" bigint NOT NULL DEFAULT 0;
-- modify "hypervisors" table
ALTER TABLE "public"."hypervisors" ADD COLUMN "observed_guests" bigint NOT NULL DEFAULT 0;

-- +goose Down
-- reverse: modify "hypervisors" table
ALTER TABLE "public"."hypervisors" DROP COLUMN "observed_guests";
-- reverse: modify "hives" table
ALTER TABLE "public"."hives" DROP COLUMN "minimum_nodes";

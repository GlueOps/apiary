-- +goose Up
-- modify "bees" table
ALTER TABLE "bees" ADD COLUMN "observed_cluster_secrets" bigint NOT NULL DEFAULT 0, ADD COLUMN "departing_root_key" character varying(1024) NOT NULL DEFAULT '';

-- +goose Down
-- reverse: modify "bees" table
ALTER TABLE "bees" DROP COLUMN "departing_root_key", DROP COLUMN "observed_cluster_secrets";

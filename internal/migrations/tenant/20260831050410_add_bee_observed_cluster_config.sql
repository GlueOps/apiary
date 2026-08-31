-- +goose Up
-- modify "bees" table
ALTER TABLE "bees" ADD COLUMN "observed_cluster_config" boolean NOT NULL DEFAULT false;

-- +goose Down
-- reverse: modify "bees" table
ALTER TABLE "bees" DROP COLUMN "observed_cluster_config";

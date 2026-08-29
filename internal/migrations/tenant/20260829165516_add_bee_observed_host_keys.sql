-- +goose Up
-- modify "bees" table
ALTER TABLE "bees" ADD COLUMN "observed_host_keys" jsonb NULL;

-- +goose Down
-- reverse: modify "bees" table
ALTER TABLE "bees" DROP COLUMN "observed_host_keys";

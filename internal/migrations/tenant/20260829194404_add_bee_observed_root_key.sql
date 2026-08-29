-- +goose Up
-- modify "bees" table
ALTER TABLE "bees" ADD COLUMN "observed_root_key" character varying(1024) NOT NULL DEFAULT '';

-- +goose Down
-- reverse: modify "bees" table
ALTER TABLE "bees" DROP COLUMN "observed_root_key";

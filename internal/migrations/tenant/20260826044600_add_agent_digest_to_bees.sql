-- +goose Up
-- modify "bees" table
ALTER TABLE "bees" ADD COLUMN "agent_digest" character varying(71) NOT NULL DEFAULT '';

-- +goose Down
-- reverse: modify "bees" table
ALTER TABLE "bees" DROP COLUMN "agent_digest";

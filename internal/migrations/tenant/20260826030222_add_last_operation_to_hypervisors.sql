-- +goose Up
-- modify "hypervisors" table
ALTER TABLE "hypervisors" ADD COLUMN "last_operation_error" text NOT NULL DEFAULT '', ADD COLUMN "last_operation" character varying(64) NOT NULL DEFAULT '', ADD COLUMN "last_operation_at" timestamptz NULL;

-- +goose Down
-- reverse: modify "hypervisors" table
ALTER TABLE "hypervisors" DROP COLUMN "last_operation_at", DROP COLUMN "last_operation", DROP COLUMN "last_operation_error";

-- +goose Up
-- modify "hypervisors" table
ALTER TABLE "hypervisors" ADD COLUMN "credential_disposition" character varying(32) NOT NULL DEFAULT '';

-- +goose Down
-- reverse: modify "hypervisors" table
ALTER TABLE "hypervisors" DROP COLUMN "credential_disposition";

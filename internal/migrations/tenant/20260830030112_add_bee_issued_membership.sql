-- +goose Up
-- modify "bees" table
ALTER TABLE "bees" ADD COLUMN "issued_membership_revision" bigint NOT NULL DEFAULT 0, ADD COLUMN "issued_membership_at" timestamptz NULL;

-- +goose Down
-- reverse: modify "bees" table
ALTER TABLE "bees" DROP COLUMN "issued_membership_at", DROP COLUMN "issued_membership_revision";

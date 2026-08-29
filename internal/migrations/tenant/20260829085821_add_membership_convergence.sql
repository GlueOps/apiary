-- +goose Up
-- modify "bees" table
ALTER TABLE "bees" ADD COLUMN "observed_cluster" character varying(100) NOT NULL DEFAULT '', ADD COLUMN "observed_members" jsonb NULL, ADD COLUMN "observed_quorate" boolean NOT NULL DEFAULT false, ADD COLUMN "observed_fingerprint" character varying(200) NOT NULL DEFAULT '', ADD COLUMN "observed_membership_at" timestamptz NULL, ADD COLUMN "applied_membership_revision" bigint NOT NULL DEFAULT 0, ADD COLUMN "applied_membership_digest" character varying(71) NOT NULL DEFAULT '', ADD COLUMN "applied_membership_at" timestamptz NULL;
-- modify "hives" table
ALTER TABLE "hives" ADD COLUMN "membership_revision" bigint NOT NULL DEFAULT 0, ADD COLUMN "membership_digest" character varying(71) NOT NULL DEFAULT '', ADD COLUMN "membership_published_at" timestamptz NULL;

-- +goose Down
-- reverse: modify "hives" table
ALTER TABLE "hives" DROP COLUMN "membership_published_at", DROP COLUMN "membership_digest", DROP COLUMN "membership_revision";
-- reverse: modify "bees" table
ALTER TABLE "bees" DROP COLUMN "applied_membership_at", DROP COLUMN "applied_membership_digest", DROP COLUMN "applied_membership_revision", DROP COLUMN "observed_membership_at", DROP COLUMN "observed_fingerprint", DROP COLUMN "observed_quorate", DROP COLUMN "observed_members", DROP COLUMN "observed_cluster";

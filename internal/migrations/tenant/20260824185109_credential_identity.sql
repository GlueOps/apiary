-- +goose Up
-- modify "credentials" table
ALTER TABLE "public"."credentials" ADD COLUMN "identity" character varying(255) NOT NULL DEFAULT '';

-- +goose Down
-- reverse: modify "credentials" table
ALTER TABLE "public"."credentials" DROP COLUMN "identity";

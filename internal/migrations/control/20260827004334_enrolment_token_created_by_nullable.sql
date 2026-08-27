-- +goose Up
-- modify "bee_enrolment_tokens" table
ALTER TABLE "bee_enrolment_tokens" ALTER COLUMN "created_by_id" DROP NOT NULL;

-- +goose Down
-- reverse: modify "bee_enrolment_tokens" table
ALTER TABLE "bee_enrolment_tokens" ALTER COLUMN "created_by_id" SET NOT NULL;

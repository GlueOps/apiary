-- +goose Up
-- modify "users" table
ALTER TABLE "users" ADD COLUMN "email_verified_at" timestamptz NULL;
-- create "email_verification_tokens" table
CREATE TABLE "email_verification_tokens" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "email" character varying(255) NOT NULL,
  "token_hash" character varying(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_email_verification_tokens_expires_at" to table: "email_verification_tokens"
CREATE INDEX "idx_email_verification_tokens_expires_at" ON "email_verification_tokens" ("expires_at");
-- create index "idx_email_verification_tokens_token_hash" to table: "email_verification_tokens"
CREATE UNIQUE INDEX "idx_email_verification_tokens_token_hash" ON "email_verification_tokens" ("token_hash");
-- create index "idx_email_verification_tokens_user_id" to table: "email_verification_tokens"
CREATE INDEX "idx_email_verification_tokens_user_id" ON "email_verification_tokens" ("user_id");

-- Grandfather every account that already existed.
--
-- The login path now refuses a user whose email_verified_at is NULL. Every row
-- predating this migration has NULL, so without this statement deploying it
-- locks out the entire existing user base — including the operators who would
-- have to log in to fix it.
--
-- These accounts registered under the previous rules, where the platform never
-- asked them to confirm. Retroactively demanding proof they were never asked
-- for is not a security gain: nobody can produce it, and the only available
-- recovery is the resend flow, which mails every existing user at once.
--
-- created_at is used as the stamp rather than now() so the timestamp keeps
-- meaning "verified since", which is what the proving and ceiling logic reads.
-- Stamping them all with now() would reset the apparent age of every
-- organization on the platform to the moment of deployment.
UPDATE "users" SET "email_verified_at" = "created_at" WHERE "email_verified_at" IS NULL;

-- +goose Down
-- reverse: create index "idx_email_verification_tokens_user_id" to table: "email_verification_tokens"
DROP INDEX "idx_email_verification_tokens_user_id";
-- reverse: create index "idx_email_verification_tokens_token_hash" to table: "email_verification_tokens"
DROP INDEX "idx_email_verification_tokens_token_hash";
-- reverse: create index "idx_email_verification_tokens_expires_at" to table: "email_verification_tokens"
DROP INDEX "idx_email_verification_tokens_expires_at";
-- reverse: create "email_verification_tokens" table
DROP TABLE "email_verification_tokens";
-- reverse: modify "users" table
ALTER TABLE "users" DROP COLUMN "email_verified_at";

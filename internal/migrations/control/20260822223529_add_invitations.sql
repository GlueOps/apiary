-- +goose Up
-- modify "auth_access_rules" table
ALTER TABLE "auth_access_rules" DROP CONSTRAINT "chk_auth_access_rules_purpose", ADD CONSTRAINT "chk_auth_access_rules_purpose" CHECK ((purpose)::text = ANY ((ARRAY['authenticate'::character varying, 'create_org'::character varying, 'join_org'::character varying])::text[]));
-- create "invitations" table
CREATE TABLE "invitations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "organization_id" uuid NOT NULL,
  "invited_by_id" uuid NOT NULL,
  "email" character varying(255) NOT NULL,
  "role" character varying(20) NOT NULL DEFAULT 'member',
  "token_hash" character varying(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "accepted_at" timestamptz NULL,
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_invitations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_invitations_expires_at" to table: "invitations"
CREATE INDEX "idx_invitations_expires_at" ON "invitations" ("expires_at");
-- create index "idx_invitations_invited_by_id" to table: "invitations"
CREATE INDEX "idx_invitations_invited_by_id" ON "invitations" ("invited_by_id");
-- create index "idx_invitations_org_email" to table: "invitations"
CREATE INDEX "idx_invitations_org_email" ON "invitations" ("organization_id", "email");
-- create index "idx_invitations_token_hash" to table: "invitations"
CREATE UNIQUE INDEX "idx_invitations_token_hash" ON "invitations" ("token_hash");

-- +goose Down
-- reverse: create index "idx_invitations_token_hash" to table: "invitations"
DROP INDEX "idx_invitations_token_hash";
-- reverse: create index "idx_invitations_org_email" to table: "invitations"
DROP INDEX "idx_invitations_org_email";
-- reverse: create index "idx_invitations_invited_by_id" to table: "invitations"
DROP INDEX "idx_invitations_invited_by_id";
-- reverse: create index "idx_invitations_expires_at" to table: "invitations"
DROP INDEX "idx_invitations_expires_at";
-- reverse: create "invitations" table
DROP TABLE "invitations";
-- reverse: modify "auth_access_rules" table
ALTER TABLE "auth_access_rules" DROP CONSTRAINT "chk_auth_access_rules_purpose", ADD CONSTRAINT "chk_auth_access_rules_purpose" CHECK ((purpose)::text = ANY (ARRAY[('authenticate'::character varying)::text, ('create_org'::character varying)::text]));

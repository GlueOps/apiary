-- +goose Up
-- modify "organizations" table
ALTER TABLE "organizations" ADD COLUMN "deleted_at" timestamptz NULL;
-- create index "idx_organizations_deleted_at" to table: "organizations"
CREATE INDEX "idx_organizations_deleted_at" ON "organizations" ("deleted_at");
-- modify "users" table
ALTER TABLE "users" ADD COLUMN "deleted_at" timestamptz NULL;
-- create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");

-- +goose Down
-- reverse: create index "idx_users_deleted_at" to table: "users"
DROP INDEX "idx_users_deleted_at";
-- reverse: modify "users" table
ALTER TABLE "users" DROP COLUMN "deleted_at";
-- reverse: create index "idx_organizations_deleted_at" to table: "organizations"
DROP INDEX "idx_organizations_deleted_at";
-- reverse: modify "organizations" table
ALTER TABLE "organizations" DROP COLUMN "deleted_at";

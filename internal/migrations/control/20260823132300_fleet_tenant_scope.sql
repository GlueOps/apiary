-- +goose Up
-- modify "users" table
ALTER TABLE "users" ALTER COLUMN "role" SET DEFAULT 'platform_customer';
-- drop index "idx_hives_name" from table: "hives"
DROP INDEX "idx_hives_name";
-- modify "hives" table
ALTER TABLE "hives" ADD COLUMN "organization_id" uuid NOT NULL, ADD CONSTRAINT "fk_hives_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- create index "idx_hives_org_name" to table: "hives"
CREATE UNIQUE INDEX "idx_hives_org_name" ON "hives" ("organization_id", "name");

-- Move existing users onto the prefixed platform roles.
--
-- The migration above only changes the DEFAULT. Without this every existing row
-- keeps a bare 'admin' or 'customer', which no longer matches any constant —
-- and the effect is silent: the platform administrator simply stops being one,
-- discovering it the next time they try to change a platform policy.
--
-- The prefix exists because both ladders used the word "admin". A users.role of
-- 'admin' and a memberships.role of 'admin' meant entirely different things, and
-- nothing in a query result said which was which.
UPDATE "users" SET "role" = 'platform_' || "role"
WHERE "role" IN ('admin', 'support', 'sales', 'customer');

-- +goose Down
-- reverse: create index "idx_hives_org_name" to table: "hives"
DROP INDEX "idx_hives_org_name";
-- reverse: modify "hives" table
ALTER TABLE "hives" DROP CONSTRAINT "fk_hives_organization", DROP COLUMN "organization_id";
-- reverse: drop index "idx_hives_name" from table: "hives"
CREATE UNIQUE INDEX "idx_hives_name" ON "hives" ("name");
-- reverse: modify "users" table
ALTER TABLE "users" ALTER COLUMN "role" SET DEFAULT 'customer';

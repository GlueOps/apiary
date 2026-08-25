-- +goose Up
-- modify "firewall_sets" table
ALTER TABLE "firewall_sets" ADD COLUMN "purpose" text NOT NULL DEFAULT 'general', ADD COLUMN "managed" boolean NOT NULL DEFAULT false;
-- create index "idx_firewall_sets_purpose" to table: "firewall_sets"
CREATE INDEX "idx_firewall_sets_purpose" ON "firewall_sets" ("purpose");
-- modify "ssh_keys" table
ALTER TABLE "ssh_keys" ADD COLUMN "purpose" text NOT NULL DEFAULT 'fleet', ADD COLUMN "expires_at" timestamptz NULL, ADD COLUMN "scope_host_id" uuid NULL, ADD COLUMN "scope_task_id" uuid NULL;
-- create index "idx_ssh_keys_expires_at" to table: "ssh_keys"
CREATE INDEX "idx_ssh_keys_expires_at" ON "ssh_keys" ("expires_at");
-- create index "idx_ssh_keys_purpose" to table: "ssh_keys"
CREATE INDEX "idx_ssh_keys_purpose" ON "ssh_keys" ("purpose");
-- create index "idx_ssh_keys_scope_host_id" to table: "ssh_keys"
CREATE INDEX "idx_ssh_keys_scope_host_id" ON "ssh_keys" ("scope_host_id");
-- create index "idx_ssh_keys_scope_task_id" to table: "ssh_keys"
CREATE INDEX "idx_ssh_keys_scope_task_id" ON "ssh_keys" ("scope_task_id");

-- +goose Down
-- reverse: create index "idx_ssh_keys_scope_task_id" to table: "ssh_keys"
DROP INDEX "idx_ssh_keys_scope_task_id";
-- reverse: create index "idx_ssh_keys_scope_host_id" to table: "ssh_keys"
DROP INDEX "idx_ssh_keys_scope_host_id";
-- reverse: create index "idx_ssh_keys_purpose" to table: "ssh_keys"
DROP INDEX "idx_ssh_keys_purpose";
-- reverse: create index "idx_ssh_keys_expires_at" to table: "ssh_keys"
DROP INDEX "idx_ssh_keys_expires_at";
-- reverse: modify "ssh_keys" table
ALTER TABLE "ssh_keys" DROP COLUMN "scope_task_id", DROP COLUMN "scope_host_id", DROP COLUMN "expires_at", DROP COLUMN "purpose";
-- reverse: create index "idx_firewall_sets_purpose" to table: "firewall_sets"
DROP INDEX "idx_firewall_sets_purpose";
-- reverse: modify "firewall_sets" table
ALTER TABLE "firewall_sets" DROP COLUMN "managed", DROP COLUMN "purpose";

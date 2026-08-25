-- +goose Up
--
-- HAND-EDITED IN TWO WAYS after generation, both of which must survive the next
-- `atlas migrate diff`:
--
--  1. STATEMENT ORDER. Atlas emitted the `bees` DROP COLUMN first, before
--     `hypervisors` had anywhere to put what those columns hold. The drop is
--     moved to the end so the backfill below can run between them.
--  2. THE BACKFILL ITSELF is hand-written DATA DML. Atlas diffs schema, not
--     rows, so it will never generate this and — per the same rule — will never
--     remove it either. Hand-written SCHEMA in a generated migration is what
--     gets silently dropped later; data statements are safe.
--
-- Both edits are safe under regeneration because Atlas replays the Up direction
-- to compute state and the resulting schema is identical either way.

-- drop index "idx_firewall_bindings_hive_ruleset" from table: "firewall_bindings"
DROP INDEX "public"."idx_firewall_bindings_hive_ruleset";
-- drop index "idx_firewall_bindings_org_ruleset" from table: "firewall_bindings"
DROP INDEX "public"."idx_firewall_bindings_org_ruleset";
-- modify "firewall_bindings" table
ALTER TABLE "public"."firewall_bindings" ADD CONSTRAINT "chk_firewall_bindings_swarm_needs_hive" CHECK ((swarm_id IS NULL) OR (hive_id IS NOT NULL)), ADD COLUMN "swarm_id" uuid NULL, ADD CONSTRAINT "fk_firewall_bindings_swarm" FOREIGN KEY ("hive_id", "swarm_id") REFERENCES "public"."swarms" ("hive_id", "id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "idx_firewall_bindings_hive_ruleset" to table: "firewall_bindings"
CREATE UNIQUE INDEX "idx_firewall_bindings_hive_ruleset" ON "public"."firewall_bindings" ("hive_id", "ruleset_id") WHERE ((hive_id IS NOT NULL) AND (swarm_id IS NULL));
-- create index "idx_firewall_bindings_org_ruleset" to table: "firewall_bindings"
CREATE UNIQUE INDEX "idx_firewall_bindings_org_ruleset" ON "public"."firewall_bindings" ("ruleset_id") WHERE ((hive_id IS NULL) AND (swarm_id IS NULL));
-- create index "idx_firewall_bindings_swarm_id" to table: "firewall_bindings"
CREATE INDEX "idx_firewall_bindings_swarm_id" ON "public"."firewall_bindings" ("swarm_id");
-- create index "idx_firewall_bindings_swarm_ruleset" to table: "firewall_bindings"
CREATE UNIQUE INDEX "idx_firewall_bindings_swarm_ruleset" ON "public"."firewall_bindings" ("swarm_id", "ruleset_id") WHERE (swarm_id IS NOT NULL);
-- drop index "ux_hypervisor_hive_name" from table: "hypervisors"
DROP INDEX "public"."ux_hypervisor_hive_name";
-- modify "hypervisors" table
ALTER TABLE "public"."hypervisors" DROP CONSTRAINT "fk_hypervisors_hive", ADD CONSTRAINT "ck_hypervisors_provisioning_state" CHECK ((provisioning_state)::text = ANY ((ARRAY['pending'::character varying, 'provisioning'::character varying, 'provisioned'::character varying, 'failed'::character varying])::text[])), ADD CONSTRAINT "ck_hypervisors_swarm_needs_hive" CHECK ((swarm_id IS NULL) OR (hive_id IS NOT NULL)), ALTER COLUMN "hive_id" DROP NOT NULL, ALTER COLUMN "name" DROP NOT NULL, ADD COLUMN "machine_id" character varying(64) NULL, ADD COLUMN "address" character varying(253) NOT NULL DEFAULT '', ADD COLUMN "provisioning_state" character varying(16) NOT NULL DEFAULT 'provisioned', ADD COLUMN "provisioning_lease_expires_at" timestamptz NULL, ADD COLUMN "provisioning_attempt" bigint NOT NULL DEFAULT 0, ADD COLUMN "provisioning_error" text NOT NULL DEFAULT '', ADD COLUMN "provisioning_started_at" timestamptz NULL, ADD CONSTRAINT "fk_hypervisors_hive" FOREIGN KEY ("hive_id") REFERENCES "public"."hives" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "ux_hypervisor_hive_name" to table: "hypervisors"
CREATE UNIQUE INDEX "ux_hypervisor_hive_name" ON "public"."hypervisors" ("hive_id", "name") WHERE ((hive_id IS NOT NULL) AND (name IS NOT NULL));
-- create index "idx_hypervisors_hive_id" to table: "hypervisors"
CREATE INDEX "idx_hypervisors_hive_id" ON "public"."hypervisors" ("hive_id");
-- create index "ux_hypervisor_machine" to table: "hypervisors"
CREATE UNIQUE INDEX "ux_hypervisor_machine" ON "public"."hypervisors" ("machine_id") WHERE (machine_id IS NOT NULL);

-- DATA: carry every machine fact off "bees" before those columns are dropped.
--
-- Without this, every enrolled agent in every tenant loses its machine identity:
-- machine_id is what makes re-enrolment idempotent, so a rebuilt host would come
-- back as a second record instead of repointing its own, and address is the only
-- way anything reaches the box.
--
-- The unique index above is created BEFORE these run on purpose. bees.machine_id
-- was itself unique, so a collision here is impossible in theory; if theory is
-- wrong the migration must stop rather than quietly keep one row.

-- Machines that already have a hypervisor row: fill it in.
--
-- In practice this updates NOTHING. hypervisors.bee_id is written by no
-- production code path at all — the only writer in the tree is a lab test — so
-- the link is null everywhere real. It is written anyway because the link is
-- being fixed in this same workstream, and a migration that assumed the bug
-- would be silently wrong on the first environment where it had been fixed.
UPDATE "public"."hypervisors" h
   SET "machine_id" = b."machine_id",
       "address"    = b."hostname"
  FROM "public"."bees" b
 WHERE h."bee_id" = b."id";

-- Every other enrolled agent has no machine record. Create one.
--
-- hive_id and name are left NULL rather than copied from bees.hive_id, and that
-- is deliberate. A node name is the capacity sync's half of the identity and we
-- do not have one for these rows; writing hive_id without it would produce a row
-- that reads as clustered, matches nothing the sync looks up, and gets a
-- duplicate created beside it on the next tick. NULL hive_id makes the machine
-- show up as ours-but-unclustered, which is both true and the state the merge
-- path is designed to resolve.
--
-- provisioning_state is 'provisioned' because a bee row existing IS the evidence
-- that an agent was installed. It is also why the column's default is
-- 'provisioned' and not 'pending': defaulting to pending would enqueue an SSH
-- onboarding run against every hypervisor in every tenant on the first tick
-- after deploy.
INSERT INTO "public"."hypervisors" ("bee_id", "machine_id", "address", "hive_id", "name", "provisioning_state")
SELECT b."id", b."machine_id", b."hostname", NULL, NULL, 'provisioned'
  FROM "public"."bees" b
 WHERE NOT EXISTS (
       SELECT 1 FROM "public"."hypervisors" h WHERE h."bee_id" = b."id"
 );

-- modify "bees" table
--
-- Last, and only now that everything above has read them. Postgres drops
-- idx_bees_hive_hostname, idx_bees_machine_id and fk_bees_hive along with the
-- columns they cover.
ALTER TABLE "public"."bees" DROP COLUMN "hive_id", DROP COLUMN "hostname", DROP COLUMN "machine_id";

-- +goose Down
--
-- HAND-EDITED. The generated reverse re-added bees.machine_id and bees.hostname
-- as NOT NULL with no default and restored neither the index nor the foreign key
-- — it could only ever have succeeded against an empty bees table. Rewritten to
-- add the columns nullable, copy the values back off the machine record, and
-- only then re-impose NOT NULL.
--
-- It is still lossy in one direction that cannot be helped: a machine record
-- created by the INSERT above, or one with no bee, has nowhere to go back to and
-- is left behind.

-- reverse: modify "bees" table
ALTER TABLE "public"."bees" ADD COLUMN "machine_id" character varying(64) NULL, ADD COLUMN "hostname" character varying(253) NULL, ADD COLUMN "hive_id" uuid NULL;
UPDATE "public"."bees" b
   SET "machine_id" = h."machine_id",
       "hostname"   = h."address",
       "hive_id"    = h."hive_id"
  FROM "public"."hypervisors" h
 WHERE h."bee_id" = b."id";
ALTER TABLE "public"."bees" ALTER COLUMN "machine_id" SET NOT NULL, ALTER COLUMN "hostname" SET NOT NULL, ADD CONSTRAINT "fk_bees_hive" FOREIGN KEY ("hive_id") REFERENCES "public"."hives" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
CREATE INDEX "idx_bees_hive_hostname" ON "public"."bees" ("hive_id", "hostname");
CREATE UNIQUE INDEX "idx_bees_machine_id" ON "public"."bees" ("machine_id");
-- A machine in no cluster CANNOT BE REPRESENTED by the schema being restored,
-- where hive_id and name are both NOT NULL. Every such row is dropped, and that
-- is the irreducible loss of this rollback: the rows the backfill above created
-- for enrolled agents, plus any machine the sync has since cleared.
--
-- If a placement still references one, fk_placements_hypervisor is RESTRICT and
-- this fails loudly rather than deleting the ledger's referent. That is the
-- correct outcome — it needs a human, not a silent cascade.
DELETE FROM "public"."hypervisors" WHERE "hive_id" IS NULL OR "name" IS NULL;
-- reverse: create index "ux_hypervisor_machine" to table: "hypervisors"
DROP INDEX "public"."ux_hypervisor_machine";
-- reverse: create index "idx_hypervisors_hive_id" to table: "hypervisors"
DROP INDEX "public"."idx_hypervisors_hive_id";
-- reverse: create index "ux_hypervisor_hive_name" to table: "hypervisors"
DROP INDEX "public"."ux_hypervisor_hive_name";
-- reverse: modify "hypervisors" table
--
-- REORDERED from what Atlas generated. It emitted DROP COLUMN
-- "provisioning_state" ahead of DROP CONSTRAINT
-- "ck_hypervisors_provisioning_state" in the same ALTER; Postgres executes
-- subcommands in order and had already removed the check along with its column,
-- so the explicit drop failed with "constraint does not exist" and aborted the
-- rollback halfway. Both DROP CONSTRAINTs now come first.
ALTER TABLE "public"."hypervisors" DROP CONSTRAINT "fk_hypervisors_hive", DROP CONSTRAINT "ck_hypervisors_provisioning_state", DROP CONSTRAINT "ck_hypervisors_swarm_needs_hive", DROP COLUMN "provisioning_started_at", DROP COLUMN "provisioning_error", DROP COLUMN "provisioning_attempt", DROP COLUMN "provisioning_lease_expires_at", DROP COLUMN "provisioning_state", DROP COLUMN "address", DROP COLUMN "machine_id", ALTER COLUMN "name" SET NOT NULL, ALTER COLUMN "hive_id" SET NOT NULL, ADD CONSTRAINT "fk_hypervisors_hive" FOREIGN KEY ("hive_id") REFERENCES "public"."hives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: drop index "ux_hypervisor_hive_name" from table: "hypervisors"
CREATE UNIQUE INDEX "ux_hypervisor_hive_name" ON "public"."hypervisors" ("hive_id", "name");
-- reverse: create index "idx_firewall_bindings_swarm_ruleset" to table: "firewall_bindings"
DROP INDEX "public"."idx_firewall_bindings_swarm_ruleset";
-- reverse: create index "idx_firewall_bindings_swarm_id" to table: "firewall_bindings"
DROP INDEX "public"."idx_firewall_bindings_swarm_id";
-- reverse: create index "idx_firewall_bindings_org_ruleset" to table: "firewall_bindings"
DROP INDEX "public"."idx_firewall_bindings_org_ruleset";
-- reverse: create index "idx_firewall_bindings_hive_ruleset" to table: "firewall_bindings"
DROP INDEX "public"."idx_firewall_bindings_hive_ruleset";
-- reverse: modify "firewall_bindings" table
--
-- REORDERED for the same reason as the hypervisors ALTER above: Atlas put
-- DROP COLUMN "swarm_id" ahead of the DROP CONSTRAINT for a check that
-- REFERENCES swarm_id, and Postgres had already removed the check with the
-- column by the time the explicit drop ran.
ALTER TABLE "public"."firewall_bindings" DROP CONSTRAINT "fk_firewall_bindings_swarm", DROP CONSTRAINT "chk_firewall_bindings_swarm_needs_hive", DROP COLUMN "swarm_id";
-- reverse: drop index "idx_firewall_bindings_org_ruleset" from table: "firewall_bindings"
CREATE UNIQUE INDEX "idx_firewall_bindings_org_ruleset" ON "public"."firewall_bindings" ("ruleset_id") WHERE (hive_id IS NULL);
-- reverse: drop index "idx_firewall_bindings_hive_ruleset" from table: "firewall_bindings"
CREATE UNIQUE INDEX "idx_firewall_bindings_hive_ruleset" ON "public"."firewall_bindings" ("hive_id", "ruleset_id") WHERE (hive_id IS NOT NULL);

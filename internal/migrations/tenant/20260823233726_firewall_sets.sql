-- +goose Up
-- create "firewall_sets" table
CREATE TABLE "firewall_sets" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "family" text NOT NULL DEFAULT 'ip',
  "description" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_firewall_sets_name" to table: "firewall_sets"
CREATE UNIQUE INDEX "idx_firewall_sets_name" ON "firewall_sets" ("name");
-- modify "firewall_rules" table
ALTER TABLE "firewall_rules" ADD COLUMN "src_set_id" uuid NULL, ADD COLUMN "dst_set_id" uuid NULL, ADD CONSTRAINT "fk_firewall_rules_dst_set" FOREIGN KEY ("dst_set_id") REFERENCES "firewall_sets" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT, ADD CONSTRAINT "fk_firewall_rules_src_set" FOREIGN KEY ("src_set_id") REFERENCES "firewall_sets" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "idx_firewall_rules_dst_set_id" to table: "firewall_rules"
CREATE INDEX "idx_firewall_rules_dst_set_id" ON "firewall_rules" ("dst_set_id");
-- create index "idx_firewall_rules_src_set_id" to table: "firewall_rules"
CREATE INDEX "idx_firewall_rules_src_set_id" ON "firewall_rules" ("src_set_id");
-- create "firewall_set_elements" table
CREATE TABLE "firewall_set_elements" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "set_id" uuid NOT NULL,
  "value" text NOT NULL,
  "comment" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_firewall_sets_elements" FOREIGN KEY ("set_id") REFERENCES "firewall_sets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "ux_firewall_set_elements" to table: "firewall_set_elements"
CREATE UNIQUE INDEX "ux_firewall_set_elements" ON "firewall_set_elements" ("set_id", "value");

-- +goose Down
-- reverse: create index "ux_firewall_set_elements" to table: "firewall_set_elements"
DROP INDEX "ux_firewall_set_elements";
-- reverse: create "firewall_set_elements" table
DROP TABLE "firewall_set_elements";
-- reverse: create index "idx_firewall_rules_src_set_id" to table: "firewall_rules"
DROP INDEX "idx_firewall_rules_src_set_id";
-- reverse: create index "idx_firewall_rules_dst_set_id" to table: "firewall_rules"
DROP INDEX "idx_firewall_rules_dst_set_id";
-- reverse: modify "firewall_rules" table
ALTER TABLE "firewall_rules" DROP CONSTRAINT "fk_firewall_rules_src_set", DROP CONSTRAINT "fk_firewall_rules_dst_set", DROP COLUMN "dst_set_id", DROP COLUMN "src_set_id";
-- reverse: create index "idx_firewall_sets_name" to table: "firewall_sets"
DROP INDEX "idx_firewall_sets_name";
-- reverse: create "firewall_sets" table
DROP TABLE "firewall_sets";

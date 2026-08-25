-- +goose Up
-- modify "firewall_set_elements" table
ALTER TABLE "firewall_set_elements" ADD COLUMN "label" text NOT NULL DEFAULT '';
-- create index "idx_firewall_set_elements_label" to table: "firewall_set_elements"
CREATE INDEX "idx_firewall_set_elements_label" ON "firewall_set_elements" ("label");

-- +goose Down
-- reverse: create index "idx_firewall_set_elements_label" to table: "firewall_set_elements"
DROP INDEX "idx_firewall_set_elements_label";
-- reverse: modify "firewall_set_elements" table
ALTER TABLE "firewall_set_elements" DROP COLUMN "label";

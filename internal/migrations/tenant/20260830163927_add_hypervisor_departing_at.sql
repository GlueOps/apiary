-- +goose Up
-- modify "hypervisors" table
ALTER TABLE "hypervisors" ADD COLUMN "departing_at" timestamptz NULL;
-- create index "idx_hypervisors_departing_at" to table: "hypervisors"
CREATE INDEX "idx_hypervisors_departing_at" ON "hypervisors" ("departing_at");

-- +goose Down
-- reverse: create index "idx_hypervisors_departing_at" to table: "hypervisors"
DROP INDEX "idx_hypervisors_departing_at";
-- reverse: modify "hypervisors" table
ALTER TABLE "hypervisors" DROP COLUMN "departing_at";

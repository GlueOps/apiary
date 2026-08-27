-- +goose Up
-- modify "fleet_tasks" table
ALTER TABLE "fleet_tasks" ALTER COLUMN "created_by_id" DROP NOT NULL;

-- +goose Down
-- reverse: modify "fleet_tasks" table
ALTER TABLE "fleet_tasks" ALTER COLUMN "created_by_id" SET NOT NULL;

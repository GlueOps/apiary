-- +goose Up
-- create "audit_events" table
CREATE TABLE "audit_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "principal_id" uuid NOT NULL,
  "principal_kind" character varying(16) NOT NULL,
  "actor_id" uuid NULL,
  "action" character varying(64) NOT NULL,
  "target_kind" character varying(32) NOT NULL,
  "target_id" uuid NULL,
  "outcome" character varying(16) NOT NULL DEFAULT 'ok',
  "detail" text NULL,
  "session_id" uuid NULL,
  "ip_address" character varying(45) NULL,
  "user_agent" text NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_audit_created" to table: "audit_events"
CREATE INDEX "idx_audit_created" ON "audit_events" ("created_at" DESC);
-- create index "idx_audit_events_action" to table: "audit_events"
CREATE INDEX "idx_audit_events_action" ON "audit_events" ("action");
-- create index "idx_audit_events_principal_id" to table: "audit_events"
CREATE INDEX "idx_audit_events_principal_id" ON "audit_events" ("principal_id");
-- create index "idx_audit_target" to table: "audit_events"
CREATE INDEX "idx_audit_target" ON "audit_events" ("target_kind", "target_id");

-- +goose Down
-- reverse: create index "idx_audit_target" to table: "audit_events"
DROP INDEX "idx_audit_target";
-- reverse: create index "idx_audit_events_principal_id" to table: "audit_events"
DROP INDEX "idx_audit_events_principal_id";
-- reverse: create index "idx_audit_events_action" to table: "audit_events"
DROP INDEX "idx_audit_events_action";
-- reverse: create index "idx_audit_created" to table: "audit_events"
DROP INDEX "idx_audit_created";
-- reverse: create "audit_events" table
DROP TABLE "audit_events";

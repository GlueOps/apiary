-- +goose Up
-- create "platform_feature_flags" table
CREATE TABLE "public"."platform_feature_flags" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "key" character varying(120) NOT NULL,
  "value" boolean NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_platform_feature_flags_key" to table: "platform_feature_flags"
CREATE UNIQUE INDEX "idx_platform_feature_flags_key" ON "public"."platform_feature_flags" ("key");
-- create "org_feature_flags" table
CREATE TABLE "public"."org_feature_flags" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "organization_id" uuid NOT NULL,
  "key" character varying(120) NOT NULL,
  "value" boolean NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_org_feature_flags_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_org_feature_flags_org_key" to table: "org_feature_flags"
CREATE UNIQUE INDEX "idx_org_feature_flags_org_key" ON "public"."org_feature_flags" ("organization_id", "key");

-- +goose Down
-- reverse: create index "idx_org_feature_flags_org_key" to table: "org_feature_flags"
DROP INDEX "public"."idx_org_feature_flags_org_key";
-- reverse: create "org_feature_flags" table
DROP TABLE "public"."org_feature_flags";
-- reverse: create index "idx_platform_feature_flags_key" to table: "platform_feature_flags"
DROP INDEX "public"."idx_platform_feature_flags_key";
-- reverse: create "platform_feature_flags" table
DROP TABLE "public"."platform_feature_flags";

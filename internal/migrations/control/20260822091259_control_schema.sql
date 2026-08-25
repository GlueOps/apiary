-- +goose Up
-- create "auth_access_rules" table
CREATE TABLE "auth_access_rules" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "kind" character varying(10) NOT NULL,
  "value" character varying(255) NOT NULL,
  "purpose" character varying(16) NOT NULL DEFAULT 'authenticate',
  "policy" character varying(10) NOT NULL,
  "reason" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "chk_auth_access_rules_policy" CHECK ((policy)::text = ANY ((ARRAY['allow'::character varying, 'deny'::character varying])::text[])),
  CONSTRAINT "chk_auth_access_rules_purpose" CHECK ((purpose)::text = ANY ((ARRAY['authenticate'::character varying, 'create_org'::character varying])::text[]))
);
-- create index "idx_auth_access_rules_kind_value_purpose" to table: "auth_access_rules"
CREATE UNIQUE INDEX "idx_auth_access_rules_kind_value_purpose" ON "auth_access_rules" ("kind", "value", "purpose");
-- create "auth_audit_events" table
CREATE TABLE "auth_audit_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "subject_id" uuid NOT NULL,
  "subject_kind" character varying(10) NOT NULL,
  "organization_id" uuid NOT NULL,
  "session_id" uuid NULL,
  "event_type" character varying(30) NOT NULL,
  "ip_address" character varying(45) NULL,
  "user_agent" text NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_auth_audit_events_event_type" to table: "auth_audit_events"
CREATE INDEX "idx_auth_audit_events_event_type" ON "auth_audit_events" ("event_type");
-- create index "idx_auth_audit_events_subject_id" to table: "auth_audit_events"
CREATE INDEX "idx_auth_audit_events_subject_id" ON "auth_audit_events" ("subject_id");
-- create "auth_challenges" table
CREATE TABLE "auth_challenges" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "jti" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "kind" character varying(32) NOT NULL,
  "attempts" bigint NOT NULL DEFAULT 0,
  "max_attempts" bigint NOT NULL DEFAULT 5,
  "consumed_at" timestamptz NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_auth_challenges_expires_at" to table: "auth_challenges"
CREATE INDEX "idx_auth_challenges_expires_at" ON "auth_challenges" ("expires_at");
-- create index "idx_auth_challenges_jti" to table: "auth_challenges"
CREATE UNIQUE INDEX "idx_auth_challenges_jti" ON "auth_challenges" ("jti");
-- create index "idx_auth_challenges_user_id" to table: "auth_challenges"
CREATE INDEX "idx_auth_challenges_user_id" ON "auth_challenges" ("user_id");
-- create "auth_failures" table
CREATE TABLE "auth_failures" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "identifier" character varying(320) NOT NULL,
  "kind" character varying(32) NOT NULL,
  "count" bigint NOT NULL DEFAULT 0,
  "locked_until" timestamptz NULL,
  "window_start" timestamptz NOT NULL DEFAULT now(),
  "last_failed_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_auth_failures_identifier_kind" to table: "auth_failures"
CREATE UNIQUE INDEX "idx_auth_failures_identifier_kind" ON "auth_failures" ("identifier", "kind");
-- create index "idx_auth_failures_last_failed_at" to table: "auth_failures"
CREATE INDEX "idx_auth_failures_last_failed_at" ON "auth_failures" ("last_failed_at");
-- create "domain_cohort_findings" table
CREATE TABLE "domain_cohort_findings" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "registrable_domain" character varying(253) NOT NULL,
  "organizations_total" bigint NOT NULL DEFAULT 0,
  "organizations_proven" bigint NOT NULL DEFAULT 0,
  "proven_ratio" double precision NOT NULL DEFAULT 0,
  "flagged" boolean NOT NULL DEFAULT false,
  "evaluated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_domain_cohort_findings_flagged" to table: "domain_cohort_findings"
CREATE INDEX "idx_domain_cohort_findings_flagged" ON "domain_cohort_findings" ("flagged");
-- create index "idx_domain_cohort_findings_registrable_domain" to table: "domain_cohort_findings"
CREATE UNIQUE INDEX "idx_domain_cohort_findings_registrable_domain" ON "domain_cohort_findings" ("registrable_domain");
-- create "email_change_requests" table
CREATE TABLE "email_change_requests" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "new_email" character varying(255) NOT NULL,
  "token_hash" character varying(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_email_change_requests_expires_at" to table: "email_change_requests"
CREATE INDEX "idx_email_change_requests_expires_at" ON "email_change_requests" ("expires_at");
-- create index "idx_email_change_requests_token_hash" to table: "email_change_requests"
CREATE UNIQUE INDEX "idx_email_change_requests_token_hash" ON "email_change_requests" ("token_hash");
-- create index "idx_email_change_requests_user_id" to table: "email_change_requests"
CREATE INDEX "idx_email_change_requests_user_id" ON "email_change_requests" ("user_id");
-- create "impersonation_grants" table
CREATE TABLE "impersonation_grants" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "target_user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "granted_by_id" uuid NOT NULL,
  "reason" text NULL,
  "expires_at" timestamptz NOT NULL,
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_impersonation_grants_target_user_id" to table: "impersonation_grants"
CREATE INDEX "idx_impersonation_grants_target_user_id" ON "impersonation_grants" ("target_user_id");
-- create "oauth_states" table
CREATE TABLE "oauth_states" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "state_hash" character varying(64) NOT NULL,
  "provider" character varying(32) NOT NULL,
  "mode" character varying(32) NOT NULL DEFAULT '',
  "origin" text NOT NULL DEFAULT '',
  "cookie_hash" character varying(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "consumed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_oauth_states_expires_at" to table: "oauth_states"
CREATE INDEX "idx_oauth_states_expires_at" ON "oauth_states" ("expires_at");
-- create index "idx_oauth_states_state_hash" to table: "oauth_states"
CREATE UNIQUE INDEX "idx_oauth_states_state_hash" ON "oauth_states" ("state_hash");
-- create "password_reset_tokens" table
CREATE TABLE "password_reset_tokens" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "token_hash" character varying(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_password_reset_tokens_token_hash" to table: "password_reset_tokens"
CREATE UNIQUE INDEX "idx_password_reset_tokens_token_hash" ON "password_reset_tokens" ("token_hash");
-- create index "idx_password_reset_tokens_user_id" to table: "password_reset_tokens"
CREATE INDEX "idx_password_reset_tokens_user_id" ON "password_reset_tokens" ("user_id");
-- create "platform_policies" table
CREATE TABLE "platform_policies" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "kind" character varying(50) NOT NULL,
  "enabled" boolean NOT NULL DEFAULT false,
  "config" jsonb NOT NULL DEFAULT '{}',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_platform_policies_kind" to table: "platform_policies"
CREATE UNIQUE INDEX "idx_platform_policies_kind" ON "platform_policies" ("kind");
-- create "token_sessions" table
CREATE TABLE "token_sessions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "subject_id" uuid NOT NULL,
  "subject_kind" character varying(10) NOT NULL,
  "organization_id" uuid NOT NULL,
  "family_id" uuid NOT NULL,
  "access_token_jti" character varying(36) NOT NULL,
  "refresh_token_jti" character varying(36) NOT NULL,
  "access_token_hash" character varying(64) NOT NULL,
  "refresh_token_hash" character varying(64) NOT NULL,
  "actor_id" uuid NULL,
  "amr" character varying(100) NULL,
  "ip_address" character varying(45) NULL,
  "user_agent" text NULL,
  "access_expires_at" timestamptz NOT NULL,
  "refresh_expires_at" timestamptz NOT NULL,
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_token_sessions_access_token_jti" to table: "token_sessions"
CREATE UNIQUE INDEX "idx_token_sessions_access_token_jti" ON "token_sessions" ("access_token_jti");
-- create index "idx_token_sessions_family_id" to table: "token_sessions"
CREATE INDEX "idx_token_sessions_family_id" ON "token_sessions" ("family_id");
-- create index "idx_token_sessions_organization_id" to table: "token_sessions"
CREATE INDEX "idx_token_sessions_organization_id" ON "token_sessions" ("organization_id");
-- create index "idx_token_sessions_refresh_token_jti" to table: "token_sessions"
CREATE UNIQUE INDEX "idx_token_sessions_refresh_token_jti" ON "token_sessions" ("refresh_token_jti");
-- create index "idx_token_sessions_subject" to table: "token_sessions"
CREATE INDEX "idx_token_sessions_subject" ON "token_sessions" ("subject_id", "subject_kind");
-- create "user_passkeys" table
CREATE TABLE "user_passkeys" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "credential_id" text NOT NULL,
  "public_key" bytea NOT NULL,
  "attestation_type" character varying(64) NOT NULL,
  "aa_guid" character varying(36) NULL,
  "sign_count" bigint NOT NULL DEFAULT 0,
  "backup_eligible" boolean NOT NULL DEFAULT false,
  "backup_state" boolean NOT NULL DEFAULT false,
  "name" character varying(255) NOT NULL,
  "transports" jsonb NULL,
  "last_used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_user_passkeys_credential_id" to table: "user_passkeys"
CREATE UNIQUE INDEX "idx_user_passkeys_credential_id" ON "user_passkeys" ("credential_id");
-- create index "idx_user_passkeys_user_id" to table: "user_passkeys"
CREATE INDEX "idx_user_passkeys_user_id" ON "user_passkeys" ("user_id");
-- create "user_totps" table
CREATE TABLE "user_totps" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "secret" text NOT NULL,
  "verified" boolean NULL DEFAULT false,
  "last_used_step" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_user_totps_user_id" to table: "user_totps"
CREATE UNIQUE INDEX "idx_user_totps_user_id" ON "user_totps" ("user_id");
-- create "users" table
CREATE TABLE "users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "email" character varying(255) NOT NULL,
  "password" character varying(255) NOT NULL,
  "first_name" character varying(100) NOT NULL,
  "last_name" character varying(100) NOT NULL,
  "role" character varying(20) NOT NULL DEFAULT 'customer',
  "mfa_enabled" boolean NULL DEFAULT false,
  "is_active" boolean NULL DEFAULT true,
  "tokens_revoked_at" timestamptz NULL,
  "last_login_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX "idx_users_email" ON "users" ("email");
-- create "accounts" table
CREATE TABLE "accounts" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "provider" text NOT NULL,
  "subject" text NOT NULL,
  "email" text NULL,
  "email_verified" boolean NOT NULL DEFAULT false,
  "profile" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_accounts_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "idx_accounts_user_id" to table: "accounts"
CREATE INDEX "idx_accounts_user_id" ON "accounts" ("user_id");
-- create "organizations" table
CREATE TABLE "organizations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "domain" text NULL,
  "domain_verified_at" timestamptz NULL,
  "status" text NOT NULL DEFAULT 'pending',
  "connection_string" text NOT NULL DEFAULT '',
  "encrypted_tenant_key" text NOT NULL DEFAULT '',
  "tenant_key_iv" text NOT NULL DEFAULT '',
  "tenant_key_tag" text NOT NULL DEFAULT '',
  "connection_string_iv" text NOT NULL DEFAULT '',
  "connection_string_tag" text NOT NULL DEFAULT '',
  "registrable_domain" character varying(253) NULL,
  "proven_at" timestamptz NULL,
  "proving_days_seen" bigint NOT NULL DEFAULT 0,
  "proving_last_day" date NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- create index "idx_organizations_domain" to table: "organizations"
CREATE INDEX "idx_organizations_domain" ON "organizations" ("domain");
-- create index "idx_organizations_registrable_domain_proven" to table: "organizations"
CREATE INDEX "idx_organizations_registrable_domain_proven" ON "organizations" ("registrable_domain", "proven_at");
-- create "memberships" table
CREATE TABLE "memberships" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "role" text NOT NULL DEFAULT 'member',
  "support_enabled" boolean NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_memberships_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "fk_memberships_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "idx_memberships_organization_id" to table: "memberships"
CREATE INDEX "idx_memberships_organization_id" ON "memberships" ("organization_id");
-- create index "idx_memberships_user_id" to table: "memberships"
CREATE INDEX "idx_memberships_user_id" ON "memberships" ("user_id");
-- create "org_policies" table
CREATE TABLE "org_policies" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "organization_id" uuid NOT NULL,
  "kind" character varying(50) NOT NULL,
  "enabled" boolean NOT NULL DEFAULT false,
  "config" jsonb NOT NULL DEFAULT '{}',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_org_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "idx_org_policies_org_kind" to table: "org_policies"
CREATE UNIQUE INDEX "idx_org_policies_org_kind" ON "org_policies" ("organization_id", "kind");

-- +goose Down
-- reverse: create index "idx_org_policies_org_kind" to table: "org_policies"
DROP INDEX "idx_org_policies_org_kind";
-- reverse: create "org_policies" table
DROP TABLE "org_policies";
-- reverse: create index "idx_memberships_user_id" to table: "memberships"
DROP INDEX "idx_memberships_user_id";
-- reverse: create index "idx_memberships_organization_id" to table: "memberships"
DROP INDEX "idx_memberships_organization_id";
-- reverse: create "memberships" table
DROP TABLE "memberships";
-- reverse: create index "idx_organizations_registrable_domain_proven" to table: "organizations"
DROP INDEX "idx_organizations_registrable_domain_proven";
-- reverse: create index "idx_organizations_domain" to table: "organizations"
DROP INDEX "idx_organizations_domain";
-- reverse: create "organizations" table
DROP TABLE "organizations";
-- reverse: create index "idx_accounts_user_id" to table: "accounts"
DROP INDEX "idx_accounts_user_id";
-- reverse: create "accounts" table
DROP TABLE "accounts";
-- reverse: create index "idx_users_email" to table: "users"
DROP INDEX "idx_users_email";
-- reverse: create "users" table
DROP TABLE "users";
-- reverse: create index "idx_user_totps_user_id" to table: "user_totps"
DROP INDEX "idx_user_totps_user_id";
-- reverse: create "user_totps" table
DROP TABLE "user_totps";
-- reverse: create index "idx_user_passkeys_user_id" to table: "user_passkeys"
DROP INDEX "idx_user_passkeys_user_id";
-- reverse: create index "idx_user_passkeys_credential_id" to table: "user_passkeys"
DROP INDEX "idx_user_passkeys_credential_id";
-- reverse: create "user_passkeys" table
DROP TABLE "user_passkeys";
-- reverse: create index "idx_token_sessions_subject" to table: "token_sessions"
DROP INDEX "idx_token_sessions_subject";
-- reverse: create index "idx_token_sessions_refresh_token_jti" to table: "token_sessions"
DROP INDEX "idx_token_sessions_refresh_token_jti";
-- reverse: create index "idx_token_sessions_organization_id" to table: "token_sessions"
DROP INDEX "idx_token_sessions_organization_id";
-- reverse: create index "idx_token_sessions_family_id" to table: "token_sessions"
DROP INDEX "idx_token_sessions_family_id";
-- reverse: create index "idx_token_sessions_access_token_jti" to table: "token_sessions"
DROP INDEX "idx_token_sessions_access_token_jti";
-- reverse: create "token_sessions" table
DROP TABLE "token_sessions";
-- reverse: create index "idx_platform_policies_kind" to table: "platform_policies"
DROP INDEX "idx_platform_policies_kind";
-- reverse: create "platform_policies" table
DROP TABLE "platform_policies";
-- reverse: create index "idx_password_reset_tokens_user_id" to table: "password_reset_tokens"
DROP INDEX "idx_password_reset_tokens_user_id";
-- reverse: create index "idx_password_reset_tokens_token_hash" to table: "password_reset_tokens"
DROP INDEX "idx_password_reset_tokens_token_hash";
-- reverse: create "password_reset_tokens" table
DROP TABLE "password_reset_tokens";
-- reverse: create index "idx_oauth_states_state_hash" to table: "oauth_states"
DROP INDEX "idx_oauth_states_state_hash";
-- reverse: create index "idx_oauth_states_expires_at" to table: "oauth_states"
DROP INDEX "idx_oauth_states_expires_at";
-- reverse: create "oauth_states" table
DROP TABLE "oauth_states";
-- reverse: create index "idx_impersonation_grants_target_user_id" to table: "impersonation_grants"
DROP INDEX "idx_impersonation_grants_target_user_id";
-- reverse: create "impersonation_grants" table
DROP TABLE "impersonation_grants";
-- reverse: create index "idx_email_change_requests_user_id" to table: "email_change_requests"
DROP INDEX "idx_email_change_requests_user_id";
-- reverse: create index "idx_email_change_requests_token_hash" to table: "email_change_requests"
DROP INDEX "idx_email_change_requests_token_hash";
-- reverse: create index "idx_email_change_requests_expires_at" to table: "email_change_requests"
DROP INDEX "idx_email_change_requests_expires_at";
-- reverse: create "email_change_requests" table
DROP TABLE "email_change_requests";
-- reverse: create index "idx_domain_cohort_findings_registrable_domain" to table: "domain_cohort_findings"
DROP INDEX "idx_domain_cohort_findings_registrable_domain";
-- reverse: create index "idx_domain_cohort_findings_flagged" to table: "domain_cohort_findings"
DROP INDEX "idx_domain_cohort_findings_flagged";
-- reverse: create "domain_cohort_findings" table
DROP TABLE "domain_cohort_findings";
-- reverse: create index "idx_auth_failures_last_failed_at" to table: "auth_failures"
DROP INDEX "idx_auth_failures_last_failed_at";
-- reverse: create index "idx_auth_failures_identifier_kind" to table: "auth_failures"
DROP INDEX "idx_auth_failures_identifier_kind";
-- reverse: create "auth_failures" table
DROP TABLE "auth_failures";
-- reverse: create index "idx_auth_challenges_user_id" to table: "auth_challenges"
DROP INDEX "idx_auth_challenges_user_id";
-- reverse: create index "idx_auth_challenges_jti" to table: "auth_challenges"
DROP INDEX "idx_auth_challenges_jti";
-- reverse: create index "idx_auth_challenges_expires_at" to table: "auth_challenges"
DROP INDEX "idx_auth_challenges_expires_at";
-- reverse: create "auth_challenges" table
DROP TABLE "auth_challenges";
-- reverse: create index "idx_auth_audit_events_subject_id" to table: "auth_audit_events"
DROP INDEX "idx_auth_audit_events_subject_id";
-- reverse: create index "idx_auth_audit_events_event_type" to table: "auth_audit_events"
DROP INDEX "idx_auth_audit_events_event_type";
-- reverse: create "auth_audit_events" table
DROP TABLE "auth_audit_events";
-- reverse: create index "idx_auth_access_rules_kind_value_purpose" to table: "auth_access_rules"
DROP INDEX "idx_auth_access_rules_kind_value_purpose";
-- reverse: create "auth_access_rules" table
DROP TABLE "auth_access_rules";

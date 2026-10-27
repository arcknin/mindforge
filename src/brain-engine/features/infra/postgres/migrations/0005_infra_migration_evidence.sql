-- Infrastructure-owned evidence only. This migration creates no Mindforge
-- product schema, feature table, endpoint, function or product permission.
-- Runtime application remains restricted to the owner-designated VM/cluster.

CREATE SCHEMA IF NOT EXISTS infra;

CREATE TABLE IF NOT EXISTS infra.migration_target_evidence (
  evidence_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  migration_id text NOT NULL CHECK (migration_id ~ '^[0-9]{4}$'),
  migration_sha256 text NOT NULL CHECK (migration_sha256 ~ '^[0-9a-f]{64}$'),
  target_name text NOT NULL CHECK (target_name IN ('mindforge_develop', 'mindforge')),
  operator_id text NOT NULL CHECK (length(btrim(operator_id)) > 0),
  authorization_reference text NOT NULL CHECK (length(btrim(authorization_reference)) > 0),
  preflight_reference text,
  post_application_result text NOT NULL CHECK (post_application_result IN ('not_started', 'succeeded', 'failed', 'inconclusive')),
  behavior_access_verification_reference text,
  sanitized_outcome text NOT NULL CHECK (sanitized_outcome IN ('incomplete', 'succeeded', 'failed', 'inconclusive')),
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);

CREATE TABLE IF NOT EXISTS infra.migration_attempts (
  attempt_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  migration_id text NOT NULL CHECK (migration_id ~ '^[0-9]{4}$'),
  migration_sha256 text NOT NULL CHECK (migration_sha256 ~ '^[0-9a-f]{64}$'),
  target_name text NOT NULL CHECK (target_name IN ('mindforge_develop', 'mindforge')),
  operator_id text NOT NULL CHECK (length(btrim(operator_id)) > 0),
  authorization_reference text NOT NULL CHECK (length(btrim(authorization_reference)) > 0),
  preflight_reference text,
  post_application_result text NOT NULL CHECK (post_application_result IN ('not_started', 'succeeded', 'failed', 'inconclusive')),
  behavior_access_verification_reference text,
  sanitized_outcome text NOT NULL CHECK (sanitized_outcome IN ('incomplete', 'succeeded', 'failed', 'inconclusive')),
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  evidence_id uuid REFERENCES infra.migration_target_evidence (evidence_id)
);

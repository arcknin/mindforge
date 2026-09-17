-- The pinned GoTrue migrator owns Auth tables and its migration history, but
-- requires this namespace to exist before it can apply that upstream catalog.
-- This creates no Mindforge domain object.

CREATE SCHEMA IF NOT EXISTS auth;

#!/usr/bin/env bash
set -Eeuo pipefail

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${MIND_FORGE_DEVELOP_DB:?MIND_FORGE_DEVELOP_DB is required}"
: "${MIND_FORGE_DEVELOP_USER:?MIND_FORGE_DEVELOP_USER is required}"
: "${MIND_FORGE_DEVELOP_PASSWORD:?MIND_FORGE_DEVELOP_PASSWORD is required}"
: "${MIND_FORGE_MAIN_DB:?MIND_FORGE_MAIN_DB is required}"
: "${MIND_FORGE_MAIN_USER:?MIND_FORGE_MAIN_USER is required}"
: "${MIND_FORGE_MAIN_PASSWORD:?MIND_FORGE_MAIN_PASSWORD is required}"

[[ "$MIND_FORGE_DEVELOP_DB" == 'mindforge_develop' ]] || {
  printf 'bootstrap refused: unexpected develop database name\n' >&2
  exit 1
}
[[ "$MIND_FORGE_MAIN_DB" == 'mindforge' ]] || {
  printf 'bootstrap refused: unexpected main database name\n' >&2
  exit 1
}
[[ "$MIND_FORGE_DEVELOP_DB" != "$MIND_FORGE_MAIN_DB" ]] || {
  printf 'bootstrap refused: application database names must be distinct\n' >&2
  exit 1
}
[[ "$MIND_FORGE_DEVELOP_USER" != "$MIND_FORGE_MAIN_USER" ]] || {
  printf 'bootstrap refused: application database roles must be distinct\n' >&2
  exit 1
}

# This script is executed only by the PostgreSQL image on a fresh cluster.
# GoTrue migrations and feature migrations run later through their own paths.
psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -v ON_ERROR_STOP=1 <<SQL
SELECT format(
  'CREATE ROLE %I LOGIN PASSWORD %L',
  '${MIND_FORGE_DEVELOP_USER}',
  '${MIND_FORGE_DEVELOP_PASSWORD}'
)
WHERE NOT EXISTS (
  SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = '${MIND_FORGE_DEVELOP_USER}'
);
\gexec

SELECT format(
  'CREATE ROLE %I LOGIN PASSWORD %L',
  '${MIND_FORGE_MAIN_USER}',
  '${MIND_FORGE_MAIN_PASSWORD}'
)
WHERE NOT EXISTS (
  SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = '${MIND_FORGE_MAIN_USER}'
);
\gexec

SELECT format(
  'CREATE DATABASE %I OWNER %I',
  '${MIND_FORGE_DEVELOP_DB}',
  '${MIND_FORGE_DEVELOP_USER}'
)
WHERE NOT EXISTS (
  SELECT 1 FROM pg_catalog.pg_database WHERE datname = '${MIND_FORGE_DEVELOP_DB}'
);
\gexec

SELECT format(
  'CREATE DATABASE %I OWNER %I',
  '${MIND_FORGE_MAIN_DB}',
  '${MIND_FORGE_MAIN_USER}'
)
WHERE NOT EXISTS (
  SELECT 1 FROM pg_catalog.pg_database WHERE datname = '${MIND_FORGE_MAIN_DB}'
);
\gexec
SQL

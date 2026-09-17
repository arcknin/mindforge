-- Infrastructure-only extensions for the reduced PostgreSQL/Auth/pgvector stack.
-- No Mindforge domain schema, table, seed, or endpoint is defined here.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "citext";

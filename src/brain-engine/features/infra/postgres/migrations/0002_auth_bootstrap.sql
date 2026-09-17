-- GoTrue owns its own schema and migration history. Its exact pinned image runs
-- `gotrue migrate` after this infrastructure ledger step and before `gotrue serve`.
-- This migration intentionally creates no Mindforge domain object.

DO $$
BEGIN
  RAISE NOTICE 'GoTrue bootstrap is executed by the pinned gotrue migrate command';
END;
$$;

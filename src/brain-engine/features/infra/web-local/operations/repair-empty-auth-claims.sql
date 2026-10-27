-- SUPERSEDED: owner chose deletion/recreation; this prepared field repair was never applied.
-- Operational repair, not a schema migration. Run only on owner-approved VM develop.
-- Operator must privately back up id/aud/role before setting repair_target_user_id.
-- Never delete/recreate the account or alter Brain/identities/email/confirmation.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '15s';
SELECT set_config('mindforge.repair_target_user_id', :'repair_target_user_id', true) IS NOT NULL AS target_set;
DO $repair$
DECLARE
  target uuid := current_setting('mindforge.repair_target_user_id')::uuid;
  changed integer;
BEGIN
  IF current_database() <> 'mindforge_develop' THEN
    RAISE EXCEPTION 'Repair restricted to develop';
  END IF;
  UPDATE auth.users AS u
     SET aud = 'authenticated', role = 'authenticated'
   WHERE u.id = target
     AND u.aud = '' AND u.role = ''
     AND u.is_anonymous = false
     AND (u.email_confirmed_at IS NOT NULL OR u.confirmed_at IS NOT NULL)
     AND EXISTS (SELECT 1 FROM auth.identities AS i
                  WHERE i.user_id = u.id AND i.provider = 'google');
  GET DIAGNOSTICS changed = ROW_COUNT;
  IF changed <> 1 THEN
    RAISE EXCEPTION 'Repair requires exactly one confirmed Google account with empty claims';
  END IF;
END
$repair$;
COMMIT;

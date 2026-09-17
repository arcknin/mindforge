-- GoTrue's pinned upstream catalog grants read access to the PostgreSQL
-- compatibility role. Create it only when the custom initial cluster user did
-- not create it already. The role cannot log in and creates no domain object.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'postgres') THEN
    CREATE ROLE postgres NOLOGIN;
  END IF;
END;
$$;

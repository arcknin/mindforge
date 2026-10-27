DO $$
DECLARE
    schema_complete boolean;
BEGIN
WITH schema_state AS (
    SELECT
        EXISTS (
            SELECT 1
            FROM pg_catalog.pg_namespace
            WHERE nspname = 'brain_core'
        ) AS schema_exists,
        COALESCE(
            pg_catalog.has_database_privilege(
                current_user,
                current_database(),
                'CREATE'
            ),
            false
        ) AS database_create
),
relation_state AS (
    SELECT
        COALESCE(
            array_agg(c.relname::text ORDER BY c.relname)
                FILTER (WHERE c.relkind = 'r'),
            ARRAY[]::text[]
        ) = ARRAY['brains', 'nodes']::text[] AS tables_ok,
        COALESCE(
            array_agg(c.relname::text ORDER BY c.relname)
                FILTER (WHERE c.relkind = 'i'),
            ARRAY[]::text[]
        ) = ARRAY[
            'brain_core_nodes_parent_idx',
            'brain_core_one_root_per_brain',
            'brains_owner_subject_key',
            'brains_pkey',
            'nodes_pk'
        ]::text[] AS indexes_ok,
        COALESCE(bool_and(c.relkind IN ('r', 'i')), false) AS kinds_ok
    FROM pg_catalog.pg_class AS c
    JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
    WHERE n.nspname = 'brain_core'
),
column_catalog AS (
    SELECT
        c.relname::text AS table_name,
        array_agg(a.attname::text ORDER BY a.attnum) AS column_names,
        array_agg(
            pg_catalog.format_type(a.atttypid, a.atttypmod)::text
            ORDER BY a.attnum
        ) AS column_types,
        array_agg(a.attnotnull ORDER BY a.attnum) AS not_null_columns
    FROM pg_catalog.pg_class AS c
    JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
    JOIN pg_catalog.pg_attribute AS a ON a.attrelid = c.oid
    WHERE n.nspname = 'brain_core'
      AND c.relname IN ('brains', 'nodes')
      AND a.attnum > 0
      AND NOT a.attisdropped
    GROUP BY c.relname
),
column_state AS (
    SELECT
        count(*) = 2
        AND COALESCE(
            bool_and(
                CASE table_name
                    WHEN 'brains' THEN
                        column_names = ARRAY[
                            'brain_id', 'owner_subject', 'created_at'
                        ]::text[]
                        AND column_types = ARRAY[
                            'uuid', 'text', 'timestamp with time zone'
                        ]::text[]
                        AND not_null_columns = ARRAY[true, true, true]::boolean[]
                    WHEN 'nodes' THEN
                        column_names = ARRAY[
                            'brain_id', 'node_id', 'kind', 'parent_id',
                            'name', 'version', 'created_at', 'updated_at'
                        ]::text[]
                        AND column_types = ARRAY[
                            'uuid', 'uuid', 'text', 'uuid', 'text', 'bigint',
                            'timestamp with time zone',
                            'timestamp with time zone'
                        ]::text[]
                        AND not_null_columns = ARRAY[
                            true, true, true, false, false, true, true, true
                        ]::boolean[]
                    ELSE false
                END
            ),
            false
        ) AS columns_ok
    FROM column_catalog
),
constraint_catalog AS (
    SELECT
        c.conname::text AS constraint_name,
        c.contype AS constraint_type,
        child.relname::text AS child_table,
        ARRAY(
            SELECT a.attname::text
            FROM pg_catalog.unnest(c.conkey) WITH ORDINALITY AS key(attnum, ordinal)
            JOIN pg_catalog.pg_attribute AS a
              ON a.attrelid = c.conrelid
             AND a.attnum = key.attnum
            ORDER BY key.ordinal
        ) AS child_columns,
        CASE
            WHEN c.confrelid = 0 THEN NULL
            ELSE parent.relname::text
        END AS parent_table,
        ARRAY(
            SELECT a.attname::text
            FROM pg_catalog.unnest(c.confkey) WITH ORDINALITY AS key(attnum, ordinal)
            JOIN pg_catalog.pg_attribute AS a
              ON a.attrelid = c.confrelid
             AND a.attnum = key.attnum
            ORDER BY key.ordinal
        ) AS parent_columns,
        c.confdeltype AS delete_action,
        c.convalidated AS validated,
        regexp_replace(
            lower(pg_catalog.pg_get_constraintdef(c.oid)),
            '[[:space:]"]+',
            '',
            'g'
        ) AS definition
    FROM pg_catalog.pg_constraint AS c
    JOIN pg_catalog.pg_class AS child ON child.oid = c.conrelid
    JOIN pg_catalog.pg_namespace AS n ON n.oid = child.relnamespace
    LEFT JOIN pg_catalog.pg_class AS parent ON parent.oid = c.confrelid
    WHERE n.nspname = 'brain_core'
),
constraint_state AS (
    SELECT
        count(*) = 9
        AND COALESCE(
            bool_and(
                validated
                AND CASE constraint_name
                    WHEN 'brains_pkey' THEN
                        constraint_type = 'p'
                        AND child_table = 'brains'
                        AND child_columns = ARRAY['brain_id']::text[]
                    WHEN 'brains_owner_subject_key' THEN
                        constraint_type = 'u'
                        AND child_table = 'brains'
                        AND child_columns = ARRAY['owner_subject']::text[]
                    WHEN 'brains_owner_subject_not_blank' THEN
                        constraint_type = 'c'
                        AND child_table = 'brains'
                        AND position('btrim(owner_subject)' IN definition) > 0
                        AND position('<>' IN definition) > 0
                    WHEN 'nodes_pk' THEN
                        constraint_type = 'p'
                        AND child_table = 'nodes'
                        AND child_columns = ARRAY['brain_id', 'node_id']::text[]
                    WHEN 'nodes_brain_fk' THEN
                        constraint_type = 'f'
                        AND child_table = 'nodes'
                        AND child_columns = ARRAY['brain_id']::text[]
                        AND parent_table = 'brains'
                        AND parent_columns = ARRAY['brain_id']::text[]
                        AND delete_action = 'c'
                    WHEN 'nodes_parent_fk' THEN
                        constraint_type = 'f'
                        AND child_table = 'nodes'
                        AND child_columns = ARRAY['brain_id', 'parent_id']::text[]
                        AND parent_table = 'nodes'
                        AND parent_columns = ARRAY['brain_id', 'node_id']::text[]
                        AND delete_action = 'a'
                    WHEN 'nodes_kind_check' THEN
                        constraint_type = 'c'
                        AND child_table = 'nodes'
                        AND position('kind' IN definition) > 0
                        AND position('root' IN definition) > 0
                        AND position('layer' IN definition) > 0
                    WHEN 'nodes_version_check' THEN
                        constraint_type = 'c'
                        AND child_table = 'nodes'
                        AND position('version>0' IN definition) > 0
                    WHEN 'nodes_shape_check' THEN
                        constraint_type = 'c'
                        AND child_table = 'nodes'
                        AND position('kind=''root''' IN definition) > 0
                        AND position('kind=''layer''' IN definition) > 0
                        AND position('parent_idisnull' IN definition) > 0
                        AND position('parent_idisnotnull' IN definition) > 0
                        AND position('nameisnull' IN definition) > 0
                        AND position('nameisnotnull' IN definition) > 0
                        AND position('version=1' IN definition) > 0
                        AND position('btrim(name)' IN definition) > 0
                    ELSE false
                END
            ),
            false
        ) AS constraints_ok
    FROM constraint_catalog
),
expected_indexes(index_name, definition, is_primary, is_unique) AS (
    VALUES
        (
            'brains_pkey',
            'createuniqueindexbrains_pkeyonbrain_core.brainsusingbtree(brain_id)',
            true,
            true
        ),
        (
            'brains_owner_subject_key',
            'createuniqueindexbrains_owner_subject_keyonbrain_core.brainsusingbtree(owner_subject)',
            false,
            true
        ),
        (
            'nodes_pk',
            'createuniqueindexnodes_pkonbrain_core.nodesusingbtree(brain_id,node_id)',
            true,
            true
        ),
        (
            'brain_core_one_root_per_brain',
            'createuniqueindexbrain_core_one_root_per_brainonbrain_core.nodesusingbtree(brain_id)where(kind=''root''::text)',
            false,
            true
        ),
        (
            'brain_core_nodes_parent_idx',
            'createindexbrain_core_nodes_parent_idxonbrain_core.nodesusingbtree(brain_id,parent_id)',
            false,
            false
        )
),
actual_indexes AS (
    SELECT
        c.relname::text AS index_name,
        i.indisprimary AS is_primary,
        i.indisunique AS is_unique,
        i.indisvalid AS is_valid,
        i.indisready AS is_ready,
        regexp_replace(
            lower(pg_catalog.pg_get_indexdef(i.indexrelid)),
            '[[:space:]"]+',
            '',
            'g'
        ) AS definition
    FROM pg_catalog.pg_class AS c
    JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
    JOIN pg_catalog.pg_index AS i ON i.indexrelid = c.oid
    WHERE n.nspname = 'brain_core'
),
index_state AS (
    SELECT
        (SELECT count(*) FROM actual_indexes) = 5
        AND count(*) = 5
        AND COALESCE(
            bool_and(
                actual_indexes.index_name IS NOT NULL
                AND actual_indexes.is_primary = expected_indexes.is_primary
                AND actual_indexes.is_unique = expected_indexes.is_unique
                AND actual_indexes.is_valid
                AND actual_indexes.is_ready
                AND actual_indexes.definition = expected_indexes.definition
            ),
            false
        ) AS indexes_ok
    FROM expected_indexes
    LEFT JOIN actual_indexes USING (index_name)
),
shape_state AS (
    SELECT
        relation_state.tables_ok
        AND relation_state.indexes_ok
        AND relation_state.kinds_ok
        AND column_state.columns_ok
        AND constraint_state.constraints_ok
        AND index_state.indexes_ok AS shape_complete
    FROM relation_state
    CROSS JOIN column_state
    CROSS JOIN constraint_state
    CROSS JOIN index_state
)
SELECT schema_state.schema_exists AND shape_state.shape_complete
INTO schema_complete
FROM schema_state
CROSS JOIN shape_state;

IF NOT COALESCE(schema_complete, false) THEN
    RAISE EXCEPTION 'Brain Core schema does not match the approved migration shape';
END IF;
END
$$;

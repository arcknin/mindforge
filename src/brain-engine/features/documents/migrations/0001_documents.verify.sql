WITH column_catalog AS (
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
    WHERE n.nspname = 'brain_documents'
      AND c.relname IN ('documents', 'document_links')
      AND c.relkind = 'r'
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
                    WHEN 'documents' THEN
                        column_names = ARRAY[
                            'document_id', 'brain_id', 'layer_id', 'title',
                            'markdown', 'version', 'created_at', 'updated_at',
                            'deleted_at', 'expires_at', 'tombstoned_at'
                        ]::text[]
                        AND column_types = ARRAY[
                            'uuid', 'uuid', 'uuid', 'text', 'text', 'bigint',
                            'timestamp with time zone', 'timestamp with time zone',
                            'timestamp with time zone', 'timestamp with time zone',
                            'timestamp with time zone'
                        ]::text[]
                        AND not_null_columns = ARRAY[
                            true, true, false, true, true, true,
                            true, true, false, false, false
                        ]::boolean[]
                    WHEN 'document_links' THEN
                        column_names = ARRAY[
                            'brain_id', 'source_document_id',
                            'target_document_id', 'position'
                        ]::text[]
                        AND column_types = ARRAY[
                            'uuid', 'uuid', 'uuid', 'bigint'
                        ]::text[]
                        AND not_null_columns = ARRAY[true, true, true, true]::boolean[]
                    ELSE false
                END
            ),
            false
        ) AS columns_present
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
        CASE
            WHEN c.confrelid = 0 THEN NULL
            ELSE parent_namespace.nspname::text
        END AS parent_schema,
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
            '[[:space:]\"]+',
            '',
            'g'
        ) AS definition
    FROM pg_catalog.pg_constraint AS c
    JOIN pg_catalog.pg_class AS child ON child.oid = c.conrelid
    JOIN pg_catalog.pg_namespace AS n ON n.oid = child.relnamespace
    LEFT JOIN pg_catalog.pg_class AS parent ON parent.oid = c.confrelid
    LEFT JOIN pg_catalog.pg_namespace AS parent_namespace ON parent_namespace.oid = parent.relnamespace
    WHERE n.nspname = 'brain_documents'
),
constraint_state AS (
    SELECT
        count(*) = 10
        AND COALESCE(
            bool_and(
                validated
                AND CASE constraint_name
                    WHEN 'documents_pkey' THEN
                        constraint_type = 'p'
                        AND child_table = 'documents'
                        AND child_columns = ARRAY['document_id']::text[]
                    WHEN 'documents_brain_key' THEN
                        constraint_type = 'u'
                        AND child_table = 'documents'
                        AND child_columns = ARRAY['brain_id', 'document_id']::text[]
                    WHEN 'documents_brain_fk' THEN
                        constraint_type = 'f'
                        AND child_table = 'documents'
                        AND child_columns = ARRAY['brain_id']::text[]
                        AND parent_table = 'brains'
                        AND parent_schema = 'brain_core'
                        AND parent_columns = ARRAY['brain_id']::text[]
                        AND delete_action = 'a'
                    WHEN 'documents_brain_layer_fk' THEN
                        constraint_type = 'f'
                        AND child_table = 'documents'
                        AND child_columns = ARRAY['brain_id', 'layer_id']::text[]
                        AND parent_table = 'nodes'
                        AND parent_schema = 'brain_core'
                        AND parent_columns = ARRAY['brain_id', 'node_id']::text[]
                        AND delete_action = 'a'
                    WHEN 'documents_version_check' THEN
                        constraint_type = 'c'
                        AND child_table = 'documents'
                        AND position('version>0' IN definition) > 0
                    WHEN 'documents_content_check' THEN
                        constraint_type = 'c'
                        AND child_table = 'documents'
                        AND position('tombstoned_at' IN definition) > 0
                        AND position('layer_id' IN definition) > 0
                        AND position('markdown' IN definition) > 0
                    WHEN 'document_links_pk' THEN
                        constraint_type = 'p'
                        AND child_table = 'document_links'
                        AND child_columns = ARRAY[
                            'brain_id', 'source_document_id', 'target_document_id'
                        ]::text[]
                    WHEN 'document_links_position_check' THEN
                        constraint_type = 'c'
                        AND child_table = 'document_links'
                        AND position('position>=0' IN definition) > 0
                    WHEN 'document_links_source_fk' THEN
                        constraint_type = 'f'
                        AND child_table = 'document_links'
                        AND child_columns = ARRAY['brain_id', 'source_document_id']::text[]
                        AND parent_table = 'documents'
                        AND parent_schema = 'brain_documents'
                        AND parent_columns = ARRAY['brain_id', 'document_id']::text[]
                        AND delete_action = 'a'
                    WHEN 'document_links_target_fk' THEN
                        constraint_type = 'f'
                        AND child_table = 'document_links'
                        AND child_columns = ARRAY['brain_id', 'target_document_id']::text[]
                        AND parent_table = 'documents'
                        AND parent_schema = 'brain_documents'
                        AND parent_columns = ARRAY['brain_id', 'document_id']::text[]
                        AND delete_action = 'a'
                    ELSE false
                END
            ),
            false
        ) AS constraints_present
    FROM constraint_catalog
),
relation_state AS (
    SELECT
        EXISTS (
            SELECT 1
            FROM pg_catalog.pg_class AS c
            JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
            WHERE n.nspname = 'brain_documents'
              AND c.relname = 'documents'
              AND c.relkind = 'r'
        ) AS documents_table,
        EXISTS (
            SELECT 1
            FROM pg_catalog.pg_class AS c
            JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
            WHERE n.nspname = 'brain_documents'
              AND c.relname = 'document_links'
              AND c.relkind = 'r'
        ) AS targets_table,
        (SELECT columns_present FROM column_state) AS columns_present,
        (SELECT constraints_present FROM constraint_state) AS constraints_present,
        EXISTS (
            SELECT 1
            FROM pg_catalog.pg_trigger AS t
            JOIN pg_catalog.pg_class AS c ON c.oid = t.tgrelid
            JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
            JOIN pg_catalog.pg_proc AS p ON p.oid = t.tgfoid
            JOIN pg_catalog.pg_namespace AS pn ON pn.oid = p.pronamespace
            WHERE n.nspname = 'brain_documents'
              AND c.relname = 'documents'
              AND t.tgname = 'documents_validate_layer'
              AND t.tgenabled = 'O'
              AND p.proname = 'validate_document_layer'
              AND pn.nspname = 'brain_documents'
              AND NOT t.tgisinternal
        ) AS layer_trigger,
        (
            SELECT count(*) = 6
                   AND COALESCE(bool_and(i.indisvalid AND i.indisready), false)
                   AND array_agg(c.relname::text ORDER BY c.relname) = ARRAY[
                       'document_links_pk',
                       'document_links_source_idx',
                       'documents_brain_key',
                       'documents_brain_layer_idx',
                       'documents_owner_idx',
                       'documents_pkey'
                   ]::text[]
            FROM pg_catalog.pg_class AS c
            JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
            JOIN pg_catalog.pg_index AS i ON i.indexrelid = c.oid
            WHERE n.nspname = 'brain_documents' AND c.relkind = 'i'
        ) AS indexes_present
)
SELECT CASE
    WHEN documents_table AND targets_table AND columns_present AND layer_trigger AND indexes_present AND constraints_present
        THEN 'complete'
    ELSE 'partial_divergent'
END AS status,
documents_table,
targets_table,
columns_present,
layer_trigger,
indexes_present,
constraints_present
FROM relation_state;

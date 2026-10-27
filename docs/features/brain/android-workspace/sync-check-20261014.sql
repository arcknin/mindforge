BEGIN TRANSACTION READ ONLY;

SELECT json_build_object(
    'current_database', current_database(),
    'current_user', current_user
)::text AS connection_context;

SELECT json_build_object(
    'total_users', count(*)
)::text AS user_count
FROM auth.users;

SELECT json_build_object(
    'total_brains', count(*),
    'distinct_brain_owners', count(DISTINCT owner_subject),
    'nonblank_owner_subjects', count(*) FILTER (WHERE btrim(owner_subject) <> ''),
    'owners_with_auth_user', count(*) FILTER (WHERE u.id IS NOT NULL),
    'owners_without_auth_user', count(*) FILTER (WHERE u.id IS NULL)
)::text AS brain_ownership_count
FROM brain_core.brains AS b
LEFT JOIN auth.users AS u
    ON u.id::text = b.owner_subject;

SELECT json_build_object(
    'total_documents', count(*),
    'active_documents', count(*) FILTER (
        WHERE d.tombstoned_at IS NULL AND d.deleted_at IS NULL
    ),
    'trash_documents', count(*) FILTER (
        WHERE d.tombstoned_at IS NULL
          AND d.deleted_at IS NOT NULL
          AND d.expires_at > clock_timestamp()
    ),
    'expired_trash_rows', count(*) FILTER (
        WHERE d.tombstoned_at IS NULL
          AND d.deleted_at IS NOT NULL
          AND (d.expires_at IS NULL OR d.expires_at <= clock_timestamp())
    ),
    'tombstoned_rows', count(*) FILTER (WHERE d.tombstoned_at IS NOT NULL),
    'documents_without_brain', count(*) FILTER (WHERE b.brain_id IS NULL),
    'documents_without_owner', count(*) FILTER (
        WHERE b.brain_id IS NULL
           OR btrim(b.owner_subject) = ''
           OR u.id IS NULL
    ),
    'documents_without_valid_layer', count(*) FILTER (
        WHERE d.tombstoned_at IS NULL AND n.kind IS DISTINCT FROM 'layer'
    ),
    'owner_coherent', coalesce(bool_and(
        b.brain_id IS NOT NULL
        AND btrim(b.owner_subject) <> ''
        AND u.id IS NOT NULL
        AND (
            (d.tombstoned_at IS NULL AND n.kind = 'layer')
            OR (d.tombstoned_at IS NOT NULL AND d.layer_id IS NULL)
        )
    ), true)
)::text AS document_ownership_aggregate
FROM brain_documents.documents AS d
LEFT JOIN brain_core.brains AS b
    ON b.brain_id = d.brain_id
LEFT JOIN auth.users AS u
    ON u.id::text = b.owner_subject
LEFT JOIN brain_core.nodes AS n
    ON n.brain_id = d.brain_id
   AND n.node_id = d.layer_id;

SELECT json_build_object(
    'document_id_short', left(d.document_id::text, 8),
    'title', d.title,
    'version', d.version,
    'created_at_utc', to_char(d.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'updated_at_utc', to_char(d.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'markdown_state', CASE WHEN d.markdown = '' THEN 'empty' ELSE 'nonempty' END,
    'markdown_bytes', octet_length(d.markdown),
    'markdown_chars', char_length(d.markdown),
    'state', CASE
        WHEN d.tombstoned_at IS NOT NULL THEN 'tombstoned'
        WHEN d.deleted_at IS NULL THEN 'active'
        WHEN d.expires_at > clock_timestamp() THEN 'trash'
        ELSE 'expired_trash'
    END,
    'owner_coherence', CASE
        WHEN b.brain_id IS NOT NULL
         AND btrim(b.owner_subject) <> ''
         AND u.id IS NOT NULL
         AND (
             (d.tombstoned_at IS NULL AND n.kind = 'layer')
             OR (d.tombstoned_at IS NOT NULL AND d.layer_id IS NULL)
         )
            THEN 'coherent'
        ELSE 'incoherent'
    END
)::text AS document_metadata
FROM brain_documents.documents AS d
LEFT JOIN brain_core.brains AS b
    ON b.brain_id = d.brain_id
LEFT JOIN auth.users AS u
    ON u.id::text = b.owner_subject
LEFT JOIN brain_core.nodes AS n
    ON n.brain_id = d.brain_id
   AND n.node_id = d.layer_id
ORDER BY d.created_at, d.document_id;

ROLLBACK;

-- Document content is feature-owned. The infra operator registers this file
-- in its global catalog, but never substitutes it for the Brain Core schema.

CREATE SCHEMA brain_documents;

CREATE TABLE brain_documents.documents (
    document_id uuid PRIMARY KEY,
    brain_id uuid NOT NULL,
    layer_id uuid,
    title text NOT NULL DEFAULT '',
    markdown text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    deleted_at timestamptz,
    expires_at timestamptz,
    tombstoned_at timestamptz,
    CONSTRAINT documents_brain_key UNIQUE (brain_id, document_id),
    CONSTRAINT documents_brain_fk
        FOREIGN KEY (brain_id)
        REFERENCES brain_core.brains (brain_id),
    CONSTRAINT documents_brain_layer_fk
        FOREIGN KEY (brain_id, layer_id)
        REFERENCES brain_core.nodes (brain_id, node_id),
    CONSTRAINT documents_version_check CHECK (version > 0),
    CONSTRAINT documents_content_check CHECK (
        (
            tombstoned_at IS NULL
            AND layer_id IS NOT NULL
            AND char_length(title) BETWEEN 1 AND 255
            AND title !~ '[[:cntrl:]]'
            AND octet_length(markdown) <= 1048576
            AND (
                (deleted_at IS NULL AND expires_at IS NULL)
                OR (deleted_at IS NOT NULL AND expires_at IS NOT NULL AND expires_at > deleted_at)
            )
        )
        OR (
            tombstoned_at IS NOT NULL
            AND layer_id IS NULL
            AND title = ''
            AND markdown = ''
            AND deleted_at IS NOT NULL
            AND expires_at IS NULL
        )
    )
);

CREATE INDEX documents_owner_idx
    ON brain_documents.documents (brain_id, document_id);

CREATE INDEX documents_brain_layer_idx
    ON brain_documents.documents (brain_id, layer_id, document_id);

CREATE TABLE brain_documents.document_links (
    brain_id uuid NOT NULL,
    source_document_id uuid NOT NULL,
    target_document_id uuid NOT NULL,
    position bigint NOT NULL,
    CONSTRAINT document_links_pk
        PRIMARY KEY (brain_id, source_document_id, target_document_id),
    CONSTRAINT document_links_position_check CHECK (position >= 0),
    CONSTRAINT document_links_source_fk
        FOREIGN KEY (brain_id, source_document_id)
        REFERENCES brain_documents.documents (brain_id, document_id),
    CONSTRAINT document_links_target_fk
        FOREIGN KEY (brain_id, target_document_id)
        REFERENCES brain_documents.documents (brain_id, document_id)
);

CREATE INDEX document_links_source_idx
    ON brain_documents.document_links (brain_id, source_document_id, position);

CREATE OR REPLACE FUNCTION brain_documents.validate_document_layer()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.tombstoned_at IS NOT NULL THEN
        RETURN NEW;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM brain_core.nodes AS n
        WHERE n.brain_id = NEW.brain_id
          AND n.node_id = NEW.layer_id
          AND n.kind = 'layer'
    ) THEN
        RAISE EXCEPTION 'document layer must be an owned layer'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER documents_validate_layer
    BEFORE INSERT OR UPDATE OF brain_id, layer_id, tombstoned_at
    ON brain_documents.documents
    FOR EACH ROW
    EXECUTE FUNCTION brain_documents.validate_document_layer();


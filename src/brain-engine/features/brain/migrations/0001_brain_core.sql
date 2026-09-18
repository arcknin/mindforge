CREATE SCHEMA IF NOT EXISTS brain_core;

CREATE TABLE brain_core.brains (
    brain_id uuid PRIMARY KEY,
    owner_subject text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT brains_owner_subject_not_blank CHECK (btrim(owner_subject) <> '')
);

CREATE TABLE brain_core.nodes (
    brain_id uuid NOT NULL,
    node_id uuid NOT NULL,
    kind text NOT NULL,
    parent_id uuid,
    name text,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT nodes_pk PRIMARY KEY (brain_id, node_id),
    CONSTRAINT nodes_brain_fk
        FOREIGN KEY (brain_id)
        REFERENCES brain_core.brains (brain_id)
        ON DELETE CASCADE,
    CONSTRAINT nodes_parent_fk
        FOREIGN KEY (brain_id, parent_id)
        REFERENCES brain_core.nodes (brain_id, node_id),
    CONSTRAINT nodes_kind_check CHECK (kind IN ('root', 'layer')),
    CONSTRAINT nodes_version_check CHECK (version > 0),
    CONSTRAINT nodes_shape_check CHECK (
        (kind = 'root' AND parent_id IS NULL AND name IS NULL AND version = 1)
        OR
        (kind = 'layer' AND parent_id IS NOT NULL AND name IS NOT NULL AND btrim(name) <> '')
    )
);

CREATE UNIQUE INDEX brain_core_one_root_per_brain
    ON brain_core.nodes (brain_id)
    WHERE kind = 'root';

CREATE INDEX brain_core_nodes_parent_idx
    ON brain_core.nodes (brain_id, parent_id);

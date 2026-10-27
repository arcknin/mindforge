package documents

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// row and rows deliberately contain only the pgx operations this adapter
// needs. The production pool is wrapped below, so tests can exercise the
// transaction boundary and database outcomes without a PostgreSQL process.
type row interface {
	Scan(dest ...any) error
}

type rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}

type transaction interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (rows, error)
	QueryRow(context.Context, string, ...any) row
	Commit(context.Context) error
	Rollback(context.Context) error
}

type database interface {
	BeginTx(context.Context, pgx.TxOptions) (transaction, error)
}

type poolDatabase struct {
	pool *pgxpool.Pool
}

func (p poolDatabase) BeginTx(ctx context.Context, options pgx.TxOptions) (transaction, error) {
	tx, err := p.pool.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return poolTransaction{tx: tx}, nil
}

type poolTransaction struct {
	tx pgx.Tx
}

func (t poolTransaction) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

func (t poolTransaction) Query(ctx context.Context, sql string, args ...any) (rows, error) {
	result, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return poolRows{rows: result}, nil
}

func (t poolTransaction) QueryRow(ctx context.Context, sql string, args ...any) row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t poolTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t poolTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

type poolRows struct {
	rows pgx.Rows
}

func (r poolRows) Next() bool             { return r.rows.Next() }
func (r poolRows) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r poolRows) Close()                 { r.rows.Close() }
func (r poolRows) Err() error             { return r.rows.Err() }

// PostgresStore is the authoritative persistence adapter for documents. It
// never applies migrations; schema lifecycle belongs to the infra operator.
type PostgresStore struct {
	db      database
	closeFn func()
}

const maxDocumentVersion int64 = 1<<63 - 1

// OpenPostgresStore opens a verified PostgreSQL pool and never runs DDL.
// A malformed or unreachable DSN fails closed with an opaque internal error.
func OpenPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	if ctx == nil || strings.TrimSpace(dsn) == "" {
		return nil, brain.ErrInternal
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, brain.ErrInternal
	}
	config.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, brain.ErrInternal
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, brain.ErrInternal
	}
	return &PostgresStore{
		db:      poolDatabase{pool: pool},
		closeFn: pool.Close,
	}, nil
}

// Close releases the pool. It is safe to call on a nil or already closed
// store.
func (s *PostgresStore) Close() {
	if s == nil || s.closeFn == nil {
		return
	}
	s.closeFn()
	s.closeFn = nil
}

// newPostgresStore is intentionally private: production construction goes
// through OpenPostgresStore, while package tests inject the narrow database
// port above.
func newPostgresStore(db database) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Apply(ctx context.Context, owner brain.Principal, mutation Mutation) (Document, error) {
	if !s.ready() {
		return Document{}, brain.ErrInternal
	}
	if ctx == nil {
		return Document{}, brain.ErrInternal
	}
	if strings.TrimSpace(string(owner)) == "" {
		return Document{}, brain.ErrUnauthenticated
	}
	if err := validateMutation(mutation); err != nil {
		return Document{}, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Document{}, brain.ErrInternal
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := purgeExpired(ctx, tx, owner); err != nil {
		return Document{}, err
	}

	var document Document
	switch mutation.Kind {
	case KindCreate:
		document, err = s.applyCreate(ctx, tx, owner, mutation)
	case KindUpdate:
		document, err = s.applyUpdate(ctx, tx, owner, mutation)
	case KindTrash:
		document, err = s.applyTrash(ctx, tx, owner, mutation)
	case KindRestore:
		document, err = s.applyRestore(ctx, tx, owner, mutation)
	default:
		return Document{}, ErrInvalidDocument
	}
	if err != nil {
		return Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, brain.ErrInternal
	}
	return document, nil
}

func (s *PostgresStore) Get(ctx context.Context, owner brain.Principal, id string) (Document, error) {
	if !s.ready() {
		return Document{}, brain.ErrInternal
	}
	if ctx == nil {
		return Document{}, brain.ErrInternal
	}
	if strings.TrimSpace(string(owner)) == "" {
		return Document{}, brain.ErrUnauthenticated
	}
	if err := validateUUID(id); err != nil {
		return Document{}, ErrInvalidDocument
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Document{}, brain.ErrInternal
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := purgeExpired(ctx, tx, owner); err != nil {
		return Document{}, err
	}
	stored, err := loadOwned(ctx, tx, owner, id, false, false)
	if err != nil {
		return Document{}, err
	}
	if stored.tombstoned {
		if err := tx.Commit(ctx); err != nil {
			return Document{}, brain.ErrInternal
		}
		return Document{}, brain.ErrNotFound
	}
	if stored.document.DeletedAt != nil {
		var unexpired bool
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(
				deleted_at IS NULL
				OR (expires_at IS NOT NULL AND expires_at > clock_timestamp()),
				false
			)
			FROM brain_documents.documents
			WHERE document_id = $1::uuid`, id).Scan(&unexpired); err != nil {
			return Document{}, mapPostgresError(err)
		}
		if !unexpired {
			if err := purgeExpired(ctx, tx, owner); err != nil {
				return Document{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return Document{}, brain.ErrInternal
			}
			return Document{}, brain.ErrNotFound
		}
	}
	if err := loadLinks(ctx, tx, &stored.document); err != nil {
		return Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, brain.ErrInternal
	}
	return stored.document, nil
}

func (s *PostgresStore) List(ctx context.Context, owner brain.Principal, query Query) (Page, error) {
	if !s.ready() {
		return Page{}, brain.ErrInternal
	}
	if ctx == nil {
		return Page{}, brain.ErrInternal
	}
	if strings.TrimSpace(string(owner)) == "" {
		return Page{}, brain.ErrUnauthenticated
	}
	query, err := normalizeStoreQuery(query)
	if err != nil {
		return Page{}, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Page{}, brain.ErrInternal
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := purgeExpired(ctx, tx, owner); err != nil {
		return Page{}, err
	}
	brainID, err := brainIDForOwner(ctx, tx, owner)
	if err != nil {
		return Page{}, err
	}
	var layerExistsForQuery bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM brain_core.nodes
			WHERE brain_id = $1::uuid AND node_id = $2::uuid AND kind = 'layer'
		)`, brainID, query.LayerID).Scan(&layerExistsForQuery); err != nil {
		return Page{}, mapPostgresError(err)
	}
	if !layerExistsForQuery {
		return Page{}, brain.ErrNotFound
	}

	rows, err := tx.Query(ctx, `
		WITH RECURSIVE owned_brain AS (
			SELECT brain_id
			FROM brain_core.brains
			WHERE brain_id = $1::uuid AND owner_subject = $2
		), scope_layers AS (
			SELECT n.brain_id, n.node_id
			FROM brain_core.nodes AS n
			JOIN owned_brain AS b ON b.brain_id = n.brain_id
			WHERE n.node_id = $3::uuid AND n.kind = 'layer'
			UNION ALL
			SELECT child.brain_id, child.node_id
			FROM brain_core.nodes AS child
			JOIN scope_layers AS parent
			  ON parent.brain_id = child.brain_id
			 AND parent.node_id = child.parent_id
			WHERE $4 = 'subtree'
		)
		SELECT d.document_id::text,
		       d.layer_id::text,
		       d.title,
		       d.version,
		       d.created_at,
		       d.updated_at,
		       d.deleted_at,
		       d.expires_at
		FROM brain_documents.documents AS d
		JOIN scope_layers AS l
		  ON l.brain_id = d.brain_id AND l.node_id = d.layer_id
		WHERE d.tombstoned_at IS NULL
		  AND (($5 = 'active' AND d.deleted_at IS NULL)
		    OR ($5 = 'trash' AND d.deleted_at IS NOT NULL
		        AND d.expires_at > clock_timestamp()))
		  AND ($6::uuid IS NULL OR d.document_id > $6::uuid)
		ORDER BY d.document_id ASC
		LIMIT $7`, brainID, string(owner), query.LayerID, query.Scope, query.State, nullableUUID(query.After), query.Limit+1)
	if err != nil {
		return Page{}, mapPostgresError(err)
	}
	documents := make([]Document, 0, query.Limit)
	nextCursor := ""
	for rows.Next() {
		var document Document
		var deletedAt, expiresAt *time.Time
		if err := rows.Scan(
			&document.ID,
			&document.LayerID,
			&document.Title,
			&document.Version,
			&document.CreatedAt,
			&document.UpdatedAt,
			&deletedAt,
			&expiresAt,
		); err != nil {
			rows.Close()
			return Page{}, brain.ErrInternal
		}
		document.DeletedAt = deletedAt
		document.ExpiresAt = expiresAt
		if len(documents) < query.Limit {
			documents = append(documents, document)
		} else {
			nextCursor = documents[len(documents)-1].ID
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Page{}, brain.ErrInternal
	}
	if err := tx.Commit(ctx); err != nil {
		return Page{}, brain.ErrInternal
	}
	return Page{Documents: documents, NextCursor: nextCursor}, nil
}

type storedDocument struct {
	document   Document
	brainID    string
	tombstoned bool
}

func (s *PostgresStore) applyCreate(ctx context.Context, tx transaction, owner brain.Principal, mutation Mutation) (Document, error) {
	brainID, err := brainIDForOwner(ctx, tx, owner)
	if err != nil {
		return Document{}, err
	}
	if exists, err := layerExists(ctx, tx, brainID, mutation.LayerID); err != nil {
		return Document{}, err
	} else if !exists {
		return Document{}, brain.ErrNotFound
	}
	if exists, err := documentExists(ctx, tx, mutation.ID); err != nil {
		return Document{}, err
	} else if exists {
		return Document{}, brain.ErrConflict
	}
	if err := txExecInsert(ctx, tx, mutation, brainID); err != nil {
		return Document{}, err
	}
	if err := replaceLinks(ctx, tx, brainID, mutation.ID, mutation.Targets); err != nil {
		return Document{}, err
	}
	return loadAndProject(ctx, tx, owner, mutation.ID)
}

func (s *PostgresStore) applyUpdate(ctx context.Context, tx transaction, owner brain.Principal, mutation Mutation) (Document, error) {
	stored, err := loadOwned(ctx, tx, owner, mutation.ID, true, false)
	if err != nil {
		return Document{}, err
	}
	if stored.tombstoned {
		return Document{}, brain.ErrNotFound
	}
	if stored.document.DeletedAt != nil {
		return Document{}, brain.ErrConflict
	}
	if stored.document.Version != mutation.ExpectedVersion {
		return Document{}, brain.ErrConflict
	}
	if stored.document.Version == maxDocumentVersion {
		return Document{}, brain.ErrConflict
	}
	if exists, err := layerExists(ctx, tx, stored.brainID, mutation.LayerID); err != nil {
		return Document{}, err
	} else if !exists {
		return Document{}, brain.ErrNotFound
	}
	command, err := tx.Exec(ctx, `
		UPDATE brain_documents.documents
		SET layer_id = $1::uuid,
		    title = $2,
		    markdown = $3,
		    version = version + 1,
		    updated_at = clock_timestamp()
		WHERE document_id = $4::uuid
		  AND brain_id = $5::uuid
		  AND version = $6
		  AND deleted_at IS NULL
		  AND tombstoned_at IS NULL`, mutation.LayerID, mutation.Title, mutation.Markdown, mutation.ID, stored.brainID, mutation.ExpectedVersion)
	if err != nil {
		return Document{}, mapPostgresError(err)
	}
	if command.RowsAffected() != 1 {
		return Document{}, brain.ErrConflict
	}
	if err := replaceLinks(ctx, tx, stored.brainID, mutation.ID, mutation.Targets); err != nil {
		return Document{}, err
	}
	return loadAndProject(ctx, tx, owner, mutation.ID)
}

func (s *PostgresStore) applyTrash(ctx context.Context, tx transaction, owner brain.Principal, mutation Mutation) (Document, error) {
	stored, err := loadOwned(ctx, tx, owner, mutation.ID, true, false)
	if err != nil {
		return Document{}, err
	}
	if stored.tombstoned {
		return Document{}, brain.ErrNotFound
	}
	if stored.document.Version != mutation.ExpectedVersion {
		return Document{}, brain.ErrConflict
	}
	if stored.document.DeletedAt != nil {
		return loadWithLinks(ctx, tx, stored)
	}
	if stored.document.Version == maxDocumentVersion {
		return Document{}, brain.ErrConflict
	}
	command, err := tx.Exec(ctx, `
		UPDATE brain_documents.documents
		SET deleted_at = statement_timestamp(),
		    expires_at = statement_timestamp() + interval '240 hours',
		    version = version + 1,
		    updated_at = statement_timestamp()
		WHERE document_id = $1::uuid
		  AND brain_id = $2::uuid
		  AND version = $3
		  AND deleted_at IS NULL
		  AND tombstoned_at IS NULL`, mutation.ID, stored.brainID, mutation.ExpectedVersion)
	if err != nil {
		return Document{}, mapPostgresError(err)
	}
	if command.RowsAffected() != 1 {
		return Document{}, brain.ErrConflict
	}
	return loadAndProject(ctx, tx, owner, mutation.ID)
}

func (s *PostgresStore) applyRestore(ctx context.Context, tx transaction, owner brain.Principal, mutation Mutation) (Document, error) {
	stored, err := loadOwned(ctx, tx, owner, mutation.ID, true, false)
	if err != nil {
		return Document{}, err
	}
	if stored.tombstoned {
		return Document{}, brain.ErrNotFound
	}
	if stored.document.Version != mutation.ExpectedVersion {
		return Document{}, brain.ErrConflict
	}
	if stored.document.DeletedAt == nil {
		return Document{}, brain.ErrConflict
	}
	if stored.document.Version == maxDocumentVersion {
		return Document{}, brain.ErrConflict
	}
	var unexpired bool
	if err := tx.QueryRow(ctx, `
		SELECT expires_at > clock_timestamp()
		FROM brain_documents.documents
		WHERE document_id = $1::uuid AND brain_id = $2::uuid`, mutation.ID, stored.brainID).Scan(&unexpired); err != nil {
		return Document{}, mapPostgresError(err)
	}
	if !unexpired {
		return Document{}, brain.ErrNotFound
	}
	destination := stored.document.LayerID
	if mutation.LayerID != "" {
		destination = mutation.LayerID
	}
	if exists, err := layerExists(ctx, tx, stored.brainID, destination); err != nil {
		return Document{}, err
	} else if !exists {
		return Document{}, brain.ErrNotFound
	}
	command, err := tx.Exec(ctx, `
		UPDATE brain_documents.documents
		SET layer_id = $1::uuid,
		    deleted_at = NULL,
		    expires_at = NULL,
		    version = version + 1,
		    updated_at = clock_timestamp()
		WHERE document_id = $2::uuid
		  AND brain_id = $3::uuid
		  AND version = $4
		  AND deleted_at IS NOT NULL
		  AND tombstoned_at IS NULL
		  AND expires_at > clock_timestamp()`, destination, mutation.ID, stored.brainID, mutation.ExpectedVersion)
	if err != nil {
		return Document{}, mapPostgresError(err)
	}
	if command.RowsAffected() != 1 {
		return Document{}, brain.ErrNotFound
	}
	return loadAndProject(ctx, tx, owner, mutation.ID)
}

func (s *PostgresStore) ready() bool {
	return s != nil && s.db != nil
}

func txExecInsert(ctx context.Context, tx transaction, mutation Mutation, brainID string) error {
	command, err := tx.Exec(ctx, `
		INSERT INTO brain_documents.documents
		    (document_id, brain_id, layer_id, title, markdown, version, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, 1, clock_timestamp(), clock_timestamp())`, mutation.ID, brainID, mutation.LayerID, mutation.Title, mutation.Markdown)
	if err != nil {
		return mapPostgresError(err)
	}
	if command.RowsAffected() != 1 {
		return brain.ErrInternal
	}
	return nil
}

func loadAndProject(ctx context.Context, tx transaction, owner brain.Principal, id string) (Document, error) {
	stored, err := loadOwned(ctx, tx, owner, id, true, false)
	if err != nil {
		return Document{}, err
	}
	if stored.tombstoned {
		return Document{}, brain.ErrNotFound
	}
	return loadWithLinks(ctx, tx, stored)
}

func loadWithLinks(ctx context.Context, tx transaction, stored storedDocument) (Document, error) {
	if stored.tombstoned {
		return Document{}, brain.ErrNotFound
	}
	if err := loadLinks(ctx, tx, &stored.document); err != nil {
		return Document{}, err
	}
	return stored.document, nil
}

func loadOwned(ctx context.Context, tx transaction, owner brain.Principal, id string, forUpdate, visibleOnly bool) (storedDocument, error) {
	lock := ""
	if forUpdate {
		lock = " FOR NO KEY UPDATE OF d"
	}
	visibility := ""
	if visibleOnly {
		visibility = "\n  AND d.tombstoned_at IS NULL\n  AND (d.deleted_at IS NULL OR d.expires_at > clock_timestamp())"
	}
	var stored storedDocument
	var deletedAt, expiresAt, tombstonedAt *time.Time
	err := tx.QueryRow(ctx, `
		SELECT d.document_id::text,
		       d.brain_id::text,
		       COALESCE(d.layer_id::text, ''),
		       d.title,
		       d.markdown,
		       d.version,
		       d.created_at,
		       d.updated_at,
		       d.deleted_at,
		       d.expires_at,
		       d.tombstoned_at
		FROM brain_documents.documents AS d
		JOIN brain_core.brains AS b ON b.brain_id = d.brain_id
		WHERE b.owner_subject = $1
		  AND d.document_id = $2::uuid`+visibility+lock, string(owner), id).Scan(
		&stored.document.ID,
		&stored.brainID,
		&stored.document.LayerID,
		&stored.document.Title,
		&stored.document.Markdown,
		&stored.document.Version,
		&stored.document.CreatedAt,
		&stored.document.UpdatedAt,
		&deletedAt,
		&expiresAt,
		&tombstonedAt,
	)
	if err != nil {
		return storedDocument{}, mapPostgresError(err)
	}
	stored.document.DeletedAt = deletedAt
	stored.document.ExpiresAt = expiresAt
	stored.tombstoned = tombstonedAt != nil
	return stored, nil
}

func loadLinks(ctx context.Context, tx transaction, document *Document) error {
	rows, err := tx.Query(ctx, `
		SELECT t.target_document_id::text,
		       CASE
			   WHEN target.document_id IS NULL OR target.tombstoned_at IS NOT NULL
			    OR (target.deleted_at IS NOT NULL
			        AND (target.expires_at IS NULL OR target.expires_at <= statement_timestamp())) THEN ''
			   ELSE target.title
		       END,
		       CASE
			   WHEN target.document_id IS NULL OR target.tombstoned_at IS NOT NULL
			    OR (target.deleted_at IS NOT NULL
			        AND (target.expires_at IS NULL OR target.expires_at <= statement_timestamp())) THEN ''
			   ELSE COALESCE(target.layer_id::text, '')
		       END,
		       CASE
			   WHEN target.document_id IS NULL THEN 'missing'
			   WHEN target.tombstoned_at IS NOT NULL THEN 'missing'
			   WHEN target.deleted_at IS NOT NULL
			    AND (target.expires_at IS NULL OR target.expires_at <= statement_timestamp()) THEN 'missing'
			   WHEN target.deleted_at IS NOT NULL THEN 'trash'
			   ELSE 'available'
		       END
		FROM brain_documents.document_links AS t
		LEFT JOIN brain_documents.documents AS target
		  ON target.brain_id = t.brain_id
		 AND target.document_id = t.target_document_id
		WHERE t.brain_id = (SELECT brain_id FROM brain_documents.documents WHERE document_id = $1::uuid)
		  AND t.source_document_id = $1::uuid
		ORDER BY t.position ASC`, document.ID)
	if err != nil {
		return mapPostgresError(err)
	}
	defer rows.Close()
	document.Links = make([]Link, 0)
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.TargetID, &link.Title, &link.LayerID, &link.Status); err != nil {
			return brain.ErrInternal
		}
		document.Links = append(document.Links, link)
	}
	if err := rows.Err(); err != nil {
		return brain.ErrInternal
	}
	return nil
}

func replaceLinks(ctx context.Context, tx transaction, brainID, sourceID string, targets []string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM brain_documents.document_links
		WHERE brain_id = $1::uuid AND source_document_id = $2::uuid`, brainID, sourceID); err != nil {
		return mapPostgresError(err)
	}
	uniqueTargets := uniqueUUIDs(targets)
	for position, targetID := range uniqueTargets {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM brain_documents.documents
				WHERE brain_id = $1::uuid AND document_id = $2::uuid
			)`, brainID, targetID).Scan(&exists); err != nil {
			return mapPostgresError(err)
		}
		if !exists {
			return brain.ErrNotFound
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO brain_documents.document_links
			    (brain_id, source_document_id, target_document_id, position)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4)`, brainID, sourceID, targetID, position); err != nil {
			return mapPostgresError(err)
		}
	}
	return nil
}

func purgeExpired(ctx context.Context, tx transaction, owner brain.Principal) error {
	if _, err := tx.Exec(ctx, `
		WITH owned AS (
			SELECT brain_id FROM brain_core.brains WHERE owner_subject = $1
		), expired AS (
			SELECT d.brain_id, d.document_id
			FROM brain_documents.documents AS d
			JOIN owned AS b ON b.brain_id = d.brain_id
			WHERE d.deleted_at IS NOT NULL
			  AND d.expires_at IS NOT NULL
			  AND d.expires_at <= clock_timestamp()
			  AND d.tombstoned_at IS NULL
			ORDER BY d.document_id
			FOR NO KEY UPDATE OF d
		), removed_links AS (
			DELETE FROM brain_documents.document_links AS t
			USING expired AS e
			WHERE t.brain_id = e.brain_id
			  AND t.source_document_id = e.document_id
			RETURNING t.source_document_id
		), purged AS (
			UPDATE brain_documents.documents AS d
			SET title = '',
			    markdown = '',
			    layer_id = NULL,
			    expires_at = NULL,
			    tombstoned_at = statement_timestamp(),
			    updated_at = statement_timestamp()
			FROM expired AS e
			WHERE d.brain_id = e.brain_id
			  AND d.document_id = e.document_id
			  AND d.deleted_at IS NOT NULL
			  AND d.expires_at IS NOT NULL
			  AND d.expires_at <= clock_timestamp()
			  AND d.tombstoned_at IS NULL
			RETURNING d.document_id
		)
		SELECT count(*) FROM purged`, string(owner)); err != nil {
		return mapPostgresError(err)
	}
	return nil
}

func brainIDForOwner(ctx context.Context, tx transaction, owner brain.Principal) (string, error) {
	var id string
	if err := tx.QueryRow(ctx, `
		SELECT brain_id::text
		FROM brain_core.brains
		WHERE owner_subject = $1`, string(owner)).Scan(&id); err != nil {
		return "", mapPostgresError(err)
	}
	if err := validateUUID(id); err != nil {
		return "", brain.ErrInternal
	}
	return id, nil
}

func layerExists(ctx context.Context, tx transaction, brainID, layerID string) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM brain_core.nodes
			WHERE brain_id = $1::uuid AND node_id = $2::uuid AND kind = 'layer'
		)`, brainID, layerID).Scan(&exists); err != nil {
		return false, mapPostgresError(err)
	}
	return exists, nil
}

func documentExists(ctx context.Context, tx transaction, id string) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM brain_documents.documents WHERE document_id = $1::uuid
		)`, id).Scan(&exists); err != nil {
		return false, mapPostgresError(err)
	}
	return exists, nil
}

func validateMutation(mutation Mutation) error {
	switch mutation.Kind {
	case KindCreate:
		if mutation.ExpectedVersion != 0 || mutation.LayerID == "" {
			return ErrInvalidDocument
		}
	case KindUpdate:
		if mutation.LayerID == "" || mutation.ExpectedVersion <= 0 {
			return ErrInvalidDocument
		}
	case KindTrash, KindRestore:
		if mutation.ExpectedVersion <= 0 {
			return ErrInvalidDocument
		}
	default:
		return ErrInvalidDocument
	}
	if err := validateUUID(mutation.ID); err != nil {
		return ErrInvalidDocument
	}
	if mutation.LayerID != "" {
		if err := validateUUID(mutation.LayerID); err != nil {
			return ErrInvalidDocument
		}
	}
	if mutation.Kind == KindCreate || mutation.Kind == KindUpdate {
		if err := validateText(mutation.Title, true); err != nil {
			return err
		}
		if err := validateText(mutation.Markdown, false); err != nil {
			return err
		}
	}
	for _, target := range mutation.Targets {
		if err := validateUUID(target); err != nil {
			return ErrInvalidDocument
		}
	}
	return nil
}

func normalizeStoreQuery(query Query) (Query, error) {
	if query.Scope == "" {
		query.Scope = "direct"
	}
	if query.State == "" {
		query.State = "active"
	}
	if query.Limit == 0 {
		query.Limit = 50
	}
	if query.LayerID == "" || (query.Scope != "direct" && query.Scope != "subtree") || (query.State != "active" && query.State != "trash") || query.Limit < 1 || query.Limit > 100 {
		return Query{}, ErrInvalidQuery
	}
	if err := validateUUID(query.LayerID); err != nil {
		return Query{}, ErrInvalidQuery
	}
	if query.After != "" {
		if err := validateUUID(query.After); err != nil {
			return Query{}, ErrInvalidQuery
		}
	}
	return query, nil
}

func validateText(value string, title bool) error {
	if !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
		return ErrInvalidDocument
	}
	if title {
		if len([]rune(value)) < 1 || len([]rune(value)) > 255 {
			return ErrInvalidDocument
		}
		if strings.TrimSpace(value) != value {
			return ErrInvalidDocument
		}
		for _, r := range value {
			if unicode.IsControl(r) {
				return ErrInvalidDocument
			}
		}
		return nil
	}
	if len([]byte(value)) > 1<<20 {
		return ErrInvalidDocument
	}
	return nil
}

func validateUUID(value string) error {
	if len(value) != 36 || strings.ToLower(value) != value {
		return errors.New("invalid canonical UUID")
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return errors.New("invalid canonical UUID")
			}
			continue
		}
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return errors.New("invalid canonical UUID")
		}
	}
	return nil
}

func uniqueUUIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return brain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return brain.ErrConflict
		case "23503":
			return brain.ErrNotFound
		case "23514", "22P02":
			return ErrInvalidDocument
		}
	}
	return brain.ErrInternal
}

var _ Store = (*PostgresStore)(nil)

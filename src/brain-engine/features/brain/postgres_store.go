package brain

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the product-persistence adapter for Brain Core. The
// application model and Store port remain independent of pgx types.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// OpenPostgresStore opens and verifies a pool. It never applies migrations.
func OpenPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(dsn) == "" {
		return nil, ErrInternal
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, ErrInternal
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, ErrInternal
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, ErrInternal
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *PostgresStore) EnsureBrain(ctx context.Context, owner Principal, candidate Brain) (Brain, error) {
	if !validPrincipal(owner) {
		return Brain{}, ErrUnauthenticated
	}
	if !s.ready() || candidate.Owner != owner || !validBrain(candidate, owner) {
		return Brain{}, ErrInternal
	}
	brainID, err := uuidValue(string(candidate.ID))
	if err != nil {
		return Brain{}, ErrInternal
	}
	rootID, err := uuidValue(string(candidate.Root.ID))
	if err != nil {
		return Brain{}, ErrInternal
	}
	ctx = normalizeContext(ctx)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Brain{}, ErrInternal
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO brain_core.brains (brain_id, owner_subject)
		VALUES ($1, $2)
		ON CONFLICT (owner_subject) DO NOTHING`, brainID, string(owner)); err != nil {
		return Brain{}, mapPostgresError(err)
	}

	var storedBrain pgtype.UUID
	if err := tx.QueryRow(ctx, `
		SELECT brain_id
		FROM brain_core.brains
		WHERE owner_subject = $1`, string(owner)).Scan(&storedBrain); err != nil {
		return Brain{}, mapPostgresError(err)
	}
	if !storedBrain.Valid {
		return Brain{}, ErrInternal
	}

	var storedRoot pgtype.UUID
	rootErr := tx.QueryRow(ctx, `
		SELECT node_id
		FROM brain_core.nodes
		WHERE brain_id = $1 AND kind = 'root'`, storedBrain).Scan(&storedRoot)
	if errors.Is(rootErr, pgx.ErrNoRows) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO brain_core.nodes (brain_id, node_id, kind, parent_id, name, version)
			VALUES ($1, $2, 'root', NULL, NULL, 1)
			ON CONFLICT DO NOTHING`, storedBrain, rootID); err != nil {
			return Brain{}, mapPostgresError(err)
		}
		rootErr = tx.QueryRow(ctx, `
			SELECT node_id
			FROM brain_core.nodes
			WHERE brain_id = $1 AND kind = 'root'`, storedBrain).Scan(&storedRoot)
	}
	if rootErr != nil {
		return Brain{}, mapPostgresError(rootErr)
	}
	if !storedRoot.Valid {
		return Brain{}, ErrInternal
	}
	if err := tx.Commit(ctx); err != nil {
		return Brain{}, ErrInternal
	}

	resolvedBrain, err := brainIDString(storedBrain)
	if err != nil {
		return Brain{}, ErrInternal
	}
	resolvedRoot, err := rootIDString(storedRoot)
	if err != nil {
		return Brain{}, ErrInternal
	}
	return Brain{
		ID:    BrainID(resolvedBrain),
		Owner: owner,
		Root: Root{
			ID:      RootID(resolvedRoot),
			BrainID: BrainID(resolvedBrain),
		},
	}, nil
}

func (s *PostgresStore) Observe(ctx context.Context, owner Principal) (Hierarchy, error) {
	if !validPrincipal(owner) {
		return Hierarchy{}, ErrUnauthenticated
	}
	if !s.ready() {
		return Hierarchy{}, ErrInternal
	}
	ctx = normalizeContext(ctx)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Hierarchy{}, ErrInternal
	}
	defer tx.Rollback(ctx)

	var brainUUID pgtype.UUID
	if err := tx.QueryRow(ctx, `
		SELECT brain_id
		FROM brain_core.brains
		WHERE owner_subject = $1`, string(owner)).Scan(&brainUUID); err != nil {
		return Hierarchy{}, mapPostgresError(err)
	}
	var rootUUID pgtype.UUID
	if err := tx.QueryRow(ctx, `
		SELECT node_id
		FROM brain_core.nodes
		WHERE brain_id = $1 AND kind = 'root'`, brainUUID).Scan(&rootUUID); err != nil {
		return Hierarchy{}, mapPostgresError(err)
	}
	brainID, err := brainIDString(brainUUID)
	if err != nil {
		return Hierarchy{}, ErrInternal
	}
	rootID, err := rootIDString(rootUUID)
	if err != nil {
		return Hierarchy{}, ErrInternal
	}

	rows, err := tx.Query(ctx, `
		SELECT n.node_id, n.parent_id, n.name, n.version, p.kind
		FROM brain_core.nodes AS n
		JOIN brain_core.nodes AS p
		  ON p.brain_id = n.brain_id AND p.node_id = n.parent_id
		WHERE n.brain_id = $1 AND n.kind = 'layer'
		ORDER BY n.node_id`, brainUUID)
	if err != nil {
		return Hierarchy{}, mapPostgresError(err)
	}
	defer rows.Close()

	layers := make([]Layer, 0)
	for rows.Next() {
		var layerUUID, parentUUID pgtype.UUID
		var name, parentKind string
		var version int64
		if err := rows.Scan(&layerUUID, &parentUUID, &name, &version, &parentKind); err != nil {
			return Hierarchy{}, ErrInternal
		}
		layer, err := scanLayer(layerUUID, parentUUID, parentKind, name, version, BrainID(brainID))
		if err != nil {
			return Hierarchy{}, err
		}
		layers = append(layers, layer)
	}
	if err := rows.Err(); err != nil {
		return Hierarchy{}, ErrInternal
	}
	if err := tx.Commit(ctx); err != nil {
		return Hierarchy{}, ErrInternal
	}

	brain := Brain{
		ID:    BrainID(brainID),
		Owner: owner,
		Root: Root{
			ID:      RootID(rootID),
			BrainID: BrainID(brainID),
		},
	}
	return Hierarchy{Brain: brain, Root: brain.Root, Layers: layers}, nil
}

func (s *PostgresStore) InsertLayer(ctx context.Context, owner Principal, layer Layer) (Layer, error) {
	if !validPrincipal(owner) {
		return Layer{}, ErrUnauthenticated
	}
	if !s.ready() {
		return Layer{}, ErrInternal
	}
	if !validLayer(layer, layer.BrainID) || layer.Version != 1 {
		if _, ok := normalizeLayerName(layer.Name); !ok {
			return Layer{}, ErrInvalidName
		}
		return Layer{}, ErrInternal
	}
	normalizedName, ok := normalizeLayerName(layer.Name)
	if !ok || normalizedName != layer.Name {
		return Layer{}, ErrInvalidName
	}
	brainUUID, err := uuidValue(string(layer.BrainID))
	if err != nil {
		return Layer{}, ErrInternal
	}
	layerUUID, err := uuidValue(string(layer.ID))
	if err != nil {
		return Layer{}, ErrInternal
	}
	parentUUID, err := uuidValue(layer.Parent.ID)
	if err != nil {
		return Layer{}, ErrNotFound
	}
	parentKind := string(layer.Parent.Kind)
	ctx = normalizeContext(ctx)
	command, err := s.pool.Exec(ctx, `
		INSERT INTO brain_core.nodes (brain_id, node_id, kind, parent_id, name, version)
		SELECT $1, $2, 'layer', $3, $4, 1
		WHERE EXISTS (
			SELECT 1
			FROM brain_core.brains
			WHERE brain_id = $1 AND owner_subject = $5
		)
		AND EXISTS (
			SELECT 1
			FROM brain_core.nodes
			WHERE brain_id = $1 AND node_id = $3 AND kind = $6
		)`, brainUUID, layerUUID, parentUUID, normalizedName, string(owner), parentKind)
	if err != nil {
		return Layer{}, mapPostgresError(err)
	}
	if command.RowsAffected() != 1 {
		return Layer{}, ErrNotFound
	}
	return layer, nil
}

func (s *PostgresStore) CompareAndSwapLayerName(ctx context.Context, owner Principal, brainID BrainID, layerID LayerID, expectedVersion LayerVersion, name string) (Layer, error) {
	if !validPrincipal(owner) {
		return Layer{}, ErrUnauthenticated
	}
	if !s.ready() {
		return Layer{}, ErrInternal
	}
	normalizedName, ok := normalizeLayerName(name)
	if !ok {
		return Layer{}, ErrInvalidName
	}
	if normalizedName != name {
		return Layer{}, ErrInvalidName
	}
	if layerID == "" || expectedVersion > LayerVersion(1<<63-1) {
		return Layer{}, ErrConflict
	}
	brainUUID, err := uuidValue(string(brainID))
	if err != nil {
		return Layer{}, ErrNotFound
	}
	layerUUID, err := uuidValue(string(layerID))
	if err != nil {
		return Layer{}, ErrNotFound
	}
	ctx = normalizeContext(ctx)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Layer{}, ErrInternal
	}
	defer tx.Rollback(ctx)

	var parentUUID pgtype.UUID
	var version int64
	command, err := tx.Exec(ctx, `
		UPDATE brain_core.nodes AS n
		SET name = $1, version = n.version + 1, updated_at = now()
		FROM brain_core.brains AS b
		WHERE b.brain_id = n.brain_id
		  AND b.owner_subject = $2
		  AND n.brain_id = $3
		  AND n.node_id = $4
		  AND n.kind = 'layer'
		  AND n.version = $5`, normalizedName, string(owner), brainUUID, layerUUID, int64(expectedVersion))
	if err != nil {
		return Layer{}, mapPostgresError(err)
	}
	if command.RowsAffected() != 1 {
		var exists bool
		existenceErr := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM brain_core.nodes AS n
				JOIN brain_core.brains AS b ON b.brain_id = n.brain_id
				WHERE b.owner_subject = $1
				  AND n.brain_id = $2
				  AND n.node_id = $3
				  AND n.kind = 'layer'
			)`, string(owner), brainUUID, layerUUID).Scan(&exists)
		if existenceErr != nil {
			return Layer{}, mapPostgresError(existenceErr)
		}
		if exists {
			return Layer{}, ErrConflict
		}
		return Layer{}, ErrNotFound
	}

	if err := tx.QueryRow(ctx, `
		SELECT parent_id, version
		FROM brain_core.nodes
		WHERE brain_id = $1 AND node_id = $2 AND kind = 'layer'`, brainUUID, layerUUID).Scan(&parentUUID, &version); err != nil {
		return Layer{}, mapPostgresError(err)
	}
	var parentKind string
	if err := tx.QueryRow(ctx, `
		SELECT kind
		FROM brain_core.nodes
		WHERE brain_id = $1 AND node_id = $2`, brainUUID, parentUUID).Scan(&parentKind); err != nil {
		return Layer{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Layer{}, ErrInternal
	}
	updated, err := scanLayer(layerUUID, parentUUID, parentKind, normalizedName, version, brainID)
	if err != nil {
		return Layer{}, err
	}
	return updated, nil
}

func (s *PostgresStore) ready() bool {
	return s != nil && s.pool != nil
}

func uuidValue(value string) (pgtype.UUID, error) {
	var result pgtype.UUID
	if err := result.Scan(strings.TrimSpace(value)); err != nil {
		return pgtype.UUID{}, err
	}
	if !result.Valid {
		return pgtype.UUID{}, errors.New("invalid UUID")
	}
	return result, nil
}

func brainIDString(value pgtype.UUID) (string, error) {
	if !value.Valid {
		return "", errors.New("invalid brain UUID")
	}
	return value.String(), nil
}

func rootIDString(value pgtype.UUID) (string, error) {
	if !value.Valid {
		return "", errors.New("invalid root UUID")
	}
	return value.String(), nil
}

func scanLayer(layerUUID, parentUUID pgtype.UUID, parentKind, name string, version int64, brainID BrainID) (Layer, error) {
	if !layerUUID.Valid || !parentUUID.Valid || version <= 0 {
		return Layer{}, ErrInternal
	}
	var kind NodeKind
	switch parentKind {
	case string(NodeRoot):
		kind = NodeRoot
	case string(NodeLayer):
		kind = NodeLayer
	default:
		return Layer{}, ErrInternal
	}
	layerID, err := brainIDString(layerUUID)
	if err != nil {
		return Layer{}, ErrInternal
	}
	parentID, err := rootIDString(parentUUID)
	if err != nil {
		return Layer{}, ErrInternal
	}
	return Layer{
		ID:      LayerID(layerID),
		BrainID: brainID,
		Parent:  NodeRef{Kind: kind, ID: parentID},
		Name:    name,
		Version: LayerVersion(version),
	}, nil
}

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrNotFound
		case "23514":
			return ErrInvalidName
		}
	}
	return ErrInternal
}

var _ Store = (*PostgresStore)(nil)

// Package brainproof runs a develop-only persistence behavior check through
// the real deterministic Brain Engine and Store boundary. Synthetic rows are
// durable; this package never performs cleanup.
package brainproof

import (
	"context"
	"errors"
	"net"
	"net/url"
	"regexp"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/jackc/pgx/v5"
)

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

var (
	ErrInvalidInput     = errors.New("Brain Core verification input is invalid")
	ErrIdentityCheck    = errors.New("Brain Core database identity check failed")
	ErrTargetMismatch   = errors.New("Brain Core verifier target mismatch")
	ErrStoreUnavailable = errors.New("Brain Core store is unavailable")
	ErrBehavior         = errors.New("Brain Core behavior verification failed")
)

// Store is the real Brain Core persistence port plus its pool lifecycle.
type Store interface {
	brain.Store
	Close()
}

// CurrentDatabaseProbe must query current_database() over a short-lived
// connection before StoreFactory opens the pool used for synthetic writes.
type CurrentDatabaseProbe func(context.Context, config.TargetSettings) (string, error)

// StoreFactory opens the application Store only after database identity passes.
type StoreFactory func(context.Context, config.TargetSettings) (Store, error)

// Dependencies allow local tests to inject the identity and Store boundaries.
type Dependencies struct {
	CurrentDatabase CurrentDatabaseProbe
	OpenStore       StoreFactory
}

// PostgresDependencies provides the production connection boundary. It reads
// the database identity on a short-lived pgx connection before opening the
// PostgresStore pool; no DSN is accepted from command-line arguments.
func PostgresDependencies() Dependencies {
	return Dependencies{
		CurrentDatabase: probeCurrentDatabase,
		OpenStore:       openPostgresStore,
	}
}

type Options struct {
	OperatorID           string
	AuthorizationRef     string
	RunID                string
	AllowSyntheticWrites bool
}

type Checks struct {
	BootstrapIdempotent   bool `json:"bootstrap_idempotent"`
	OwnObserve            bool `json:"own_observe"`
	UnauthorizedIsolation bool `json:"unauthorized_isolation"`
	NestedCreate          bool `json:"nested_create"`
	RenameIdentityStable  bool `json:"rename_identity_stable"`
	StaleRenameConflict   bool `json:"stale_rename_conflict"`
}

// Result reports only the selected target and the outcome of each check. The
// Brain, root, layer, owner and database identifiers are intentionally omitted.
type Result struct {
	Status                 string `json:"status"`
	Target                 string `json:"target"`
	RunID                  string `json:"run_id"`
	OperatorID             string `json:"operator_id"`
	AuthorizationReference string `json:"authorization_reference"`
	SyntheticRowsDurable   bool   `json:"synthetic_rows_durable"`
	Cleanup                string `json:"cleanup"`
	Checks                 Checks `json:"checks"`
}

// Verify refuses every target except mindforge_develop, checks the connected
// database identity before opening the application Store, then exercises the
// actual Brain Engine against a reserved synthetic owner prefix. Writes remain
// durable, and no cleanup is attempted.
func Verify(ctx context.Context, settings config.TargetSettings, options Options, dependencies Dependencies) (Result, error) {
	if settings.Target != config.TargetDevelop || settings.DatabaseName != string(config.TargetDevelop) {
		return Result{}, ErrTargetMismatch
	}
	if !options.AllowSyntheticWrites || !runIDPattern.MatchString(options.RunID) {
		return Result{}, ErrInvalidInput
	}
	if _, err := command.NewOperationRequest(config.TargetDevelop, options.OperatorID, options.AuthorizationRef); err != nil {
		return Result{}, ErrInvalidInput
	}
	if dependencies.CurrentDatabase == nil || dependencies.OpenStore == nil {
		return Result{}, ErrInvalidInput
	}
	ctx = normalizeContext(ctx)

	currentDatabase, err := dependencies.CurrentDatabase(ctx, settings)
	if err != nil {
		return Result{}, ErrIdentityCheck
	}
	if currentDatabase != string(config.TargetDevelop) {
		return Result{}, ErrTargetMismatch
	}
	store, err := dependencies.OpenStore(ctx, settings)
	if err != nil || store == nil {
		return Result{}, ErrStoreUnavailable
	}
	defer store.Close()

	engine := brain.NewEngine(store, brain.NewUUIDSource())
	owner := brain.Principal("verification/brain-core/" + options.RunID)
	checks, err := verifyBehavior(ctx, engine, owner, options.RunID)
	if err != nil {
		return Result{}, ErrBehavior
	}
	return Result{
		Status:                 "succeeded",
		Target:                 string(config.TargetDevelop),
		RunID:                  options.RunID,
		OperatorID:             options.OperatorID,
		AuthorizationReference: options.AuthorizationRef,
		SyntheticRowsDurable:   true,
		Cleanup:                "not_performed",
		Checks:                 checks,
	}, nil
}

func verifyBehavior(ctx context.Context, engine *brain.Engine, owner brain.Principal, runID string) (Checks, error) {
	var checks Checks
	first, err := engine.Bootstrap(ctx, owner)
	if err != nil {
		return checks, err
	}
	second, err := engine.Bootstrap(ctx, owner)
	if err != nil || first.ID != second.ID || first.Root.ID != second.Root.ID {
		return checks, ErrBehavior
	}
	checks.BootstrapIdempotent = true

	observed, err := engine.Observe(ctx, owner)
	if err != nil || observed.Brain.ID != first.ID || observed.Root.ID != first.Root.ID {
		return checks, ErrBehavior
	}
	checks.OwnObserve = true

	otherOwner := brain.Principal("verification/brain-core/" + runID + "/unauthorized")
	if _, err := engine.Observe(ctx, otherOwner); !errors.Is(err, brain.ErrNotFound) {
		return checks, ErrBehavior
	}
	checks.UnauthorizedIsolation = true

	rootParent := brain.RootRef(first.Root.ID)
	outerName := "verify-" + runID + "-outer"
	outer, err := ensureLayer(ctx, engine, owner, rootParent, outerName)
	if err != nil {
		return checks, err
	}
	if _, err := engine.RenameLayer(ctx, otherOwner, outer.ID, outer.Version, outer.Name+"-unauthorized"); !errors.Is(err, brain.ErrNotFound) {
		return checks, ErrBehavior
	}
	nestedName := "verify-" + runID + "-nested"
	nested, err := ensureLayer(ctx, engine, owner, brain.LayerRef(outer.ID), nestedName)
	if err != nil || nested.Parent != brain.LayerRef(outer.ID) || nested.BrainID != first.ID {
		return checks, ErrBehavior
	}
	checks.NestedCreate = true

	renameLayer, err := ensureRenameLayer(ctx, engine, owner, rootParent, runID)
	if err != nil {
		return checks, err
	}
	previous := renameLayer
	targetName := "verify-" + runID + "-rename-a"
	if previous.Name == targetName {
		targetName = "verify-" + runID + "-rename-b"
	}
	updated, err := engine.RenameLayer(ctx, owner, previous.ID, previous.Version, targetName)
	if err != nil || updated.ID != previous.ID || updated.Parent != previous.Parent || updated.BrainID != previous.BrainID || updated.Name != targetName || updated.Version != previous.Version+1 {
		return checks, ErrBehavior
	}
	checks.RenameIdentityStable = true

	if _, err := engine.RenameLayer(ctx, owner, previous.ID, previous.Version, targetName+"-stale"); !errors.Is(err, brain.ErrConflict) {
		return checks, ErrBehavior
	}
	checks.StaleRenameConflict = true
	return checks, nil
}

func ensureLayer(ctx context.Context, engine *brain.Engine, owner brain.Principal, parent brain.NodeRef, name string) (brain.Layer, error) {
	hierarchy, err := engine.Observe(ctx, owner)
	if err != nil {
		return brain.Layer{}, err
	}
	for _, layer := range hierarchy.Layers {
		if layer.Parent == parent && layer.Name == name {
			return layer, nil
		}
	}
	return engine.CreateLayer(ctx, owner, parent, name)
}

func ensureRenameLayer(ctx context.Context, engine *brain.Engine, owner brain.Principal, parent brain.NodeRef, runID string) (brain.Layer, error) {
	nameA := "verify-" + runID + "-rename-a"
	nameB := "verify-" + runID + "-rename-b"
	hierarchy, err := engine.Observe(ctx, owner)
	if err != nil {
		return brain.Layer{}, err
	}
	for _, layer := range hierarchy.Layers {
		if layer.Parent == parent && (layer.Name == nameA || layer.Name == nameB) {
			return layer, nil
		}
	}
	return engine.CreateLayer(ctx, owner, parent, nameA)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func probeCurrentDatabase(ctx context.Context, settings config.TargetSettings) (string, error) {
	dsn, err := databaseDSN(settings)
	if err != nil {
		return "", err
	}
	connection, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return "", err
	}
	defer connection.Close(ctx)
	var database string
	if err := connection.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		return "", err
	}
	return database, nil
}

func openPostgresStore(ctx context.Context, settings config.TargetSettings) (Store, error) {
	dsn, err := databaseDSN(settings)
	if err != nil {
		return nil, err
	}
	store, err := brain.OpenPostgresStore(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func databaseDSN(settings config.TargetSettings) (string, error) {
	if settings.Target != config.TargetDevelop || settings.DatabaseName != string(config.TargetDevelop) || settings.DatabaseHost == "" || settings.DatabasePort == "" || settings.DatabaseUser == "" || settings.DatabasePassword == "" {
		return "", ErrInvalidInput
	}
	parsed := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(settings.DatabaseUser, settings.DatabasePassword),
		Host:   net.JoinHostPort(settings.DatabaseHost, settings.DatabasePort),
		Path:   "/" + settings.DatabaseName,
	}
	query := url.Values{}
	query.Set("sslmode", "disable")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

package brain

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestPostgresStoreImplementsStoreAndDoesNotAutoMigrate(t *testing.T) {
	var _ Store = (*PostgresStore)(nil)

	if _, err := OpenPostgresStore(context.Background(), ""); !errors.Is(err, ErrInternal) {
		t.Fatalf("empty DSN error = %v, want opaque internal error", err)
	}
}

func TestPostgresStoreIntegrationMatrix(t *testing.T) {
	engine, store := openDisposablePostgres(t)
	owner := Principal("b02-bootstrap-owner")

	start := make(chan struct{})
	brains := make(chan Brain, 2)
	errorsCh := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			brain, err := engine.Bootstrap(context.Background(), owner)
			if err != nil {
				errorsCh <- err
				return
			}
			brains <- brain
		}()
	}
	close(start)
	wg.Wait()
	close(brains)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent PostgreSQL bootstrap: %v", err)
	}
	var resolved []Brain
	for brain := range brains {
		resolved = append(resolved, brain)
	}
	if len(resolved) != 2 || resolved[0].ID != resolved[1].ID || resolved[0].Root.ID != resolved[1].Root.ID {
		t.Fatalf("concurrent bootstrap results diverged: %#v", resolved)
	}
	if got := queryCount(t, store, "SELECT count(*) FROM brain_core.brains WHERE owner_subject = $1", string(owner)); got != 1 {
		t.Fatalf("stored brains for owner = %d, want 1", got)
	}
	if got := queryCount(t, store, "SELECT count(*) FROM brain_core.nodes WHERE brain_id = $1::uuid AND kind = 'root'", string(resolved[0].ID)); got != 1 {
		t.Fatalf("stored roots for brain = %d, want 1", got)
	}

	hierarchy := mustObserve(t, engine, string(owner))
	if len(hierarchy.Layers) != 0 {
		t.Fatalf("initial PostgreSQL visible layers = %d, want 0", len(hierarchy.Layers))
	}

	createStart := make(chan struct{})
	created := make(chan Layer, 2)
	createErrors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-createStart
			layer, err := engine.CreateLayer(context.Background(), owner, RootRef(hierarchy.Root.ID), "same")
			if err != nil {
				createErrors <- err
				return
			}
			created <- layer
		}()
	}
	close(createStart)
	wg.Wait()
	close(created)
	close(createErrors)
	for err := range createErrors {
		t.Fatalf("concurrent PostgreSQL create: %v", err)
	}
	var layers []Layer
	for layer := range created {
		layers = append(layers, layer)
	}
	if len(layers) != 2 || layers[0].ID == layers[1].ID {
		t.Fatalf("concurrent creates = %#v, want two distinct layers", layers)
	}

	otherOwner := Principal("b02-other-owner")
	otherBrain := mustBootstrap(t, engine, string(otherOwner))
	foreign := mustCreate(t, engine, string(otherOwner), RootRef(otherBrain.Root.ID), "foreign-secret")
	for _, parent := range []NodeRef{RootRef(otherBrain.Root.ID), LayerRef(foreign.ID)} {
		_, err := engine.CreateLayer(context.Background(), owner, parent, "cross-brain")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("cross-Brain parent %#v error = %v, want not-found", parent, err)
		}
		if containsAny(errString(err), string(otherBrain.ID), string(otherBrain.Root.ID), string(foreign.ID), foreign.Name) {
			t.Fatalf("cross-Brain error leaked details: %v", err)
		}
	}
	if _, err := store.pool.Exec(context.Background(), `
		INSERT INTO brain_core.nodes (brain_id, node_id, kind, parent_id, name, version)
		VALUES ($1::uuid, $2::uuid, 'layer', $3::uuid, $4, 1)`,
		resolved[0].ID, foreign.ID, otherBrain.Root.ID, "direct-cross-brain"); err == nil {
		t.Fatal("database accepted a cross-Brain parent despite the composite foreign key")
	}
	if got := len(mustObserve(t, engine, string(owner)).Layers); got != 2 {
		t.Fatalf("owner layers after cross-Brain attempts = %d, want 2", got)
	}

	parent := layers[0]
	child := mustCreate(t, engine, string(owner), LayerRef(parent.ID), "child")
	sibling := mustCreate(t, engine, string(owner), RootRef(hierarchy.Root.ID), "same")
	renameStart := make(chan struct{})
	renameResults := make(chan Layer, 2)
	renameErrors := make(chan error, 2)
	for _, name := range []string{"first-rename", "second-rename"} {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-renameStart
			updated, err := engine.RenameLayer(context.Background(), owner, parent.ID, parent.Version, name)
			if err != nil {
				renameErrors <- err
				return
			}
			renameResults <- updated
		}()
	}
	close(renameStart)
	wg.Wait()
	close(renameResults)
	close(renameErrors)
	var winners []Layer
	conflicts := 0
	for result := range renameResults {
		winners = append(winners, result)
	}
	for err := range renameErrors {
		if errors.Is(err, ErrConflict) {
			conflicts++
			continue
		}
		t.Fatalf("concurrent PostgreSQL rename: %v", err)
	}
	if len(winners) != 1 || conflicts != 1 {
		t.Fatalf("rename winners=%d conflicts=%d, want one each", len(winners), conflicts)
	}

	after := mustObserve(t, engine, string(owner))
	gotParent, ok := findLayer(after.Layers, parent.ID)
	if !ok || gotParent.ID != parent.ID || gotParent.Parent != parent.Parent || gotParent.Version != winners[0].Version {
		t.Fatalf("parent identity/position/version changed unexpectedly: %#v", gotParent)
	}
	gotChild, ok := findLayer(after.Layers, child.ID)
	if !ok || gotChild.Parent != LayerRef(parent.ID) || gotChild.ID != child.ID {
		t.Fatalf("child continuity lost after rename: %#v", gotChild)
	}
	gotSibling, ok := findLayer(after.Layers, sibling.ID)
	if !ok || gotSibling.Name != sibling.Name || gotSibling.ID == gotParent.ID {
		t.Fatalf("sibling continuity lost after rename: %#v", gotSibling)
	}
	retry := mustRename(t, engine, string(owner), gotParent, "retry-rename")
	if retry.Name != "retry-rename" || retry.Version == gotParent.Version {
		t.Fatalf("rename retry = %#v, want version advance", retry)
	}
}

func openDisposablePostgres(t *testing.T) (*Engine, *PostgresStore) {
	t.Helper()
	dsn := disposableDSN(t)
	store, err := OpenPostgresStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open disposable PostgreSQL: %v", err)
	}
	t.Cleanup(store.Close)

	sqlBytes, err := os.ReadFile("migrations/0001_brain_core.sql")
	if err != nil {
		t.Fatalf("read migration for disposable test: %v", err)
	}
	ctx := context.Background()
	if _, err := store.pool.Exec(ctx, "DROP SCHEMA IF EXISTS brain_core CASCADE"); err != nil {
		t.Fatalf("reset disposable Brain Core schema: %v", err)
	}
	if _, err := store.pool.Exec(ctx, string(sqlBytes)); err != nil {
		t.Fatalf("apply Brain Core migration to disposable target: %v", err)
	}
	return NewEngine(store, NewUUIDSource()), store
}

func disposableDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("BRAIN_CORE_DATABASE_URL")
	if dsn == "" {
		t.Skip("BRAIN_CORE_DATABASE_URL is not set; integration evidence unavailable")
	}
	if os.Getenv("BRAIN_CORE_DATABASE_DISPOSABLE") != "1" {
		t.Skip("integration requires BRAIN_CORE_DATABASE_DISPOSABLE=1")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Skip("integration DSN is not a URL; target was not proven local")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Skip("integration target is not loopback; target was not proven local")
	}
	return dsn
}

func queryCount(t *testing.T, store *PostgresStore, query string, args ...any) int64 {
	t.Helper()
	var count int64
	if err := store.pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	return count
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func containsAny(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if fragment != "" && strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

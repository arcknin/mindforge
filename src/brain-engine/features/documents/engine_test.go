package documents

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
)

const (
	layerA = "20000000-0000-4000-8000-000000000001"
	layerB = "20000000-0000-4000-8000-000000000002"
	layerC = "20000000-0000-4000-8000-000000000003"
)

func TestEngineRejectsInvalidDocumentBeforeStore(t *testing.T) {
	engine := NewEngine(nil)
	_, err := engine.Create(context.Background(), brain.Principal("owner-1"), Mutation{
		ID:       "not-a-uuid",
		LayerID:  layerA,
		Title:    "Title",
		Markdown: "body",
	})
	if !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("Create() error = %v, want ErrInvalidDocument", err)
	}
}

func TestNilEngineReturnsInternalWithoutPanicking(t *testing.T) {
	var engine *Engine
	valid := Mutation{ID: docA, LayerID: layerA, Title: "Title"}
	owner := brain.Principal("owner-1")
	checks := []struct {
		name string
		err  error
	}{
		{name: "create", err: func() error { _, err := engine.Create(context.Background(), owner, valid); return err }()},
		{name: "update", err: func() error {
			_, err := engine.Update(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "Title", ExpectedVersion: 1})
			return err
		}()},
		{name: "trash", err: func() error { _, err := engine.Trash(context.Background(), owner, docA, 1); return err }()},
		{name: "restore", err: func() error { _, err := engine.Restore(context.Background(), owner, docA, 1, layerA); return err }()},
		{name: "get", err: func() error { _, err := engine.Get(context.Background(), owner, docA); return err }()},
		{name: "list", err: func() error { _, err := engine.List(context.Background(), owner, Query{LayerID: layerA}); return err }()},
	}
	for _, check := range checks {
		if !errors.Is(check.err, brain.ErrInternal) {
			t.Errorf("%s error = %v, want brain.ErrInternal", check.name, check.err)
		}
	}
}

func TestEngineNormalizesSnapshotAndDerivesTargets(t *testing.T) {
	owner := brain.Principal("owner-1")
	store := newMemoryStore(time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))
	store.addBrain(owner, "brain-1")
	store.addLayer(owner, layerA, "root-1")
	engine := NewEngine(store)

	if _, err := engine.Create(context.Background(), owner, Mutation{
		ID:      docA,
		LayerID: layerA,
		Title:   "Target",
	}); err != nil {
		t.Fatalf("create target: %v", err)
	}
	got, err := engine.Create(context.Background(), owner, Mutation{
		Kind:    KindTrash,
		ID:      docB,
		LayerID: layerA,
		Title:   "  Draft  ",
		Markdown: "[[doc:" + docA + "]] [[Título]] \\[[doc:" +
			docB + "]]",
		ExpectedVersion: 99,
		Targets:         []string{docB},
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if got.Title != "Draft" || got.Version != 1 {
		t.Fatalf("snapshot = %#v, want trimmed title and version 1", got)
	}
	if len(got.Links) != 1 || got.Links[0].TargetID != docA || got.Links[0].Status != "available" {
		t.Fatalf("links = %#v, want only resolved target %s", got.Links, docA)
	}
	if store.documents[docB].document.Version != 1 {
		t.Fatalf("create carried an expected version: %#v", store.documents[docB].document)
	}
}

func TestEngineRejectsInvalidSnapshotsBeforeStore(t *testing.T) {
	store := &recordingStore{}
	engine := NewEngine(store)
	valid := Mutation{ID: docA, LayerID: layerA, Title: "Title", Markdown: "body"}
	cases := []Mutation{
		{ID: "not-a-uuid", LayerID: valid.LayerID, Title: valid.Title, Markdown: valid.Markdown},
		{ID: valid.ID, LayerID: valid.LayerID, Title: "\nTitle", Markdown: valid.Markdown},
		{ID: valid.ID, LayerID: valid.LayerID, Title: strings.Repeat("界", 256), Markdown: valid.Markdown},
		{ID: valid.ID, LayerID: valid.LayerID, Title: valid.Title, Markdown: "bad\x00body"},
		{ID: valid.ID, LayerID: valid.LayerID, Title: valid.Title, Markdown: strings.Repeat("x", 1<<20+1)},
	}
	for i, input := range cases {
		if _, err := engine.Create(context.Background(), brain.Principal("owner-1"), input); !errors.Is(err, ErrInvalidDocument) {
			t.Errorf("case %d: Create() error = %v, want ErrInvalidDocument", i, err)
		}
	}
	if store.applyCalls != 0 {
		t.Fatalf("invalid snapshots reached Store.Apply %d times", store.applyCalls)
	}
}

func TestEngineRejectsNonCanonicalLayerIDsBeforeStore(t *testing.T) {
	store := &recordingStore{}
	engine := NewEngine(store)
	for _, layer := range []string{"layer-1", "20000000000040008000000000000001", "20000000-0000-4000-8000-00000000000A"} {
		if _, err := engine.Create(context.Background(), brain.Principal("owner-1"), Mutation{ID: docA, LayerID: layer, Title: "Title"}); !errors.Is(err, ErrInvalidDocument) {
			t.Errorf("Create(layer=%q) error = %v, want ErrInvalidDocument", layer, err)
		}
		if _, err := engine.Update(context.Background(), brain.Principal("owner-1"), Mutation{ID: docA, LayerID: layer, Title: "Title", ExpectedVersion: 1}); !errors.Is(err, ErrInvalidDocument) {
			t.Errorf("Update(layer=%q) error = %v, want ErrInvalidDocument", layer, err)
		}
		if _, err := engine.Restore(context.Background(), brain.Principal("owner-1"), docA, 1, layer); !errors.Is(err, ErrInvalidDocument) {
			t.Errorf("Restore(layer=%q) error = %v, want ErrInvalidDocument", layer, err)
		}
		if _, err := engine.List(context.Background(), brain.Principal("owner-1"), Query{LayerID: layer}); !errors.Is(err, ErrInvalidQuery) {
			t.Errorf("List(layer=%q) error = %v, want ErrInvalidQuery", layer, err)
		}
	}
	if store.applyCalls != 0 || store.listCalls != 0 {
		t.Fatalf("invalid layer IDs reached Store: apply=%d list=%d", store.applyCalls, store.listCalls)
	}
}

func TestEngineRejectsLeakyMissingLinkProjection(t *testing.T) {
	store := &recordingStore{getResult: Document{
		ID:      docA,
		LayerID: layerA,
		Title:   "Source",
		Version: 1,
		Links: []Link{{
			TargetID: docB,
			Title:    "Leaked title",
			LayerID:  layerB,
			Status:   "missing",
		}},
	}}
	if _, err := NewEngine(store).Get(context.Background(), brain.Principal("owner-1"), docA); !errors.Is(err, brain.ErrInternal) {
		t.Fatalf("Get() error = %v, want brain.ErrInternal", err)
	}
}

func TestEngineListDefaultsAndValidatesCursor(t *testing.T) {
	store := &recordingStore{}
	engine := NewEngine(store)
	if _, err := engine.List(context.Background(), brain.Principal("owner-1"), Query{LayerID: layerA}); err != nil {
		t.Fatalf("List() with defaults: %v", err)
	}
	if store.query.Scope != ScopeDirect || store.query.State != StateActive || store.query.Limit != DefaultLimit {
		t.Fatalf("query defaults = %#v", store.query)
	}
	if _, err := engine.List(context.Background(), brain.Principal("owner-1"), Query{LayerID: layerA, After: "UPPER"}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("invalid cursor error = %v, want ErrInvalidQuery", err)
	}
	if store.listCalls != 1 {
		t.Fatalf("invalid query reached Store.List; calls = %d", store.listCalls)
	}
}

func TestMemoryStoreCASOwnershipAndAtomicFailure(t *testing.T) {
	owner := brain.Principal("owner-1")
	foreign := brain.Principal("owner-2")
	store := newMemoryStore(time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))
	store.addBrain(owner, "brain-1")
	store.addBrain(foreign, "brain-2")
	store.addLayer(owner, layerA, "root-1")
	store.addLayer(foreign, layerB, "root-2")
	engine := NewEngine(store)
	if _, err := engine.Create(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "A"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	store.failOnce(errors.New("transaction failed"))
	if _, err := engine.Create(context.Background(), owner, Mutation{ID: docB, LayerID: layerA, Title: "B"}); !errors.Is(err, brain.ErrInternal) {
		t.Fatalf("failed transaction error = %v, want brain.ErrInternal", err)
	}
	if _, err := engine.Get(context.Background(), owner, docB); !errors.Is(err, brain.ErrNotFound) {
		t.Fatalf("failed transaction left document: %v", err)
	}
	if _, err := engine.Create(context.Background(), foreign, Mutation{
		ID: docB, LayerID: layerB, Title: "B", Markdown: "[[doc:" + docA + "]]",
	}); !errors.Is(err, brain.ErrNotFound) {
		t.Fatalf("foreign link error = %v, want brain.ErrNotFound", err)
	}

	first, err := engine.Update(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "first", ExpectedVersion: 1})
	if err != nil {
		t.Fatalf("first update: %v", err)
	}
	if _, err := engine.Update(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "stale", ExpectedVersion: 1}); !errors.Is(err, brain.ErrConflict) {
		t.Fatalf("stale update error = %v, want brain.ErrConflict", err)
	}
	if first.Version != 2 {
		t.Fatalf("first update version = %d, want 2", first.Version)
	}
}

func TestMemoryStoreTrashRestoreExpiryAndTombstone(t *testing.T) {
	owner := brain.Principal("owner-1")
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore(now)
	store.addBrain(owner, "brain-1")
	store.addLayer(owner, layerA, "root-1")
	engine := NewEngine(store)
	if _, err := engine.Create(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "A"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	trashed, err := engine.Trash(context.Background(), owner, docA, 1)
	if err != nil {
		t.Fatalf("trash: %v", err)
	}
	repeated, err := engine.Trash(context.Background(), owner, docA, trashed.Version)
	if err != nil {
		t.Fatalf("repeat trash: %v", err)
	}
	if repeated.Version != trashed.Version || !repeated.DeletedAt.Equal(*trashed.DeletedAt) {
		t.Fatal("repeat trash renewed or advanced the tombstone")
	}
	restored, err := engine.Restore(context.Background(), owner, docA, trashed.Version, "")
	if err != nil || restored.DeletedAt != nil || restored.Version != trashed.Version+1 {
		t.Fatalf("restore = %#v, error %v", restored, err)
	}
	trashed, err = engine.Trash(context.Background(), owner, docA, restored.Version)
	if err != nil {
		t.Fatalf("trash after restore: %v", err)
	}
	store.advanceTo(now.Add(documentTrashTTL))
	if _, err := engine.Get(context.Background(), owner, docA); !errors.Is(err, brain.ErrNotFound) {
		t.Fatalf("expired Get() error = %v, want brain.ErrNotFound", err)
	}
	if _, err := engine.Create(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "reuse"}); !errors.Is(err, brain.ErrConflict) {
		t.Fatalf("recreate after purge error = %v, want brain.ErrConflict", err)
	}
}

func TestMemoryStoreSubtreePaginationAndLinkProjection(t *testing.T) {
	owner := brain.Principal("owner-1")
	store := newMemoryStore(time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))
	store.addBrain(owner, "brain-1")
	store.addLayer(owner, layerA, "root-1")
	store.addLayer(owner, layerB, layerA)
	store.addLayer(owner, layerC, "root-1")
	engine := NewEngine(store)
	for id, layer := range map[string]string{docA: layerA, docB: layerB, "00000000-0000-4000-8000-000000000003": layerC} {
		if _, err := engine.Create(context.Background(), owner, Mutation{ID: id, LayerID: layer, Title: id, Markdown: "body"}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	page, err := engine.List(context.Background(), owner, Query{LayerID: layerA, Scope: ScopeSubtree, Limit: 1})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(page.Documents) != 1 || page.NextCursor != docA || page.Documents[0].Markdown != "" {
		t.Fatalf("first page = %#v, want one metadata document and cursor %s", page, docA)
	}
	page, err = engine.List(context.Background(), owner, Query{LayerID: layerA, Scope: ScopeSubtree, Limit: 1, After: page.NextCursor})
	if err != nil || len(page.Documents) != 1 || page.Documents[0].ID != docB || page.NextCursor != "" {
		t.Fatalf("second page = %#v, error %v", page, err)
	}
}

func TestLinkProjectionFollowsRenameAndMoveByID(t *testing.T) {
	owner := brain.Principal("owner-1")
	store := newMemoryStore(time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))
	store.addBrain(owner, "brain-1")
	store.addLayer(owner, layerA, "root-1")
	store.addLayer(owner, layerB, "root-1")
	engine := NewEngine(store)
	if _, err := engine.Create(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "Before"}); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if _, err := engine.Create(context.Background(), owner, Mutation{ID: docB, LayerID: layerA, Title: "Source", Markdown: "[[doc:" + docA + "]]"}); err != nil {
		t.Fatalf("create source: %v", err)
	}
	if _, err := engine.Update(context.Background(), owner, Mutation{ID: docA, LayerID: layerB, Title: "After", ExpectedVersion: 1}); err != nil {
		t.Fatalf("rename/move target: %v", err)
	}
	source, err := engine.Get(context.Background(), owner, docB)
	if err != nil {
		t.Fatalf("get source: %v", err)
	}
	if source.Version != 1 || len(source.Links) != 1 || source.Links[0].Title != "After" || source.Links[0].LayerID != layerB {
		t.Fatalf("source link projection = %#v, source version = %d", source.Links, source.Version)
	}
}

func TestEngineConcurrentCASHasOneWinner(t *testing.T) {
	owner := brain.Principal("owner-1")
	store := newMemoryStore(time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))
	store.addBrain(owner, "brain-1")
	store.addLayer(owner, layerA, "root-1")
	engine := NewEngine(store)
	if _, err := engine.Create(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: "A"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, title := range []string{"first", "second"} {
		wg.Add(1)
		go func(title string) {
			defer wg.Done()
			_, err := engine.Update(context.Background(), owner, Mutation{ID: docA, LayerID: layerA, Title: title, ExpectedVersion: 1})
			results <- err
		}(title)
	}
	wg.Wait()
	close(results)
	var successes, conflicts int
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, brain.ErrConflict) {
			conflicts++
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent CAS results: successes=%d conflicts=%d", successes, conflicts)
	}
}

type recordingStore struct {
	applyCalls int
	listCalls  int
	mutation   Mutation
	query      Query
	getResult  Document
}

func (s *recordingStore) Apply(_ context.Context, _ brain.Principal, mutation Mutation) (Document, error) {
	s.applyCalls++
	s.mutation = mutation
	return Document{ID: mutation.ID, LayerID: mutation.LayerID, Title: mutation.Title, Markdown: mutation.Markdown, Version: 1}, nil
}

func (s *recordingStore) Get(_ context.Context, _ brain.Principal, id string) (Document, error) {
	if s.getResult.ID != "" {
		return s.getResult, nil
	}
	return Document{ID: id, LayerID: layerA, Title: "Title", Version: 1}, nil
}

func (s *recordingStore) List(_ context.Context, _ brain.Principal, query Query) (Page, error) {
	s.listCalls++
	s.query = query
	return Page{}, nil
}

var _ Store = (*recordingStore)(nil)

package brain

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestATDD01OneBrainPerOwner(t *testing.T) {
	engine, _ := newTestEngine()

	brain, err := engine.Bootstrap(context.Background(), Principal("owner-1"))
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if brain.ID == "" || brain.Root.ID == "" {
		t.Fatalf("bootstrap returned incomplete brain: %#v", brain)
	}
	if brain.Owner != Principal("owner-1") {
		t.Fatalf("brain owner = %q, want owner-1", brain.Owner)
	}
	if brain.Root.BrainID != brain.ID {
		t.Fatalf("root brain = %q, want %q", brain.Root.BrainID, brain.ID)
	}
}

func TestATDD02RepeatedBootstrapConverges(t *testing.T) {
	engine, store := newTestEngine()

	first, err := engine.Bootstrap(context.Background(), Principal("owner-1"))
	if err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	for i := 0; i < 5; i++ {
		got, err := engine.Bootstrap(context.Background(), Principal("owner-1"))
		if err != nil {
			t.Fatalf("bootstrap %d: %v", i, err)
		}
		if got.ID != first.ID || got.Root.ID != first.Root.ID {
			t.Fatalf("bootstrap %d = %#v, want same brain/root as %#v", i, got, first)
		}
	}
	if got := store.brainCount(); got != 1 {
		t.Fatalf("stored brains = %d, want 1", got)
	}
}

func TestATDD03ConcurrentBootstrapConverges(t *testing.T) {
	engine, store := newTestEngine()
	const callers = 2

	start := make(chan struct{})
	results := make(chan Brain, callers)
	errorsCh := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			brain, err := engine.Bootstrap(context.Background(), Principal("owner-1"))
			if err != nil {
				errorsCh <- err
				return
			}
			results <- brain
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent bootstrap: %v", err)
	}

	var brains []Brain
	for brain := range results {
		brains = append(brains, brain)
	}
	if len(brains) != callers {
		t.Fatalf("resolved brains = %d, want %d", len(brains), callers)
	}
	if brains[0].ID != brains[1].ID || brains[0].Root.ID != brains[1].Root.ID {
		t.Fatalf("concurrent bootstraps diverged: %#v and %#v", brains[0], brains[1])
	}
	if got := store.brainCount(); got != 1 {
		t.Fatalf("stored brains = %d, want 1", got)
	}
}

func TestATDD04BootstrapHasOnlyStructuralRoot(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")

	before := mustObserve(t, engine, "owner-1")
	if before.Brain.ID != brain.ID || before.Root.ID != brain.Root.ID {
		t.Fatalf("hierarchy root/brain mismatch: %#v", before)
	}
	if len(before.Layers) != 0 {
		t.Fatalf("initial visible layers = %d, want 0", len(before.Layers))
	}

	first := mustCreate(t, engine, "owner-1", RootRef(before.Root.ID), "First layer")
	after := mustObserve(t, engine, "owner-1")
	if len(after.Layers) != 1 || after.Layers[0].ID != first.ID {
		t.Fatalf("visible layers after first create = %#v", after.Layers)
	}
	for _, layer := range after.Layers {
		if strings.EqualFold(layer.Name, "inbox") || strings.Contains(strings.ToLower(layer.Name), "starter") {
			t.Fatalf("unexpected default layer: %#v", layer)
		}
	}
}

func TestATDD05OwnerOnlyWithoutLeakage(t *testing.T) {
	engine, _ := newTestEngine()
	ownerBrain := mustBootstrap(t, engine, "owner-1")
	ownerHierarchy := mustObserve(t, engine, "owner-1")
	secretName := "Owner private layer"
	_ = mustCreate(t, engine, "owner-1", RootRef(ownerHierarchy.Root.ID), secretName)

	if _, err := engine.Observe(context.Background(), Principal("other")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other observe error = %v, want opaque not-found", err)
	} else if leaks(err, string(ownerBrain.ID), string(ownerBrain.Root.ID), secretName, "owner-1") {
		t.Fatalf("other observe leaked private details: %v", err)
	}
	if _, err := engine.CreateLayer(context.Background(), Principal("other"), RootRef(ownerBrain.Root.ID), "intrusion"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other create error = %v, want opaque not-found", err)
	} else if leaks(err, string(ownerBrain.ID), string(ownerBrain.Root.ID), secretName, "owner-1") {
		t.Fatalf("other create leaked private details: %v", err)
	}
	if got := len(mustObserve(t, engine, "owner-1").Layers); got != 1 {
		t.Fatalf("owner layers after unauthorized attempt = %d, want 1", got)
	}
}

func TestATDD06CreateDirectlyUnderRoot(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")

	layer := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), "Work")
	if layer.BrainID != brain.ID {
		t.Fatalf("layer brain = %q, want %q", layer.BrainID, brain.ID)
	}
	if layer.Parent != RootRef(brain.Root.ID) {
		t.Fatalf("layer parent = %#v, want root %q", layer.Parent, brain.Root.ID)
	}
}

func TestATDD07ArbitraryNesting(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")
	parent := RootRef(brain.Root.ID)
	const depth = 64

	for i := 0; i < depth; i++ {
		layer := mustCreate(t, engine, "owner-1", parent, "level")
		if layer.Parent != parent {
			t.Fatalf("level %d parent = %#v, want %#v", i, layer.Parent, parent)
		}
		parent = LayerRef(layer.ID)
	}
	if got := len(mustObserve(t, engine, "owner-1").Layers); got != depth {
		t.Fatalf("layers = %d, want %d", got, depth)
	}
}

func TestATDD08ForeignBrainParentRejected(t *testing.T) {
	engine, _ := newTestEngine()
	first := mustBootstrap(t, engine, "owner-1")
	second := mustBootstrap(t, engine, "owner-2")
	foreign := mustCreate(t, engine, "owner-2", RootRef(second.Root.ID), "Foreign secret")

	for _, parent := range []NodeRef{RootRef(second.Root.ID), LayerRef(foreign.ID)} {
		_, err := engine.CreateLayer(context.Background(), Principal("owner-1"), parent, "cross-brain")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("foreign parent %#v error = %v, want opaque not-found", parent, err)
		}
		if leaks(err, string(second.ID), string(second.Root.ID), string(foreign.ID), foreign.Name) {
			t.Fatalf("foreign parent error leaked details: %v", err)
		}
	}
	if got := len(mustObserve(t, engine, "owner-1").Layers); got != 0 {
		t.Fatalf("owner-1 layers after cross-brain attempts = %d, want 0", got)
	}
	if got := len(mustObserve(t, engine, "owner-2").Layers); got != 1 {
		t.Fatalf("owner-2 layers after cross-brain attempts = %d, want 1", got)
	}
	_ = first
}

func TestATDD09NoMandatoryTaxonomy(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")

	name := "Arbitrary / 42 / 🧠"
	layer := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), name)
	if layer.Name != name {
		t.Fatalf("layer name = %q, want %q", layer.Name, name)
	}
}

func TestATDD10EmptyOrWhitespaceNameRejected(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")
	for _, name := range []string{"", "   ", "\t\n"} {
		if _, err := engine.CreateLayer(context.Background(), Principal("owner-1"), RootRef(brain.Root.ID), name); !errors.Is(err, ErrInvalidName) {
			t.Fatalf("name %q error = %v, want invalid-name", name, err)
		}
	}
	if got := len(mustObserve(t, engine, "owner-1").Layers); got != 0 {
		t.Fatalf("layers after invalid names = %d, want 0", got)
	}
}

func TestATDD11TrimOuterPreserveInnerValue(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")
	input := "  Café  com  Espaço ÁÇ  "
	expected := "Café  com  Espaço ÁÇ"

	layer := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), input)
	if layer.Name != expected {
		t.Fatalf("normalized name = %q, want %q", layer.Name, expected)
	}
}

func TestATDD12DuplicateSiblingNamesHaveDistinctIdentity(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")

	first := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), "same")
	second := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), "same")
	if first.ID == second.ID {
		t.Fatalf("duplicate siblings share identity: %q", first.ID)
	}
	if first.Name != second.Name || first.Parent != second.Parent {
		t.Fatalf("duplicate siblings differ unexpectedly: %#v and %#v", first, second)
	}
}

func TestATDD13ConcurrentCreatesAreNotDeduplicated(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")
	const callers = 2

	start := make(chan struct{})
	results := make(chan Layer, callers)
	errorsCh := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			layer, err := engine.CreateLayer(context.Background(), Principal("owner-1"), RootRef(brain.Root.ID), "same")
			if err != nil {
				errorsCh <- err
				return
			}
			results <- layer
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent create: %v", err)
	}

	var layers []Layer
	for layer := range results {
		layers = append(layers, layer)
	}
	if len(layers) != callers || layers[0].ID == layers[1].ID {
		t.Fatalf("concurrent creates = %#v, want two distinct layers", layers)
	}
}

func TestATDD14RenamePreservesIdentityParentAndChildren(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")
	parent := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), "Parent")
	child := mustCreate(t, engine, "owner-1", LayerRef(parent.ID), "Child")

	rename := mustRename(t, engine, "owner-1", parent, "Renamed")
	if rename.ID != parent.ID || rename.BrainID != parent.BrainID || rename.Parent != parent.Parent {
		t.Fatalf("rename changed identity/brain/parent: before=%#v after=%#v", parent, rename)
	}
	if rename.Name != "Renamed" || rename.Version == parent.Version {
		t.Fatalf("rename result = %#v, want new name and version", rename)
	}
	after := mustObserve(t, engine, "owner-1")
	gotChild, ok := findTestLayer(after.Layers, child.ID)
	if !ok || gotChild.Parent != LayerRef(parent.ID) || gotChild.ID != child.ID {
		t.Fatalf("child changed after parent rename: %#v", gotChild)
	}
}

func TestATDD15ConcurrentRenameUsesCompareAndSwap(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")
	layer := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), "Original")

	start := make(chan struct{})
	results := make(chan Layer, 2)
	errorsCh := make(chan error, 2)
	var wg sync.WaitGroup
	for _, name := range []string{"First", "Second"} {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			updated, err := engine.RenameLayer(context.Background(), Principal("owner-1"), layer.ID, layer.Version, name)
			if err != nil {
				errorsCh <- err
				return
			}
			results <- updated
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errorsCh)

	var winners []Layer
	for result := range results {
		winners = append(winners, result)
	}
	var conflicts int
	for err := range errorsCh {
		if errors.Is(err, ErrConflict) {
			conflicts++
			continue
		}
		t.Fatalf("unexpected concurrent rename error: %v", err)
	}
	if len(winners) != 1 || conflicts != 1 {
		t.Fatalf("rename winners=%d conflicts=%d, want one each", len(winners), conflicts)
	}

	current := mustObserve(t, engine, "owner-1")
	final, ok := findTestLayer(current.Layers, layer.ID)
	if !ok || final.Name != winners[0].Name || final.Version != winners[0].Version {
		t.Fatalf("stored rename = %#v, winner = %#v", final, winners[0])
	}
	retry := mustRename(t, engine, "owner-1", final, "Retry")
	if retry.Name != "Retry" || retry.Version == final.Version {
		t.Fatalf("retry rename = %#v, want version advance", retry)
	}
}

func TestATDD16IdentityIndependentOfNameAndPosition(t *testing.T) {
	engine, _ := newTestEngine()
	brain := mustBootstrap(t, engine, "owner-1")
	first := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), "same")
	parent := mustCreate(t, engine, "owner-1", RootRef(brain.Root.ID), "parent")
	second := mustCreate(t, engine, "owner-1", LayerRef(parent.ID), "same")

	updated := mustRename(t, engine, "owner-1", first, "renamed")
	if updated.ID != first.ID || updated.Parent != first.Parent {
		t.Fatalf("renamed identity/position changed: before=%#v after=%#v", first, updated)
	}
	after := mustObserve(t, engine, "owner-1")
	gotSecond, ok := findTestLayer(after.Layers, second.ID)
	if !ok || gotSecond.Name != "same" || gotSecond.Parent != LayerRef(parent.ID) {
		t.Fatalf("sibling identity/position changed: %#v", gotSecond)
	}
}

func newTestEngine() (*Engine, *memoryStore) {
	store := newMemoryStore()
	return NewEngine(store, NewUUIDSource()), store
}

func mustBootstrap(t *testing.T, engine *Engine, owner string) Brain {
	t.Helper()
	brain, err := engine.Bootstrap(context.Background(), Principal(owner))
	if err != nil {
		t.Fatalf("bootstrap %q: %v", owner, err)
	}
	return brain
}

func mustObserve(t *testing.T, engine *Engine, owner string) Hierarchy {
	t.Helper()
	hierarchy, err := engine.Observe(context.Background(), Principal(owner))
	if err != nil {
		t.Fatalf("observe %q: %v", owner, err)
	}
	return hierarchy
}

func mustCreate(t *testing.T, engine *Engine, owner string, parent NodeRef, name string) Layer {
	t.Helper()
	layer, err := engine.CreateLayer(context.Background(), Principal(owner), parent, name)
	if err != nil {
		t.Fatalf("create %q under %#v: %v", name, parent, err)
	}
	return layer
}

func mustRename(t *testing.T, engine *Engine, owner string, layer Layer, name string) Layer {
	t.Helper()
	updated, err := engine.RenameLayer(context.Background(), Principal(owner), layer.ID, layer.Version, name)
	if err != nil {
		t.Fatalf("rename %q: %v", name, err)
	}
	return updated
}

func findTestLayer(layers []Layer, id LayerID) (Layer, bool) {
	for _, layer := range layers {
		if layer.ID == id {
			return layer, true
		}
	}
	return Layer{}, false
}

func leaks(err error, values ...string) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	for _, value := range values {
		if value != "" && strings.Contains(message, value) {
			return true
		}
	}
	return false
}

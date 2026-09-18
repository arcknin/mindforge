package brain

import (
	"context"
	"errors"
	"strings"
)

// Engine is the deterministic application boundary for the Brain Core. It
// accepts an already-authenticated opaque principal and delegates atomic state
// transitions to Store.
type Engine struct {
	store Store
	ids   IDSource
}

func NewEngine(store Store, ids IDSource) *Engine {
	if ids == nil {
		ids = UUIDSource{}
	}
	return &Engine{store: store, ids: ids}
}

func (e *Engine) Bootstrap(ctx context.Context, owner Principal) (Brain, error) {
	if !validPrincipal(owner) {
		return Brain{}, ErrUnauthenticated
	}
	if !e.ready() {
		return Brain{}, ErrInternal
	}
	brainID, err := e.ids.NewBrainID()
	if err != nil {
		return Brain{}, ErrInternal
	}
	rootID, err := e.ids.NewRootID()
	if err != nil {
		return Brain{}, ErrInternal
	}
	candidate := Brain{
		ID:    brainID,
		Owner: owner,
		Root: Root{
			ID:      rootID,
			BrainID: brainID,
		},
	}
	brain, err := e.store.EnsureBrain(normalizeContext(ctx), owner, candidate)
	if err != nil {
		return Brain{}, mapStoreError(err)
	}
	if !validBrain(brain, owner) {
		return Brain{}, ErrInternal
	}
	return brain, nil
}

func (e *Engine) Observe(ctx context.Context, owner Principal) (Hierarchy, error) {
	if !validPrincipal(owner) {
		return Hierarchy{}, ErrUnauthenticated
	}
	if !e.ready() {
		return Hierarchy{}, ErrInternal
	}
	hierarchy, err := e.store.Observe(normalizeContext(ctx), owner)
	if err != nil {
		return Hierarchy{}, mapStoreError(err)
	}
	if !validHierarchy(hierarchy, owner) {
		return Hierarchy{}, ErrInternal
	}
	return hierarchy, nil
}

func (e *Engine) CreateLayer(ctx context.Context, owner Principal, parent NodeRef, name string) (Layer, error) {
	if !validPrincipal(owner) {
		return Layer{}, ErrUnauthenticated
	}
	name, ok := normalizeLayerName(name)
	if !ok {
		return Layer{}, ErrInvalidName
	}
	if !validNodeRef(parent) {
		return Layer{}, ErrNotFound
	}
	if !e.ready() {
		return Layer{}, ErrInternal
	}
	hierarchy, err := e.Observe(ctx, owner)
	if err != nil {
		return Layer{}, err
	}
	if !containsParent(hierarchy, parent) {
		return Layer{}, ErrNotFound
	}
	layerID, err := e.ids.NewLayerID()
	if err != nil {
		return Layer{}, ErrInternal
	}
	candidate := Layer{
		ID:      layerID,
		BrainID: hierarchy.Brain.ID,
		Parent:  parent,
		Name:    name,
		Version: 1,
	}
	layer, err := e.store.InsertLayer(normalizeContext(ctx), owner, candidate)
	if err != nil {
		return Layer{}, mapStoreError(err)
	}
	if !validLayer(layer, hierarchy.Brain.ID) {
		return Layer{}, ErrInternal
	}
	return layer, nil
}

func (e *Engine) RenameLayer(ctx context.Context, owner Principal, layerID LayerID, expectedVersion LayerVersion, name string) (Layer, error) {
	if !validPrincipal(owner) {
		return Layer{}, ErrUnauthenticated
	}
	name, ok := normalizeLayerName(name)
	if !ok {
		return Layer{}, ErrInvalidName
	}
	if layerID == "" {
		return Layer{}, ErrNotFound
	}
	if !e.ready() {
		return Layer{}, ErrInternal
	}
	hierarchy, err := e.Observe(ctx, owner)
	if err != nil {
		return Layer{}, err
	}
	if _, ok := findLayer(hierarchy.Layers, layerID); !ok {
		return Layer{}, ErrNotFound
	}
	layer, err := e.store.CompareAndSwapLayerName(normalizeContext(ctx), owner, hierarchy.Brain.ID, layerID, expectedVersion, name)
	if err != nil {
		return Layer{}, mapStoreError(err)
	}
	if !validLayer(layer, hierarchy.Brain.ID) {
		return Layer{}, ErrInternal
	}
	return layer, nil
}

func (e *Engine) ready() bool {
	return e != nil && e.store != nil && e.ids != nil
}

func validPrincipal(owner Principal) bool {
	return owner != ""
}

func normalizeLayerName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	return name, name != ""
}

func validNodeRef(ref NodeRef) bool {
	return ref.ID != "" && (ref.Kind == NodeRoot || ref.Kind == NodeLayer)
}

func validBrain(brain Brain, owner Principal) bool {
	return brain.ID != "" && brain.Owner == owner && brain.Root.ID != "" && brain.Root.BrainID == brain.ID
}

func validHierarchy(hierarchy Hierarchy, owner Principal) bool {
	if !validBrain(hierarchy.Brain, owner) || hierarchy.Root != hierarchy.Brain.Root {
		return false
	}
	for _, layer := range hierarchy.Layers {
		if !validLayer(layer, hierarchy.Brain.ID) {
			return false
		}
	}
	return true
}

func validLayer(layer Layer, brainID BrainID) bool {
	return layer.ID != "" && layer.BrainID == brainID && validNodeRef(layer.Parent) && layer.Name != "" && layer.Version > 0
}

func containsParent(hierarchy Hierarchy, parent NodeRef) bool {
	if parent == RootRef(hierarchy.Root.ID) {
		return true
	}
	if parent.Kind != NodeLayer {
		return false
	}
	_, ok := findLayer(hierarchy.Layers, LayerID(parent.ID))
	return ok
}

func findLayer(layers []Layer, id LayerID) (Layer, bool) {
	for _, layer := range layers {
		if layer.ID == id {
			return layer, true
		}
	}
	return Layer{}, false
}

func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrUnauthenticated):
		return ErrUnauthenticated
	case errors.Is(err, ErrConflict):
		return ErrConflict
	case errors.Is(err, ErrInvalidName):
		return ErrInvalidName
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbidden):
		return ErrNotFound
	default:
		return ErrInternal
	}
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

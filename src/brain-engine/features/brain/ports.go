package brain

import "context"

// Store is the persistence boundary for the deterministic Brain Core. Its
// implementations own atomicity for bootstrap, creation and rename CAS.
type Store interface {
	EnsureBrain(ctx context.Context, owner Principal, candidate Brain) (Brain, error)
	Observe(ctx context.Context, owner Principal) (Hierarchy, error)
	InsertLayer(ctx context.Context, owner Principal, layer Layer) (Layer, error)
	CompareAndSwapLayerName(ctx context.Context, owner Principal, brainID BrainID, layerID LayerID, expectedVersion LayerVersion, name string) (Layer, error)
}

// IDSource keeps identity generation inside the Engine while allowing a
// future persistence adapter and tests to inject the source explicitly.
type IDSource interface {
	NewBrainID() (BrainID, error)
	NewRootID() (RootID, error)
	NewLayerID() (LayerID, error)
}

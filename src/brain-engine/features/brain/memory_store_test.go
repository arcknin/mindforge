package brain

import (
	"context"
	"sync"
)

type memoryStore struct {
	mu     sync.RWMutex
	brains map[Principal]Brain
	layers map[LayerID]Layer
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		brains: make(map[Principal]Brain),
		layers: make(map[LayerID]Layer),
	}
}

func (s *memoryStore) EnsureBrain(_ context.Context, owner Principal, candidate Brain) (Brain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.brains[owner]; ok {
		return existing, nil
	}
	if candidate.Owner != owner || candidate.ID == "" || candidate.Root.ID == "" || candidate.Root.BrainID != candidate.ID {
		return Brain{}, ErrInternal
	}
	s.brains[owner] = candidate
	return candidate, nil
}

func (s *memoryStore) Observe(_ context.Context, owner Principal) (Hierarchy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	brain, ok := s.brains[owner]
	if !ok {
		return Hierarchy{}, ErrNotFound
	}
	return Hierarchy{Brain: brain, Root: brain.Root, Layers: s.layersFor(brain.ID)}, nil
}

func (s *memoryStore) InsertLayer(_ context.Context, owner Principal, layer Layer) (Layer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	brain, ok := s.brains[owner]
	if !ok || brain.ID != layer.BrainID {
		return Layer{}, ErrNotFound
	}
	if layer.ID == "" || layer.Version != 1 || layer.Name == "" {
		return Layer{}, ErrInternal
	}
	if !s.parentExistsLocked(brain, layer.Parent) {
		return Layer{}, ErrNotFound
	}
	if _, exists := s.layers[layer.ID]; exists {
		return Layer{}, ErrConflict
	}
	s.layers[layer.ID] = layer
	return layer, nil
}

func (s *memoryStore) CompareAndSwapLayerName(_ context.Context, owner Principal, brainID BrainID, layerID LayerID, expectedVersion LayerVersion, name string) (Layer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	brain, ok := s.brains[owner]
	if !ok || brain.ID != brainID {
		return Layer{}, ErrNotFound
	}
	layer, ok := s.layers[layerID]
	if !ok || layer.BrainID != brainID {
		return Layer{}, ErrNotFound
	}
	if layer.Version != expectedVersion {
		return Layer{}, ErrConflict
	}
	layer.Name = name
	layer.Version++
	s.layers[layerID] = layer
	return layer, nil
}

func (s *memoryStore) brainCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.brains)
}

func (s *memoryStore) layersFor(brainID BrainID) []Layer {
	var layers []Layer
	for _, layer := range s.layers {
		if layer.BrainID == brainID {
			layers = append(layers, layer)
		}
	}
	return layers
}

func (s *memoryStore) parentExistsLocked(brain Brain, parent NodeRef) bool {
	switch parent.Kind {
	case NodeRoot:
		return RootID(parent.ID) == brain.Root.ID
	case NodeLayer:
		layer, ok := s.layers[LayerID(parent.ID)]
		return ok && layer.BrainID == brain.ID
	default:
		return false
	}
}

var _ Store = (*memoryStore)(nil)

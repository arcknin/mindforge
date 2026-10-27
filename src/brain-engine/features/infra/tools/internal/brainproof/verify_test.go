package brainproof

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
)

func TestVerifyRejectsUnsafeInputsBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(*Options, *config.TargetSettings){
		"missing synthetic write opt-in": func(options *Options, _ *config.TargetSettings) {
			options.AllowSyntheticWrites = false
		},
		"missing operator": func(options *Options, _ *config.TargetSettings) {
			options.OperatorID = ""
		},
		"missing authority": func(options *Options, _ *config.TargetSettings) {
			options.AuthorizationRef = ""
		},
		"unsafe run ID": func(options *Options, _ *config.TargetSettings) {
			options.RunID = "../private"
		},
		"main target": func(_ *Options, settings *config.TargetSettings) {
			settings.Target = config.TargetMain
			settings.DatabaseName = string(config.TargetMain)
		},
		"develop target name mismatch": func(_ *Options, settings *config.TargetSettings) {
			settings.DatabaseName = string(config.TargetMain)
		},
	} {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			options := validOptions()
			settings := validDevelopSettings()
			mutate(&options, &settings)
			probed := false
			opened := false
			_, err := Verify(context.Background(), settings, options, Dependencies{
				CurrentDatabase: func(context.Context, config.TargetSettings) (string, error) {
					probed = true
					return "mindforge_develop", nil
				},
				OpenStore: func(context.Context, config.TargetSettings) (Store, error) {
					opened = true
					return newFakeStore(), nil
				},
			})
			if err == nil {
				t.Fatal("Verify() error = nil, want guarded input rejection")
			}
			if probed || opened {
				t.Fatalf("database access occurred before input rejection: probed=%t opened=%t", probed, opened)
			}
		})
	}
}

func TestVerifyChecksDatabaseIdentityBeforeOpeningStore(t *testing.T) {
	t.Parallel()

	opened := false
	_, err := Verify(context.Background(), validDevelopSettings(), validOptions(), Dependencies{
		CurrentDatabase: func(_ context.Context, settings config.TargetSettings) (string, error) {
			if settings.DatabaseHost != "127.0.0.1" {
				t.Fatalf("identity probe host = %q, want configured client host", settings.DatabaseHost)
			}
			return "mindforge", nil
		},
		OpenStore: func(context.Context, config.TargetSettings) (Store, error) {
			opened = true
			return newFakeStore(), nil
		},
	})
	if err == nil {
		t.Fatal("Verify() error = nil for the main database identity")
	}
	if opened {
		t.Fatal("Store factory ran after database identity mismatch")
	}
	if strings.Contains(err.Error(), "mindforge") {
		t.Fatalf("database identity mismatch leaked the observed database name: %v", err)
	}
}

func TestVerifySanitizesDatabaseIdentityAndStoreFailures(t *testing.T) {
	t.Parallel()

	const privateMarker = "synthetic-password-must-not-appear"
	for name, deps := range map[string]Dependencies{
		"identity query": {
			CurrentDatabase: func(context.Context, config.TargetSettings) (string, error) {
				return "", fmt.Errorf("driver diagnostic %s", privateMarker)
			},
			OpenStore: func(context.Context, config.TargetSettings) (Store, error) {
				return newFakeStore(), nil
			},
		},
		"store open": {
			CurrentDatabase: func(context.Context, config.TargetSettings) (string, error) {
				return string(config.TargetDevelop), nil
			},
			OpenStore: func(context.Context, config.TargetSettings) (Store, error) {
				return nil, fmt.Errorf("pool diagnostic %s", privateMarker)
			},
		},
	} {
		name, deps := name, deps
		t.Run(name, func(t *testing.T) {
			_, err := Verify(context.Background(), validDevelopSettings(), validOptions(), deps)
			if err == nil {
				t.Fatal("Verify() error = nil, want dependency failure")
			}
			if strings.Contains(err.Error(), privateMarker) {
				t.Fatalf("Verify() leaked a driver error: %v", err)
			}
		})
	}
}

func TestVerifyRunsBrainEngineBoundariesAndIsIdempotentForRunID(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	dependencies := Dependencies{
		CurrentDatabase: func(_ context.Context, settings config.TargetSettings) (string, error) {
			if settings.Target != config.TargetDevelop || settings.DatabaseName != string(config.TargetDevelop) {
				t.Fatalf("identity probe received wrong target: %#v", settings)
			}
			return string(config.TargetDevelop), nil
		},
		OpenStore: func(_ context.Context, settings config.TargetSettings) (Store, error) {
			if settings.Target != config.TargetDevelop || settings.DatabaseName != string(config.TargetDevelop) {
				t.Fatalf("store factory received wrong target: %#v", settings)
			}
			return store, nil
		},
	}

	first, err := Verify(context.Background(), validDevelopSettings(), validOptions(), dependencies)
	if err != nil {
		t.Fatalf("Verify() first run: %v", err)
	}
	second, err := Verify(context.Background(), validDevelopSettings(), validOptions(), dependencies)
	if err != nil {
		t.Fatalf("Verify() repeated run: %v", err)
	}
	if first.Target != string(config.TargetDevelop) || second.Target != string(config.TargetDevelop) {
		t.Fatalf("verification targets = %q and %q", first.Target, second.Target)
	}
	if first.RunID != validOptions().RunID || second.RunID != first.RunID {
		t.Fatalf("run IDs = %q and %q", first.RunID, second.RunID)
	}
	if !reflect.DeepEqual(first.Checks, second.Checks) {
		t.Fatalf("verification checks changed between runs: %#v / %#v", first.Checks, second.Checks)
	}
	if !first.Checks.BootstrapIdempotent || !first.Checks.OwnObserve || !first.Checks.UnauthorizedIsolation || !first.Checks.NestedCreate || !first.Checks.RenameIdentityStable || !first.Checks.StaleRenameConflict {
		t.Fatalf("verification omitted a required Brain Engine check: %#v", first.Checks)
	}
	if len(store.brains) != 1 {
		t.Fatalf("synthetic brains = %d, want exactly one across repeated run", len(store.brains))
	}
	if len(store.layers) != 3 {
		t.Fatalf("synthetic layers = %d, want deterministic reuse of three rows", len(store.layers))
	}
	if first.OperatorID != validOptions().OperatorID || first.AuthorizationReference != validOptions().AuthorizationRef {
		t.Fatalf("verification omitted authority references: %#v", first)
	}
}

func TestVerifyDoesNotExposeStoreErrorOrSyntheticOwner(t *testing.T) {
	t.Parallel()

	const privateMarker = "synthetic-owner/verification/brain-core/run-private"
	store := newFakeStore()
	store.failInsert = true
	_, err := Verify(context.Background(), validDevelopSettings(), validOptions(), Dependencies{
		CurrentDatabase: func(context.Context, config.TargetSettings) (string, error) {
			return string(config.TargetDevelop), nil
		},
		OpenStore: func(context.Context, config.TargetSettings) (Store, error) {
			return store, nil
		},
	})
	if err == nil {
		t.Fatal("Verify() error = nil, want injected store failure")
	}
	if strings.Contains(err.Error(), privateMarker) || strings.Contains(err.Error(), "private database error") {
		t.Fatalf("Verify() leaked synthetic or storage detail: %v", err)
	}
}

func validOptions() Options {
	return Options{
		OperatorID:           "operator-test",
		AuthorizationRef:     "owner-auth-2026-10-08",
		RunID:                "run-20261008-1",
		AllowSyntheticWrites: true,
	}
}

func validDevelopSettings() config.TargetSettings {
	return config.TargetSettings{
		Target:           config.TargetDevelop,
		DatabaseHost:     "127.0.0.1",
		DatabasePort:     "15432",
		DatabaseName:     string(config.TargetDevelop),
		DatabaseUser:     "mf_develop",
		DatabasePassword: "synthetic-password",
	}
}

type fakeStore struct {
	brains     map[brain.Principal]brain.Brain
	layers     map[brain.LayerID]brain.Layer
	failInsert bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{brains: make(map[brain.Principal]brain.Brain), layers: make(map[brain.LayerID]brain.Layer)}
}

func (s *fakeStore) Close() {}

func (s *fakeStore) EnsureBrain(_ context.Context, owner brain.Principal, candidate brain.Brain) (brain.Brain, error) {
	if existing, ok := s.brains[owner]; ok {
		return existing, nil
	}
	s.brains[owner] = candidate
	return candidate, nil
}

func (s *fakeStore) Observe(_ context.Context, owner brain.Principal) (brain.Hierarchy, error) {
	stored, ok := s.brains[owner]
	if !ok {
		return brain.Hierarchy{}, brain.ErrNotFound
	}
	layers := make([]brain.Layer, 0)
	for _, layer := range s.layers {
		if layer.BrainID == stored.ID {
			layers = append(layers, layer)
		}
	}
	sort.Slice(layers, func(i, j int) bool { return layers[i].ID < layers[j].ID })
	return brain.Hierarchy{Brain: stored, Root: stored.Root, Layers: layers}, nil
}

func (s *fakeStore) InsertLayer(_ context.Context, owner brain.Principal, layer brain.Layer) (brain.Layer, error) {
	if s.failInsert {
		return brain.Layer{}, fmt.Errorf("private database error")
	}
	stored, ok := s.brains[owner]
	if !ok || stored.ID != layer.BrainID {
		return brain.Layer{}, brain.ErrNotFound
	}
	if layer.Parent == brain.RootRef(stored.Root.ID) {
		s.layers[layer.ID] = layer
		return layer, nil
	}
	for _, parent := range s.layers {
		if parent.BrainID == stored.ID && layer.Parent == brain.LayerRef(parent.ID) {
			s.layers[layer.ID] = layer
			return layer, nil
		}
	}
	return brain.Layer{}, brain.ErrNotFound
}

func (s *fakeStore) CompareAndSwapLayerName(_ context.Context, owner brain.Principal, brainID brain.BrainID, layerID brain.LayerID, expectedVersion brain.LayerVersion, name string) (brain.Layer, error) {
	stored, ok := s.brains[owner]
	if !ok || stored.ID != brainID {
		return brain.Layer{}, brain.ErrNotFound
	}
	layer, ok := s.layers[layerID]
	if !ok || layer.BrainID != brainID {
		return brain.Layer{}, brain.ErrNotFound
	}
	if layer.Version != expectedVersion {
		return brain.Layer{}, brain.ErrConflict
	}
	layer.Name = name
	layer.Version++
	s.layers[layerID] = layer
	return layer, nil
}

var _ Store = (*fakeStore)(nil)

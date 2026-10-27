package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/auth"
	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
)

type fakeVerifier struct {
	principal brain.Principal
	err       error
	panicWith any
	calls     atomic.Int32
	token     atomic.Value
}

func (v *fakeVerifier) Verify(_ context.Context, token string) (brain.Principal, error) {
	v.calls.Add(1)
	v.token.Store(token)
	if v.panicWith != nil {
		panic(v.panicWith)
	}
	return v.principal, v.err
}

type fakeStore struct {
	mu                     sync.Mutex
	brains                 map[brain.Principal]brain.Brain
	layers                 map[brain.LayerID]brain.Layer
	ensureErrAfterMutation error
	ensureCalls            int
	observeCalls           int
	insertCalls            int
	renameCalls            int
}

func newFakeStore() *fakeStore {
	return &fakeStore{brains: make(map[brain.Principal]brain.Brain), layers: make(map[brain.LayerID]brain.Layer)}
}

func (s *fakeStore) EnsureBrain(_ context.Context, owner brain.Principal, candidate brain.Brain) (brain.Brain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCalls++
	if existing, ok := s.brains[owner]; ok {
		return existing, nil
	}
	s.brains[owner] = candidate
	if s.ensureErrAfterMutation != nil {
		return brain.Brain{}, s.ensureErrAfterMutation
	}
	return candidate, nil
}

func (s *fakeStore) Observe(_ context.Context, owner brain.Principal) (brain.Hierarchy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observeCalls++
	ownedBrain, ok := s.brains[owner]
	if !ok {
		return brain.Hierarchy{}, brain.ErrNotFound
	}
	layers := make([]brain.Layer, 0)
	for _, layer := range s.layers {
		if layer.BrainID == ownedBrain.ID {
			layers = append(layers, layer)
		}
	}
	return brain.Hierarchy{Brain: ownedBrain, Root: ownedBrain.Root, Layers: layers}, nil
}

func (s *fakeStore) InsertLayer(_ context.Context, owner brain.Principal, layer brain.Layer) (brain.Layer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.insertCalls++
	ownedBrain, ok := s.brains[owner]
	if !ok || ownedBrain.ID != layer.BrainID || !s.parentExistsLocked(ownedBrain, layer.Parent) {
		return brain.Layer{}, brain.ErrNotFound
	}
	s.layers[layer.ID] = layer
	return layer, nil
}

func (s *fakeStore) CompareAndSwapLayerName(_ context.Context, owner brain.Principal, brainID brain.BrainID, layerID brain.LayerID, expectedVersion brain.LayerVersion, name string) (brain.Layer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.renameCalls++
	ownedBrain, ok := s.brains[owner]
	layer, layerOK := s.layers[layerID]
	if !ok || ownedBrain.ID != brainID || !layerOK || layer.BrainID != brainID {
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

func (s *fakeStore) parentExistsLocked(ownedBrain brain.Brain, parent brain.NodeRef) bool {
	switch parent.Kind {
	case brain.NodeRoot:
		return string(ownedBrain.Root.ID) == parent.ID
	case brain.NodeLayer:
		layer, ok := s.layers[brain.LayerID(parent.ID)]
		return ok && layer.BrainID == ownedBrain.ID
	default:
		return false
	}
}

func (s *fakeStore) snapshot() (ensures, observes, inserts, renames int, layers map[brain.LayerID]brain.Layer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyLayers := make(map[brain.LayerID]brain.Layer, len(s.layers))
	for id, layer := range s.layers {
		copyLayers[id] = layer
	}
	return s.ensureCalls, s.observeCalls, s.insertCalls, s.renameCalls, copyLayers
}

func newTestAPI() (*fakeVerifier, *fakeStore, *brain.Engine, http.Handler) {
	verifier := &fakeVerifier{principal: brain.Principal("550e8400-e29b-41d4-a716-446655440000")}
	store := newFakeStore()
	engine := brain.NewEngine(store, nil)
	return verifier, store, engine, NewHandler(engine, verifier)
}

func performRequest(t *testing.T, handler http.Handler, method, target, token, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

func decodeError(t *testing.T, response *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var result errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode error response: %v; body=%q", err, response.Body.String())
	}
	if result.Error.Code == "" || result.Error.Message == "" || result.RequestID == "" {
		t.Fatalf("incomplete error envelope: %#v", result)
	}
	return result
}

func TestHandlerRejectsMissingAndInvalidCredentialsBeforeEngine(t *testing.T) {
	verifier, store, _, handler := newTestAPI()
	verifier.err = auth.ErrUnauthenticated

	missing := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "", "", "")
	if missing.Code != http.StatusUnauthorized || decodeError(t, missing).Error.Code != "unauthenticated" {
		t.Fatalf("missing token response = %d %s", missing.Code, missing.Body.String())
	}
	if got := verifier.calls.Load(); got != 0 {
		t.Fatalf("verifier calls for missing token = %d, want 0", got)
	}

	invalid := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "bad-token", "", "")
	if invalid.Code != http.StatusUnauthorized || decodeError(t, invalid).Error.Code != "unauthenticated" {
		t.Fatalf("invalid token response = %d %s", invalid.Code, invalid.Body.String())
	}
	if _, _, _, _, layers := store.snapshot(); len(layers) != 0 {
		t.Fatalf("invalid identity changed layers: %#v", layers)
	}
	ensures, observes, inserts, renames, _ := store.snapshot()
	if ensures+observes+inserts+renames != 0 {
		t.Fatalf("Engine reached Store before auth rejection: ensure=%d observe=%d insert=%d rename=%d", ensures, observes, inserts, renames)
	}
}

func TestHandlerRejectsMalformedOrDuplicateAuthorizationHeaders(t *testing.T) {
	verifier, store, _, handler := newTestAPI()
	for name, authorization := range map[string][]string{
		"wrong scheme": {"Basic c2VjcmV0"},
		"empty bearer": {"Bearer   "},
		"duplicate":    {"Bearer first", "Bearer second"},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/v1/me/brain", nil)
			for _, value := range authorization {
				req.Header.Add("Authorization", value)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != http.StatusUnauthorized || decodeError(t, response).Error.Code != "unauthenticated" {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
	if got := verifier.calls.Load(); got != 0 {
		t.Fatalf("verifier calls for invalid headers = %d, want 0", got)
	}
	ensures, observes, inserts, renames, _ := store.snapshot()
	if ensures+observes+inserts+renames != 0 {
		t.Fatalf("invalid Authorization headers reached Store: ensure=%d observe=%d insert=%d rename=%d", ensures, observes, inserts, renames)
	}
}

func TestHandlerMapsAuthUnavailableTo503WithoutEngineEffects(t *testing.T) {
	verifier, store, _, handler := newTestAPI()
	verifier.err = auth.ErrUnavailable
	response := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "valid-shaped-token", "", "")
	if response.Code != http.StatusServiceUnavailable || decodeError(t, response).Error.Code != "auth_unavailable" {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	ensures, observes, inserts, renames, _ := store.snapshot()
	if ensures+observes+inserts+renames != 0 {
		t.Fatalf("Engine reached Store during Auth outage: ensure=%d observe=%d insert=%d rename=%d", ensures, observes, inserts, renames)
	}
}

func TestHandlerDoesNotTurnPostEngineUnauthenticatedIntoReplayable401(t *testing.T) {
	_, store, _, handler := newTestAPI()
	store.ensureErrAfterMutation = fmt.Errorf("%w: private store detail", brain.ErrUnauthenticated)

	response := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "valid-access-token", "", "")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("late Engine unauthenticated response = %d %s; want opaque internal error, never replayable 401", response.Code, response.Body.String())
	}
	envelope := decodeError(t, response)
	if envelope.Error.Code != "internal" || strings.Contains(response.Body.String(), "private store detail") {
		t.Fatalf("late Engine error leaked details or wrong code: %s", response.Body.String())
	}
	store.mu.Lock()
	mutated := len(store.brains) == 1
	store.mu.Unlock()
	if !mutated {
		t.Fatal("fixture did not simulate an Engine-side mutation before ErrUnauthenticated")
	}
}

func TestBootstrapConvergesAndObserveDoesNotBootstrap(t *testing.T) {
	verifier, store, _, handler := newTestAPI()
	before := performRequest(t, handler, http.MethodGet, "/v1/me/brain", "access-token", "", "")
	if before.Code != http.StatusNotFound || decodeError(t, before).Error.Code != "not_found" {
		t.Fatalf("observe before bootstrap = %d %s", before.Code, before.Body.String())
	}
	ensures, _, _, _, _ := store.snapshot()
	if ensures != 0 {
		t.Fatalf("GET bootstrapped Brain: EnsureBrain calls = %d", ensures)
	}

	first := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "access-token", "", "")
	second := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "access-token", "", "")
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("bootstrap statuses = %d, %d; bodies=%s / %s", first.Code, second.Code, first.Body.String(), second.Body.String())
	}
	if got := verifier.token.Load(); got != "access-token" {
		t.Fatalf("verifier received token %v, want request bearer", got)
	}
	var firstBody map[string]any
	var secondBody map[string]any
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("decode first bootstrap: %v", err)
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("decode second bootstrap: %v", err)
	}
	firstBrain := firstBody["brain"].(map[string]any)
	secondBrain := secondBody["brain"].(map[string]any)
	if firstBrain["id"] != secondBrain["id"] {
		t.Fatalf("Brain IDs differ across convergent bootstrap: %#v / %#v", firstBrain, secondBrain)
	}
	if firstBrain["root"].(map[string]any)["kind"] != "root" {
		t.Fatalf("root kind missing from bootstrap: %#v", firstBrain)
	}

	observed := performRequest(t, handler, http.MethodGet, "/v1/me/brain", "access-token", "", "")
	if observed.Code != http.StatusOK {
		t.Fatalf("observe status = %d, body=%s", observed.Code, observed.Body.String())
	}
	var hierarchy struct {
		Brain  map[string]any `json:"brain"`
		Layers []any          `json:"layers"`
	}
	if err := json.Unmarshal(observed.Body.Bytes(), &hierarchy); err != nil {
		t.Fatalf("decode hierarchy: %v", err)
	}
	if hierarchy.Layers == nil || len(hierarchy.Layers) != 0 {
		t.Fatalf("initial layers = %#v, want non-nil empty array", hierarchy.Layers)
	}
}

func TestCreateNestedLayerAndRenamePreserveIdentityAndParent(t *testing.T) {
	_, _, _, handler := newTestAPI()
	bootstrap := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "access-token", "", "")
	if bootstrap.Code != http.StatusOK {
		t.Fatalf("bootstrap: %d %s", bootstrap.Code, bootstrap.Body.String())
	}
	var brainResponse struct {
		Brain struct {
			Root struct {
				ID string `json:"id"`
			} `json:"root"`
		} `json:"brain"`
	}
	if err := json.Unmarshal(bootstrap.Body.Bytes(), &brainResponse); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	rootID := brainResponse.Brain.Root.ID

	create := func(parentKind, parentID, name string) struct {
		ID     string `json:"id"`
		Parent struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		} `json:"parent"`
		Name    string `json:"name"`
		Version int64  `json:"version"`
	} {
		t.Helper()
		body := fmt.Sprintf(`{"parent":{"kind":%q,"id":%q},"name":%q}`, parentKind, parentID, name)
		response := performRequest(t, handler, http.MethodPost, "/v1/me/brain/layers", "access-token", "application/json", body)
		if response.Code != http.StatusCreated {
			t.Fatalf("create layer response = %d %s", response.Code, response.Body.String())
		}
		var envelope struct {
			Layer struct {
				ID     string `json:"id"`
				Parent struct {
					Kind string `json:"kind"`
					ID   string `json:"id"`
				} `json:"parent"`
				Name    string `json:"name"`
				Version int64  `json:"version"`
			} `json:"layer"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode layer: %v", err)
		}
		return envelope.Layer
	}

	outer := create("root", rootID, "  Repeated  ")
	if outer.ID == "" || outer.Parent.Kind != "root" || outer.Parent.ID != rootID || outer.Name != "Repeated" || outer.Version != 1 {
		t.Fatalf("root layer = %#v", outer)
	}
	inner := create("layer", outer.ID, "Repeated")
	if inner.ID == "" || inner.ID == outer.ID || inner.Parent.Kind != "layer" || inner.Parent.ID != outer.ID || inner.Name != outer.Name {
		t.Fatalf("nested layer = %#v; outer=%#v", inner, outer)
	}

	renameBody := fmt.Sprintf(`{"name":"Renamed","expected_version":%d}`, inner.Version)
	rename := performRequest(t, handler, http.MethodPatch, "/v1/me/brain/layers/"+inner.ID, "access-token", "application/json", renameBody)
	if rename.Code != http.StatusOK {
		t.Fatalf("rename = %d %s", rename.Code, rename.Body.String())
	}
	var renamed struct {
		Layer struct {
			ID     string `json:"id"`
			Parent struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"parent"`
			Name    string `json:"name"`
			Version int64  `json:"version"`
		} `json:"layer"`
	}
	if err := json.Unmarshal(rename.Body.Bytes(), &renamed); err != nil {
		t.Fatalf("decode rename: %v", err)
	}
	if renamed.Layer.ID != inner.ID || renamed.Layer.Parent != inner.Parent || renamed.Layer.Name != "Renamed" || renamed.Layer.Version != 2 {
		t.Fatalf("renamed layer = %#v; before=%#v", renamed.Layer, inner)
	}
}

func TestHandlerStrictlyValidatesProtocolBeforeMutation(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		target      string
		contentType string
		body        string
		status      int
		code        string
	}{
		{name: "unknown query", method: http.MethodPut, target: "/v1/me/brain?owner=other", status: 400, code: "invalid_request"},
		{name: "put body", method: http.MethodPut, target: "/v1/me/brain", body: `{}`, status: 400, code: "invalid_request"},
		{name: "unknown owner field", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"owner":"other","parent":{"kind":"root","id":"root"},"name":"A"}`, status: 400, code: "invalid_request"},
		{name: "case-mismatched field", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"Parent":{"kind":"root","id":"root"},"name":"A"}`, status: 400, code: "invalid_request"},
		{name: "unknown nested field", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"root","id":"root","owner":"other"},"name":"A"}`, status: 400, code: "invalid_request"},
		{name: "duplicate field", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"root","id":"root"},"name":"A","name":"B"}`, status: 400, code: "invalid_request"},
		{name: "invalid UTF-8", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"root","id":"root"},"name":"` + string([]byte{0xff}) + `"}`, status: 400, code: "invalid_request"},
		{name: "wrong type", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"root","id":"root"},"name":5}`, status: 400, code: "invalid_request"},
		{name: "empty parent ID", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"root","id":""},"name":"A"}`, status: 400, code: "invalid_request"},
		{name: "unknown parent kind", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"brain","id":"root"},"name":"A"}`, status: 400, code: "invalid_request"},
		{name: "trailing JSON", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"root","id":"root"},"name":"A"}{}`, status: 400, code: "invalid_request"},
		{name: "whitespace name", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json", body: `{"parent":{"kind":"root","id":"root"},"name":"  \t "}`, status: 422, code: "invalid_name"},
		{name: "missing media type", method: http.MethodPost, target: "/v1/me/brain/layers", body: `{"parent":{"kind":"root","id":"root"},"name":"A"}`, status: 415, code: "unsupported_media_type"},
		{name: "unsupported charset", method: http.MethodPost, target: "/v1/me/brain/layers", contentType: "application/json; charset=iso-8859-1", body: `{"parent":{"kind":"root","id":"root"},"name":"A"}`, status: 415, code: "unsupported_media_type"},
		{name: "patch empty layer ID", method: http.MethodPatch, target: "/v1/me/brain/layers/", contentType: "application/json", body: `{"name":"A","expected_version":1}`, status: 400, code: "invalid_request"},
		{name: "patch zero version", method: http.MethodPatch, target: "/v1/me/brain/layers/layer", contentType: "application/json", body: `{"name":"A","expected_version":0}`, status: 400, code: "invalid_request"},
		{name: "patch version overflow", method: http.MethodPatch, target: "/v1/me/brain/layers/layer", contentType: "application/json", body: `{"name":"A","expected_version":9223372036854775808}`, status: 400, code: "invalid_request"},
	}
	_, store, _, handler := newTestAPI()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(t, handler, test.method, test.target, "access-token", test.contentType, test.body)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if got := decodeError(t, response).Error.Code; got != test.code {
				t.Fatalf("error code = %q, want %q", got, test.code)
			}
		})
	}
	_, _, inserts, renames, layers := store.snapshot()
	if inserts+renames != 0 || len(layers) != 0 {
		t.Fatalf("invalid requests changed Store: insert=%d rename=%d layers=%#v", inserts, renames, layers)
	}
}

func TestHandlerEnforcesPayloadLimitAndMethodErrors(t *testing.T) {
	_, store, _, handler := newTestAPI()
	tooLarge := `{"parent":{"kind":"root","id":"root"},"name":"` + strings.Repeat("x", 1<<20) + `"}`
	large := performRequest(t, handler, http.MethodPost, "/v1/me/brain/layers", "access-token", "application/json", tooLarge)
	if large.Code != http.StatusRequestEntityTooLarge || decodeError(t, large).Error.Code != "payload_too_large" {
		t.Fatalf("oversize response = %d %s", large.Code, large.Body.String())
	}

	unsupported := performRequest(t, handler, http.MethodDelete, "/v1/me/brain", "access-token", "", "")
	if unsupported.Code != http.StatusMethodNotAllowed || decodeError(t, unsupported).Error.Code != "method_not_allowed" {
		t.Fatalf("method response = %d %s", unsupported.Code, unsupported.Body.String())
	}
	ensures, _, inserts, _, layers := store.snapshot()
	if ensures+inserts != 0 || len(layers) != 0 {
		t.Fatalf("protocol errors changed Store: ensure=%d insert=%d layers=%#v", ensures, inserts, layers)
	}
}

func TestHandlerMapsConflictsAndHidesForeignResources(t *testing.T) {
	_, store, engine, handler := newTestAPI()
	ownerA := brain.Principal("550e8400-e29b-41d4-a716-446655440000")
	ownerB := brain.Principal("550e8400-e29b-41d4-a716-446655440001")
	foreignBrain, err := engine.Bootstrap(context.Background(), ownerB)
	if err != nil {
		t.Fatalf("bootstrap fixture owner B: %v", err)
	}
	foreignLayer, err := engine.CreateLayer(context.Background(), ownerB, brain.RootRef(foreignBrain.Root.ID), "private")
	if err != nil {
		t.Fatalf("create fixture owner B layer: %v", err)
	}
	if string(ownerA) != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatal("test owner fixture drifted")
	}

	_, err = engine.Bootstrap(context.Background(), ownerA)
	if err != nil {
		t.Fatalf("bootstrap owner A: %v", err)
	}
	foreign := performRequest(t, handler, http.MethodPatch, "/v1/me/brain/layers/"+string(foreignLayer.ID), "access-token", "application/json", `{"name":"leak","expected_version":1}`)
	if foreign.Code != http.StatusNotFound || decodeError(t, foreign).Error.Code != "not_found" {
		t.Fatalf("foreign layer response = %d %s", foreign.Code, foreign.Body.String())
	}
	missing := performRequest(t, handler, http.MethodPatch, "/v1/me/brain/layers/unknown-id", "access-token", "application/json", `{"name":"leak","expected_version":1}`)
	if missing.Code != foreign.Code {
		t.Fatalf("foreign and absent statuses differ: %d / %d", foreign.Code, missing.Code)
	}
	foreignError, missingError := decodeError(t, foreign), decodeError(t, missing)
	if foreignError.Error != missingError.Error {
		t.Fatalf("foreign and absent details differ: %#v / %#v", foreignError.Error, missingError.Error)
	}
	if strings.Contains(foreign.Body.String(), string(foreignLayer.ID)) || strings.Contains(foreign.Body.String(), "private") {
		t.Fatalf("foreign resource response disclosed private data: %s", foreign.Body.String())
	}

	_, _, _, renames, layers := store.snapshot()
	if renames != 0 || layers[foreignLayer.ID].Name != "private" {
		t.Fatalf("foreign rename mutated Store: renames=%d layer=%#v", renames, layers[foreignLayer.ID])
	}
}

func TestHandlerRecoversPanicWithSanitizedInternalError(t *testing.T) {
	verifier, _, _, handler := newTestAPI()
	verifier.panicWith = "sensitive internal detail"
	response := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "access-token", "", "")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("panic response status = %d; body=%s", response.Code, response.Body.String())
	}
	envelope := decodeError(t, response)
	if envelope.Error.Code != "internal" || strings.Contains(response.Body.String(), "sensitive internal detail") {
		t.Fatalf("panic response leaked details or wrong code: %s", response.Body.String())
	}
}

func TestConcurrentHTTPBootstrapConverges(t *testing.T) {
	_, store, _, handler := newTestAPI()
	const callers = 8
	start := make(chan struct{})
	results := make(chan string, callers)
	errs := make(chan error, callers)
	var wait sync.WaitGroup
	for i := 0; i < callers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			response := performRequest(t, handler, http.MethodPut, "/v1/me/brain", "access-token", "", "")
			if response.Code != http.StatusOK {
				errs <- fmt.Errorf("bootstrap status %d: %s", response.Code, response.Body.String())
				return
			}
			var payload struct {
				Brain struct {
					ID string `json:"id"`
				} `json:"brain"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				errs <- err
				return
			}
			results <- payload.Brain.ID
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var brainID string
	for result := range results {
		if brainID == "" {
			brainID = result
		} else if result != brainID {
			t.Fatalf("concurrent bootstraps diverged: %q / %q", brainID, result)
		}
	}
	ensures, _, _, _, _ := store.snapshot()
	if ensures != callers {
		t.Fatalf("bootstrap calls reaching Store = %d, want %d convergent calls", ensures, callers)
	}
}

func TestAuthAndBrainHTTPContractIntegration(t *testing.T) {
	const (
		userOneID = "550e8400-e29b-41d4-a716-446655440010"
		userTwoID = "550e8400-e29b-41d4-a716-446655440020"
		tokenOne  = "eyJhbGciOiJIUzI1NiJ9.synthetic.user-one.signature"
		tokenTwo  = "eyJhbGciOiJIUzI1NiJ9.synthetic.user-two.signature"
	)

	userPayload := func(id string) []byte {
		t.Helper()
		payload := struct {
			ID               string `json:"id"`
			Audience         string `json:"aud"`
			Role             string `json:"role"`
			EmailConfirmedAt string `json:"email_confirmed_at"`
			AppMetadata      struct {
				Provider  string   `json:"provider"`
				Providers []string `json:"providers"`
			} `json:"app_metadata"`
			Identities []struct {
				Provider string `json:"provider"`
			} `json:"identities"`
			IsAnonymous bool `json:"is_anonymous"`
		}{
			ID:               id,
			Audience:         "authenticated",
			Role:             "authenticated",
			EmailConfirmedAt: "2026-10-01T12:00:00Z",
			IsAnonymous:      false,
		}
		payload.AppMetadata.Provider = "google"
		payload.AppMetadata.Providers = []string{"google"}
		payload.Identities = []struct {
			Provider string `json:"provider"`
		}{{Provider: "google"}}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal synthetic Auth user: %v", err)
		}
		return body
	}
	userOnePayload := userPayload(userOneID)
	userTwoPayload := userPayload(userTwoID)

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/user" || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Header.Get("Authorization") {
		case "Bearer " + tokenOne:
			_, _ = w.Write(userOnePayload)
		case "Bearer " + tokenTwo:
			_, _ = w.Write(userTwoPayload)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer authServer.Close()

	verifier, err := auth.NewGoTrueVerifier(authServer.URL+"/user", authServer.Client())
	if err != nil {
		t.Fatalf("NewGoTrueVerifier: %v", err)
	}
	store := newFakeStore()
	engine := brain.NewEngine(store, nil)
	apiServer := httptest.NewServer(NewHandler(engine, verifier))
	defer apiServer.Close()
	apiClient := apiServer.Client()

	request := func(method, path, token, body string) (int, []byte) {
		t.Helper()
		apiRequest, err := http.NewRequest(method, apiServer.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatalf("build API request: %v", err)
		}
		if token != "" {
			apiRequest.Header.Set("Authorization", "Bearer "+token)
		}
		if body != "" {
			apiRequest.Header.Set("Content-Type", "application/json")
		}
		response, err := apiClient.Do(apiRequest)
		if err != nil {
			t.Fatalf("send %s %s: %v", method, path, err)
		}
		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read %s %s: %v", method, path, err)
		}
		return response.StatusCode, responseBody
	}

	var brainOne struct {
		Brain struct {
			ID   string `json:"id"`
			Root struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"root"`
		} `json:"brain"`
	}
	status, body := request(http.MethodPut, brainPath, tokenOne, "")
	if status != http.StatusOK {
		t.Fatalf("user one bootstrap = %d %s", status, body)
	}
	if err := json.Unmarshal(body, &brainOne); err != nil {
		t.Fatalf("decode user one Brain: %v", err)
	}
	if brainOne.Brain.ID == "" || brainOne.Brain.Root.ID == "" || brainOne.Brain.Root.Kind != "root" {
		t.Fatalf("incomplete user one Brain: %#v", brainOne)
	}

	status, body = request(http.MethodGet, brainPath, tokenOne, "")
	if status != http.StatusOK {
		t.Fatalf("initial hierarchy = %d %s", status, body)
	}
	var initialHierarchy struct {
		Layers []json.RawMessage `json:"layers"`
	}
	if err := json.Unmarshal(body, &initialHierarchy); err != nil || initialHierarchy.Layers == nil || len(initialHierarchy.Layers) != 0 {
		t.Fatalf("initial hierarchy layers = %#v, err=%v", initialHierarchy.Layers, err)
	}

	type layerResponse struct {
		Layer struct {
			ID     string `json:"id"`
			Parent struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"parent"`
			Name    string `json:"name"`
			Version int64  `json:"version"`
		} `json:"layer"`
	}
	createLayer := func(parentKind, parentID, name string) layerResponse {
		t.Helper()
		input, err := json.Marshal(struct {
			Parent struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"parent"`
			Name string `json:"name"`
		}{
			Parent: struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			}{Kind: parentKind, ID: parentID},
			Name: name,
		})
		if err != nil {
			t.Fatalf("marshal create layer: %v", err)
		}
		status, body := request(http.MethodPost, layersPath, tokenOne, string(input))
		if status != http.StatusCreated {
			t.Fatalf("create %q = %d %s", name, status, body)
		}
		var result layerResponse
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("decode created layer: %v", err)
		}
		if result.Layer.ID == "" {
			t.Fatalf("created layer has no ID: %#v", result)
		}
		return result
	}
	outer := createLayer("root", brainOne.Brain.Root.ID, "outer")
	inner := createLayer("layer", outer.Layer.ID, "inner")
	if outer.Layer.Parent.Kind != "root" || outer.Layer.Parent.ID != brainOne.Brain.Root.ID || inner.Layer.Parent.Kind != "layer" || inner.Layer.Parent.ID != outer.Layer.ID {
		t.Fatalf("nested parent references = outer:%#v inner:%#v", outer.Layer.Parent, inner.Layer.Parent)
	}

	renameInput, err := json.Marshal(struct {
		Name            string `json:"name"`
		ExpectedVersion int64  `json:"expected_version"`
	}{Name: "renamed", ExpectedVersion: inner.Layer.Version})
	if err != nil {
		t.Fatalf("marshal rename: %v", err)
	}
	layerPath := layersPath + "/" + inner.Layer.ID
	status, body = request(http.MethodPatch, layerPath, tokenOne, string(renameInput))
	if status != http.StatusOK {
		t.Fatalf("rename = %d %s", status, body)
	}
	var renamed layerResponse
	if err := json.Unmarshal(body, &renamed); err != nil {
		t.Fatalf("decode renamed layer: %v", err)
	}
	if renamed.Layer.ID != inner.Layer.ID || renamed.Layer.Parent.ID != outer.Layer.ID || renamed.Layer.Name != "renamed" || renamed.Layer.Version != inner.Layer.Version+1 {
		t.Fatalf("renamed layer = %#v", renamed.Layer)
	}
	status, body = request(http.MethodPatch, layerPath, tokenOne, string(renameInput))
	if status != http.StatusConflict {
		t.Fatalf("stale rename = %d %s, want 409", status, body)
	}
	status, body = request(http.MethodGet, brainPath, tokenOne, "")
	if status != http.StatusOK {
		t.Fatalf("hierarchy after stale rename = %d %s", status, body)
	}
	var afterConflict struct {
		Layers []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Version int64  `json:"version"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(body, &afterConflict); err != nil {
		t.Fatalf("decode hierarchy after stale rename: %v", err)
	}
	var childAfterConflict bool
	for _, layer := range afterConflict.Layers {
		if layer.ID == inner.Layer.ID {
			childAfterConflict = layer.Name == "renamed" && layer.Version == inner.Layer.Version+1
		}
	}
	if !childAfterConflict {
		t.Fatalf("stale rename changed the confirmed layer: %#v", afterConflict.Layers)
	}

	status, body = request(http.MethodPut, brainPath, tokenTwo, "")
	if status != http.StatusOK {
		t.Fatalf("user two bootstrap = %d %s", status, body)
	}
	_, _, _, renamesBeforeForeign, _ := store.snapshot()
	status, body = request(http.MethodPatch, layerPath, tokenTwo, string(renameInput))
	if status != http.StatusNotFound {
		t.Fatalf("user two foreign layer response = %d %s, want 404", status, body)
	}
	foreignError := decodeHTTPError(t, body)
	if foreignError.Error.Code != "not_found" || strings.Contains(string(body), inner.Layer.ID) || strings.Contains(string(body), "renamed") {
		t.Fatalf("foreign layer response disclosed state: %s", body)
	}
	_, _, _, renamesAfterForeign, layersAfterForeign := store.snapshot()
	if renamesAfterForeign != renamesBeforeForeign || layersAfterForeign[brain.LayerID(inner.Layer.ID)].Name != "renamed" {
		t.Fatalf("foreign rename had an effect: rename calls=%d→%d layer=%#v", renamesBeforeForeign, renamesAfterForeign, layersAfterForeign[brain.LayerID(inner.Layer.ID)])
	}

	ensureBefore, observeBefore, insertBefore, renameBefore, layersBefore := store.snapshot()
	invalidBody, err := json.Marshal(struct {
		Parent struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		} `json:"parent"`
		Name string `json:"name"`
	}{Parent: struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}{Kind: "root", ID: brainOne.Brain.Root.ID}, Name: "must-not-create"})
	if err != nil {
		t.Fatalf("marshal invalid-token request: %v", err)
	}
	status, body = request(http.MethodPost, layersPath, "invalid-token", string(invalidBody))
	if status != http.StatusUnauthorized || decodeHTTPError(t, body).Error.Code != "unauthenticated" {
		t.Fatalf("invalid token response = %d %s, want 401", status, body)
	}
	ensureAfter, observeAfter, insertAfter, renameAfter, layersAfter := store.snapshot()
	if ensureAfter != ensureBefore || observeAfter != observeBefore || insertAfter != insertBefore || renameAfter != renameBefore || len(layersAfter) != len(layersBefore) {
		t.Fatalf("invalid token reached Brain Engine: before=(%d,%d,%d,%d,%d) after=(%d,%d,%d,%d,%d)", ensureBefore, observeBefore, insertBefore, renameBefore, len(layersBefore), ensureAfter, observeAfter, insertAfter, renameAfter, len(layersAfter))
	}
}

func decodeHTTPError(t *testing.T, body []byte) errorEnvelope {
	t.Helper()
	var result errorEnvelope
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode HTTP error: %v; body=%q", err, body)
	}
	if result.Error.Code == "" || result.Error.Message == "" || result.RequestID == "" {
		t.Fatalf("incomplete HTTP error: %#v", result)
	}
	return result
}

var _ brain.Store = (*fakeStore)(nil)
var _ auth.Verifier = (*fakeVerifier)(nil)

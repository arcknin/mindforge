package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/arcknin/mindforge/src/brain-engine/auth"
	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/arcknin/mindforge/src/brain-engine/features/documents"
)

const (
	brainPath  = "/v1/me/brain"
	layersPath = brainPath + "/layers"
	maxBody    = 1 << 20
)

type handler struct {
	engine     *brain.Engine
	documents  *documents.Engine
	verifier   auth.Verifier
	authFacade *auth.Facade
}

// NewHandler exposes the approved manual Brain operations through a strict
// HTTP boundary. It does not expose infrastructure or health routes.
func NewHandler(engine *brain.Engine, verifier auth.Verifier) http.Handler {
	return NewHandlerWithDocuments(engine, nil, verifier)
}

// NewHandlerWithDocuments composes the approved document API without changing Core contracts.
func NewHandlerWithDocuments(engine *brain.Engine, docEngine *documents.Engine, verifier auth.Verifier) http.Handler {
	return NewHandlerWithAuth(engine, docEngine, verifier, nil)
}

// NewHandlerWithAuth composes the five approved public Auth routes with the
// existing Core/document boundary. Auth routes never reach the Brain Engine.
func NewHandlerWithAuth(engine *brain.Engine, docEngine *documents.Engine, verifier auth.Verifier, facade *auth.Facade) http.Handler {
	h := &handler{engine: engine, documents: docEngine, verifier: verifier, authFacade: facade}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		defer func() {
			if recover() != nil {
				if r != nil && r.URL != nil && (r.URL.EscapedPath() == authPrefix || strings.HasPrefix(r.URL.EscapedPath(), authPrefix+"/")) {
					writeAuthError(w, http.StatusInternalServerError, "internal", requestID)
					return
				}
				writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
			}
		}()
		h.serve(w, r, requestID)
	})
}

func (h *handler) serve(w http.ResponseWriter, r *http.Request, requestID string) {
	if r == nil || r.URL == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return
	}
	if authOp, authStatus, authAllow, recognized := routeAuth(r.Method, r.URL.EscapedPath()); recognized {
		if authStatus != 0 {
			if authAllow != "" {
				w.Header().Set("Allow", authAllow)
			}
			writeAuthProtocolError(w, authStatus, requestID)
			return
		}
		if h.authFacade == nil {
			writeAuthError(w, http.StatusNotFound, "not_found", requestID)
			return
		}
		h.serveAuth(w, r, authOp, requestID)
		return
	}
	docOp, docID, docLayerID, docStatus, docAllow, isDocument := routeDocument(r.Method, r.URL.EscapedPath())
	op, layerID, status, allow := route(r.Method, r.URL.EscapedPath())
	if isDocument {
		status, allow = docStatus, docAllow
		if h.documents == nil {
			status, allow = http.StatusNotFound, ""
		}
	}
	if status != 0 {
		if allow != "" {
			w.Header().Set("Allow", allow)
		}
		writeProtocolError(w, status, requestID)
		return
	}
	if (r.URL.RawQuery != "" || r.URL.ForceQuery) && !(isDocument && docOp == documentList) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return
	}

	token, ok := bearerToken(r.Header)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", requestID)
		return
	}
	if h.verifier == nil {
		writeError(w, http.StatusServiceUnavailable, "auth_unavailable", "Authentication service is unavailable.", requestID)
		return
	}
	principal, err := h.verifier.Verify(r.Context(), token)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthenticated) {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.", requestID)
			return
		}
		writeError(w, http.StatusServiceUnavailable, "auth_unavailable", "Authentication service is unavailable.", requestID)
		return
	}
	if principal == "" {
		writeError(w, http.StatusServiceUnavailable, "auth_unavailable", "Authentication service is unavailable.", requestID)
		return
	}
	if h.engine == nil {
		writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
		return
	}

	if isDocument {
		h.serveDocuments(w, r, principal, docOp, docID, docLayerID, requestID)
		return
	}
	switch op {
	case operationBootstrap:
		if rejectNoBody(w, r, requestID) {
			return
		}
		result, err := h.engine.Bootstrap(r.Context(), principal)
		if err != nil {
			writeBrainError(w, err, requestID)
			return
		}
		brainDTO, ok := toBrainSummary(result.ID, result.Root.ID)
		if !ok {
			writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
			return
		}
		writeJSON(w, http.StatusOK, brainEnvelope{Brain: brainDTO})
	case operationObserve:
		if rejectNoBody(w, r, requestID) {
			return
		}
		result, err := h.engine.Observe(r.Context(), principal)
		if err != nil {
			writeBrainError(w, err, requestID)
			return
		}
		brainDTO, ok := toBrainSummary(result.Brain.ID, result.Root.ID)
		if !ok || result.Brain.Root.ID != result.Root.ID || result.Brain.Root.BrainID != result.Brain.ID {
			writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
			return
		}
		layers := make([]layerDTO, 0, len(result.Layers))
		for _, layer := range result.Layers {
			converted, ok := toLayerDTO(layer)
			if !ok || layer.BrainID != result.Brain.ID {
				writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
				return
			}
			layers = append(layers, converted)
		}
		writeJSON(w, http.StatusOK, hierarchyEnvelope{Brain: brainDTO, Layers: layers})
	case operationCreateLayer:
		var request createLayerRequest
		if decodeRequest(w, r, &request, requestID) {
			return
		}
		if request.Parent == nil || request.Name == nil || request.Parent.ID == "" || !validParentKind(request.Parent.Kind) {
			writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
			return
		}
		parent := brain.NodeRef{Kind: brain.NodeKind(request.Parent.Kind), ID: request.Parent.ID}
		result, err := h.engine.CreateLayer(r.Context(), principal, parent, *request.Name)
		if err != nil {
			writeBrainError(w, err, requestID)
			return
		}
		layer, ok := toLayerDTO(result)
		if !ok {
			writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
			return
		}
		writeJSON(w, http.StatusCreated, layerEnvelope{Layer: layer})
	case operationRenameLayer:
		var request renameLayerRequest
		if decodeRequest(w, r, &request, requestID) {
			return
		}
		if request.Name == nil || request.ExpectedVersion == nil || layerID == "" || *request.ExpectedVersion < 1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
			return
		}
		result, err := h.engine.RenameLayer(r.Context(), principal, brain.LayerID(layerID), brain.LayerVersion(*request.ExpectedVersion), *request.Name)
		if err != nil {
			writeBrainError(w, err, requestID)
			return
		}
		layer, ok := toLayerDTO(result)
		if !ok {
			writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
			return
		}
		writeJSON(w, http.StatusOK, layerEnvelope{Layer: layer})
	default:
		writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
	}
}

type operation uint8

const (
	operationInvalid operation = iota
	operationBootstrap
	operationObserve
	operationCreateLayer
	operationRenameLayer
)

func route(method, escapedPath string) (operation, string, int, string) {
	switch escapedPath {
	case brainPath:
		switch method {
		case http.MethodPut:
			return operationBootstrap, "", 0, ""
		case http.MethodGet:
			return operationObserve, "", 0, ""
		default:
			return operationInvalid, "", http.StatusMethodNotAllowed, "GET, PUT"
		}
	case layersPath:
		if method == http.MethodPost {
			return operationCreateLayer, "", 0, ""
		}
		return operationInvalid, "", http.StatusMethodNotAllowed, http.MethodPost
	}

	prefix := layersPath + "/"
	if !strings.HasPrefix(escapedPath, prefix) {
		return operationInvalid, "", http.StatusNotFound, ""
	}
	if method != http.MethodPatch {
		return operationInvalid, "", http.StatusMethodNotAllowed, http.MethodPatch
	}
	segment := strings.TrimPrefix(escapedPath, prefix)
	if segment == "" || strings.Contains(segment, "/") {
		return operationInvalid, "", http.StatusBadRequest, ""
	}
	decoded, err := url.PathUnescape(segment)
	if err != nil || decoded == "" || strings.Contains(decoded, "/") || strings.ContainsRune(decoded, '\x00') {
		return operationInvalid, "", http.StatusBadRequest, ""
	}
	return operationRenameLayer, decoded, 0, ""
}

func bearerToken(header http.Header) (string, bool) {
	values := header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func rejectNoBody(w http.ResponseWriter, r *http.Request, requestID string) bool {
	if r.ContentLength > maxBody {
		writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large.", requestID)
		return true
	}
	if r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return true
	}
	if r.Body == nil || r.Body == http.NoBody {
		return false
	}
	var probe [1]byte
	n, err := io.ReadFull(r.Body, probe[:])
	if n != 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return true
	}
	if err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return true
	}
	return false
}

type parentRefRequest struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type createLayerRequest struct {
	Parent *parentRefRequest `json:"parent"`
	Name   *string           `json:"name"`
}

type renameLayerRequest struct {
	Name            *string `json:"name"`
	ExpectedVersion *int64  `json:"expected_version"`
}

func validParentKind(kind string) bool {
	return kind == string(brain.NodeRoot) || kind == string(brain.NodeLayer)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, destination any, requestID string) bool {
	return decodeRequestLimit(w, r, destination, requestID, maxBody)
}

func decodeRequestLimit(w http.ResponseWriter, r *http.Request, destination any, requestID string, bodyLimit int64) bool {
	if r.ContentLength > bodyLimit {
		writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large.", requestID)
		return true
	}
	mediaType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	charset := parameters["charset"]
	if err != nil || mediaType != "application/json" || (charset != "" && !strings.EqualFold(charset, "utf-8")) {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json.", requestID)
		return true
	}
	bodyReader := r.Body
	if bodyReader == nil {
		bodyReader = http.NoBody
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, bodyReader, bodyLimit))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large.", requestID)
		} else {
			writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		}
		return true
	}
	if len(body) == 0 || !utf8.Valid(body) || !validExactObjectShape(body, destination) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return true
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return true
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return true
	}
	if hasDuplicateJSONKeys(body) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request is invalid.", requestID)
		return true
	}
	return false
}

func validExactObjectShape(body []byte, destination any) bool {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil || object == nil {
		return false
	}
	switch destination.(type) {
	case *createLayerRequest:
		if !hasExactKeys(object, "parent", "name") {
			return false
		}
		var parent map[string]json.RawMessage
		if err := json.Unmarshal(object["parent"], &parent); err != nil || parent == nil {
			return false
		}
		return hasExactKeys(parent, "kind", "id")
	case *renameLayerRequest:
		return hasExactKeys(object, "name", "expected_version")
	default:
		return validDocumentObjectShape(object, destination)
	}
}

func hasExactKeys(object map[string]json.RawMessage, allowed ...string) bool {
	if len(object) != len(allowed) {
		return false
	}
	for _, key := range allowed {
		if _, ok := object[key]; !ok {
			return false
		}
	}
	return true
}

func hasDuplicateJSONKeys(body []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := consumeJSONValue(decoder, 0); err != nil {
		return true
	}
	return decoder.Decode(new(any)) != io.EOF
}

func consumeJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("JSON nesting is too deep")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid object key")
			}
			if _, duplicate := seen[key]; duplicate {
				return errors.New("duplicate object key")
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("unterminated object")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("unterminated array")
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	return nil
}

type rootRefDTO struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type parentRefDTO struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type brainSummaryDTO struct {
	ID   string     `json:"id"`
	Root rootRefDTO `json:"root"`
}

type brainEnvelope struct {
	Brain brainSummaryDTO `json:"brain"`
}

type hierarchyEnvelope struct {
	Brain  brainSummaryDTO `json:"brain"`
	Layers []layerDTO      `json:"layers"`
}

type layerDTO struct {
	ID      string       `json:"id"`
	Parent  parentRefDTO `json:"parent"`
	Name    string       `json:"name"`
	Version int64        `json:"version"`
}

type layerEnvelope struct {
	Layer layerDTO `json:"layer"`
}

func toBrainSummary(id brain.BrainID, rootID brain.RootID) (brainSummaryDTO, bool) {
	if id == "" || rootID == "" {
		return brainSummaryDTO{}, false
	}
	return brainSummaryDTO{ID: string(id), Root: rootRefDTO{Kind: string(brain.NodeRoot), ID: string(rootID)}}, true
}

func toLayerDTO(layer brain.Layer) (layerDTO, bool) {
	if layer.ID == "" || layer.Parent.ID == "" || !validParentKind(string(layer.Parent.Kind)) || layer.Name == "" || layer.Version == 0 || layer.Version > math.MaxInt64 {
		return layerDTO{}, false
	}
	return layerDTO{
		ID:      string(layer.ID),
		Parent:  parentRefDTO{Kind: string(layer.Parent.Kind), ID: layer.Parent.ID},
		Name:    layer.Name,
		Version: int64(layer.Version),
	}, true
}

func writeBrainError(w http.ResponseWriter, err error, requestID string) {
	switch {
	case errors.Is(err, brain.ErrUnauthenticated):
		// Authentication has already succeeded before the Engine is called. A
		// late domain unauthenticated error cannot promise the no-effect
		// semantics clients rely on when refreshing and replaying after 401.
		writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
	case errors.Is(err, brain.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "The resource changed; reload before retrying.", requestID)
	case errors.Is(err, brain.ErrInvalidName):
		writeError(w, http.StatusUnprocessableEntity, "invalid_name", "Name is invalid.", requestID)
	case errors.Is(err, brain.ErrNotFound), errors.Is(err, brain.ErrForbidden):
		writeError(w, http.StatusNotFound, "not_found", "Resource was not found.", requestID)
	default:
		writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
	}
}

func writeProtocolError(w http.ResponseWriter, status int, requestID string) {
	switch status {
	case http.StatusMethodNotAllowed:
		writeError(w, status, "method_not_allowed", "Method is not allowed.", requestID)
	case http.StatusBadRequest:
		writeError(w, status, "invalid_request", "Request is invalid.", requestID)
	case http.StatusNotFound:
		writeError(w, status, "not_found", "Resource was not found.", requestID)
	default:
		writeError(w, http.StatusInternalServerError, "internal", "Internal error.", requestID)
	}
}

func writeError(w http.ResponseWriter, status int, code, message, requestID string) {
	writeJSON(w, status, struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: code, Message: message},
		RequestID: requestID,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

var requestIDFallback atomic.Uint64

func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("fallback-%d-%d", time.Now().UnixNano(), requestIDFallback.Add(1))
}

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/arcknin/mindforge/src/brain-engine/auth"
)

const (
	authPrefix    = "/v1/auth"
	authAuthorize = authPrefix + "/authorize"
	authCallback  = authPrefix + "/callback"
	authToken     = authPrefix + "/token"
	authUser      = authPrefix + "/user"
	authLogout    = authPrefix + "/logout"
	maxAuthQuery  = 8192
	maxAuthBody   = 16384
)

type authOperation uint8

const (
	authOperationInvalid authOperation = iota
	authOperationAuthorize
	authOperationCallback
	authOperationToken
	authOperationUser
	authOperationLogout
)

func routeAuth(method, escapedPath string) (authOperation, int, string, bool) {
	if escapedPath != authPrefix && !strings.HasPrefix(escapedPath, authPrefix+"/") {
		return authOperationInvalid, 0, "", false
	}
	switch escapedPath {
	case authAuthorize:
		if method != http.MethodGet {
			return authOperationInvalid, http.StatusMethodNotAllowed, http.MethodGet, true
		}
		return authOperationAuthorize, 0, "", true
	case authCallback:
		if method != http.MethodGet {
			return authOperationInvalid, http.StatusMethodNotAllowed, http.MethodGet, true
		}
		return authOperationCallback, 0, "", true
	case authToken:
		if method != http.MethodPost {
			return authOperationInvalid, http.StatusMethodNotAllowed, http.MethodPost, true
		}
		return authOperationToken, 0, "", true
	case authUser:
		if method != http.MethodGet {
			return authOperationInvalid, http.StatusMethodNotAllowed, http.MethodGet, true
		}
		return authOperationUser, 0, "", true
	case authLogout:
		if method != http.MethodPost {
			return authOperationInvalid, http.StatusMethodNotAllowed, http.MethodPost, true
		}
		return authOperationLogout, 0, "", true
	default:
		return authOperationInvalid, http.StatusNotFound, "", true
	}
}

func (h *handler) serveAuth(w http.ResponseWriter, r *http.Request, operation authOperation, requestID string) {
	setAuthHeaders(w)
	if (operation == authOperationToken || operation == authOperationUser || operation == authOperationLogout) && !h.authFacade.ValidateOrigin(r.Header) {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return
	}
	switch operation {
	case authOperationAuthorize:
		query, status := parseAuthQuery(r, map[string]bool{"provider": true, "flow_type": true, "redirect_to": true, "code_challenge": true, "code_challenge_method": true})
		if status != 0 {
			writeAuthProtocolError(w, status, requestID)
			return
		}
		if rejectAuthNoBody(w, r, requestID) {
			return
		}
		location, err := h.authFacade.Authorize(r.Context(), auth.AuthorizeRequest{
			Provider:            query.Get("provider"),
			FlowType:            query.Get("flow_type"),
			RedirectTo:          query.Get("redirect_to"),
			CodeChallenge:       query.Get("code_challenge"),
			CodeChallengeMethod: query.Get("code_challenge_method"),
		})
		if err != nil {
			writeFacadeError(w, err, requestID)
			return
		}
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusFound)
	case authOperationCallback:
		query, status := parseAuthQuery(r, map[string]bool{"state": true, "code": true, "error": true, "error_description": true, "error_uri": true, "scope": true, "authuser": true, "prompt": true, "hd": true, "iss": true})
		if status != 0 {
			writeAuthProtocolError(w, status, requestID)
			return
		}
		_, hasCode := query["code"]
		_, hasError := query["error"]
		if hasCode == hasError || hasCode && query.Get("code") == "" || hasError && query.Get("error") == "" {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
			return
		}
		if rejectAuthNoBody(w, r, requestID) {
			return
		}
		location, err := h.authFacade.Callback(r.Context(), auth.CallbackRequest{
			State:            query.Get("state"),
			Code:             query.Get("code"),
			Error:            query.Get("error"),
			ErrorDescription: query.Get("error_description"),
			ErrorURI:         query.Get("error_uri"),
			Scope:            query.Get("scope"),
			AuthUser:         query.Get("authuser"),
			Prompt:           query.Get("prompt"),
			HD:               query.Get("hd"),
			Issuer:           query.Get("iss"),
		})
		if err != nil {
			writeFacadeError(w, err, requestID)
			return
		}
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusFound)
	case authOperationToken:
		query, status := parseAuthQuery(r, map[string]bool{"grant_type": true})
		if status != 0 {
			writeAuthProtocolError(w, status, requestID)
			return
		}
		request, status := decodeTokenAuthRequest(w, r, query.Get("grant_type"), requestID)
		if status != 0 {
			return
		}
		session, err := h.authFacade.Exchange(r.Context(), request)
		if err != nil {
			writeFacadeError(w, err, requestID)
			return
		}
		writeAuthJSON(w, http.StatusOK, session)
	case authOperationUser:
		if _, status := parseAuthQuery(r, nil); status != 0 {
			writeAuthProtocolError(w, status, requestID)
			return
		}
		if rejectAuthNoBody(w, r, requestID) {
			return
		}
		token, ok := bearerToken(r.Header)
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "unauthenticated", requestID)
			return
		}
		user, err := h.authFacade.User(r.Context(), token)
		if err != nil {
			writeFacadeError(w, err, requestID)
			return
		}
		writeAuthJSON(w, http.StatusOK, user)
	case authOperationLogout:
		query, status := parseAuthQuery(r, map[string]bool{"scope": true})
		if status != 0 {
			writeAuthProtocolError(w, status, requestID)
			return
		}
		if query.Get("scope") != "local" {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
			return
		}
		if decodeLogoutBody(w, r, requestID) {
			return
		}
		token, ok := bearerToken(r.Header)
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "unauthenticated", requestID)
			return
		}
		if err := h.authFacade.Logout(r.Context(), token); err != nil {
			writeFacadeError(w, err, requestID)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeAuthError(w, http.StatusNotFound, "not_found", requestID)
	}
}

func parseAuthQuery(r *http.Request, allowed map[string]bool) (url.Values, int) {
	if r == nil || r.URL == nil {
		return nil, http.StatusBadRequest
	}
	if len(r.URL.RawQuery) > maxAuthQuery {
		return nil, http.StatusRequestURITooLong
	}
	if r.URL.RawQuery == "" && !r.URL.ForceQuery {
		if len(allowed) == 0 {
			return url.Values{}, 0
		}
		return url.Values{}, http.StatusBadRequest
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, http.StatusBadRequest
	}
	if r.URL.RawQuery != "" && len(values) == 0 {
		return nil, http.StatusBadRequest
	}
	for key, entries := range values {
		if allowed == nil || !allowed[key] || len(entries) != 1 {
			return nil, http.StatusBadRequest
		}
	}
	return values, 0
}

func rejectAuthNoBody(w http.ResponseWriter, r *http.Request, requestID string) bool {
	if r == nil {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return true
	}
	if r.ContentLength > maxAuthBody {
		writeAuthError(w, http.StatusRequestEntityTooLarge, "payload_too_large", requestID)
		return true
	}
	if r.ContentLength == 0 && (r.Body == nil || r.Body == http.NoBody) {
		return false
	}
	reader := r.Body
	if reader == nil {
		reader = http.NoBody
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxAuthBody+1))
	if err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return true
	}
	if len(body) > maxAuthBody {
		writeAuthError(w, http.StatusRequestEntityTooLarge, "payload_too_large", requestID)
		return true
	}
	if len(body) > 0 {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return true
	}
	return false
}

func decodeTokenAuthRequest(w http.ResponseWriter, r *http.Request, grantType, requestID string) (auth.TokenRequest, int) {
	if r.ContentLength > maxAuthBody {
		writeAuthError(w, http.StatusRequestEntityTooLarge, "payload_too_large", requestID)
		return auth.TokenRequest{}, http.StatusRequestEntityTooLarge
	}
	mediaType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" || (parameters["charset"] != "" && !strings.EqualFold(parameters["charset"], "utf-8")) {
		writeAuthError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", requestID)
		return auth.TokenRequest{}, http.StatusUnsupportedMediaType
	}
	reader := r.Body
	if reader == nil {
		reader = http.NoBody
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxAuthBody+1))
	if err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return auth.TokenRequest{}, http.StatusBadRequest
	}
	if len(body) > maxAuthBody {
		writeAuthError(w, http.StatusRequestEntityTooLarge, "payload_too_large", requestID)
		return auth.TokenRequest{}, http.StatusRequestEntityTooLarge
	}
	if len(body) == 0 || !utf8.Valid(body) || hasDuplicateJSONKeys(body) {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return auth.TokenRequest{}, http.StatusBadRequest
	}
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&object); err != nil || object == nil || decoder.Decode(new(any)) != io.EOF {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return auth.TokenRequest{}, http.StatusBadRequest
	}
	request := auth.TokenRequest{GrantType: grantType}
	if grantType == "pkce" {
		if !hasExactKeys(object, "auth_code", "code_verifier") || !decodeStringField(object["auth_code"], &request.AuthCode) || !decodeStringField(object["code_verifier"], &request.CodeVerifier) {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
			return auth.TokenRequest{}, http.StatusBadRequest
		}
	} else if grantType == "refresh_token" {
		if !hasExactKeys(object, "refresh_token") || !decodeStringField(object["refresh_token"], &request.RefreshToken) {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
			return auth.TokenRequest{}, http.StatusBadRequest
		}
	} else {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return auth.TokenRequest{}, http.StatusBadRequest
	}
	return request, 0
}

func decodeStringField(raw json.RawMessage, destination *string) bool {
	if len(raw) == 0 || destination == nil {
		return false
	}
	return json.Unmarshal(raw, destination) == nil
}

func decodeLogoutBody(w http.ResponseWriter, r *http.Request, requestID string) bool {
	if r.ContentLength == 0 && (r.Body == nil || r.Body == http.NoBody) {
		return false
	}
	if r.ContentLength > maxAuthBody {
		writeAuthError(w, http.StatusRequestEntityTooLarge, "payload_too_large", requestID)
		return true
	}
	mediaType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" || (parameters["charset"] != "" && !strings.EqualFold(parameters["charset"], "utf-8")) {
		writeAuthError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", requestID)
		return true
	}
	reader := r.Body
	if reader == nil {
		reader = http.NoBody
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxAuthBody+1))
	if err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return true
	}
	if len(body) > maxAuthBody {
		writeAuthError(w, http.StatusRequestEntityTooLarge, "payload_too_large", requestID)
		return true
	}
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(body))
	if !utf8.Valid(body) || hasDuplicateJSONKeys(body) || decoder.Decode(&object) != nil || object == nil || len(object) != 0 || decoder.Decode(new(any)) != io.EOF {
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
		return true
	}
	return false
}

func writeFacadeError(w http.ResponseWriter, err error, requestID string) {
	var authErr *auth.AuthError
	if errors.As(err, &authErr) && authErr != nil {
		writeAuthError(w, authErr.Status(), authErr.Code(), requestID)
		return
	}
	writeAuthError(w, http.StatusInternalServerError, "internal", requestID)
}

func writeAuthProtocolError(w http.ResponseWriter, status int, requestID string) {
	switch status {
	case http.StatusMethodNotAllowed:
		writeAuthError(w, status, "method_not_allowed", requestID)
	case http.StatusNotFound:
		writeAuthError(w, status, "not_found", requestID)
	case http.StatusRequestURITooLong:
		writeAuthError(w, http.StatusRequestEntityTooLarge, "payload_too_large", requestID)
	default:
		writeAuthError(w, http.StatusBadRequest, "invalid_request", requestID)
	}
}

func setAuthHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func writeAuthError(w http.ResponseWriter, status int, code, requestID string) {
	setAuthHeaders(w)
	message := map[string]string{
		"invalid_request":        "Request is invalid.",
		"unauthenticated":        "Authentication is required.",
		"not_found":              "Resource was not found.",
		"method_not_allowed":     "Method is not allowed.",
		"payload_too_large":      "Request is too large.",
		"unsupported_media_type": "Content-Type must be application/json.",
		"internal":               "Internal error.",
		"auth_unavailable":       "Authentication service is unavailable.",
	}[code]
	if message == "" {
		message = "Request failed."
	}
	payload := struct {
		Code      string `json:"code"`
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	}{Code: code, ErrorCode: code, Message: message, RequestID: requestID}
	writeAuthJSON(w, status, payload)
}

func writeAuthJSON(w http.ResponseWriter, status int, payload any) {
	setAuthHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/auth"
)

const (
	authTestUserID = "550e8400-e29b-41d4-a716-446655440000"
	authTestState  = "550e8400-e29b-41d4-a716-446655440002"
)

func authTestFacade(t *testing.T, handler http.HandlerFunc) (*auth.Facade, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		handler(w, r)
	}))
	t.Cleanup(upstream.Close)
	facade, err := auth.NewFacade(auth.FacadeConfig{
		UpstreamURL:        upstream.URL,
		CallbackURL:        "http://127.0.0.1:11001/v1/auth/callback",
		RedirectAllowlist:  []string{"mindforge-dev://auth/callback", "http://web-mind.home.arpa/login/callback"},
		GoogleAuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
	}, upstream.Client())
	if err != nil {
		t.Fatalf("NewFacade: %v", err)
	}
	return facade, &calls
}

func authTestRequest(t *testing.T, handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestAuthHandlerRejectsUnknownAndDuplicateQueriesBeforeUpstream(t *testing.T) {
	facade, calls := authTestFacade(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unexpected upstream call", http.StatusInternalServerError)
	})
	handler := NewHandlerWithAuth(nil, nil, facade, facade)
	for _, target := range []string{
		"/v1/auth/authorize?provider=google&flow_type=pkce&redirect_to=mindforge-dev%3A%2F%2Fauth%2Fcallback&code_challenge=" + strings.Repeat("A", 43) + "&code_challenge_method=s256&provider=google",
		"/v1/auth/token?grant_type=pkce&grant_type=refresh_token",
		"/v1/auth/nope",
	} {
		method := http.MethodGet
		if strings.Contains(target, "/token?") {
			method = http.MethodPost
		}
		response := authTestRequest(t, handler, method, target, "")
		if response.Code != http.StatusBadRequest && !strings.HasSuffix(target, "/nope") {
			t.Fatalf("target %s status = %d, want 400", target, response.Code)
		}
		if strings.HasSuffix(target, "/nope") && response.Code != http.StatusNotFound {
			t.Fatalf("unknown route status = %d, want 404", response.Code)
		}
		var envelope map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode %s: %v", target, err)
		}
		if _, nested := envelope["error"]; nested {
			t.Fatalf("Auth error used Core nested envelope: %s", response.Body.String())
		}
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("upstream calls = %d, want 0", got)
	}
}

func TestAuthHandlerAuthorizeAndCallbackUseValidatedRedirects(t *testing.T) {
	facade, _ := authTestFacade(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authorize":
			location := "https://accounts.google.com/o/oauth2/v2/auth?client_id=client&redirect_uri=" + url.QueryEscape("http://127.0.0.1:11001/v1/auth/callback") + "&response_type=code&state=" + authTestState + "&scope=email%20profile&redirect_to=mindforge-dev%3A%2F%2Fauth%2Fcallback"
			w.Header().Set("Location", location)
			w.WriteHeader(http.StatusFound)
		case "/callback":
			w.Header().Set("Location", "mindforge-dev://auth/callback?code=opaque-code&state="+authTestState)
			w.WriteHeader(http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	})
	handler := NewHandlerWithAuth(nil, nil, facade, facade)
	authorizeTarget := "/v1/auth/authorize?provider=google&flow_type=pkce&redirect_to=mindforge-dev%3A%2F%2Fauth%2Fcallback&code_challenge=" + strings.Repeat("A", 43) + "&code_challenge_method=s256"
	authorize := authTestRequest(t, handler, http.MethodGet, authorizeTarget, "")
	if authorize.Code != http.StatusFound || !strings.HasPrefix(authorize.Header().Get("Location"), "https://accounts.google.com/o/oauth2/v2/auth?") {
		t.Fatalf("authorize = %d Location=%q", authorize.Code, authorize.Header().Get("Location"))
	}
	callback := authTestRequest(t, handler, http.MethodGet, "/v1/auth/callback?state="+authTestState+"&code=google-code", "")
	if callback.Code != http.StatusFound || strings.Contains(callback.Header().Get("Location"), "#") || !strings.Contains(callback.Header().Get("Location"), "code=opaque-code") {
		t.Fatalf("callback = %d Location=%q", callback.Code, callback.Header().Get("Location"))
	}
}

func TestAuthHandlerTokenReturnsFlatSanitizedSessionAndUserVerifiesBearer(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).Unix()
	accessToken := jwtWithSessionIDAndExpiry("550e8400-e29b-41d4-a716-446655440001", expiresAt)
	facade, _ := authTestFacade(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			_, _ = fmt.Fprintf(w, `{"access_token":%q,"refresh_token":"next-refresh","token_type":"bearer","expires_in":3600,"expires_at":%d,"provider_token":"provider-secret","user":{"id":%q}}`, accessToken, expiresAt, authTestUserID)
		case "/user":
			_, _ = fmt.Fprintf(w, `{"id":%q,"aud":"authenticated","role":"authenticated","is_anonymous":false,"created_at":"2026-10-01T12:00:00Z","email_confirmed_at":"2026-10-01T12:00:00Z","app_metadata":{"provider":"google","providers":["google"]},"user_metadata":{"name":"Ada","secret":"omit"},"email":"ada@example.com","session_id":"550e8400-e29b-41d4-a716-446655440001"}`, authTestUserID)
		default:
			http.NotFound(w, r)
		}
	})
	handler := NewHandlerWithAuth(nil, nil, facade, facade)
	body := `{"auth_code":"one-time-code","code_verifier":"` + strings.Repeat("A", 43) + `"}`
	response := authTestRequest(t, handler, http.MethodPost, "/v1/auth/token?grant_type=pkce", body)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "provider-secret") || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("token response = %d %s", response.Code, response.Body.String())
	}
	var session auth.Session
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil || session.User.ID != authTestUserID {
		t.Fatalf("session = %#v err=%v", session, err)
	}
	// Build the authenticated request directly because authTestRequest keeps
	// request headers deliberately minimal for malformed-request tests.
	request := httptest.NewRequest(http.MethodGet, "/v1/auth/user", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	userResponse := httptest.NewRecorder()
	handler.ServeHTTP(userResponse, request)
	if userResponse.Code != http.StatusOK || strings.Contains(userResponse.Body.String(), "secret") {
		t.Fatalf("user response = %d %s", userResponse.Code, userResponse.Body.String())
	}
}

func TestAuthHandlerLogoutRequiresSessionIDAndNeverFallsBackToGlobalLogout(t *testing.T) {
	var logoutCalls atomic.Int32
	facade, _ := authTestFacade(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/logout" {
			logoutCalls.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + authTestUserID + `","aud":"authenticated","role":"authenticated","is_anonymous":false,"created_at":"2026-10-01T12:00:00Z","email_confirmed_at":"2026-10-01T12:00:00Z","app_metadata":{"provider":"google","providers":["google"]}}`))
	})
	handler := NewHandlerWithAuth(nil, nil, facade, facade)
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/logout?scope=local", nil)
	request.Header.Set("Authorization", "Bearer opaque-token-without-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || logoutCalls.Load() != 0 {
		t.Fatalf("logout = %d calls=%d body=%s", response.Code, logoutCalls.Load(), response.Body.String())
	}
}

func TestAuthJSONRoutesRejectInvalidOriginBeforeUpstream(t *testing.T) {
	facade, calls := authTestFacade(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unexpected upstream call", http.StatusInternalServerError)
	})
	handler := NewHandlerWithAuth(nil, nil, facade, facade)
	cases := []struct {
		name   string
		method string
		target string
		body   string
		origin []string
	}{
		{name: "token evil", method: http.MethodPost, target: "/v1/auth/token?grant_type=pkce", body: `{"auth_code":"code","code_verifier":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`, origin: []string{"http://evil.example"}},
		{name: "user null", method: http.MethodGet, target: "/v1/auth/user", origin: []string{"null"}},
		{name: "logout duplicate", method: http.MethodPost, target: "/v1/auth/logout?scope=local", origin: []string{"http://web-mind.home.arpa", "http://web-mind.home.arpa"}},
		{name: "user comma list", method: http.MethodGet, target: "/v1/auth/user", origin: []string{"http://web-mind.home.arpa,http://evil.example"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			for _, origin := range tc.origin {
				request.Header.Add("Origin", origin)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body=%s, want 400", response.Code, response.Body.String())
			}
		})
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("upstream calls = %d, want 0 for rejected origins", got)
	}
}

func TestAuthJSONRouteAllowsAbsentOrExactConfiguredWebOrigin(t *testing.T) {
	facade, calls := authTestFacade(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	handler := NewHandlerWithAuth(nil, nil, facade, facade)
	for _, origin := range []string{"", "http://web-mind.home.arpa"} {
		request := httptest.NewRequest(http.MethodPost, "/v1/auth/token?grant_type=pkce", strings.NewReader(`{"auth_code":"code","code_verifier":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`))
		request.Header.Set("Content-Type", "application/json")
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("origin %q status = %d body=%s, want upstream 503", origin, response.Code, response.Body.String())
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}
}

func jwtWithSessionID(sessionID string) string {
	return jwtWithSessionIDAndExpiry(sessionID, time.Now().Add(time.Hour).Unix())
}

func jwtWithSessionIDAndExpiry(sessionID string, expiresAt int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + authTestUserID + `","exp":` + fmt.Sprint(expiresAt) + `,"session_id":"` + sessionID + `"}`))
	return header + "." + payload + ".signature"
}

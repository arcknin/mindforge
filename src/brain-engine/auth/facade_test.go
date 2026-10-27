package auth

import (
	"context"
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
)

const facadeCallback = "http://127.0.0.1:11001/v1/auth/callback"

func TestFacadeAuthorizeRejectsUnapprovedProviderAndRedirectBeforeUpstream(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()

	facade, err := NewFacade(FacadeConfig{
		UpstreamURL:        upstream.URL,
		CallbackURL:        facadeCallback,
		RedirectAllowlist:  []string{"mindforge-dev://auth/callback"},
		GoogleAuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
	}, upstream.Client())
	if err != nil {
		t.Fatalf("NewFacade: %v", err)
	}

	_, err = facade.Authorize(context.Background(), AuthorizeRequest{
		Provider:            "github",
		FlowType:            "pkce",
		RedirectTo:          "https://attacker.example/callback",
		CodeChallenge:       strings.Repeat("A", 43),
		CodeChallengeMethod: "plain",
	})
	if !IsCode(err, "invalid_request") {
		t.Fatalf("Authorize error = %v, want invalid_request", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("upstream calls = %d, want 0", got)
	}
}

func TestFacadeAuthorizeReturnsOnlyValidatedGoogleLocation(t *testing.T) {
	var received url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authorize" {
			t.Errorf("upstream path = %q, want /authorize", r.URL.Path)
		}
		received = r.URL.Query()
		w.Header().Set("Location", "https://accounts.google.com/o/oauth2/v2/auth?client_id=client&redirect_uri="+url.QueryEscape(facadeCallback)+"&response_type=code&state=550e8400-e29b-41d4-a716-446655440002&scope=openid%20email%20profile&redirect_to=mindforge-dev%3A%2F%2Fauth%2Fcallback")
		w.WriteHeader(http.StatusFound)
	}))
	defer upstream.Close()

	facade, err := NewFacade(FacadeConfig{
		UpstreamURL:        upstream.URL,
		CallbackURL:        facadeCallback,
		RedirectAllowlist:  []string{"mindforge-dev://auth/callback"},
		GoogleAuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
	}, upstream.Client())
	if err != nil {
		t.Fatalf("NewFacade: %v", err)
	}

	location, err := facade.Authorize(context.Background(), AuthorizeRequest{
		Provider:            "google",
		FlowType:            "pkce",
		RedirectTo:          "mindforge-dev://auth/callback",
		CodeChallenge:       strings.Repeat("A", 43),
		CodeChallengeMethod: "s256",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if !strings.HasPrefix(location, "https://accounts.google.com/o/oauth2/v2/auth?") || strings.Contains(location, "provider=") {
		t.Fatalf("Location = %q, want validated Google authorize URL", location)
	}
	if received.Get("provider") != "google" || received.Get("redirect_to") != "mindforge-dev://auth/callback" || received.Get("code_challenge") != strings.Repeat("A", 43) || received.Get("code_challenge_method") != "s256" || received.Get("flow_type") != "" {
		t.Fatalf("upstream query = %v, want restricted Google PKCE query", received)
	}
}

func TestFacadeTokenProjectsUserAndDropsProviderCredentials(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).Unix()
	accessToken := facadeJWT(testUserID, expiresAt, "550e8400-e29b-41d4-a716-446655440001")
	refreshToken := "refresh-token"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			if r.URL.Query().Get("grant_type") != "pkce" {
				t.Errorf("grant_type = %q, want pkce", r.URL.Query().Get("grant_type"))
			}
			_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","refresh_token":"` + refreshToken + `","token_type":"bearer","expires_in":3600,"expires_at":` + fmt.Sprint(expiresAt) + `,"provider_token":"google-secret","user":{"id":"550e8400-e29b-41d4-a716-446655440000","aud":"authenticated","role":"authenticated","is_anonymous":false,"created_at":"2026-10-01T12:00:00Z","email_confirmed_at":"2026-10-01T12:00:00Z","app_metadata":{"provider":"google","providers":["google"]},"user_metadata":{"name":"Ada","full_name":"Ada Lovelace","secret":"drop-me"},"email":"ada@example.com"}}`))
		case "/user":
			if r.Header.Get("Authorization") != "Bearer "+accessToken {
				t.Errorf("verification Authorization = %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","aud":"authenticated","role":"authenticated","is_anonymous":false,"created_at":"2026-10-01T12:00:00Z","email_confirmed_at":"2026-10-01T12:00:00Z","app_metadata":{"provider":"google","providers":["google"]},"user_metadata":{"name":"Ada","full_name":"Ada Lovelace","secret":"drop-me"},"email":"ada@example.com","session_id":"550e8400-e29b-41d4-a716-446655440001"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	facade, err := NewFacade(FacadeConfig{UpstreamURL: upstream.URL, CallbackURL: facadeCallback, RedirectAllowlist: []string{"mindforge-dev://auth/callback"}, GoogleAuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth"}, upstream.Client())
	if err != nil {
		t.Fatalf("NewFacade: %v", err)
	}
	session, err := facade.Exchange(context.Background(), TokenRequest{GrantType: "pkce", AuthCode: "code", CodeVerifier: strings.Repeat("A", 43)})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	encoded, _ := json.Marshal(session)
	body := string(encoded)
	for _, forbidden := range []string{"provider_token", "secret", "identities"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("session leaked %q: %s", forbidden, body)
		}
	}
	if session.AccessToken != accessToken || session.RefreshToken != refreshToken || session.User.ID != testUserID || session.User.AppMetadata.Provider != "google" {
		t.Fatalf("session projection = %#v", session)
	}
}

func TestFacadeExchangeMapsPKCEFlowStateFailuresToInvalidRequest(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusUnprocessableEntity} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/token" {
					w.WriteHeader(status)
					return
				}
				http.NotFound(w, r)
			}))
			defer upstream.Close()
			facade, err := NewFacade(FacadeConfig{UpstreamURL: upstream.URL, CallbackURL: facadeCallback, RedirectAllowlist: []string{"mindforge-dev://auth/callback"}}, upstream.Client())
			if err != nil {
				t.Fatalf("NewFacade: %v", err)
			}
			_, err = facade.Exchange(context.Background(), TokenRequest{GrantType: "pkce", AuthCode: "code", CodeVerifier: strings.Repeat("A", 43)})
			if !IsCode(err, "invalid_request") {
				t.Fatalf("Exchange error = %v, want invalid_request", err)
			}
		})
	}
}

func TestFacadeExchangeRejectsNonJSONTokenResponse(t *testing.T) {
	var userCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			userCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"id":%q,"aud":"authenticated","role":"authenticated","is_anonymous":false,"created_at":"2026-10-01T12:00:00Z","email_confirmed_at":"2026-10-01T12:00:00Z","app_metadata":{"provider":"google","providers":["google"]}}`, testUserID)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"opaque","refresh_token":"refresh","token_type":"bearer","expires_in":3600,"expires_at":9999999999,"user":{"id":"` + testUserID + `"}}`))
	}))
	defer upstream.Close()
	facade, err := NewFacade(FacadeConfig{UpstreamURL: upstream.URL, CallbackURL: facadeCallback, RedirectAllowlist: []string{"mindforge-dev://auth/callback"}}, upstream.Client())
	if err != nil {
		t.Fatalf("NewFacade: %v", err)
	}
	_, err = facade.Exchange(context.Background(), TokenRequest{GrantType: "pkce", AuthCode: "code", CodeVerifier: strings.Repeat("A", 43)})
	if !IsCode(err, "auth_unavailable") {
		t.Fatalf("Exchange error = %v, want auth_unavailable", err)
	}
	if userCalls.Load() != 0 {
		t.Fatalf("user verification calls = %d, want 0 for non-JSON token response", userCalls.Load())
	}
}

func TestFacadeExchangeRequiresJWTSubjectAndExpiryToMatchSessionResponse(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).Unix()
	for name, accessToken := range map[string]string{
		"subject mismatch": facadeJWT("550e8400-e29b-41d4-a716-446655440001", expiresAt, "550e8400-e29b-41d4-a716-446655440001"),
		"expiry mismatch":  facadeJWT(testUserID, expiresAt+1, "550e8400-e29b-41d4-a716-446655440001"),
	} {
		t.Run(name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/token":
					_, _ = fmt.Fprintf(w, `{"access_token":%q,"refresh_token":"refresh","token_type":"bearer","expires_in":3600,"expires_at":%d,"user":{"id":%q}}`, accessToken, expiresAt, testUserID)
				case "/user":
					_, _ = fmt.Fprintf(w, `{"id":%q,"aud":"authenticated","role":"authenticated","is_anonymous":false,"created_at":"2026-10-01T12:00:00Z","email_confirmed_at":"2026-10-01T12:00:00Z","app_metadata":{"provider":"google","providers":["google"]}}`, testUserID)
				default:
					http.NotFound(w, r)
				}
			}))
			defer upstream.Close()
			facade, err := NewFacade(FacadeConfig{UpstreamURL: upstream.URL, CallbackURL: facadeCallback, RedirectAllowlist: []string{"mindforge-dev://auth/callback"}}, upstream.Client())
			if err != nil {
				t.Fatalf("NewFacade: %v", err)
			}
			_, err = facade.Exchange(context.Background(), TokenRequest{GrantType: "pkce", AuthCode: "code", CodeVerifier: strings.Repeat("A", 43)})
			if !IsCode(err, "auth_unavailable") {
				t.Fatalf("Exchange error = %v, want auth_unavailable", err)
			}
		})
	}
}

func TestFacadeCallbackRejectsTokenAndUnknownFragmentFields(t *testing.T) {
	for name, fragment := range map[string]string{
		"provider token":   "provider_token=secret",
		"provider refresh": "provider_refresh_token=secret",
		"nonempty sb":      "error=access_denied&sb=marker",
		"duplicate sb":     "error=access_denied&sb=&sb=",
		"unknown":          "unexpected=value",
	} {
		t.Run(name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "mindforge-dev://auth/callback?error=access_denied&error_code=provider_error#"+fragment)
				w.WriteHeader(http.StatusFound)
			}))
			defer upstream.Close()
			facade, err := NewFacade(FacadeConfig{UpstreamURL: upstream.URL, CallbackURL: facadeCallback, RedirectAllowlist: []string{"mindforge-dev://auth/callback"}}, upstream.Client())
			if err != nil {
				t.Fatalf("NewFacade: %v", err)
			}
			_, err = facade.Callback(context.Background(), CallbackRequest{State: "550e8400-e29b-41d4-a716-446655440002", Error: "access_denied"})
			if !IsCode(err, "auth_unavailable") {
				t.Fatalf("Callback error = %v, want auth_unavailable", err)
			}
		})
	}
}

func TestFacadeCallbackAcceptsEmptyGoTrueSBMarkerAndReturnsQueryOnly(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "mindforge-dev://auth/callback?error=access_denied#error=access_denied&sb=")
		w.WriteHeader(http.StatusFound)
	}))
	defer upstream.Close()
	facade, err := NewFacade(FacadeConfig{UpstreamURL: upstream.URL, CallbackURL: facadeCallback, RedirectAllowlist: []string{"mindforge-dev://auth/callback"}}, upstream.Client())
	if err != nil {
		t.Fatalf("NewFacade: %v", err)
	}
	location, err := facade.Callback(context.Background(), CallbackRequest{State: "550e8400-e29b-41d4-a716-446655440002", Error: "access_denied"})
	if err != nil {
		t.Fatalf("Callback: %v", err)
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse sanitized callback: %v", err)
	}
	if parsed.Fragment != "" || parsed.Query().Get("error") != "access_denied" || parsed.Query().Get("sb") != "" {
		t.Fatalf("sanitized callback = %q, want query-only access_denied without sb", location)
	}
}

func TestAuthOperationContextHasExplicitTenSecondBudget(t *testing.T) {
	ctx, cancel := newAuthOperationContext(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("operation context has no deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 9*time.Second || remaining > authOperationTimeout {
		t.Fatalf("operation budget = %v, want just under 10 seconds", remaining)
	}
}

func facadeJWT(subject string, expiry int64, sessionID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := fmt.Sprintf(`{"sub":%q,"exp":%d,"session_id":%q}`, subject, expiry, sessionID)
	return header + "." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".signature"
}

func TestFacadeLogoutRejectsBearerWithoutNonzeroSessionBeforeUpstream(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(confirmedGoogleUser()))
	}))
	defer upstream.Close()

	facade, err := NewFacade(FacadeConfig{UpstreamURL: upstream.URL, CallbackURL: facadeCallback, RedirectAllowlist: []string{"mindforge-dev://auth/callback"}, GoogleAuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth"}, upstream.Client())
	if err != nil {
		t.Fatalf("NewFacade: %v", err)
	}
	if err := facade.Logout(context.Background(), "access-token"); !IsCode(err, "unauthenticated") {
		t.Fatalf("Logout error = %v, want unauthenticated", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("upstream calls = %d, want 0 for missing session_id", got)
	}
}

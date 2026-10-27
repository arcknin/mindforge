package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testUserID = "550e8400-e29b-41d4-a716-446655440000"

type deadlineTransport struct {
	t        *testing.T
	delegate http.RoundTripper
	calls    *atomic.Int32
}

func (transport deadlineTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	transport.t.Helper()
	deadline, ok := request.Context().Deadline()
	if !ok || time.Until(deadline) > 3*time.Second || time.Until(deadline) < 2*time.Second {
		transport.t.Errorf("upstream request deadline = %v, want approximately 3 seconds", deadline)
	}
	return transport.delegate.RoundTrip(request)
}

func confirmedGoogleUser() string {
	return fmt.Sprintf(`{"id":%q,"aud":"authenticated","role":"authenticated","email_confirmed_at":"2026-10-01T12:00:00Z","confirmed_at":null,"app_metadata":{"provider":"google","providers":["google"]},"identities":[{"provider":"google"}],"is_anonymous":false}`, testUserID)
}

func TestVerifyForwardsBearerToFixedUserEndpointAndAcceptsConfirmedGoogleUser(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/user" || r.URL.RawQuery != "" {
			t.Errorf("request = %s %s, want GET /user without query", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-access-token" {
			t.Errorf("Authorization = %q, want forwarded bearer", got)
		}
		if got := strings.ToLower(r.Header.Get("Cache-Control")); !strings.Contains(got, "no-cache") && !strings.Contains(got, "no-store") {
			t.Errorf("Cache-Control = %q, want a cache bypass directive", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(confirmedGoogleUser()))
	}))
	defer server.Close()

	client := server.Client()
	var transportCalls atomic.Int32
	client.Transport = deadlineTransport{t: t, delegate: client.Transport, calls: &transportCalls}
	verifier, err := NewGoTrueVerifier(server.URL+"/user", client)
	if err != nil {
		t.Fatalf("NewGoTrueVerifier: %v", err)
	}

	for i := 0; i < 2; i++ {
		principal, err := verifier.Verify(context.Background(), "secret-access-token")
		if err != nil {
			t.Fatalf("Verify call %d: %v", i, err)
		}
		if string(principal) != testUserID {
			t.Fatalf("principal = %q, want Auth user ID", principal)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("Auth calls = %d, want one uncached call per verification", got)
	}
	if got := transportCalls.Load(); got != 2 {
		t.Fatalf("injected transport calls = %d, want 2; constructor replaced caller transport", got)
	}
}

func TestDefaultVerifierClonesTransportWithoutAmbientProxy(t *testing.T) {
	original := http.DefaultTransport
	transport, ok := original.(*http.Transport)
	if !ok {
		t.Skip("default transport is a custom RoundTripper")
	}
	proxyDefault := transport.Clone()
	proxyDefault.Proxy = http.ProxyFromEnvironment
	http.DefaultTransport = proxyDefault
	t.Cleanup(func() { http.DefaultTransport = original })

	verifier, err := NewGoTrueVerifier("http://auth.internal.test/user", nil)
	if err != nil {
		t.Fatalf("NewGoTrueVerifier: %v", err)
	}
	cloned, ok := verifier.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("default transport type = %T, want *http.Transport", verifier.client.Transport)
	}
	if cloned == proxyDefault || cloned.Proxy != nil {
		t.Fatal("default verifier did not clone the transport with proxy disabled")
	}
	if proxyDefault.Proxy == nil {
		t.Fatal("constructor mutated the process default transport")
	}
}

func TestNewGoTrueVerifierRequiresAFullUserURL(t *testing.T) {
	for _, endpoint := range []string{
		"",
		"/user",
		"http://user:password@example.test/user",
		"http://example.test/",
		"http://example.test/user?next=/admin",
		"http://example.test/user#fragment",
		"ftp://example.test/user",
	} {
		t.Run(fmt.Sprintf("%q", endpoint), func(t *testing.T) {
			if _, err := NewGoTrueVerifier(endpoint, nil); err == nil {
				t.Fatal("NewGoTrueVerifier accepted an invalid configured endpoint")
			}
		})
	}
}

func TestVerifyRejectsEmptyTokenWithoutCallingAuth(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	verifier, err := NewGoTrueVerifier(server.URL+"/user", server.Client())
	if err != nil {
		t.Fatalf("NewGoTrueVerifier: %v", err)
	}

	if _, err := verifier.Verify(context.Background(), " \t"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Verify error = %v, want ErrUnauthenticated", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("Auth calls = %d, want 0", got)
	}
}

func TestVerifyMapsAuthRejectionsToUnauthenticated(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			verifier, err := NewGoTrueVerifier(server.URL+"/user", server.Client())
			if err != nil {
				t.Fatalf("NewGoTrueVerifier: %v", err)
			}
			if _, err := verifier.Verify(context.Background(), "rejected-token"); !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("Verify error = %v, want ErrUnauthenticated", err)
			}
		})
	}
}

func TestVerifyRejectsIneligibleButWellFormedUsersAsUnauthenticated(t *testing.T) {
	tests := map[string]func(string) string{
		"email-only": func(body string) string {
			body = strings.Replace(body, `"provider":"google"`, `"provider":"email"`, 1)
			body = strings.Replace(body, `"providers":["google"]`, `"providers":["email"]`, 1)
			return strings.Replace(body, `"provider":"google"`, `"provider":"email"`, 1)
		},
		"unconfirmed": func(body string) string {
			return strings.Replace(body, `"email_confirmed_at":"2026-10-01T12:00:00Z"`, `"email_confirmed_at":null`, 1)
		},
		"anonymous": func(body string) string {
			return strings.Replace(body, `"is_anonymous":false`, `"is_anonymous":true`, 1)
		},
		"wrong-audience": func(body string) string { return strings.Replace(body, `"aud":"authenticated"`, `"aud":"anon"`, 1) },
		"wrong-role": func(body string) string {
			return strings.Replace(body, `"role":"authenticated"`, `"role":"service_role"`, 1)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			payload := mutate(confirmedGoogleUser())
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(payload))
			}))
			defer server.Close()
			verifier, err := NewGoTrueVerifier(server.URL+"/user", server.Client())
			if err != nil {
				t.Fatalf("NewGoTrueVerifier: %v", err)
			}
			if _, err := verifier.Verify(context.Background(), "valid-token"); !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("Verify error = %v, want ErrUnauthenticated", err)
			}
		})
	}
}

func TestVerifyAcceptsConfirmedAtWhenEmailConfirmationIsNull(t *testing.T) {
	payload := strings.Replace(confirmedGoogleUser(), `"email_confirmed_at":"2026-10-01T12:00:00Z"`, `"email_confirmed_at":null`, 1)
	payload = strings.Replace(payload, `"confirmed_at":null`, `"confirmed_at":"2026-10-01T12:00:00Z"`, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()
	verifier, err := NewGoTrueVerifier(server.URL+"/user", server.Client())
	if err != nil {
		t.Fatalf("NewGoTrueVerifier: %v", err)
	}
	principal, err := verifier.Verify(context.Background(), "valid-token")
	if err != nil || string(principal) != testUserID {
		t.Fatalf("Verify = %q, %v; want Auth user ID", principal, err)
	}
}

func TestVerifyTreatsMalformedOrUnavailableAuthResponsesAsUnavailable(t *testing.T) {
	tests := map[string]func(http.ResponseWriter, *http.Request){
		"redirect": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "/followed")
			w.WriteHeader(http.StatusFound)
		},
		"server-error":   func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) },
		"malformed-json": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{")) },
		"invalid-user-id": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Replace(confirmedGoogleUser(), testUserID, "not-a-uuid", 1)))
		},
		"missing-audience": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Replace(confirmedGoogleUser(), `,"aud":"authenticated"`, "", 1)))
		},
		"malformed-confirmation-time": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Replace(confirmedGoogleUser(), "2026-10-01T12:00:00Z", "yesterday", 1)))
		},
		"duplicate-user-id": func(w http.ResponseWriter, _ *http.Request) {
			payload := strings.Replace(confirmedGoogleUser(), `"id":"`+testUserID+`",`, `"id":"`+testUserID+`","id":"`+testUserID+`",`, 1)
			_, _ = w.Write([]byte(payload))
		},
		"invalid-utf8": func(w http.ResponseWriter, _ *http.Request) {
			payload := strings.Replace(confirmedGoogleUser(), "2026-10-01", "2026-"+string([]byte{0xff}), 1)
			_, _ = w.Write([]byte(payload))
		},
		"missing-anonymous-flag": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Replace(confirmedGoogleUser(), `,"is_anonymous":false`, "", 1)))
		},
	}
	for name, respond := range tests {
		t.Run(name, func(t *testing.T) {
			var followed atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/followed" {
					followed.Add(1)
					_, _ = w.Write([]byte(confirmedGoogleUser()))
					return
				}
				respond(w, r)
			}))
			defer server.Close()
			verifier, err := NewGoTrueVerifier(server.URL+"/user", server.Client())
			if err != nil {
				t.Fatalf("NewGoTrueVerifier: %v", err)
			}
			if _, err := verifier.Verify(context.Background(), "valid-token"); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("Verify error = %v, want ErrUnavailable", err)
			}
			if got := followed.Load(); got != 0 {
				t.Fatalf("redirect follow count = %d, want 0", got)
			}
		})
	}
}

func TestVerifyMapsTransportFailureToUnavailableWithoutLeakingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := server.URL + "/user"
	server.Close()
	verifier, err := NewGoTrueVerifier(endpoint, server.Client())
	if err != nil {
		t.Fatalf("NewGoTrueVerifier: %v", err)
	}

	const token = "do-not-leak-this-token"
	if _, err := verifier.Verify(context.Background(), token); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Verify error = %v, want ErrUnavailable", err)
	} else if strings.Contains(err.Error(), token) {
		t.Fatalf("Verify error leaked token: %v", err)
	}
}

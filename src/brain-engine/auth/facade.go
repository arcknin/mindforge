package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
)

const (
	defaultGoogleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	authOperationTimeout      = 10 * time.Second
	upstreamTimeout           = 5 * time.Second
	maxUpstreamResponseSize   = 1 << 20
	maxAuthCodeBytes          = 4096
	maxRefreshTokenBytes      = 8192
)

// FacadeConfig describes the private, fixed GoTrue connection and the exact
// browser/native callback destinations accepted by the public API.
type FacadeConfig struct {
	UpstreamURL        string
	CallbackURL        string
	RedirectAllowlist  []string
	GoogleAuthorizeURL string
}

// Facade is the intentionally small public Auth surface. It is not a proxy:
// each method validates its own input, selects one upstream operation, and
// returns a sanitized projection.
type Facade struct {
	config   facadeConfig
	client   *http.Client
	verifier *GoTrueVerifier
}

type facadeConfig struct {
	upstreamBase       string
	callbackURL        string
	redirectAllowlist  map[string]struct{}
	webOrigin          string
	googleAuthorizeURL *url.URL
}

// NewFacade constructs a Google-only Auth facade against one fixed GoTrue
// origin. The caller must provide a private server-side upstream URL.
func NewFacade(config FacadeConfig, client *http.Client) (*Facade, error) {
	base, err := validateUpstreamBase(config.UpstreamURL)
	if err != nil {
		return nil, err
	}
	callback, err := validateCallbackURL(config.CallbackURL)
	if err != nil {
		return nil, err
	}
	allowlist := make(map[string]struct{}, len(config.RedirectAllowlist))
	webOrigin := ""
	if len(config.RedirectAllowlist) == 0 {
		return nil, errors.New("auth: redirect allowlist is required")
	}
	for _, redirect := range config.RedirectAllowlist {
		validated, err := validateFinalRedirect(redirect)
		if err != nil {
			return nil, err
		}
		allowlist[validated] = struct{}{}
		parsedRedirect, _ := url.Parse(validated)
		if parsedRedirect.Scheme == "http" || parsedRedirect.Scheme == "https" {
			webOrigin = parsedRedirect.Scheme + "://" + parsedRedirect.Host
		}
	}
	googleURL := config.GoogleAuthorizeURL
	if googleURL == "" {
		googleURL = defaultGoogleAuthorizeURL
	}
	parsedGoogle, err := url.Parse(googleURL)
	if err != nil || parsedGoogle.Scheme != "https" || parsedGoogle.Hostname() != "accounts.google.com" || parsedGoogle.Host != "accounts.google.com" || parsedGoogle.Path != "/o/oauth2/v2/auth" || parsedGoogle.RawQuery != "" || parsedGoogle.Fragment != "" || parsedGoogle.User != nil {
		return nil, errors.New("auth: invalid configured Google authorize endpoint")
	}

	transportClient := cloneSafeClient(client)
	verifier, err := NewGoTrueVerifier(base+"/user", transportClient)
	if err != nil {
		return nil, err
	}
	return &Facade{
		config: facadeConfig{
			upstreamBase:       base,
			callbackURL:        callback,
			redirectAllowlist:  allowlist,
			webOrigin:          webOrigin,
			googleAuthorizeURL: parsedGoogle,
		},
		client:   transportClient,
		verifier: verifier,
	}, nil
}

func newAuthOperationContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, authOperationTimeout)
}

// ValidateOrigin accepts no Origin for native traffic and exactly the
// configured Web origin for browser JSON requests. It never consults Host or
// forwarding headers.
func (f *Facade) ValidateOrigin(header http.Header) bool {
	if f == nil {
		return false
	}
	values := header.Values("Origin")
	if len(values) == 0 {
		return true
	}
	if len(values) != 1 || f.config.webOrigin == "" {
		return false
	}
	origin := values[0]
	if origin == "" || origin == "null" || strings.TrimSpace(origin) != origin || strings.ContainsAny(origin, "\r\n\x00") {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" || parsed.Opaque != "" {
		return false
	}
	return origin == f.config.webOrigin
}

func validateUpstreamBase(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("auth: invalid configured upstream")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("auth: invalid configured upstream path")
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func validateCallbackURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" || parsed.Host != "127.0.0.1:11001" || parsed.Path != "/v1/auth/callback" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" || parsed.User != nil {
		return "", errors.New("auth: invalid configured callback URL")
	}
	return parsed.String(), nil
}

func validateFinalRedirect(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" || parsed.Path == "" {
		return "", errors.New("auth: invalid configured redirect")
	}
	if parsed.Scheme != "mindforge-dev" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("auth: invalid configured redirect")
	}
	if parsed.Scheme == "mindforge-dev" {
		if parsed.Host != "auth" || parsed.Path != "/callback" {
			return "", errors.New("auth: invalid configured redirect")
		}
	} else if parsed.Hostname() == "" || parsed.Host != "web-mind.home.arpa" || parsed.Path != "/login/callback" {
		return "", errors.New("auth: invalid configured redirect")
	}
	return parsed.String(), nil
}

func cloneSafeClient(client *http.Client) *http.Client {
	copyClient := http.Client{}
	if client != nil {
		copyClient = *client
	}
	if copyClient.Transport == nil {
		copyClient.Transport = defaultTransportWithoutProxy()
	} else if transport, ok := copyClient.Transport.(*http.Transport); ok {
		cloned := transport.Clone()
		cloned.Proxy = nil
		copyClient.Transport = cloned
	}
	copyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	copyClient.Jar = nil
	copyClient.Timeout = 0
	return &copyClient
}

// Verify implements the Core auth.Verifier interface.
func (f *Facade) Verify(ctx context.Context, accessToken string) (brain.Principal, error) {
	if f == nil || f.verifier == nil {
		return "", ErrUnavailable
	}
	return f.verifier.Verify(ctx, accessToken)
}

// AuthorizeRequest is the allowlisted input for Google PKCE initiation.
type AuthorizeRequest struct {
	Provider            string
	FlowType            string
	RedirectTo          string
	CodeChallenge       string
	CodeChallengeMethod string
}

// CallbackRequest is the filtered result of the browser's Google callback.
type CallbackRequest struct {
	State            string
	Code             string
	Error            string
	ErrorDescription string
	ErrorURI         string
	Scope            string
	AuthUser         string
	Prompt           string
	HD               string
	Issuer           string
}

// TokenRequest selects exactly one supported GoTrue grant.
type TokenRequest struct {
	GrantType    string
	AuthCode     string
	CodeVerifier string
	RefreshToken string
}

// AppMetadata is the fixed eligibility projection returned by the facade.
type AppMetadata struct {
	Provider  string   `json:"provider"`
	Providers []string `json:"providers"`
}

// UserMetadata contains only presentational fields accepted from GoTrue.
type UserMetadata struct {
	Name     string `json:"name,omitempty"`
	FullName string `json:"full_name,omitempty"`
}

// User is the minimal stable user projection exposed by the facade.
type User struct {
	ID               string       `json:"id"`
	Audience         string       `json:"aud"`
	Role             string       `json:"role"`
	IsAnonymous      bool         `json:"is_anonymous"`
	CreatedAt        string       `json:"created_at"`
	EmailConfirmedAt string       `json:"email_confirmed_at"`
	Email            string       `json:"email,omitempty"`
	AppMetadata      AppMetadata  `json:"app_metadata"`
	UserMetadata     UserMetadata `json:"user_metadata"`
	sessionID        string
}

// Session is the only credential projection returned by token exchange.
type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	User         User   `json:"user"`
}

// AuthError is an opaque error that the API layer maps to the approved flat
// Auth envelope. It intentionally carries no upstream message or URL.
type AuthError struct {
	status int
	code   string
}

func (e *AuthError) Error() string {
	if e == nil {
		return "auth: unknown error"
	}
	return "auth: " + e.code
}

func (e *AuthError) Status() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	return e.status
}

func (e *AuthError) Code() string {
	if e == nil {
		return "internal"
	}
	return e.code
}

// IsCode is useful to tests and adapters without exposing upstream details.
func IsCode(err error, code string) bool {
	var authErr *AuthError
	return errors.As(err, &authErr) && authErr != nil && authErr.code == code
}

func invalidRequest() error {
	return &AuthError{status: http.StatusBadRequest, code: "invalid_request"}
}
func unauthenticated() error {
	return &AuthError{status: http.StatusUnauthorized, code: "unauthenticated"}
}
func unavailable() error {
	return &AuthError{status: http.StatusServiceUnavailable, code: "auth_unavailable"}
}
func internalError() error {
	return &AuthError{status: http.StatusInternalServerError, code: "internal"}
}

// Authorize validates the public request, starts exactly one upstream Google
// FlowState, and returns a validated Google authorization Location.
func (f *Facade) Authorize(ctx context.Context, request AuthorizeRequest) (string, error) {
	if f == nil {
		return "", unavailable()
	}
	operationCtx, cancel := newAuthOperationContext(ctx)
	defer cancel()
	if request.Provider != "google" || request.FlowType != "pkce" || request.CodeChallengeMethod != "s256" || !validPKCEChallenge(request.CodeChallenge) {
		return "", invalidRequest()
	}
	redirect, err := validateFinalRedirect(request.RedirectTo)
	if err != nil {
		return "", invalidRequest()
	}
	if _, ok := f.config.redirectAllowlist[redirect]; !ok {
		return "", invalidRequest()
	}
	query := url.Values{}
	query.Set("provider", "google")
	query.Set("redirect_to", redirect)
	query.Set("code_challenge", request.CodeChallenge)
	query.Set("code_challenge_method", "s256")
	response, body, err := f.doUpstream(operationCtx, http.MethodGet, "/authorize?"+query.Encode(), "", nil, false)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusFound {
		return "", mapUpstreamStatus(response.StatusCode)
	}
	locations := response.Header.Values("Location")
	if len(locations) != 1 {
		return "", unavailable()
	}
	location := locations[0]
	if !f.validGoogleAuthorizeLocation(location, redirect) {
		return "", unavailable()
	}
	_ = body
	return location, nil
}

func validPKCEChallenge(value string) bool {
	if len(value) != 43 || strings.TrimSpace(value) != value {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func (f *Facade) validGoogleAuthorizeLocation(value, expectedRedirect string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host != f.config.googleAuthorizeURL.Host || parsed.Path != f.config.googleAuthorizeURL.Path || parsed.User != nil || parsed.Fragment != "" || parsed.RawFragment != "" {
		return false
	}
	query, queryErr := url.ParseQuery(parsed.RawQuery)
	if queryErr != nil || hasDuplicateQueryValues(query) || query.Get("response_type") != "code" || query.Get("redirect_uri") != f.config.callbackURL || query.Get("redirect_to") != expectedRedirect || !validCanonicalUUID(query.Get("state")) || query.Get("client_id") == "" {
		return false
	}
	allowed := map[string]bool{"client_id": true, "redirect_uri": true, "response_type": true, "state": true, "scope": true, "redirect_to": true, "access_type": true, "prompt": true}
	for key := range query {
		if !allowed[key] {
			return false
		}
	}
	scopes := strings.Fields(query.Get("scope"))
	seen := map[string]bool{}
	for _, scope := range scopes {
		seen[scope] = true
		if scope != "openid" && scope != "email" && scope != "profile" {
			return false
		}
	}
	return seen["email"] && seen["profile"]
}

// Callback forwards only the state and the selected success/error fields to
// GoTrue, then reconstructs a final allowlisted redirect.
func (f *Facade) Callback(ctx context.Context, request CallbackRequest) (string, error) {
	if f == nil || !validCanonicalUUID(request.State) || (request.Code == "") == (request.Error == "") || len(request.Code) > maxAuthCodeBytes || len(request.Error) > maxAuthCodeBytes || len(request.ErrorDescription) > maxAuthCodeBytes || len(request.ErrorURI) > maxAuthCodeBytes || len(request.Scope) > maxAuthCodeBytes || len(request.AuthUser) > maxAuthCodeBytes || len(request.Prompt) > maxAuthCodeBytes || len(request.HD) > maxAuthCodeBytes || request.Issuer != "" && request.Issuer != "https://accounts.google.com" {
		return "", invalidRequest()
	}
	operationCtx, cancel := newAuthOperationContext(ctx)
	defer cancel()
	query := url.Values{}
	query.Set("state", request.State)
	if request.Code != "" {
		query.Set("code", request.Code)
	} else {
		query.Set("error", request.Error)
		if request.ErrorDescription != "" {
			query.Set("error_description", request.ErrorDescription)
		}
	}
	response, _, err := f.doUpstream(operationCtx, http.MethodGet, "/callback?"+query.Encode(), "", nil, false)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusFound {
		return "", mapUpstreamStatus(response.StatusCode)
	}
	locations := response.Header.Values("Location")
	if len(locations) != 1 {
		return "", unavailable()
	}
	return f.sanitizeCallbackLocation(locations[0])
}

func (f *Facade) sanitizeCallbackLocation(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.RawQuery == "" {
		return "", unavailable()
	}
	if parsed.Fragment != "" || parsed.RawFragment != "" {
		fragmentValues, fragmentErr := url.ParseQuery(parsed.Fragment)
		if fragmentErr != nil || !validCallbackFragment(fragmentValues) {
			return "", unavailable()
		}
		parsed.Fragment = ""
		parsed.RawFragment = ""
	}
	base := *parsed
	base.RawQuery = ""
	base.ForceQuery = false
	if _, ok := f.config.redirectAllowlist[base.String()]; !ok {
		return "", unavailable()
	}
	query, queryErr := url.ParseQuery(parsed.RawQuery)
	if queryErr != nil || hasDuplicateQueryValues(query) {
		return "", unavailable()
	}
	if query.Get("code") != "" {
		if query.Get("error") != "" || len(query.Get("code")) > maxAuthCodeBytes || (query.Get("state") != "" && !validCanonicalUUID(query.Get("state"))) {
			return "", unavailable()
		}
		allowed := map[string]bool{"code": true, "state": true}
		for key := range query {
			if !allowed[key] {
				return "", unavailable()
			}
		}
		output := url.Values{"code": []string{query.Get("code")}}
		if query.Get("state") != "" {
			output.Set("state", query.Get("state"))
		}
		return base.String() + "?" + output.Encode(), nil
	}
	if query.Get("error") == "" || query.Get("code") != "" || (query.Get("state") != "" && !validCanonicalUUID(query.Get("state"))) {
		return "", unavailable()
	}
	allowedError := "server_error"
	if query.Get("error") == "access_denied" {
		allowedError = "access_denied"
	}
	for key := range query {
		if key != "error" && key != "error_code" && key != "error_description" && key != "state" {
			return "", unavailable()
		}
	}
	output := url.Values{"error": []string{allowedError}, "error_description": []string{"Authentication was not completed."}}
	if query.Get("state") != "" {
		output.Set("state", query.Get("state"))
	}
	return base.String() + "?" + output.Encode(), nil
}

func validCanonicalUUID(value string) bool {
	return validUUID(value) && strings.ToLower(value) == value
}

// Exchange performs one selected GoTrue grant and validates the returned
// access token with the fixed internal /user endpoint before releasing it.
func (f *Facade) Exchange(ctx context.Context, request TokenRequest) (Session, error) {
	if f == nil || (request.GrantType != "pkce" && request.GrantType != "refresh_token") {
		return Session{}, invalidRequest()
	}
	operationCtx, cancel := newAuthOperationContext(ctx)
	defer cancel()
	var payload any
	switch request.GrantType {
	case "pkce":
		if len(request.AuthCode) == 0 || len(request.AuthCode) > maxAuthCodeBytes || !validVerifier(request.CodeVerifier) {
			return Session{}, invalidRequest()
		}
		payload = struct {
			AuthCode     string `json:"auth_code"`
			CodeVerifier string `json:"code_verifier"`
		}{request.AuthCode, request.CodeVerifier}
	case "refresh_token":
		if len(request.RefreshToken) == 0 || len(request.RefreshToken) > maxRefreshTokenBytes || strings.TrimSpace(request.RefreshToken) != request.RefreshToken {
			return Session{}, invalidRequest()
		}
		payload = struct {
			RefreshToken string `json:"refresh_token"`
		}{request.RefreshToken}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Session{}, internalError()
	}
	response, upstreamBody, err := f.doUpstream(operationCtx, http.MethodPost, "/token?grant_type="+url.QueryEscape(request.GrantType), "application/json", bytes.NewReader(body), true)
	if err != nil {
		return Session{}, err
	}
	if response.StatusCode != http.StatusOK {
		return Session{}, mapTokenStatus(response.StatusCode, request.GrantType == "refresh_token")
	}
	if !isJSONContentType(response.Header.Get("Content-Type")) {
		return Session{}, unavailable()
	}
	var token tokenResponse
	if !decodeStrictJSON(upstreamBody, &token) || token.AccessToken == "" || token.RefreshToken == "" || token.TokenType != "bearer" || token.ExpiresIn <= 0 || token.ExpiresAt <= 0 || token.User.ID == "" {
		return Session{}, unavailable()
	}
	if !coherentExpiry(token.ExpiresIn, token.ExpiresAt) {
		return Session{}, unavailable()
	}
	verified, verifyErr := f.verifier.VerifyUser(operationCtx, token.AccessToken)
	if verifyErr != nil {
		if errors.Is(verifyErr, ErrUnauthenticated) {
			return Session{}, unauthenticated()
		}
		return Session{}, unavailable()
	}
	if verified.ID != token.User.ID {
		return Session{}, unavailable()
	}
	claims, claimsOK := decodeAccessTokenClaims(token.AccessToken)
	if !claimsOK || !validUUID(claims.Subject) || claims.Subject != verified.ID || claims.Expiry != token.ExpiresAt || claims.Expiry <= time.Now().Unix() {
		return Session{}, unavailable()
	}
	return Session{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, TokenType: "bearer", ExpiresIn: token.ExpiresIn, ExpiresAt: token.ExpiresAt, User: verified}, nil
}

func validVerifier(value string) bool {
	if len(value) < 43 || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for _, char := range value {
		if !(char >= 'A' && char <= 'Z') && !(char >= 'a' && char <= 'z') && !(char >= '0' && char <= '9') && !strings.ContainsRune("-._~", char) {
			return false
		}
	}
	return true
}

type tokenResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresIn    int64      `json:"expires_in"`
	ExpiresAt    int64      `json:"expires_at"`
	User         goTrueUser `json:"user"`
}

func coherentExpiry(expiresIn, expiresAt int64) bool {
	now := time.Now().Unix()
	if expiresAt <= now || expiresIn <= 0 {
		return false
	}
	// GoTrue computes expires_at from the same clock as expires_in. A bounded
	// tolerance avoids rejecting a response while retaining a future expiry.
	delta := expiresAt - now - expiresIn
	return delta >= -60 && delta <= 60
}

// User validates the bearer with GoTrue and returns the same sanitized DTO as
// a token exchange.
func (f *Facade) User(ctx context.Context, accessToken string) (User, error) {
	if f == nil || f.verifier == nil {
		return User{}, unavailable()
	}
	operationCtx, cancel := newAuthOperationContext(ctx)
	defer cancel()
	user, err := f.verifier.VerifyUser(operationCtx, accessToken)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			return User{}, unauthenticated()
		}
		return User{}, unavailable()
	}
	return user, nil
}

// Logout revokes only the current session. The session_id claim is inspected
// only as a nonzero routing guard; identity and signature remain GoTrue's job.
func (f *Facade) Logout(ctx context.Context, accessToken string) error {
	if f == nil || f.verifier == nil {
		return unavailable()
	}
	sessionID, ok := sessionIDFromAccessToken(accessToken)
	if !ok || isZeroUUID(sessionID) {
		return unauthenticated()
	}
	operationCtx, cancel := newAuthOperationContext(ctx)
	defer cancel()
	user, err := f.verifier.VerifyUser(operationCtx, accessToken)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			return unauthenticated()
		}
		return unavailable()
	}
	if user.ID == "" {
		return unavailable()
	}
	if user.sessionID != "" && user.sessionID != sessionID {
		return unauthenticated()
	}
	response, _, err := f.doUpstream(operationCtx, http.MethodPost, "/logout?scope=local", "", nil, bearerOption{token: accessToken})
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return unauthenticated()
	}
	if response.StatusCode != http.StatusNoContent {
		return mapUpstreamStatus(response.StatusCode)
	}
	return nil
}

type bearerOption struct{ token string }

func (f *Facade) doUpstream(ctx context.Context, method, path, contentType string, body io.Reader, option any) (*http.Response, []byte, error) {
	if f == nil || f.client == nil {
		return nil, nil, unavailable()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	operationCtx, cancel := context.WithTimeout(ctx, upstreamTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(operationCtx, method, f.config.upstreamBase+path, body)
	if err != nil {
		return nil, nil, unavailable()
	}
	request.Header.Set("Accept", "application/json")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if bearer, ok := option.(bearerOption); ok {
		request.Header.Set("Authorization", "Bearer "+bearer.token)
	}
	response, err := f.client.Do(request)
	if err != nil {
		return nil, nil, unavailable()
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxUpstreamResponseSize+1))
	if err != nil || len(data) > maxUpstreamResponseSize || !utf8.Valid(data) {
		return response, nil, unavailable()
	}
	return response, data, nil
}

func mapUpstreamStatus(status int) error {
	switch status {
	case http.StatusBadRequest:
		return invalidRequest()
	case http.StatusUnauthorized, http.StatusForbidden:
		return unauthenticated()
	default:
		return unavailable()
	}
}

func mapTokenStatus(status int, refresh bool) error {
	switch status {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusUnauthorized, http.StatusForbidden:
		if refresh {
			return unauthenticated()
		}
		return invalidRequest()
	default:
		return unavailable()
	}
}

func decodeStrictJSON(body []byte, destination any) bool {
	if len(body) == 0 || !utf8.Valid(body) || hasDuplicateJSONKeys(body) {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(destination); err != nil {
		return false
	}
	return decoder.Decode(new(any)) == io.EOF
}

func hasDuplicateQueryValues(query url.Values) bool {
	for _, values := range query {
		if len(values) > 1 {
			return true
		}
	}
	return false
}

func validCallbackFragment(fragment url.Values) bool {
	allowed := map[string]bool{
		"error":             true,
		"error_code":        true,
		"error_description": true,
		"error_uri":         true,
		"sb":                true,
		"state":             true,
	}
	for key, values := range fragment {
		if !allowed[key] {
			return false
		}
		if key == "sb" && (len(values) != 1 || values[0] != "") {
			return false
		}
	}
	return true
}

func isZeroUUID(value string) bool {
	return value == "00000000-0000-0000-0000-000000000000"
}

func sessionIDFromAccessToken(token string) (string, bool) {
	claims, ok := decodeAccessTokenClaims(token)
	if !ok || !validCanonicalUUID(claims.SessionID) {
		return "", false
	}
	return claims.SessionID, true
}

type accessTokenClaims struct {
	Subject   string `json:"sub"`
	Expiry    int64  `json:"exp"`
	SessionID string `json:"session_id"`
}

func decodeAccessTokenClaims(token string) (accessTokenClaims, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return accessTokenClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !utf8.Valid(payload) || hasDuplicateJSONKeys(payload) {
		return accessTokenClaims{}, false
	}
	var claims accessTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return accessTokenClaims{}, false
	}
	return claims, true
}

var _ Verifier = (*Facade)(nil)

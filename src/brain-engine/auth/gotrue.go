package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
)

const (
	verificationTimeout = 3 * time.Second
	maxUserResponseSize = 1 << 20
)

type ErrorCode string

const (
	ErrorUnauthenticated ErrorCode = "unauthenticated"
	ErrorUnavailable     ErrorCode = "unavailable"
)

// Error is a stable, opaque authentication boundary error. It never contains
// a token, user ID, upstream body, or transport detail.
type Error struct {
	code ErrorCode
}

func (e *Error) Error() string {
	if e == nil {
		return "auth: unknown error"
	}
	return "auth: " + string(e.code)
}

func (e *Error) Code() ErrorCode {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && e != nil && other != nil && e.code == other.code
}

var (
	ErrUnauthenticated = &Error{code: ErrorUnauthenticated}
	ErrUnavailable     = &Error{code: ErrorUnavailable}
)

// Verifier converts a bearer credential into the opaque principal consumed by
// Brain Core. Implementations must fail closed when identity cannot be trusted.
type Verifier interface {
	Verify(ctx context.Context, accessToken string) (brain.Principal, error)
}

// GoTrueVerifier verifies each access token against the configured GoTrue
// /user endpoint. It deliberately does not cache positive identity results.
type GoTrueVerifier struct {
	userURL string
	client  *http.Client
}

// NewGoTrueVerifier creates a verifier for one fixed absolute /user URL. The
// endpoint must come from private server configuration, never from a request.
func NewGoTrueVerifier(userURL string, client *http.Client) (*GoTrueVerifier, error) {
	parsed, err := url.Parse(userURL)
	if err != nil || !validUserURL(parsed) {
		return nil, errors.New("auth: invalid configured user endpoint")
	}

	clientCopy := http.Client{Transport: defaultTransportWithoutProxy()}
	if client != nil {
		clientCopy = *client
		if clientCopy.Transport == nil {
			clientCopy.Transport = defaultTransportWithoutProxy()
		}
	}
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	clientCopy.Jar = nil

	return &GoTrueVerifier{userURL: parsed.String(), client: &clientCopy}, nil
}

func defaultTransportWithoutProxy() http.RoundTripper {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultTransport
	}
	clone := transport.Clone()
	clone.Proxy = nil
	return clone
}

func validUserURL(parsed *url.URL) bool {
	if parsed == nil || !parsed.IsAbs() || parsed.Opaque != "" || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" {
		return false
	}
	return parsed.Path == "/user" && parsed.EscapedPath() == "/user"
}

// Verify asks GoTrue to validate the bearer token and returns only its user ID
// after checking the account is confirmed, authenticated, Google-backed, and
// nonanonymous.
func (v *GoTrueVerifier) Verify(ctx context.Context, accessToken string) (brain.Principal, error) {
	if v == nil || v.client == nil || strings.TrimSpace(accessToken) == "" || strings.ContainsAny(accessToken, " \t\r\n") {
		return "", ErrUnauthenticated
	}
	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx, cancel := context.WithTimeout(ctx, verificationTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, v.userURL, nil)
	if err != nil {
		return "", ErrUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-cache, no-store")
	request.Header.Set("Pragma", "no-cache")
	response, err := v.client.Do(request)
	if err != nil {
		return "", ErrUnavailable
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", ErrUnauthenticated
	case http.StatusOK:
	default:
		return "", ErrUnavailable
	}
	if !isJSONContentType(response.Header.Get("Content-Type")) {
		return "", ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxUserResponseSize+1))
	if err != nil || len(body) > maxUserResponseSize || !utf8.Valid(body) {
		return "", ErrUnavailable
	}
	var user goTrueUser
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&user); err != nil {
		return "", ErrUnavailable
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "", ErrUnavailable
	}
	if hasDuplicateJSONKeys(body) || !validUUID(user.ID) || user.Audience == nil || user.Role == nil || user.IsAnonymous == nil || !user.confirmationFieldsValid() {
		return "", ErrUnavailable
	}
	if *user.Audience != "authenticated" || *user.Role != "authenticated" || *user.IsAnonymous || !user.isConfirmed() || !user.hasGoogleIdentity() {
		return "", ErrUnauthenticated
	}
	return brain.Principal(user.ID), nil
}

// VerifyUser validates a bearer against the fixed GoTrue /user endpoint and
// returns the same sanitized user projection exposed by the Auth facade.
func (v *GoTrueVerifier) VerifyUser(ctx context.Context, accessToken string) (User, error) {
	if v == nil || v.client == nil || strings.TrimSpace(accessToken) == "" || strings.ContainsAny(accessToken, " \t\r\n") {
		return User{}, ErrUnauthenticated
	}
	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx, cancel := context.WithTimeout(ctx, verificationTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, v.userURL, nil)
	if err != nil {
		return User{}, ErrUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-cache, no-store")
	request.Header.Set("Pragma", "no-cache")

	response, err := v.client.Do(request)
	if err != nil {
		return User{}, ErrUnavailable
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return User{}, ErrUnauthenticated
	case http.StatusOK:
	default:
		return User{}, ErrUnavailable
	}
	if !isJSONContentType(response.Header.Get("Content-Type")) {
		return User{}, ErrUnavailable
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxUserResponseSize+1))
	if err != nil || len(body) > maxUserResponseSize || !utf8.Valid(body) {
		return User{}, ErrUnavailable
	}

	var user goTrueUser
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&user); err != nil {
		return User{}, ErrUnavailable
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return User{}, ErrUnavailable
	}
	if hasDuplicateJSONKeys(body) || !validUUID(user.ID) || user.Audience == nil || user.Role == nil || user.IsAnonymous == nil || !user.confirmationFieldsValid() {
		return User{}, ErrUnavailable
	}
	if *user.Audience != "authenticated" || *user.Role != "authenticated" || *user.IsAnonymous || !user.isConfirmed() || !user.hasGoogleIdentity() {
		return User{}, ErrUnauthenticated
	}
	projection, ok := user.projection()
	if !ok {
		return User{}, ErrUnavailable
	}
	return projection, nil
}

type goTrueUser struct {
	ID               string                     `json:"id"`
	Audience         *string                    `json:"aud"`
	Role             *string                    `json:"role"`
	CreatedAt        *string                    `json:"created_at"`
	Email            *string                    `json:"email"`
	EmailConfirmedAt *string                    `json:"email_confirmed_at"`
	ConfirmedAt      *string                    `json:"confirmed_at"`
	AppMetadata      goTrueAppMetadata          `json:"app_metadata"`
	Identities       []goTrueIdentity           `json:"identities"`
	IsAnonymous      *bool                      `json:"is_anonymous"`
	UserMetadata     map[string]json.RawMessage `json:"user_metadata"`
	SessionID        *string                    `json:"session_id"`
}

type goTrueAppMetadata struct {
	Provider  string   `json:"provider"`
	Providers []string `json:"providers"`
}

type goTrueIdentity struct {
	Provider string `json:"provider"`
}

func (u goTrueUser) projection() (User, bool) {
	if u.Audience == nil || u.Role == nil || u.IsAnonymous == nil || u.CreatedAt == nil {
		return User{}, false
	}
	created, err := time.Parse(time.RFC3339Nano, *u.CreatedAt)
	if err != nil {
		return User{}, false
	}
	confirmation := ""
	if validConfirmationTime(u.EmailConfirmedAt) {
		confirmation = *u.EmailConfirmedAt
	} else if validConfirmationTime(u.ConfirmedAt) {
		confirmation = *u.ConfirmedAt
	}
	confirmedAt, err := time.Parse(time.RFC3339Nano, confirmation)
	if err != nil {
		return User{}, false
	}
	if *u.Audience != "authenticated" || *u.Role != "authenticated" || *u.IsAnonymous || !u.hasGoogleIdentity() {
		return User{}, false
	}
	projection := User{
		ID:               strings.ToLower(u.ID),
		Audience:         "authenticated",
		Role:             "authenticated",
		IsAnonymous:      false,
		CreatedAt:        created.UTC().Format(time.RFC3339Nano),
		EmailConfirmedAt: confirmedAt.UTC().Format(time.RFC3339Nano),
		AppMetadata:      AppMetadata{Provider: "google", Providers: []string{"google"}},
		UserMetadata:     UserMetadata{},
	}
	if u.Email != nil {
		if !validOptionalString(*u.Email, 320) {
			return User{}, false
		}
		projection.Email = *u.Email
	}
	metadata, ok := projectUserMetadata(u.UserMetadata)
	if !ok {
		return User{}, false
	}
	projection.UserMetadata = metadata
	if u.SessionID != nil {
		if !validCanonicalUUID(*u.SessionID) {
			return User{}, false
		}
		projection.sessionID = *u.SessionID
	}
	return projection, true
}

func validOptionalString(value string, max int) bool {
	return utf8.ValidString(value) && len(value) <= max && !strings.ContainsRune(value, '\x00')
}

func projectUserMetadata(raw map[string]json.RawMessage) (UserMetadata, bool) {
	var result UserMetadata
	for key, value := range raw {
		if key != "name" && key != "full_name" {
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err != nil || !validOptionalString(text, 255) {
			return UserMetadata{}, false
		}
		switch key {
		case "name":
			result.Name = text
		case "full_name":
			result.FullName = text
		}
	}
	return result, true
}

func (u goTrueUser) isConfirmed() bool {
	return validConfirmationTime(u.EmailConfirmedAt) || validConfirmationTime(u.ConfirmedAt)
}

func (u goTrueUser) confirmationFieldsValid() bool {
	return wellFormedConfirmationTime(u.EmailConfirmedAt) && wellFormedConfirmationTime(u.ConfirmedAt)
}

func wellFormedConfirmationTime(value *string) bool {
	if value == nil || strings.TrimSpace(*value) == "" {
		return true
	}
	_, err := time.Parse(time.RFC3339Nano, *value)
	return err == nil
}

func validConfirmationTime(value *string) bool {
	if value == nil || strings.TrimSpace(*value) == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, *value)
	return err == nil
}

func (u goTrueUser) hasGoogleIdentity() bool {
	if u.AppMetadata.Provider == "google" {
		return true
	}
	for _, provider := range u.AppMetadata.Providers {
		if provider == "google" {
			return true
		}
	}
	for _, identity := range u.Identities {
		if identity.Provider == "google" {
			return true
		}
	}
	return false
}

func isJSONContentType(value string) bool {
	mediaType, parameters, err := mime.ParseMediaType(value)
	if err != nil || mediaType != "application/json" {
		return false
	}
	charset := parameters["charset"]
	return charset == "" || strings.EqualFold(charset, "utf-8")
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i := 0; i < len(value); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		c := value[i]
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') && !('A' <= c && c <= 'F') {
			return false
		}
	}
	return true
}

func hasDuplicateJSONKeys(body []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if consumeJSONValue(decoder, 0) != nil {
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

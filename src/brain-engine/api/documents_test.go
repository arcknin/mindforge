package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/arcknin/mindforge/src/brain-engine/features/documents"
)

const testDocumentID = "10000000-0000-4000-8000-000000000001"
const testLayerID = "20000000-0000-4000-8000-000000000001"

type documentHTTPStore struct {
	calls    int
	owner    brain.Principal
	mutation documents.Mutation
	query    documents.Query
	err      error
}

func (s *documentHTTPStore) result() (documents.Document, error) {
	if s.err != nil {
		return documents.Document{}, s.err
	}
	return documents.Document{ID: testDocumentID, LayerID: testLayerID, Title: "Investimentos", Markdown: "# Nota", Version: 9007199254740993, CreatedAt: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC), Links: []documents.Link{}}, nil
}
func (s *documentHTTPStore) Apply(_ context.Context, o brain.Principal, m documents.Mutation) (documents.Document, error) {
	s.calls++
	s.owner = o
	s.mutation = m
	return s.result()
}
func (s *documentHTTPStore) Get(_ context.Context, o brain.Principal, id string) (documents.Document, error) {
	s.calls++
	s.owner = o
	return s.result()
}
func (s *documentHTTPStore) List(_ context.Context, o brain.Principal, q documents.Query) (documents.Page, error) {
	s.calls++
	s.owner = o
	s.query = q
	d, e := s.result()
	return documents.Page{Documents: []documents.Document{d}}, e
}
func documentRequest(t *testing.T, s *documentHTTPStore, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewHandlerWithDocuments(brain.NewEngine(newFakeStore(), nil), documents.NewEngine(s), &fakeVerifier{principal: "owner"})
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestDocumentHTTPApprovedOperations(t *testing.T) {
	base := "/v1/me/brain/documents/" + testDocumentID
	for _, tc := range []struct {
		method, path, body string
		status             int
		kind               string
	}{
		{"POST", "/v1/me/brain/documents", `{"id":"` + testDocumentID + `","layer_id":"` + testLayerID + `","title":"Nota","markdown":""}`, 201, "create"},
		{"GET", base, "", 200, ""},
		{"GET", "/v1/me/brain/layers/" + testLayerID + "/documents?scope=subtree&state=trash&limit=2&after=" + testDocumentID, "", 200, ""},
		{"PATCH", base, `{"expected_version":1,"layer_id":"` + testLayerID + `","title":"Nota","markdown":"# Nota"}`, 200, "update"},
		{"POST", base + "/trash", `{"expected_version":1}`, 200, "trash"},
		{"POST", base + "/restore", `{"expected_version":1}`, 200, "restore"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			s := &documentHTTPStore{}
			w := documentRequest(t, s, tc.method, tc.path, tc.body, "token")
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if s.calls != 1 || s.owner != "owner" {
				t.Fatal("auth principal not propagated")
			}
			if tc.kind != "" && string(s.mutation.Kind) != tc.kind {
				t.Fatalf("mutation=%+v", s.mutation)
			}
			if strings.Contains(tc.path, "?scope") {
				if s.query.Scope != "subtree" || s.query.State != "trash" || s.query.Limit != 2 || s.query.After != testDocumentID {
					t.Fatalf("query=%+v", s.query)
				}
				if strings.Contains(w.Body.String(), `"markdown"`) || strings.Contains(w.Body.String(), `"links"`) {
					t.Fatal("list leaked full bodies")
				}
				if !strings.Contains(w.Body.String(), `"next_cursor":null`) {
					t.Fatal("missing terminal cursor")
				}
			} else {
				if !strings.Contains(w.Body.String(), `9007199254740993`) {
					t.Fatal("version precision lost")
				}
				if !strings.Contains(w.Body.String(), `"deleted_at":null`) || !strings.Contains(w.Body.String(), `"links":[]`) {
					t.Fatal("response shape drift")
				}
			}
		})
	}
}
func TestDocumentHTTPStrictRequestsAndAuth(t *testing.T) {
	base := "/v1/me/brain/documents/" + testDocumentID
	for _, tc := range []struct {
		method, path, body, token string
		status                    int
	}{
		{"GET", base, "", "", 401},
		{"GET", base + "?unexpected=1", "", "token", 400},
		{"GET", base, "{}", "token", 400},
		{"POST", base + "/trash", `{"expected_version":1,"expected_version":2}`, "token", 400},
		{"POST", base + "/trash", `{"expected_version":null}`, "token", 400},
		{"POST", base + "/trash", `{"Expected_Version":1}`, "token", 400},
		{"POST", base + "/trash", `{"expected_version":1,"owner":"other"}`, "token", 400},
		{"POST", base + "/restore", `{"expected_version":1,"layer_id":null}`, "token", 400},
		{"GET", "/v1/me/brain/layers/" + testLayerID + "/documents?limit=1&limit=2", "", "token", 400},
		{"GET", "/v1/me/brain/layers/" + testLayerID + "/documents?limit=-1", "", "token", 400},
		{"GET", "/v1/me/brain/layers/" + testLayerID + "/documents?scope=other", "", "token", 400},
		{"GET", "/v1/me/brain/documents/bad-id", "", "token", 400},
		{"DELETE", base, "", "token", 405},
		{"POST", base + "/purge", "{}", "token", 404},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			s := &documentHTTPStore{}
			w := documentRequest(t, s, tc.method, tc.path, tc.body, tc.token)
			if w.Code != tc.status {
				t.Fatalf("status=%d want%d body=%s", w.Code, tc.status, w.Body.String())
			}
			if s.calls != 0 {
				t.Fatal("invalid request reached store")
			}
		})
	}
}
func TestDocumentHTTPDomainErrorsStayOpaque(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{brain.ErrNotFound, 404, "not_found"}, {brain.ErrForbidden, 404, "not_found"}, {brain.ErrConflict, 409, "conflict"}, {documents.ErrInvalidDocument, 422, "invalid_document"}, {errors.New("private-sql-marker"), 500, "internal"},
	} {
		s := &documentHTTPStore{err: tc.err}
		w := documentRequest(t, s, "GET", "/v1/me/brain/documents/"+testDocumentID, "", "token")
		if w.Code != tc.status {
			t.Fatalf("status=%d want%d", w.Code, tc.status)
		}
		var e errorEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil || e.Error.Code != tc.code || e.RequestID == "" {
			t.Fatal("bad error envelope")
		}
		if strings.Contains(w.Body.String(), "private-sql-marker") {
			t.Fatal("internal error disclosed")
		}
	}
}

func TestDocumentHTTPBodyValidationAndJSONEscapes(t *testing.T) {
	base := "/v1/me/brain/documents/" + testDocumentID
	cases := []struct {
		body string
		want int
	}{
		{`{"expected_version":1,"layer_id":"` + testLayerID + `","title":" ","markdown":""}`, 422},
		{`{"expected_version":0,"layer_id":"` + testLayerID + `","title":"A","markdown":""}`, 422},
		{`{"expected_version":1.5,"layer_id":"` + testLayerID + `","title":"A","markdown":""}`, 400},
		{`{"expected_version":9223372036854775808,"layer_id":"` + testLayerID + `","title":"A","markdown":""}`, 400},
		{`{"expected_version":1,"layer_id":"` + testLayerID + `","title":"A","markdown":null}`, 400},
	}
	for _, tc := range cases {
		s := &documentHTTPStore{}
		w := documentRequest(t, s, "PATCH", base, tc.body, "token")
		if w.Code != tc.want || s.calls != 0 {
			t.Fatalf("status=%d want%d calls=%d body=%s", w.Code, tc.want, s.calls, w.Body.String())
		}
	}
	payload, _ := json.Marshal(map[string]any{"expected_version": 1, "layer_id": testLayerID, "title": "Nota", "markdown": strings.Repeat("\n", 1<<20)})
	s := &documentHTTPStore{}
	w := documentRequest(t, s, "PATCH", base, string(payload), "token")
	if w.Code != 200 {
		t.Fatalf("valid escaped1MiB markdown rejected: %d %s", w.Code, w.Body.String())
	}
	s = &documentHTTPStore{}
	w = documentRequest(t, s, "POST", base+"/restore", `{"expected_version":1,"layer_id":""}`, "token")
	if w.Code != 422 || s.calls != 0 {
		t.Fatal("explicit invalid restore destination accepted")
	}
	h := NewHandlerWithDocuments(brain.NewEngine(newFakeStore(), nil), documents.NewEngine(&documentHTTPStore{}), &fakeVerifier{principal: "owner"})
	for _, tc := range []struct {
		body, media string
		status      int
	}{{`{"expected_version":1}`, "text/plain", 415}, {strings.Repeat("x", (8<<20)+1), "application/json", 413}} {
		r := httptest.NewRequest("PATCH", base, strings.NewReader(tc.body))
		r.Header.Set("Authorization", "Bearer token")
		r.Header.Set("Content-Type", tc.media)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("status=%d want%d", w.Code, tc.status)
		}
	}
}

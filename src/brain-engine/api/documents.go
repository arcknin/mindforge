package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/arcknin/mindforge/src/brain-engine/features/documents"
)

const documentsPath = brainPath + "/documents"

type documentOperation uint8

const (
	documentCreate documentOperation = iota + 1
	documentGet
	documentUpdate
	documentList
	documentTrash
	documentRestore
)

func routeDocument(method, path string) (documentOperation, string, string, int, string, bool) {
	if path == documentsPath {
		if method == http.MethodPost {
			return documentCreate, "", "", 0, "", true
		}
		return 0, "", "", 405, "POST", true
	}
	if strings.HasPrefix(path, documentsPath+"/") {
		segments := strings.Split(strings.TrimPrefix(path, documentsPath+"/"), "/")
		if len(segments) > 2 || (len(segments) == 2 && segments[1] != "trash" && segments[1] != "restore") {
			return 0, "", "", 404, "", true
		}
		id, err := url.PathUnescape(segments[0])
		if err != nil || !documents.ValidID(id) {
			return 0, "", "", 400, "", true
		}
		if len(segments) == 1 {
			switch method {
			case http.MethodGet:
				return documentGet, id, "", 0, "", true
			case http.MethodPatch:
				return documentUpdate, id, "", 0, "", true
			default:
				return 0, "", "", 405, "GET, PATCH", true
			}
		}
		if method != http.MethodPost {
			return 0, "", "", 405, "POST", true
		}
		if segments[1] == "trash" {
			return documentTrash, id, "", 0, "", true
		}
		return documentRestore, id, "", 0, "", true
	}
	if strings.HasPrefix(path, layersPath+"/") && strings.HasSuffix(path, "/documents") {
		raw := strings.TrimSuffix(strings.TrimPrefix(path, layersPath+"/"), "/documents")
		id, err := url.PathUnescape(raw)
		if err != nil || !documents.ValidID(id) {
			return 0, "", "", 400, "", true
		}
		if method != http.MethodGet {
			return 0, "", "", 405, "GET", true
		}
		return documentList, "", id, 0, "", true
	}
	return 0, "", "", 0, "", false
}

type documentCreateRequest struct {
	ID       string `json:"id"`
	LayerID  string `json:"layer_id"`
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}
type documentUpdateRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	LayerID         string `json:"layer_id"`
	Title           string `json:"title"`
	Markdown        string `json:"markdown"`
}
type documentVersionRequest struct {
	ExpectedVersion int64 `json:"expected_version"`
}
type documentRestoreRequest struct {
	ExpectedVersion int64   `json:"expected_version"`
	LayerID         *string `json:"layer_id,omitempty"`
}

func validDocumentObjectShape(object map[string]json.RawMessage, destination any) bool {
	for _, value := range object {
		if strings.TrimSpace(string(value)) == "null" {
			return false
		}
	}
	switch destination.(type) {
	case *documentCreateRequest:
		return hasExactKeys(object, "id", "layer_id", "title", "markdown")
	case *documentUpdateRequest:
		return hasExactKeys(object, "expected_version", "layer_id", "title", "markdown")
	case *documentVersionRequest:
		return hasExactKeys(object, "expected_version")
	case *documentRestoreRequest:
		return hasExactKeys(object, "expected_version") || hasExactKeys(object, "expected_version", "layer_id")
	}
	return false
}
func documentQuery(r *http.Request, layer string) (documents.Query, error) {
	q := documents.Query{LayerID: layer}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return q, documents.ErrInvalidQuery
	}
	for key, v := range values {
		if len(v) != 1 || v[0] == "" {
			return q, documents.ErrInvalidQuery
		}
		switch key {
		case "scope":
			q.Scope = v[0]
		case "state":
			q.State = v[0]
		case "after":
			q.After = v[0]
		case "limit":
			n, e := strconv.Atoi(v[0])
			if e != nil || n < 1 || n > 100 || strconv.Itoa(n) != v[0] {
				return q, documents.ErrInvalidQuery
			}
			q.Limit = n
		default:
			return q, documents.ErrInvalidQuery
		}
	}
	return q, nil
}
func (h *handler) serveDocuments(w http.ResponseWriter, r *http.Request, owner brain.Principal, op documentOperation, id, layer, requestID string) {
	var d documents.Document
	var err error
	switch op {
	case documentGet:
		if rejectNoBody(w, r, requestID) {
			return
		}
		d, err = h.documents.Get(r.Context(), owner, id)
	case documentList:
		if rejectNoBody(w, r, requestID) {
			return
		}
		q, e := documentQuery(r, layer)
		if e != nil {
			writeDocumentError(w, e, requestID)
			return
		}
		page, e := h.documents.List(r.Context(), owner, q)
		if e != nil {
			writeDocumentError(w, e, requestID)
			return
		}
		summaries := make([]documentMetadataDTO, 0, len(page.Documents))
		for _, doc := range page.Documents {
			summaries = append(summaries, toDocumentMetadata(doc))
		}
		var next *string
		if page.NextCursor != "" {
			next = &page.NextCursor
		}
		writeJSON(w, 200, struct {
			Documents  []documentMetadataDTO `json:"documents"`
			NextCursor *string               `json:"next_cursor"`
		}{summaries, next})
		return
	case documentCreate:
		var input documentCreateRequest
		if decodeRequestLimit(w, r, &input, requestID, 8<<20) {
			return
		}
		d, err = h.documents.Create(r.Context(), owner, documents.Mutation{ID: input.ID, LayerID: input.LayerID, Title: input.Title, Markdown: input.Markdown})
	case documentUpdate:
		var input documentUpdateRequest
		if decodeRequestLimit(w, r, &input, requestID, 8<<20) {
			return
		}
		d, err = h.documents.Update(r.Context(), owner, documents.Mutation{ID: id, LayerID: input.LayerID, Title: input.Title, Markdown: input.Markdown, ExpectedVersion: input.ExpectedVersion})
	case documentTrash:
		var input documentVersionRequest
		if decodeRequest(w, r, &input, requestID) {
			return
		}
		d, err = h.documents.Trash(r.Context(), owner, id, input.ExpectedVersion)
	case documentRestore:
		var input documentRestoreRequest
		if decodeRequest(w, r, &input, requestID) {
			return
		}
		destination := ""
		if input.LayerID != nil {
			if !documents.ValidID(*input.LayerID) {
				writeDocumentError(w, documents.ErrInvalidDocument, requestID)
				return
			}
			destination = *input.LayerID
		}
		d, err = h.documents.Restore(r.Context(), owner, id, input.ExpectedVersion, destination)
	default:
		writeError(w, 500, "internal", "Internal error.", requestID)
		return
	}
	if err != nil {
		writeDocumentError(w, err, requestID)
		return
	}
	status := 200
	if op == documentCreate {
		status = 201
	}
	writeJSON(w, status, struct {
		Document documentDTO `json:"document"`
	}{toDocumentDTO(d)})
}
func writeDocumentError(w http.ResponseWriter, err error, requestID string) {
	if errors.Is(err, documents.ErrInvalidDocument) {
		writeError(w, 422, "invalid_document", "Document is invalid.", requestID)
		return
	}
	if errors.Is(err, documents.ErrInvalidQuery) {
		writeError(w, 400, "invalid_request", "Request is invalid.", requestID)
		return
	}
	writeBrainError(w, err, requestID)
}

type documentMetadataDTO struct {
	ID        string     `json:"id"`
	LayerID   string     `json:"layer_id"`
	Title     string     `json:"title"`
	Version   int64      `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}
type documentLinkDTO struct {
	TargetID string `json:"target_id"`
	Title    string `json:"title"`
	LayerID  string `json:"layer_id"`
	Status   string `json:"status"`
}
type documentDTO struct {
	documentMetadataDTO
	Markdown string            `json:"markdown"`
	Links    []documentLinkDTO `json:"links"`
}

func utcPointer(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	t := v.UTC()
	return &t
}
func toDocumentMetadata(d documents.Document) documentMetadataDTO {
	return documentMetadataDTO{ID: d.ID, LayerID: d.LayerID, Title: d.Title, Version: d.Version, CreatedAt: d.CreatedAt.UTC(), UpdatedAt: d.UpdatedAt.UTC(), DeletedAt: utcPointer(d.DeletedAt), ExpiresAt: utcPointer(d.ExpiresAt)}
}
func toDocumentDTO(d documents.Document) documentDTO {
	links := make([]documentLinkDTO, 0, len(d.Links))
	for _, l := range d.Links {
		links = append(links, documentLinkDTO{l.TargetID, l.Title, l.LayerID, l.Status})
	}
	return documentDTO{toDocumentMetadata(d), d.Markdown, links}
}

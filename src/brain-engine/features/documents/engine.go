package documents

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
)

const (
	ScopeDirect   = "direct"
	ScopeSubtree  = "subtree"
	StateActive   = "active"
	StateTrash    = "trash"
	DefaultLimit  = 50
	MaxLimit      = 100
	maxMarkdown   = 1 << 20
	maxTitleRunes = 255
)

// Engine is the deterministic domain boundary for documents. It owns
// authentication checks, snapshot validation and link target derivation;
// Store owns persistence, ownership, clock and atomicity.
type Engine struct {
	store Store
}

func NewEngine(store Store) *Engine {
	return &Engine{store: store}
}

func (e *Engine) Create(ctx context.Context, owner brain.Principal, input Mutation) (Document, error) {
	if !validPrincipal(owner) {
		return Document{}, brain.ErrUnauthenticated
	}
	mutation, err := e.snapshotMutation(input, KindCreate, false)
	if err != nil {
		return Document{}, err
	}
	return e.apply(ctx, owner, mutation)
}

func (e *Engine) Update(ctx context.Context, owner brain.Principal, input Mutation) (Document, error) {
	if !validPrincipal(owner) {
		return Document{}, brain.ErrUnauthenticated
	}
	mutation, err := e.snapshotMutation(input, KindUpdate, true)
	if err != nil {
		return Document{}, err
	}
	return e.apply(ctx, owner, mutation)
}

func (e *Engine) Trash(ctx context.Context, owner brain.Principal, id string, version int64) (Document, error) {
	if !validPrincipal(owner) {
		return Document{}, brain.ErrUnauthenticated
	}
	if !ValidID(id) || version < 1 {
		return Document{}, ErrInvalidDocument
	}
	return e.apply(ctx, owner, Mutation{
		Kind:            KindTrash,
		ID:              id,
		ExpectedVersion: version,
	})
}

func (e *Engine) Restore(ctx context.Context, owner brain.Principal, id string, version int64, layer string) (Document, error) {
	if !validPrincipal(owner) {
		return Document{}, brain.ErrUnauthenticated
	}
	if !ValidID(id) || version < 1 || (layer != "" && !validLayerID(layer)) {
		return Document{}, ErrInvalidDocument
	}
	return e.apply(ctx, owner, Mutation{
		Kind:            KindRestore,
		ID:              id,
		LayerID:         layer,
		ExpectedVersion: version,
	})
}

func (e *Engine) Get(ctx context.Context, owner brain.Principal, id string) (Document, error) {
	if !validPrincipal(owner) {
		return Document{}, brain.ErrUnauthenticated
	}
	if !ValidID(id) {
		return Document{}, ErrInvalidDocument
	}
	if !e.ready() {
		return Document{}, brain.ErrInternal
	}
	document, err := e.store.Get(normalizeContext(ctx), owner, id)
	if err != nil {
		return Document{}, mapStoreError(err)
	}
	if !validDocumentSnapshot(document) {
		return Document{}, brain.ErrInternal
	}
	return document, nil
}

func (e *Engine) List(ctx context.Context, owner brain.Principal, input Query) (Page, error) {
	if !validPrincipal(owner) {
		return Page{}, brain.ErrUnauthenticated
	}
	query, err := normalizeQuery(input)
	if err != nil {
		return Page{}, err
	}
	if !e.ready() {
		return Page{}, brain.ErrInternal
	}
	page, err := e.store.List(normalizeContext(ctx), owner, query)
	if err != nil {
		return Page{}, mapStoreError(err)
	}
	if page.NextCursor != "" && !ValidID(page.NextCursor) {
		return Page{}, brain.ErrInternal
	}
	for _, document := range page.Documents {
		if !validDocumentSnapshot(document) {
			return Page{}, brain.ErrInternal
		}
	}
	return page, nil
}

// snapshotMutation validates a complete create/update snapshot and always
// derives Targets from Markdown. The caller's Kind and Targets are ignored.
func (e *Engine) snapshotMutation(input Mutation, kind MutationKind, requireVersion bool) (Mutation, error) {
	if !ValidID(input.ID) || !validLayerID(input.LayerID) {
		return Mutation{}, ErrInvalidDocument
	}
	title, ok := normalizeTitle(input.Title)
	if !ok || !validMarkdown(input.Markdown) {
		return Mutation{}, ErrInvalidDocument
	}
	if requireVersion && input.ExpectedVersion < 1 {
		return Mutation{}, ErrInvalidDocument
	}
	targets, err := ParseLinks(input.Markdown)
	if err != nil {
		return Mutation{}, ErrInvalidDocument
	}
	expectedVersion := input.ExpectedVersion
	if !requireVersion {
		expectedVersion = 0
	}
	return Mutation{
		Kind:            kind,
		ID:              input.ID,
		LayerID:         input.LayerID,
		Title:           title,
		Markdown:        input.Markdown,
		ExpectedVersion: expectedVersion,
		Targets:         targets,
	}, nil
}

func (e *Engine) apply(ctx context.Context, owner brain.Principal, mutation Mutation) (Document, error) {
	if !e.ready() {
		return Document{}, brain.ErrInternal
	}
	document, err := e.store.Apply(normalizeContext(ctx), owner, mutation)
	if err != nil {
		return Document{}, mapStoreError(err)
	}
	if !validDocumentSnapshot(document) {
		return Document{}, brain.ErrInternal
	}
	return document, nil
}

func normalizeQuery(input Query) (Query, error) {
	if !validLayerID(input.LayerID) {
		return Query{}, ErrInvalidQuery
	}
	query := input
	if query.Scope == "" {
		query.Scope = ScopeDirect
	}
	if query.State == "" {
		query.State = StateActive
	}
	if query.Scope != ScopeDirect && query.Scope != ScopeSubtree {
		return Query{}, ErrInvalidQuery
	}
	if query.State != StateActive && query.State != StateTrash {
		return Query{}, ErrInvalidQuery
	}
	if query.Limit == 0 {
		query.Limit = DefaultLimit
	}
	if query.Limit < 1 || query.Limit > MaxLimit {
		return Query{}, ErrInvalidQuery
	}
	if query.After != "" && !ValidID(query.After) {
		return Query{}, ErrInvalidQuery
	}
	return query, nil
}

func validPrincipal(owner brain.Principal) bool {
	return owner != ""
}

func validLayerID(id string) bool {
	return ValidID(id)
}

func normalizeTitle(title string) (string, bool) {
	if !utf8.ValidString(title) {
		return "", false
	}
	for _, r := range title {
		if r == 0 || unicode.IsControl(r) {
			return "", false
		}
	}
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > maxTitleRunes {
		return "", false
	}
	return title, true
}

func validMarkdown(markdown string) bool {
	return utf8.ValidString(markdown) && len(markdown) <= maxMarkdown && !strings.ContainsRune(markdown, '\x00')
}

func validDocumentSnapshot(document Document) bool {
	if !ValidID(document.ID) || !validLayerID(document.LayerID) || document.Version < 1 {
		return false
	}
	title, ok := normalizeTitle(document.Title)
	if !ok || title != document.Title || !validMarkdown(document.Markdown) {
		return false
	}
	if (document.DeletedAt == nil) != (document.ExpiresAt == nil) {
		return false
	}
	if document.DeletedAt != nil && document.ExpiresAt.Before(*document.DeletedAt) {
		return false
	}
	for _, link := range document.Links {
		if !ValidID(link.TargetID) {
			return false
		}
		switch link.Status {
		case "missing":
			if link.Title != "" || link.LayerID != "" {
				return false
			}
		case "available", "trash":
			normalized, valid := normalizeTitle(link.Title)
			if !valid || normalized != link.Title || !validLayerID(link.LayerID) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (e *Engine) ready() bool {
	return e != nil && e.store != nil
}

func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrInvalidDocument):
		return ErrInvalidDocument
	case errors.Is(err, ErrInvalidQuery):
		return ErrInvalidQuery
	case errors.Is(err, brain.ErrUnauthenticated):
		return brain.ErrUnauthenticated
	case errors.Is(err, brain.ErrConflict):
		return brain.ErrConflict
	case errors.Is(err, brain.ErrNotFound), errors.Is(err, brain.ErrForbidden):
		return brain.ErrNotFound
	case errors.Is(err, brain.ErrInternal):
		return brain.ErrInternal
	default:
		return brain.ErrInternal
	}
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

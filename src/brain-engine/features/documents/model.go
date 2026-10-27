package documents

import (
	"context"
	"errors"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
)

// MutationKind identifies the state transition requested by Engine. Stores
// must treat the complete Mutation as one atomic operation.
type MutationKind string

const (
	KindCreate  MutationKind = "create"
	KindUpdate  MutationKind = "update"
	KindTrash   MutationKind = "trash"
	KindRestore MutationKind = "restore"

	// Mutation* aliases keep the operation vocabulary close to the port name.
	MutationCreate  = KindCreate
	MutationUpdate  = KindUpdate
	MutationTrash   = KindTrash
	MutationRestore = KindRestore
)

// Mutation is the complete snapshot or lifecycle request passed to Store.
// Targets are derived by Engine from Markdown and are never trusted from an
// external caller.
type Mutation struct {
	Kind            MutationKind
	ID              string
	LayerID         string
	Title           string
	Markdown        string
	ExpectedVersion int64
	Targets         []string
}

// Document is the authoritative document projection returned by Store.
type Document struct {
	ID        string
	LayerID   string
	Title     string
	Markdown  string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	ExpiresAt *time.Time
	Links     []Link
}

// Link is a projection of a canonical document reference in Markdown.
type Link struct {
	TargetID string
	Title    string
	LayerID  string
	Status   string
}

// Query controls document listing below one layer.
type Query struct {
	LayerID string
	Scope   string
	State   string
	After   string
	Limit   int
}

// Page is the limit+1 listing result projected by Store. NextCursor is empty
// when there is no subsequent page.
type Page struct {
	Documents  []Document
	NextCursor string
}

// Store owns persistence concerns that cannot be safely reproduced by the
// Engine: ownership, atomic target indexing, CAS, PostgreSQL clock and
// opportunistic expiry/tombstone cleanup.
type Store interface {
	Apply(context.Context, brain.Principal, Mutation) (Document, error)
	Get(context.Context, brain.Principal, string) (Document, error)
	List(context.Context, brain.Principal, Query) (Page, error)
}

var (
	ErrInvalidDocument = errors.New("documents: invalid document")
	ErrInvalidQuery    = errors.New("documents: invalid query")
)

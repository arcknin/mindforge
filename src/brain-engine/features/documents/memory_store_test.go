package documents

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
)

const documentTrashTTL = 240 * time.Hour

// memoryStore is deliberately test-only. It exercises the Store port's
// transactional obligations without creating a production in-memory backend.
type memoryStore struct {
	mu         sync.Mutex
	now        func() time.Time
	brains     map[brain.Principal]string
	layers     map[string]memoryLayer
	documents  map[string]memoryDocument
	tombstones map[string]memoryTombstone
	failNext   error
}

type memoryLayer struct {
	owner  brain.Principal
	brain  string
	parent string
}

type memoryDocument struct {
	owner    brain.Principal
	document Document
	targets  []string
}

type memoryTombstone struct {
	owner   brain.Principal
	version int64
}

func newMemoryStore(at ...time.Time) *memoryStore {
	now := time.Now().UTC()
	if len(at) != 0 {
		now = at[0].UTC()
	}
	return &memoryStore{
		now:        func() time.Time { return now },
		brains:     make(map[brain.Principal]string),
		layers:     make(map[string]memoryLayer),
		documents:  make(map[string]memoryDocument),
		tombstones: make(map[string]memoryTombstone),
	}
}

func (s *memoryStore) addBrain(owner brain.Principal, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.brains[owner] = id
}

func (s *memoryStore) addLayer(owner brain.Principal, id, parent string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.layers[id] = memoryLayer{owner: owner, brain: s.brains[owner], parent: parent}
}

func (s *memoryStore) advanceTo(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := at.UTC()
	s.now = func() time.Time { return now }
}

func (s *memoryStore) failOnce(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

func (s *memoryStore) Apply(_ context.Context, owner brain.Principal, mutation Mutation) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return Document{}, err
	}
	if _, ok := s.brains[owner]; !ok {
		return Document{}, brain.ErrNotFound
	}
	s.purgeExpiredLocked(owner)
	switch mutation.Kind {
	case KindCreate:
		return s.createLocked(owner, mutation)
	case KindUpdate:
		return s.updateLocked(owner, mutation)
	case KindTrash:
		return s.trashLocked(owner, mutation)
	case KindRestore:
		return s.restoreLocked(owner, mutation)
	default:
		return Document{}, brain.ErrInternal
	}
}

func (s *memoryStore) Get(_ context.Context, owner brain.Principal, id string) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.brains[owner]; !ok {
		return Document{}, brain.ErrNotFound
	}
	s.purgeExpiredLocked(owner)
	stored, ok := s.documents[id]
	if !ok || stored.owner != owner {
		return Document{}, brain.ErrNotFound
	}
	return s.projectLocked(stored), nil
}

func (s *memoryStore) List(_ context.Context, owner brain.Principal, query Query) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.brains[owner]; !ok {
		return Page{}, brain.ErrNotFound
	}
	if layer, ok := s.layers[query.LayerID]; !ok || layer.owner != owner {
		return Page{}, brain.ErrNotFound
	}
	s.purgeExpiredLocked(owner)
	ids := make([]string, 0)
	for id, stored := range s.documents {
		if stored.owner != owner || !s.stateMatches(stored.document, query.State) {
			continue
		}
		if query.Scope == ScopeSubtree {
			if !s.inSubtreeLocked(owner, stored.document.LayerID, query.LayerID) {
				continue
			}
		} else if stored.document.LayerID != query.LayerID {
			continue
		}
		if query.After != "" && id <= query.After {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	limit := query.Limit
	if limit == 0 {
		limit = DefaultLimit
	}
	page := Page{Documents: make([]Document, 0, minInt(limit, len(ids)))}
	for i, id := range ids {
		if i >= limit {
			page.NextCursor = ids[limit-1]
			break
		}
		stored := s.documents[id]
		document := s.projectLocked(stored)
		document.Markdown = ""
		document.Links = nil
		page.Documents = append(page.Documents, document)
	}
	return page, nil
}

func (s *memoryStore) createLocked(owner brain.Principal, mutation Mutation) (Document, error) {
	if _, exists := s.documents[mutation.ID]; exists {
		return Document{}, brain.ErrConflict
	}
	if _, exists := s.tombstones[mutation.ID]; exists {
		return Document{}, brain.ErrConflict
	}
	if !s.ownedLayerLocked(owner, mutation.LayerID) {
		return Document{}, brain.ErrNotFound
	}
	if !s.targetsOwnedLocked(owner, mutation.Targets) {
		return Document{}, brain.ErrNotFound
	}
	now := s.now().UTC()
	document := Document{
		ID:        mutation.ID,
		LayerID:   mutation.LayerID,
		Title:     mutation.Title,
		Markdown:  mutation.Markdown,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.documents[mutation.ID] = memoryDocument{owner: owner, document: document, targets: cloneStrings(mutation.Targets)}
	return s.projectLocked(s.documents[mutation.ID]), nil
}

func (s *memoryStore) updateLocked(owner brain.Principal, mutation Mutation) (Document, error) {
	stored, ok := s.documents[mutation.ID]
	if !ok || stored.owner != owner {
		return Document{}, brain.ErrNotFound
	}
	if stored.document.DeletedAt != nil || stored.document.Version != mutation.ExpectedVersion {
		return Document{}, brain.ErrConflict
	}
	if stored.document.Version == int64(^uint64(0)>>1) {
		return Document{}, brain.ErrConflict
	}
	if !s.ownedLayerLocked(owner, mutation.LayerID) {
		return Document{}, brain.ErrNotFound
	}
	if !s.targetsOwnedLocked(owner, mutation.Targets) {
		return Document{}, brain.ErrNotFound
	}
	now := s.now().UTC()
	document := stored.document
	document.LayerID = mutation.LayerID
	document.Title = mutation.Title
	document.Markdown = mutation.Markdown
	document.Version++
	document.UpdatedAt = now
	stored.document = document
	stored.targets = cloneStrings(mutation.Targets)
	s.documents[mutation.ID] = stored
	return s.projectLocked(stored), nil
}

func (s *memoryStore) trashLocked(owner brain.Principal, mutation Mutation) (Document, error) {
	stored, ok := s.documents[mutation.ID]
	if !ok || stored.owner != owner {
		return Document{}, brain.ErrNotFound
	}
	if stored.document.Version != mutation.ExpectedVersion {
		return Document{}, brain.ErrConflict
	}
	if stored.document.DeletedAt != nil {
		return s.projectLocked(stored), nil
	}
	if stored.document.Version == int64(^uint64(0)>>1) {
		return Document{}, brain.ErrConflict
	}
	now := s.now().UTC()
	deleted := now
	expires := now.Add(documentTrashTTL)
	document := stored.document
	document.Version++
	document.UpdatedAt = now
	document.DeletedAt = &deleted
	document.ExpiresAt = &expires
	stored.document = document
	s.documents[mutation.ID] = stored
	return s.projectLocked(stored), nil
}

func (s *memoryStore) restoreLocked(owner brain.Principal, mutation Mutation) (Document, error) {
	stored, ok := s.documents[mutation.ID]
	if !ok || stored.owner != owner {
		return Document{}, brain.ErrNotFound
	}
	if stored.document.DeletedAt == nil || stored.document.Version != mutation.ExpectedVersion {
		return Document{}, brain.ErrConflict
	}
	layer := stored.document.LayerID
	if mutation.LayerID != "" {
		layer = mutation.LayerID
	}
	if !s.ownedLayerLocked(owner, layer) {
		return Document{}, brain.ErrNotFound
	}
	if stored.document.Version == int64(^uint64(0)>>1) {
		return Document{}, brain.ErrConflict
	}
	now := s.now().UTC()
	if !now.Before(*stored.document.ExpiresAt) {
		s.purgeDocumentLocked(mutation.ID, stored)
		return Document{}, brain.ErrNotFound
	}
	document := stored.document
	document.LayerID = layer
	document.Version++
	document.UpdatedAt = now
	document.DeletedAt = nil
	document.ExpiresAt = nil
	stored.document = document
	s.documents[mutation.ID] = stored
	return s.projectLocked(stored), nil
}

func (s *memoryStore) ownedLayerLocked(owner brain.Principal, id string) bool {
	layer, ok := s.layers[id]
	return ok && layer.owner == owner
}

func (s *memoryStore) targetsOwnedLocked(owner brain.Principal, targets []string) bool {
	for _, target := range targets {
		if stored, ok := s.documents[target]; ok {
			if stored.owner != owner {
				return false
			}
			continue
		}
		if tombstone, ok := s.tombstones[target]; !ok || tombstone.owner != owner {
			return false
		}
	}
	return true
}

func (s *memoryStore) projectLocked(stored memoryDocument) Document {
	document := stored.document
	document.Links = s.linksLocked(stored.owner, stored.targets)
	return cloneDocument(document)
}

func (s *memoryStore) linksLocked(owner brain.Principal, targets []string) []Link {
	links := make([]Link, 0, len(targets))
	for _, target := range targets {
		link := Link{TargetID: target}
		if stored, ok := s.documents[target]; ok && stored.owner == owner {
			link.Title = stored.document.Title
			link.LayerID = stored.document.LayerID
			if stored.document.DeletedAt == nil {
				link.Status = "available"
			} else {
				link.Status = "trash"
			}
		} else {
			link.Status = "missing"
		}
		links = append(links, link)
	}
	return links
}

func (s *memoryStore) purgeExpiredLocked(owner brain.Principal) {
	now := s.now().UTC()
	for id, stored := range s.documents {
		if stored.owner == owner && stored.document.ExpiresAt != nil && !now.Before(*stored.document.ExpiresAt) {
			s.purgeDocumentLocked(id, stored)
		}
	}
}

func (s *memoryStore) purgeDocumentLocked(id string, stored memoryDocument) {
	delete(s.documents, id)
	s.tombstones[id] = memoryTombstone{owner: stored.owner, version: stored.document.Version}
}

func (s *memoryStore) stateMatches(document Document, state string) bool {
	if state == StateTrash {
		return document.DeletedAt != nil
	}
	return document.DeletedAt == nil
}

func (s *memoryStore) inSubtreeLocked(owner brain.Principal, candidate, root string) bool {
	visited := make(map[string]struct{})
	for current := candidate; current != ""; {
		if current == root {
			return true
		}
		if _, seen := visited[current]; seen {
			return false
		}
		visited[current] = struct{}{}
		layer, ok := s.layers[current]
		if !ok || layer.owner != owner {
			return false
		}
		current = layer.parent
	}
	return false
}

func cloneDocument(document Document) Document {
	clone := document
	if document.DeletedAt != nil {
		deleted := *document.DeletedAt
		clone.DeletedAt = &deleted
	}
	if document.ExpiresAt != nil {
		expires := *document.ExpiresAt
		clone.ExpiresAt = &expires
	}
	clone.Links = append([]Link(nil), document.Links...)
	return clone
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ Store = (*memoryStore)(nil)

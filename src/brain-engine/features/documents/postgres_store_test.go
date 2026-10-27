package documents

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	testBrainID  = "11111111-1111-4111-8111-111111111111"
	testLayerID  = "22222222-2222-4222-8222-222222222222"
	testLayerID2 = "33333333-3333-4333-8333-333333333333"
	testDocID    = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testDocID2   = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func TestPostgresStoreImplementsStoreAndDoesNotAutoMigrate(t *testing.T) {
	var _ Store = (*PostgresStore)(nil)

	if _, err := OpenPostgresStore(context.Background(), ""); !errors.Is(err, brain.ErrInternal) {
		t.Fatalf("empty DSN error = %v, want opaque internal error", err)
	}
	if _, err := OpenPostgresStore(nil, "postgresql://[invalid"); !errors.Is(err, brain.ErrInternal) {
		t.Fatalf("nil context/invalid DSN error = %v, want opaque internal error", err)
	}
}

func TestPostgresStoreApplyCreateCommitsAndDerivesOwnershipFromPrincipal(t *testing.T) {
	fake := newFakeDatabase()
	fake.tx.rows = []fakeRow{
		valuesRow(testBrainID),
		valuesRow(true),
		valuesRow(false),
		documentRow(testDocID, testBrainID, testLayerID, "Title", "body", 1, nil, nil, nil),
	}
	fake.tx.queries = []fakeRowsResult{emptyRows()}

	store := newPostgresStore(fake)
	document, err := store.Apply(context.Background(), brain.Principal("user-a"), Mutation{
		Kind:     KindCreate,
		ID:       testDocID,
		LayerID:  testLayerID,
		Title:    "Title",
		Markdown: "body",
	})
	if err != nil {
		t.Fatalf("Apply(create): %v", err)
	}
	if document.ID != testDocID || document.LayerID != testLayerID || document.Version != 1 {
		t.Fatalf("created document = %#v", document)
	}
	if !fake.tx.committed || !fake.tx.rolledBack {
		t.Fatalf("transaction state = committed %v, rolled back %v; both commit and deferred rollback are expected", fake.tx.committed, fake.tx.rolledBack)
	}
	if len(fake.tx.execSQL) < 3 || !strings.Contains(fake.tx.execSQL[0], "WITH owned AS") || !strings.Contains(fake.tx.execSQL[1], "INSERT INTO brain_documents.documents") {
		t.Fatalf("unexpected SQL sequence: %v", fake.tx.execSQL)
	}
	if !strings.Contains(fake.tx.execSQL[0], "FOR NO KEY UPDATE OF d") || !strings.Contains(fake.tx.execSQL[0], "ORDER BY d.document_id") {
		t.Fatalf("purge does not lock expired documents in deterministic order: %s", fake.tx.execSQL[0])
	}
	if owner := firstArg(fake.tx.queryArgs, "user-a"); !owner {
		t.Fatal("owner principal was not used in the ownership query")
	}
}

func TestPostgresStoreGetForeignOrAbsentDocumentIsOpaqueAndRollsBack(t *testing.T) {
	fake := newFakeDatabase()
	fake.tx.rows = []fakeRow{{err: pgx.ErrNoRows}}
	store := newPostgresStore(fake)

	_, err := store.Get(context.Background(), brain.Principal("user-a"), testDocID)
	if !errors.Is(err, brain.ErrNotFound) {
		t.Fatalf("Get(foreign/absent) error = %v, want not found", err)
	}
	if fake.tx.committed || !fake.tx.rolledBack {
		t.Fatalf("failed transaction state = committed %v, rolled back %v", fake.tx.committed, fake.tx.rolledBack)
	}
	if len(fake.tx.querySQL) < 1 || !strings.Contains(fake.tx.querySQL[0], "b.owner_subject = $1") {
		t.Fatalf("ownership predicate missing from SQL: %v", fake.tx.querySQL)
	}
}

func TestPostgresStoreGetCommitsTombstoneCleanupBeforeReturningNotFound(t *testing.T) {
	tombstoned := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	fake := newFakeDatabase()
	fake.tx.rows = []fakeRow{
		documentRow(testDocID, testBrainID, "", "", "", 9, &tombstoned, nil, &tombstoned),
	}
	store := newPostgresStore(fake)

	_, err := store.Get(context.Background(), brain.Principal("user-a"), testDocID)
	if !errors.Is(err, brain.ErrNotFound) {
		t.Fatalf("Get(tombstone) error = %v, want not found", err)
	}
	if !fake.tx.committed || !fake.tx.rolledBack {
		t.Fatalf("tombstone maintenance transaction state = committed %v, rolled back %v", fake.tx.committed, fake.tx.rolledBack)
	}
}

func TestPostgresStoreGetProjectsCurrentTargetTitle(t *testing.T) {
	fake := newFakeDatabase()
	fake.tx.rows = []fakeRow{
		documentRow(testDocID, testBrainID, testLayerID, "Source", "[[doc:"+testDocID2+"]]", 2, nil, nil, nil),
	}
	fake.tx.queries = []fakeRowsResult{rowsOf(valuesRow(testDocID2, "Renamed", testLayerID2, "available"))}
	store := newPostgresStore(fake)

	document, err := store.Get(context.Background(), brain.Principal("user-a"), testDocID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(document.Links) != 1 || document.Links[0].Title != "Renamed" || document.Links[0].TargetID != testDocID2 {
		t.Fatalf("dynamic links = %#v", document.Links)
	}
	if len(fake.tx.querySQL) < 2 || !strings.Contains(fake.tx.querySQL[1], "target.expires_at <= statement_timestamp()") {
		t.Fatalf("link projection does not recheck target expiry with server clock: %v", fake.tx.querySQL)
	}
	if !fake.tx.committed {
		t.Fatal("successful read did not commit its transaction")
	}
}

func TestPostgresStoreUpdateUsesCASAndRollsBackWhenTargetIsNotOwned(t *testing.T) {
	fake := newFakeDatabase()
	fake.tx.rows = []fakeRow{
		documentRow(testDocID, testBrainID, testLayerID, "Old", "old", 7, nil, nil, nil),
		valuesRow(true),
		valuesRow(false),
	}
	store := newPostgresStore(fake)

	_, err := store.Apply(context.Background(), brain.Principal("user-a"), Mutation{
		Kind:            KindUpdate,
		ID:              testDocID,
		LayerID:         testLayerID2,
		Title:           "New",
		Markdown:        "new",
		ExpectedVersion: 7,
		Targets:         []string{testDocID2},
	})
	if !errors.Is(err, brain.ErrNotFound) {
		t.Fatalf("Apply(update with foreign target) error = %v, want not found", err)
	}
	if fake.tx.committed || !fake.tx.rolledBack {
		t.Fatalf("failed update transaction state = committed %v, rolled back %v", fake.tx.committed, fake.tx.rolledBack)
	}
	joined := strings.Join(fake.tx.execSQL, "\n")
	if !strings.Contains(joined, "version = $6") || !strings.Contains(joined, "deleted_at IS NULL") {
		t.Fatalf("CAS predicates missing from update SQL: %s", joined)
	}
}

func TestPostgresStoreTrashIsIdempotentForCurrentTrashVersion(t *testing.T) {
	deleted := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	expires := deleted.Add(240 * time.Hour)
	fake := newFakeDatabase()
	fake.tx.rows = []fakeRow{
		documentRow(testDocID, testBrainID, testLayerID, "Trashed", "body", 8, &deleted, &expires, nil),
	}
	fake.tx.queries = []fakeRowsResult{emptyRows()}
	store := newPostgresStore(fake)

	document, err := store.Apply(context.Background(), brain.Principal("user-a"), Mutation{
		Kind:            KindTrash,
		ID:              testDocID,
		ExpectedVersion: 8,
	})
	if err != nil {
		t.Fatalf("repeat trash: %v", err)
	}
	if document.Version != 8 || document.DeletedAt == nil || document.ExpiresAt == nil {
		t.Fatalf("repeat trash changed lifecycle: %#v", document)
	}
	if strings.Contains(strings.Join(fake.tx.execSQL, "\n"), "SET deleted_at = clock_timestamp()") {
		t.Fatal("repeat trash renewed expiry/version")
	}
	if !fake.tx.committed {
		t.Fatal("repeat trash did not commit")
	}
}

func TestPostgresStoreListUsesRecursiveCTEAndExclusiveUUIDPaging(t *testing.T) {
	fake := newFakeDatabase()
	fake.tx.rows = []fakeRow{valuesRow(testBrainID), valuesRow(true)}
	fake.tx.queries = []fakeRowsResult{rowsOf(
		listRow(testDocID, testLayerID, "A", 1),
		listRow(testDocID2, testLayerID2, "B", 1),
	)}
	store := newPostgresStore(fake)

	page, err := store.List(context.Background(), brain.Principal("user-a"), Query{
		LayerID: testLayerID,
		Scope:   ScopeSubtree,
		State:   StateActive,
		After:   testDocID,
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Documents) != 1 || page.Documents[0].ID != testDocID || page.NextCursor != testDocID {
		t.Fatalf("page = %#v", page)
	}
	if len(fake.tx.querySQL) < 2 || !strings.Contains(fake.tx.querySQL[len(fake.tx.querySQL)-1], "WITH RECURSIVE") || !strings.Contains(fake.tx.querySQL[len(fake.tx.querySQL)-1], "d.document_id > $6::uuid") {
		t.Fatalf("recursive/paging SQL missing: %v", fake.tx.querySQL)
	}
	if !fake.tx.committed {
		t.Fatal("successful list did not commit")
	}
	if fake.tx.lastRows == nil || !fake.tx.lastRows.closed {
		t.Fatal("list committed before closing PostgreSQL rows")
	}
}

func TestPostgresStoreBeginAndCommitFailuresFailClosed(t *testing.T) {
	beginFailure := newFakeDatabase()
	beginFailure.beginErr = errors.New("begin failed")
	if _, err := newPostgresStore(beginFailure).Get(context.Background(), brain.Principal("user-a"), testDocID); !errors.Is(err, brain.ErrInternal) {
		t.Fatalf("begin failure = %v, want internal", err)
	}

	commitFailure := newFakeDatabase()
	commitFailure.tx.commitErr = errors.New("commit failed")
	commitFailure.tx.rows = []fakeRow{
		valuesRow(testBrainID),
		valuesRow(true),
		valuesRow(false),
		documentRow(testDocID, testBrainID, testLayerID, "Title", "body", 1, nil, nil, nil),
	}
	commitFailure.tx.queries = []fakeRowsResult{emptyRows()}
	if _, err := newPostgresStore(commitFailure).Apply(context.Background(), brain.Principal("user-a"), Mutation{
		Kind: KindCreate, ID: testDocID, LayerID: testLayerID, Title: "Title", Markdown: "body",
	}); !errors.Is(err, brain.ErrInternal) {
		t.Fatalf("commit failure = %v, want internal", err)
	}
	if !commitFailure.tx.commitCalled || !commitFailure.tx.rolledBack {
		t.Fatalf("commit failure transaction state = called %v, rolled back %v", commitFailure.tx.commitCalled, commitFailure.tx.rolledBack)
	}
}

type fakeDatabase struct {
	tx       *fakeTransaction
	beginErr error
}

func newFakeDatabase() *fakeDatabase {
	return &fakeDatabase{tx: &fakeTransaction{}}
}

func (f *fakeDatabase) BeginTx(context.Context, pgx.TxOptions) (transaction, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.tx, nil
}

type fakeTransaction struct {
	execSQL      []string
	querySQL     []string
	queryArgs    [][]any
	execErrs     []error
	queryErrs    []error
	rows         []fakeRow
	queries      []fakeRowsResult
	lastRows     *fakeRowsResult
	commitErr    error
	commitCalled bool
	committed    bool
	rolledBack   bool
}

func (f *fakeTransaction) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	f.execSQL = append(f.execSQL, sql)
	if len(f.execErrs) > 0 {
		err := f.execErrs[0]
		f.execErrs = f.execErrs[1:]
		return pgconn.CommandTag{}, err
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (f *fakeTransaction) Query(_ context.Context, sql string, args ...any) (rows, error) {
	f.querySQL = append(f.querySQL, sql)
	f.queryArgs = append(f.queryArgs, args)
	if len(f.queryErrs) > 0 {
		err := f.queryErrs[0]
		f.queryErrs = f.queryErrs[1:]
		return nil, err
	}
	if len(f.queries) == 0 {
		result := emptyRows()
		f.lastRows = &result
		return &result, nil
	}
	result := f.queries[0]
	f.queries = f.queries[1:]
	f.lastRows = &result
	return &result, nil
}

func (f *fakeTransaction) QueryRow(_ context.Context, sql string, args ...any) row {
	f.querySQL = append(f.querySQL, sql)
	f.queryArgs = append(f.queryArgs, args)
	if len(f.rows) == 0 {
		return fakeRow{err: pgx.ErrNoRows}
	}
	result := f.rows[0]
	f.rows = f.rows[1:]
	return result
}

func (f *fakeTransaction) Commit(context.Context) error {
	f.commitCalled = true
	if f.commitErr != nil {
		return f.commitErr
	}
	f.committed = true
	return nil
}

func (f *fakeTransaction) Rollback(context.Context) error {
	f.rolledBack = true
	return nil
}

type fakeRow struct {
	values []any
	err    error
}

func valuesRow(values ...any) fakeRow {
	return fakeRow{values: values}
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("fake row column count mismatch")
	}
	for index, value := range r.values {
		if err := assignFakeValue(dest[index], value); err != nil {
			return err
		}
	}
	return nil
}

func documentRow(id, brainID, layerID, title, markdown string, version int64, deletedAt, expiresAt, tombstonedAt *time.Time) fakeRow {
	return valuesRow(id, brainID, layerID, title, markdown, version, time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC(), deletedAt, expiresAt, tombstonedAt)
}

func listRow(id, layerID, title string, version int64) fakeRow {
	return valuesRow(id, layerID, title, version, time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC(), nil, nil)
}

type fakeRowsResult struct {
	rows   []fakeRow
	index  int
	err    error
	closed bool
}

func emptyRows() fakeRowsResult {
	return fakeRowsResult{}
}

func rowsOf(rows ...fakeRow) fakeRowsResult {
	return fakeRowsResult{rows: rows}
}

func (r *fakeRowsResult) Next() bool {
	return r.index < len(r.rows)
}

func (r *fakeRowsResult) Scan(dest ...any) error {
	if r.index >= len(r.rows) {
		return errors.New("fake rows exhausted")
	}
	err := r.rows[r.index].Scan(dest...)
	r.index++
	return err
}

func (r *fakeRowsResult) Close() { r.closed = true }

func (r *fakeRowsResult) Err() error { return r.err }

func assignFakeValue(destination any, value any) error {
	if point, ok := value.(*time.Time); ok && point == nil {
		if target, ok := destination.(**time.Time); ok {
			*target = nil
			return nil
		}
	}
	if value == nil {
		switch target := destination.(type) {
		case **time.Time:
			*target = nil
			return nil
		case *string:
			*target = ""
			return nil
		}
	}
	switch target := destination.(type) {
	case *string:
		*target = value.(string)
	case *int64:
		*target = value.(int64)
	case *bool:
		*target = value.(bool)
	case *time.Time:
		*target = value.(time.Time)
	case **time.Time:
		switch point := value.(type) {
		case time.Time:
			copy := point
			*target = &copy
		case *time.Time:
			*target = point
		default:
			return errors.New("unsupported fake nullable time value")
		}
	default:
		return errors.New("unsupported fake scan destination " + reflect.TypeOf(destination).String())
	}
	return nil
}

func firstArg(arguments [][]any, want string) bool {
	for _, values := range arguments {
		for _, value := range values {
			if value == want {
				return true
			}
		}
	}
	return false
}

package brain

import (
	"context"
	"errors"
	"testing"
)

func TestPostgresStoreImplementsStoreAndDoesNotAutoMigrate(t *testing.T) {
	var _ Store = (*PostgresStore)(nil)

	if _, err := OpenPostgresStore(context.Background(), ""); !errors.Is(err, ErrInternal) {
		t.Fatalf("empty DSN error = %v, want opaque internal error", err)
	}
}

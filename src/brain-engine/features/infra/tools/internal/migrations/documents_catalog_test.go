package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadCatalogRegistersDocumentsMigrationAndPinnedChecks(t *testing.T) {
	t.Parallel()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate migration package")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../../../../.."))
	catalog, err := LoadCatalog(repositoryRoot)
	if err != nil {
		t.Fatalf("LoadCatalog(source): %v", err)
	}
	if len(catalog.Migrations) != 7 {
		t.Fatalf("catalog has %d migrations, want infra 0001-0005 plus feature 0006-0007", len(catalog.Migrations))
	}
	if catalog.Migrations[5].ID != "0006" || catalog.Migrations[5].SHA256 != "1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9" {
		t.Fatalf("existing 0006 entry changed: %#v", catalog.Migrations[5])
	}
	feature := catalog.Migrations[6]
	if feature.ID != "0007" || feature.Name != "documents" || feature.Feature != "documents" || feature.Kind != KindFeature {
		t.Fatalf("documents feature migration = %#v", feature)
	}
	if feature.RelativePath != "src/brain-engine/features/documents/migrations/0001_documents.sql" {
		t.Fatalf("documents source path = %q", feature.RelativePath)
	}
	for _, pinned := range []struct {
		path string
		want string
	}{
		{path: feature.Path, want: feature.SHA256},
		{path: feature.PreflightPath, want: feature.PreflightSHA256},
		{path: feature.VerificationPath, want: feature.VerificationSHA256},
	} {
		contents, err := os.ReadFile(pinned.path)
		if err != nil {
			t.Fatalf("read pinned catalog file %s: %v", pinned.path, err)
		}
		digest := sha256.Sum256(contents)
		actual := hex.EncodeToString(digest[:])
		if actual != pinned.want {
			t.Errorf("sha256(%s) = %s, want %s", pinned.path, actual, pinned.want)
		}
	}
}

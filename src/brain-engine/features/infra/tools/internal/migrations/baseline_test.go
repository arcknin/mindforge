package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGlobalCatalogBaselineMatchesPayloadAndPreservesPriorPins(t *testing.T) {
	t.Parallel()

	baseline := GlobalMigrationLedgerBaseline()
	wantPrior := map[string]string{
		"0001": "2625ae851aa42f59d7c071a563141fcb6ecf6c7301157786fedc1ce7e7725fdb",
		"0002": "c4fdcc68396d7bff76bc6325d3171f5d73bbb0ee273709be260ff1f30d92b7cb",
		"0003": "613d12044705182f42e228fcbbac099219e24d32e0390fed5e6e8243a534a94b",
		"0004": "3b21d7e7dfaa37e4998835016c19b663a1f0fae7d3c36a2f81d9bc59078c8c09",
		"0005": "4cc60fd5c3d653f5492c2aab03220f5a74f01a58ea3b8270b25b75dd03f930ed",
		"0006": "1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9",
		"0007": "37c0378f6a4d18e98505e31baf50ebb8e36da071868beae05e4631a92fed351f",
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate migration package")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../../../../.."))
	catalog, err := LoadCatalog(repositoryRoot)
	if err != nil {
		t.Fatalf("LoadCatalog(source): %v", err)
	}
	if len(catalog.Migrations) != 7 || len(baseline) != 7 {
		t.Fatalf("global catalog has %d migrations and %d pins, want infra 0001-0005 plus feature 0006-0007", len(catalog.Migrations), len(baseline))
	}
	for _, migration := range catalog.Migrations {
		contents, err := os.ReadFile(migration.Path)
		if err != nil {
			t.Fatalf("read migration %s: %v", migration.ID, err)
		}
		digest := sha256.Sum256(contents)
		actual := hex.EncodeToString(digest[:])
		if baseline[migration.ID] != actual {
			t.Errorf("baseline[%s] = %s, source hash = %s", migration.ID, baseline[migration.ID], actual)
		}
		if expected, exists := wantPrior[migration.ID]; exists && baseline[migration.ID] != expected {
			t.Errorf("baseline[%s] changed from preserved pin %s to %s", migration.ID, expected, baseline[migration.ID])
		}
	}
	feature := catalog.Migrations[6]
	if feature.Kind != KindFeature || feature.PreflightSHA256 == "" || feature.VerificationSHA256 == "" {
		t.Fatalf("feature catalog checks are not pinned: %#v", feature)
	}
	if actual := CatalogHash(catalog); actual != GlobalCatalogHashBaseline() {
		t.Errorf("catalog hash = %s, baseline = %s", actual, GlobalCatalogHashBaseline())
	}
	if feature.ID != "0007" {
		t.Errorf("feature global ID = %s, want 0007", feature.ID)
	}
}

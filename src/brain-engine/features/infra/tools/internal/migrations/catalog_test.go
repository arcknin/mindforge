package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCatalogRegistersCanonicalFeatureSourceAndChecks(t *testing.T) {
	t.Parallel()

	root, expected := featureCatalogFixture(t)
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(): %v", err)
	}
	if len(catalog.Migrations) != 6 {
		t.Fatalf("catalog has %d migrations, want infra 0001-0005 plus feature 0006", len(catalog.Migrations))
	}
	feature := catalog.Migrations[5]
	if feature.ID != "0006" || feature.Name != "brain_core" || feature.Kind != KindFeature {
		t.Fatalf("feature migration = %#v, want global 0006 brain_core", feature)
	}
	if feature.SHA256 != expected["source"] || feature.Path != filepath.Join(catalog.Root, brainSource) {
		t.Fatalf("feature migration source = %#v, want canonical feature-owned bytes", feature)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(root): %v", err)
	}
	if catalog.Root != canonicalRoot {
		t.Fatalf("catalog root = %q, want canonical repository root %q", catalog.Root, canonicalRoot)
	}
	if _, err := os.Stat(filepath.Join(root, "src/brain-engine/features/infra/postgres/feature-migrations/0006_brain_core.sql")); !os.IsNotExist(err) {
		t.Fatalf("infra catalog contains a copied feature payload: stat error = %v", err)
	}
}

func TestLoadCatalogBindsFeatureChecksToCatalogHash(t *testing.T) {
	t.Parallel()

	root, _ := featureCatalogFixture(t)
	first, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(first): %v", err)
	}
	checkPath := filepath.Join(root, "src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql")
	if err := os.WriteFile(checkPath, []byte("SELECT 'complete';\n"), 0o600); err != nil {
		t.Fatalf("change check source: %v", err)
	}
	if _, err := LoadCatalog(root); err == nil {
		t.Fatal("LoadCatalog() accepted changed feature-check bytes with the original descriptor hash")
	}

	secondRoot, _ := featureCatalogFixture(t)
	secondCheckPath := filepath.Join(secondRoot, "src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql")
	if err := os.WriteFile(secondCheckPath, []byte("SELECT 'complete';\n"), 0o600); err != nil {
		t.Fatalf("change second check source: %v", err)
	}
	updateFeatureDescriptorHashes(t, secondRoot)
	second, err := LoadCatalog(secondRoot)
	if err != nil {
		t.Fatalf("LoadCatalog(second): %v", err)
	}
	if CatalogHash(first) == CatalogHash(second) {
		t.Fatal("CatalogHash() did not change when an approved feature check changed")
	}
}

func TestLoadCatalogRejectsInvalidFeatureDescriptorSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(t *testing.T, root string)
	}{
		{
			name: "missing verification check",
			mutate: func(t *testing.T, root string) {
				writeFeatureDescriptor(t, root, featureEntryFixture{ID: "0006", Name: "brain_core", Feature: "brain", Source: brainSource, SHA256: hashFixture("canonical brain sql"), Preflight: brainPreflight, PreflightSHA256: hashFixture("preflight brain sql")})
			},
		},
		{
			name: "missing feature source",
			mutate: func(t *testing.T, root string) {
				if err := os.Remove(filepath.Join(root, brainSource)); err != nil {
					t.Fatalf("remove canonical feature source: %v", err)
				}
			},
		},
		{
			name: "missing preflight check file",
			mutate: func(t *testing.T, root string) {
				if err := os.Remove(filepath.Join(root, brainPreflight)); err != nil {
					t.Fatalf("remove preflight check: %v", err)
				}
			},
		},
		{
			name: "unsafe source path",
			mutate: func(t *testing.T, root string) {
				entry := validFeatureEntry(root)
				entry.Source = "../brain/migrations/0001_brain_core.sql"
				writeFeatureDescriptor(t, root, entry)
			},
		},
		{
			name: "unsafe check path",
			mutate: func(t *testing.T, root string) {
				entry := validFeatureEntry(root)
				entry.Verify = "src/brain-engine/features/other/migrations/0001_brain_core.verify.sql"
				writeFeatureDescriptor(t, root, entry)
			},
		},
		{
			name: "changed source hash",
			mutate: func(t *testing.T, root string) {
				entry := validFeatureEntry(root)
				entry.SHA256 = strings.Repeat("a", 64)
				writeFeatureDescriptor(t, root, entry)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, _ := featureCatalogFixture(t)
			test.mutate(t, root)
			if _, err := LoadCatalog(root); err == nil {
				t.Fatalf("LoadCatalog() accepted %s", test.name)
			}
		})
	}
}

func TestLoadCatalogRejectsDuplicateGlobalIDsAndFeatureSymlinkEscape(t *testing.T) {
	t.Parallel()

	t.Run("duplicate global ID", func(t *testing.T) {
		root, _ := featureCatalogFixture(t)
		writeMigration(t, root, "src/brain-engine/features/infra/postgres/migrations/0006_infra_extra.sql", "-- extra\n")
		if _, err := LoadCatalog(root); err == nil {
			t.Fatal("LoadCatalog() error = nil, want duplicate global ID rejection")
		}
	})

	t.Run("noncontiguous global IDs", func(t *testing.T) {
		root, _ := featureCatalogFixture(t)
		missing := filepath.Join(root, "src/brain-engine/features/infra/postgres/migrations/0003_infra_3.sql")
		if err := os.Remove(missing); err != nil {
			t.Fatalf("remove migration gap fixture: %v", err)
		}
		if _, err := LoadCatalog(root); err == nil {
			t.Fatal("LoadCatalog() error = nil, want globally noncontiguous migration rejection")
		}
	})

	t.Run("infrastructure source symlink escapes repository", func(t *testing.T) {
		root, _ := featureCatalogFixture(t)
		source := filepath.Join(root, "src/brain-engine/features/infra/postgres/migrations/0001_infra_1.sql")
		if err := os.Remove(source); err != nil {
			t.Fatalf("remove infra fixture: %v", err)
		}
		outside := filepath.Join(t.TempDir(), "0001_infra_1.sql")
		if err := os.WriteFile(outside, []byte("-- infrastructure 1\n"), 0o600); err != nil {
			t.Fatalf("write outside migration: %v", err)
		}
		if err := os.Symlink(outside, source); err != nil {
			t.Skipf("symlink fixture unavailable: %v", err)
		}
		if _, err := LoadCatalog(root); err == nil {
			t.Fatal("LoadCatalog() error = nil, want out-of-repository infra source rejection")
		}
	})

	t.Run("feature source symlink escapes repository", func(t *testing.T) {
		root, _ := featureCatalogFixture(t)
		source := filepath.Join(root, brainSource)
		if err := os.Remove(source); err != nil {
			t.Fatalf("remove fixture source: %v", err)
		}
		outside := filepath.Join(t.TempDir(), "0001_brain_core.sql")
		contents := []byte("canonical brain sql")
		if err := os.WriteFile(outside, contents, 0o600); err != nil {
			t.Fatalf("write outside source: %v", err)
		}
		if err := os.Symlink(outside, source); err != nil {
			t.Skipf("symlink fixture unavailable: %v", err)
		}
		updateFeatureDescriptorHashes(t, root)
		if _, err := LoadCatalog(root); err == nil {
			t.Fatal("LoadCatalog() error = nil, want out-of-repository feature source rejection")
		}
	})

	t.Run("feature check symlink escapes repository", func(t *testing.T) {
		root, _ := featureCatalogFixture(t)
		check := filepath.Join(root, brainPreflight)
		if err := os.Remove(check); err != nil {
			t.Fatalf("remove preflight fixture: %v", err)
		}
		outside := filepath.Join(t.TempDir(), "0001_brain_core.preflight.sql")
		if err := os.WriteFile(outside, []byte("SELECT 'absent_eligible';\n"), 0o600); err != nil {
			t.Fatalf("write outside check: %v", err)
		}
		if err := os.Symlink(outside, check); err != nil {
			t.Skipf("symlink fixture unavailable: %v", err)
		}
		updateFeatureDescriptorHashes(t, root)
		if _, err := LoadCatalog(root); err == nil {
			t.Fatal("LoadCatalog() error = nil, want out-of-repository feature check rejection")
		}
	})
}

func TestVerifyLedgerRejectsChangedAppliedBytes(t *testing.T) {
	t.Parallel()

	root, _ := featureCatalogFixture(t)
	original, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(original): %v", err)
	}
	ledger := NewStaticLedger(original)
	path := filepath.Join(root, "src/brain-engine/features/infra/postgres/migrations/0001_infra_1.sql")
	if err := os.WriteFile(path, []byte("changed\n"), 0o600); err != nil {
		t.Fatalf("mutate migration fixture: %v", err)
	}
	changed, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(changed): %v", err)
	}
	if err := VerifyLedger(changed, ledger); err == nil {
		t.Fatal("VerifyLedger() error = nil, want changed applied bytes rejection")
	}
}

func TestCanonicalFilenameRejectsUnsafeNames(t *testing.T) {
	t.Parallel()

	if _, err := ParseFilename("0001_good.sql"); err != nil {
		t.Fatalf("ParseFilename(valid): %v", err)
	}
	for _, name := range []string{"0001-not-good.sql", "0001_Good.sql", "0001_good.psql"} {
		_, err := ParseFilename(name)
		if err == nil {
			t.Fatalf("ParseFilename(%q) error = nil", name)
		}
	}
}

const (
	brainSource    = "src/brain-engine/features/brain/migrations/0001_brain_core.sql"
	brainPreflight = "src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql"
	brainVerify    = "src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql"
)

type featureEntryFixture struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Feature         string `json:"feature"`
	Source          string `json:"source"`
	SHA256          string `json:"sha256"`
	Preflight       string `json:"preflight"`
	PreflightSHA256 string `json:"preflight_sha256"`
	Verify          string `json:"verify"`
	VerifySHA256    string `json:"verify_sha256"`
}

func featureCatalogFixture(t *testing.T) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	for _, relative := range []string{
		"src/brain-engine/features/infra/postgres/migrations",
		"src/brain-engine/features/infra/postgres/feature-migrations",
		"src/brain-engine/features/brain/migrations",
	} {
		if err := os.MkdirAll(filepath.Join(root, relative), 0o700); err != nil {
			t.Fatalf("create fixture directory: %v", err)
		}
	}
	for index := 1; index <= 5; index++ {
		writeMigration(t, root, fmt.Sprintf("src/brain-engine/features/infra/postgres/migrations/%04d_infra_%d.sql", index, index), fmt.Sprintf("-- infrastructure %d\n", index))
	}
	contents := map[string]string{
		brainSource:    "canonical brain sql",
		brainPreflight: "preflight brain sql",
		brainVerify:    "verify brain sql",
	}
	hashes := make(map[string]string)
	for relative, content := range contents {
		writeMigration(t, root, relative, content)
		if relative == brainSource {
			hashes["source"] = hashFixture(content)
		} else if relative == brainPreflight {
			hashes["preflight"] = hashFixture(content)
		} else if relative == brainVerify {
			hashes["verify"] = hashFixture(content)
		}
	}
	writeFeatureDescriptor(t, root, validFeatureEntry(root))
	return root, hashes
}

func validFeatureEntry(root string) featureEntryFixture {
	return featureEntryFixture{
		ID:              "0006",
		Name:            "brain_core",
		Feature:         "brain",
		Source:          brainSource,
		SHA256:          fileHashFixture(root, brainSource),
		Preflight:       brainPreflight,
		PreflightSHA256: fileHashFixture(root, brainPreflight),
		Verify:          brainVerify,
		VerifySHA256:    fileHashFixture(root, brainVerify),
	}
}

func updateFeatureDescriptorHashes(t *testing.T, root string) {
	t.Helper()
	writeFeatureDescriptor(t, root, validFeatureEntry(root))
}

func writeFeatureDescriptor(t *testing.T, root string, entries ...featureEntryFixture) {
	t.Helper()
	payload := struct {
		SchemaVersion int                   `json:"schema_version"`
		Migrations    []featureEntryFixture `json:"migrations"`
	}{SchemaVersion: 1, Migrations: entries}
	contents, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	path := filepath.Join(root, "src/brain-engine/features/infra/postgres/feature-migrations/catalog.json")
	if err := os.WriteFile(path, append(contents, '\n'), 0o600); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}
}

func writeMigration(t *testing.T, root, relative, contents string) string {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create parent for %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write migration %s: %v", relative, err)
	}
	return path
}

func fileHashFixture(root, relative string) string {
	contents, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		return strings.Repeat("0", 64)
	}
	return hashFixture(string(contents))
}

func hashFixture(contents string) string {
	digest := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(digest[:])
}

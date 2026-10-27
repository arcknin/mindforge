package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
)

func TestAuditRejectsProductObjectsSQLRoutingExtraServicesAndChangedBytes(t *testing.T) {
	t.Parallel()

	t.Run("accepts the reduced infrastructure boundary", func(t *testing.T) {
		root := auditFixture(t, "CREATE EXTENSION IF NOT EXISTS vector;\n", allowedCompose())
		catalog := mustCatalog(t, root)
		if err := AuditSource(root, migrations.NewStaticLedger(catalog)); err != nil {
			t.Fatalf("AuditSource(): %v", err)
		}
	})

	t.Run("accepts approved feature-owned product SQL without infra-only restrictions", func(t *testing.T) {
		root := auditFixtureWithFeature(t, allowedCompose(), "CREATE TABLE brain_core.brains (brain_id uuid);\n")
		catalog := mustCatalog(t, root)
		if err := AuditSource(root, migrations.NewStaticLedger(catalog)); err != nil {
			t.Fatalf("AuditSource(): %v", err)
		}
	})

	t.Run("pinned catalog hash detects changed feature checks", func(t *testing.T) {
		root := auditFixtureWithFeature(t, allowedCompose(), "CREATE TABLE brain_core.brains (brain_id uuid);\n")
		catalog := mustCatalog(t, root)
		pinned := migrations.CatalogHash(catalog)
		checkPath := filepath.Join(root, "src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql")
		if err := os.WriteFile(checkPath, []byte("SELECT 'changed';\n"), 0o600); err != nil {
			t.Fatalf("change feature check: %v", err)
		}
		updateAuditFeatureDescriptor(t, root)
		changed := mustCatalog(t, root)
		if err := AuditSource(root, migrations.NewStaticLedger(changed), pinned); err == nil {
			t.Fatal("AuditSource() error = nil, want pinned feature-check hash rejection")
		}
	})

	t.Run("rejects product table", func(t *testing.T) {
		root := auditFixture(t, "CREATE TABLE brains (id text);\n", allowedCompose())
		catalog := mustCatalog(t, root)
		if err := AuditSource(root, migrations.NewStaticLedger(catalog)); err == nil {
			t.Fatal("AuditSource() error = nil, want product object rejection")
		}
	})

	t.Run("rejects SQL endpoint promotion", func(t *testing.T) {
		root := auditFixture(t, "-- OpenAPI HTTP endpoint for this SQL function\nCREATE EXTENSION IF NOT EXISTS vector;\n", allowedCompose())
		catalog := mustCatalog(t, root)
		if err := AuditSource(root, migrations.NewStaticLedger(catalog)); err == nil {
			t.Fatal("AuditSource() error = nil, want SQL routing rejection")
		}
	})

	t.Run("rejects extra Supabase service", func(t *testing.T) {
		compose := strings.Replace(allowedCompose(), "volumes:\n", "  postgrest:\n    image: postgrest/postgrest:v12\nvolumes:\n", 1)
		root := auditFixture(t, "CREATE EXTENSION IF NOT EXISTS vector;\n", compose)
		catalog := mustCatalog(t, root)
		if err := AuditSource(root, migrations.NewStaticLedger(catalog)); err == nil {
			t.Fatal("AuditSource() error = nil, want extra service rejection")
		}
	})

	t.Run("rejects changed applied bytes", func(t *testing.T) {
		root := auditFixture(t, "CREATE EXTENSION IF NOT EXISTS vector;\n", allowedCompose())
		catalog := mustCatalog(t, root)
		ledger := migrations.NewStaticLedger(catalog)
		path := filepath.Join(root, "src/brain-engine/features/infra/postgres/migrations/0001_infra_init.sql")
		if err := os.WriteFile(path, []byte("CREATE EXTENSION IF NOT EXISTS citext;\n"), 0o600); err != nil {
			t.Fatalf("change migration fixture: %v", err)
		}
		if err := AuditSource(root, ledger); err == nil {
			t.Fatal("AuditSource() error = nil, want immutable ledger rejection")
		}
	})
}

func auditFixture(t *testing.T, migrationSQL, compose string) string {
	t.Helper()
	root := t.TempDir()
	for _, relative := range []string{
		"src/brain-engine/features/infra/postgres/migrations",
		"src/brain-engine/features/infra/postgres/feature-migrations",
		"contracts/openapi",
	} {
		if err := os.MkdirAll(filepath.Join(root, relative), 0o700); err != nil {
			t.Fatalf("create fixture directory: %v", err)
		}
	}
	write := func(relative, contents string) {
		path := filepath.Join(root, relative)
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write fixture %s: %v", relative, err)
		}
	}
	for index := 1; index <= 5; index++ {
		contents := "-- infrastructure catalog fixture\n"
		if index == 1 {
			contents = migrationSQL
		}
		write(filepath.Join("src/brain-engine/features/infra/postgres/migrations", fmt.Sprintf("%04d_infra_%d.sql", index, index)), contents)
	}
	write("src/brain-engine/features/infra/postgres/feature-migrations/.gitkeep", "")
	write("src/brain-engine/features/infra/postgres/feature-migrations/README.md", "Feature-owned migrations require an approved feature contract.\n")
	write("src/brain-engine/features/infra/postgres/feature-migrations/catalog.json", "{\"schema_version\":1,\"migrations\":[]}\n")
	write("src/brain-engine/features/infra/compose.yaml", compose)
	write("contracts/openapi/registry.yaml", "endpoints: []\n")
	return root
}

func auditFixtureWithFeature(t *testing.T, compose, featureSQL string) string {
	t.Helper()
	root := auditFixture(t, "CREATE EXTENSION IF NOT EXISTS vector;\n", compose)
	write := func(relative, contents string) {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("create feature fixture parent: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write feature fixture %s: %v", relative, err)
		}
	}
	const source = "src/brain-engine/features/brain/migrations/0001_brain_core.sql"
	const preflight = "src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql"
	const verify = "src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql"
	write(source, featureSQL)
	write(preflight, "SELECT 'absent_eligible';\n")
	write(verify, "DO $$ BEGIN PERFORM 1; END $$;\n")
	updateAuditFeatureDescriptor(t, root)
	return root
}

func updateAuditFeatureDescriptor(t *testing.T, root string) {
	t.Helper()
	const source = "src/brain-engine/features/brain/migrations/0001_brain_core.sql"
	const preflight = "src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql"
	const verify = "src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql"
	entry := map[string]string{
		"id": "0006", "name": "brain_core", "feature": "brain",
		"source": source, "sha256": auditFileHash(t, root, source),
		"preflight": preflight, "preflight_sha256": auditFileHash(t, root, preflight),
		"verify": verify, "verify_sha256": auditFileHash(t, root, verify),
	}
	descriptor, err := json.Marshal(struct {
		SchemaVersion int                 `json:"schema_version"`
		Migrations    []map[string]string `json:"migrations"`
	}{SchemaVersion: 1, Migrations: []map[string]string{entry}})
	if err != nil {
		t.Fatalf("marshal feature descriptor: %v", err)
	}
	path := filepath.Join(root, "src/brain-engine/features/infra/postgres/feature-migrations/catalog.json")
	if err := os.WriteFile(path, append(descriptor, '\n'), 0o600); err != nil {
		t.Fatalf("write feature descriptor: %v", err)
	}
}

func auditFileHash(t *testing.T, root, relative string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read feature fixture %s: %v", relative, err)
	}
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:])
}

func allowedCompose() string {
	return "services:\n" +
		"  db:\n    image: pgvector/pgvector@sha256:cccc\n" +
		"  auth-migrate:\n    image: supabase/gotrue:v2.196.0\n" +
		"  auth:\n    image: supabase/gotrue:v2.196.0\n" +
		"  auth-develop-migrate:\n    image: supabase/gotrue:v2.196.0\n" +
		"  auth-develop:\n    image: supabase/gotrue:v2.196.0\n" +
		"volumes:\n  mindforge-postgres-data:\n"
}

func mustCatalog(t *testing.T, root string) migrations.Catalog {
	t.Helper()
	catalog, err := migrations.LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(): %v", err)
	}
	return catalog
}

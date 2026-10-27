package brain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestBrainCoreMigrationStaticShape(t *testing.T) {
	sqlBytes, err := os.ReadFile("migrations/0001_brain_core.sql")
	if err != nil {
		t.Fatalf("read Brain Core migration: %v", err)
	}
	sql := strings.ToLower(string(sqlBytes))

	for _, fragment := range []string{
		"create schema if not exists brain_core",
		"create table brain_core.brains",
		"owner_subject",
		"unique",
		"brain_id uuid",
		"create table brain_core.nodes",
		"kind text",
		"kind in ('root', 'layer')",
		"parent_id uuid",
		"foreign key (brain_id, parent_id)",
		"create unique index",
		"version bigint",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing required fragment %q", fragment)
		}
	}

	for _, forbidden := range []string{
		"inbox",
		"documents",
		"starter",
		"create extension",
		"create table users",
		"http",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("migration contains forbidden fragment %q", forbidden)
		}
	}
}

func TestBrainCoreMigrationIsOutsideInfraLedger(t *testing.T) {
	ledger, err := os.ReadFile("../infra/tests/versioned-surface.txt")
	if err != nil {
		t.Fatalf("read infra migration ledger: %v", err)
	}
	if strings.Contains(string(ledger), "0001_brain_core.sql") {
		t.Fatal("Brain Core product migration must not enter the infra migration ledger")
	}
}

func TestBrainCoreMigrationOwnedChecks(t *testing.T) {
	payload, err := os.ReadFile("migrations/0001_brain_core.sql")
	if err != nil {
		t.Fatalf("read Brain Core payload: %v", err)
	}
	payloadHash := sha256.Sum256(payload)
	expectedHash := hex.EncodeToString(payloadHash[:])
	if expectedHash != "1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9" {
		t.Fatalf("canonical Brain Core payload changed: got sha256 %s", expectedHash)
	}

	preflight, err := os.ReadFile("migrations/0001_brain_core.preflight.sql")
	if err != nil {
		t.Fatalf("read Brain Core SQL preflight: %v", err)
	}
	verify, err := os.ReadFile("migrations/0001_brain_core.verify.sql")
	if err != nil {
		t.Fatalf("read Brain Core schema verifier: %v", err)
	}

	preflightSQL := strings.ToLower(string(preflight))
	verifySQL := strings.ToLower(string(verify))

	for _, fragment := range []string{
		"pg_catalog.pg_namespace",
		"pg_catalog.pg_class",
		"pg_catalog.pg_attribute",
		"pg_catalog.pg_constraint",
		"pg_catalog.pg_index",
		"has_database_privilege",
		"absent_eligible",
		"partial_divergent",
		"complete",
	} {
		if !strings.Contains(preflightSQL, fragment) {
			t.Errorf("read-only preflight missing required fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{"\\set", "\\if", "\\gset", "\\quit", "\\ir", "\\pset", "\\echo"} {
		if strings.Contains(preflightSQL, forbidden) {
			t.Errorf("preflight must be plain SQL without psql meta-commands, found %q", forbidden)
		}
	}
	if matched, _ := regexp.MatchString(`(?im)^\s*(create|alter|drop|insert|update|delete|truncate|grant|revoke|commit|rollback)\b|^\s*begin\s+(transaction|work)\b`, preflightSQL); matched {
		t.Fatal("preflight must only inspect catalogs and must not mutate or manage transaction state")
	}
	for _, forbidden := range []string{"from brain_core.brains", "join brain_core.brains", "from brain_core.nodes", "join brain_core.nodes", "migration_ledger", "select owner_subject", "select name"} {
		if strings.Contains(preflightSQL, forbidden) {
			t.Errorf("preflight must not inspect user data or a feature-local ledger, found %q", forbidden)
		}
	}

	for _, fragment := range []string{
		"do $$",
		"raise exception",
		"pg_catalog.pg_attribute",
		"pg_catalog.pg_constraint",
		"pg_catalog.pg_index",
		"pg_catalog.pg_get_constraintdef",
		"pg_catalog.pg_get_indexdef",
		"nodes_parent_fk",
		"nodes_brain_fk",
		"array['brain_id', 'parent_id']::text[]",
		"array['brain_id', 'node_id']::text[]",
		"nodes_shape_check",
		"nodes_version_check",
		"brain_core_one_root_per_brain",
		"brain_core_nodes_parent_idx",
		"column_names = array[",
		"column_types = array[",
		"'owner_subject'",
		"'created_at'",
		"'node_id'",
		"'kind'",
		"'root'",
		"'layer'",
		"parent_id",
		"version",
		"timestamp with time zone",
	} {
		if !strings.Contains(verifySQL, fragment) {
			t.Errorf("schema verifier missing required shape check %q", fragment)
		}
	}
	for _, forbidden := range []string{"\\set", "\\if", "\\gset", "\\quit", "\\ir", "migration_ledger", "from brain_core.brains", "from brain_core.nodes"} {
		if strings.Contains(verifySQL, forbidden) {
			t.Errorf("schema verifier must use catalogs without a psql envelope or product ledger, found %q", forbidden)
		}
	}
	if matched, _ := regexp.MatchString(`(?im)^\s*(create|alter|drop|insert|update|delete|truncate|grant|revoke|commit|rollback)\b|^\s*begin\s+(transaction|work)\b`, verifySQL); matched {
		t.Fatal("schema verifier must inspect shape without owning DDL or transaction control")
	}

	for _, obsolete := range []string{"migrations/0001_brain_core.preflight.psql", "migrations/0001_brain_core.apply.psql"} {
		if _, err := os.Stat(obsolete); err == nil {
			t.Errorf("obsolete standalone psql path still exists: %s", obsolete)
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("inspect obsolete path %s: %v", obsolete, err)
		}
	}
}

func TestPostgresStoreTestHasNoDisposableIntegrationHarness(t *testing.T) {
	sourceBytes, err := os.ReadFile("postgres_store_test.go")
	if err != nil {
		t.Fatalf("read PostgreSQL store tests: %v", err)
	}
	source := strings.ToLower(string(sourceBytes))
	for _, forbidden := range []string{
		"opendisposablepostgres",
		"disposabledsn",
		"drop schema",
		".pool.exec(",
		".pool.queryrow(",
		"pgxpool",
		"net/url",
		"sync.waitgroup",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("postgres_store_test.go still contains disposable integration harness fragment %q", forbidden)
		}
	}
}

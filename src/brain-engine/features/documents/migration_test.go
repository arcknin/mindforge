package documents

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestDocumentsMigrationStaticShape(t *testing.T) {
	sqlBytes, err := os.ReadFile("migrations/0001_documents.sql")
	if err != nil {
		t.Fatalf("read documents migration: %v", err)
	}
	sql := strings.ToLower(string(sqlBytes))
	for _, fragment := range []string{
		"create schema brain_documents",
		"create table brain_documents.documents",
		"document_id uuid primary key",
		"documents_brain_key unique (brain_id, document_id)",
		"foreign key (brain_id)",
		"references brain_core.brains",
		"foreign key (brain_id, layer_id)",
		"references brain_core.nodes",
		"create index documents_owner_idx",
		"create index documents_brain_layer_idx",
		"kind = 'layer'",
		"create table brain_documents.document_links",
		"document_links_source_fk",
		"document_links_target_fk",
		"tombstoned_at",
		"documents_validate_layer",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing required fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"create extension",
		"alter brain_core",
		"drop schema",
		"http",
		"create table users",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("migration contains forbidden fragment %q", forbidden)
		}
	}
}

func TestDocumentsMigrationChecksAreReadOnlyAndPlainSQL(t *testing.T) {
	for _, name := range []string{"0001_documents.preflight.sql", "0001_documents.verify.sql"} {
		contents, err := os.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		sql := strings.ToLower(string(contents))
		for _, fragment := range []string{
			"pg_catalog.pg_namespace",
			"pg_catalog.pg_class",
			"pg_catalog.pg_constraint",
			"pg_catalog.pg_index",
			"pg_catalog.pg_attribute",
			"pg_catalog.pg_trigger",
			"pg_catalog.pg_proc",
			"complete",
			"partial_divergent",
		} {
			if !strings.Contains(sql, fragment) {
				t.Errorf("%s missing required fragment %q", name, fragment)
			}
		}
		for _, forbidden := range []string{"\\set", "\\if", "\\gset", "\\quit", "\\ir", "\\pset", "\\echo"} {
			if strings.Contains(sql, forbidden) {
				t.Errorf("%s must be plain SQL, found psql meta-command %q", name, forbidden)
			}
		}
		if matched, _ := regexp.MatchString(`(?im)^\s*(create|alter|drop|insert|update|delete|truncate|grant|revoke|commit|rollback)\b|^\s*begin\s+(transaction|work)\b`, sql); matched {
			t.Errorf("%s must only inspect catalogs and must not mutate state", name)
		}
	}
}

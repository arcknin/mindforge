package brain

import (
	"os"
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

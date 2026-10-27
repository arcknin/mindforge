package operations

import (
	"strings"
	"testing"
)

// PostgreSQL resolves a direct FROM relation before it evaluates CASE. The
// bootstrap query must defer optional relation lookup to fixed query text.
func TestBootstrapMetadataDoesNotBindAbsentRelations(t *testing.T) {
	var outer strings.Builder
	inLiteral := false
	for i := 0; i < len(metadataQuery); i++ {
		if metadataQuery[i] == '\'' {
			if inLiteral && i+1 < len(metadataQuery) && metadataQuery[i+1] == '\'' {
				i++
				continue
			}
			inLiteral = !inLiteral
			continue
		}
		if !inLiteral {
			outer.WriteByte(metadataQuery[i])
		}
	}
	for _, relation := range []string{"auth.schema_migrations", "infra.migration_ledger"} {
		if strings.Contains(outer.String(), "FROM "+relation) {
			t.Errorf("bootstrap directly binds optional relation %s before CASE", relation)
		}
	}
	if inLiteral {
		t.Fatal("metadata query has an unterminated string literal")
	}
}

func TestBootstrapMetadataPreservesReadinessAndLedgerFields(t *testing.T) {
	if !strings.Contains(metadataQuery, "count(*)") || !strings.Contains(metadataQuery, "string_agg(id") {
		t.Fatal("metadata must retain Auth migration count and full canonical ledger")
	}
	metadata, err := parseMetadata(strings.Join([]string{
		"mindforge_develop", "mindforge,mindforge_develop", "", "", "", "", "", "", "", "", "", "0", "",
	}, "|") + "\n")
	if err != nil || metadata.Database != "mindforge_develop" || metadata.Ledger != "" {
		t.Fatalf("absent infrastructure metadata must parse without inventing readiness: %v", err)
	}
}

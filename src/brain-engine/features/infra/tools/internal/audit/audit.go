// Package audit performs local source inspection only. It has no database
// target, no VM access and no SQL execution path.
package audit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
)

var tablePattern = regexp.MustCompile(`(?im)\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?)`)
var forbiddenSQLObjectPattern = regexp.MustCompile(`(?im)\b(?:CREATE\s+(?:OR\s+REPLACE\s+)?(?:FUNCTION|PROCEDURE|VIEW)|CREATE\s+(?:TRIGGER|POLICY)|GRANT\s+|REVOKE\s+)`)
var productObjectPattern = regexp.MustCompile(`(?im)\b(?:users|brains|neural_layers|documents)\b`)
var sqlRoutingPattern = regexp.MustCompile(`(?im)\b(?:openapi|postgrest|https?://|http\s+(?:endpoint|route|operation))\b`)
var servicePattern = regexp.MustCompile(`^  ([a-z][a-z0-9-]*):\s*$`)

var allowedServices = map[string]struct{}{
	"db":                   {},
	"auth-develop-migrate": {},
	"auth-develop":         {},
	"auth-migrate":         {},
	"auth":                 {},
}

// AuditSource checks a repository source root with only local filesystem
// reads. The caller controls the immutable ledger used for the inspection.
func AuditSource(sourceRoot string, ledger migrations.Ledger, expectedCatalogHash ...string) error {
	infraRoot := filepath.Join(sourceRoot, "src/brain-engine/features/infra")
	catalog, err := migrations.LoadCatalog(sourceRoot)
	if err != nil {
		return fmt.Errorf("audit migration catalog: %w", err)
	}
	if err := migrations.VerifyExactCatalog(catalog, ledger); err != nil {
		return fmt.Errorf("audit immutable migration ledger: %w", err)
	}
	if len(expectedCatalogHash) > 1 {
		return fmt.Errorf("audit accepts only one pinned global catalog hash")
	}
	if len(expectedCatalogHash) == 1 && migrations.CatalogHash(catalog) != expectedCatalogHash[0] {
		return fmt.Errorf("audit feature sources or catalog descriptor differ from the pinned global catalog")
	}
	for _, migration := range catalog.Migrations {
		if migration.Kind == migrations.KindFeature {
			continue
		}
		if err := inspectInfrastructureSQL(migration); err != nil {
			return err
		}
	}
	if err := inspectCompose(filepath.Join(infraRoot, "compose.yaml")); err != nil {
		return err
	}
	if err := inspectOpenAPIRegistry(filepath.Join(sourceRoot, "contracts/openapi/registry.yaml")); err != nil {
		return err
	}
	return nil
}

func inspectInfrastructureSQL(migration migrations.Migration) error {
	contents, err := os.ReadFile(migration.Path)
	if err != nil {
		return fmt.Errorf("audit read infrastructure migration %s: %w", migration.ID, err)
	}
	source := string(contents)
	if sqlRoutingPattern.MatchString(source) {
		return fmt.Errorf("audit rejects SQL that declares an HTTP or OpenAPI operation in migration %s", migration.ID)
	}
	if forbiddenSQLObjectPattern.MatchString(source) {
		return fmt.Errorf("audit rejects function, RPC, policy, view or grant state in infrastructure migration %s", migration.ID)
	}
	for _, match := range tablePattern.FindAllStringSubmatch(source, -1) {
		if !strings.HasPrefix(strings.ToLower(match[1]), "infra.migration_") {
			return fmt.Errorf("audit rejects non-infrastructure table %s in migration %s", match[1], migration.ID)
		}
	}
	if productObjectPattern.MatchString(source) {
		return fmt.Errorf("audit rejects Mindforge product object vocabulary in infrastructure migration %s", migration.ID)
	}
	return nil
}

func inspectCompose(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("audit open Compose source: %w", err)
	}
	defer file.Close()

	services := make(map[string]struct{}, len(allowedServices))
	inServices := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "services:" {
			inServices = true
			continue
		}
		if !inServices {
			continue
		}
		if line != "" && line[0] != ' ' {
			break
		}
		matches := servicePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		name := matches[1]
		if _, ok := allowedServices[name]; !ok {
			return fmt.Errorf("audit rejects extra Compose service %s", name)
		}
		services[name] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("audit read Compose source: %w", err)
	}
	for name := range allowedServices {
		if _, ok := services[name]; !ok {
			return fmt.Errorf("audit requires Compose service %s", name)
		}
	}
	return nil
}

func inspectOpenAPIRegistry(path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("audit read OpenAPI registry: %w", err)
	}
	if !strings.Contains(string(contents), "endpoints: []") {
		return fmt.Errorf("audit rejects unapproved OpenAPI endpoint state")
	}
	return nil
}

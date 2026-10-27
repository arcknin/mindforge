package operations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/evidence"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
)

const bootstrapDevelopIdentityQuery = `SELECT current_database(),
current_user,
pg_get_userbyid(database_row.datdba),
actor_role.rolsuper::text,
actor_role.rolcreatedb::text,
actor_role.rolcanlogin::text,
COALESCE(infra_namespace.nspname, ''),
COALESCE(infra_owner.rolname, ''),
COALESCE(ledger_relation.relname || ':' || ledger_relation.relkind::text, ''),
COALESCE(ledger_owner.rolname, ''),
COALESCE((SELECT string_agg(extname || ':' || extversion, ',' ORDER BY extname) FROM pg_extension WHERE extname IN ('uuid-ossp', 'vector', 'citext')), ''),
COALESCE((SELECT count(*)::text FROM pg_class AS infra_object JOIN pg_namespace AS infra_schema ON infra_schema.oid = infra_object.relnamespace WHERE infra_schema.nspname = 'infra' AND infra_object.relkind IN ('r', 'p', 'v', 'm', 'S', 'f') AND infra_object.relname <> 'migration_ledger'), '0'),
COALESCE(auth_namespace.nspname, ''),
COALESCE(to_regclass('auth.users')::text, ''),
COALESCE(to_regclass('auth.identities')::text, ''),
COALESCE(to_regclass('auth.schema_migrations')::text, ''),
COALESCE((SELECT string_agg(name || ':' || default_version, ',' ORDER BY name) FROM pg_available_extensions WHERE name IN ('uuid-ossp', 'vector', 'citext')), '')
FROM pg_database AS database_row
JOIN pg_roles AS actor_role ON actor_role.rolname = current_user
LEFT JOIN pg_namespace AS infra_namespace ON infra_namespace.nspname = 'infra'
LEFT JOIN pg_roles AS infra_owner ON infra_owner.oid = infra_namespace.nspowner
LEFT JOIN pg_class AS ledger_relation ON ledger_relation.relnamespace = infra_namespace.oid AND ledger_relation.relname = 'migration_ledger'
LEFT JOIN pg_roles AS ledger_owner ON ledger_owner.oid = ledger_relation.relowner
LEFT JOIN pg_namespace AS auth_namespace ON auth_namespace.nspname = 'auth'
WHERE database_row.datname = current_database()`

var bootstrapVector08Version = regexp.MustCompile(`^0\.8\.[0-9]+$`)

// BootstrapDevelopExtensionsRequest describes the one explicit develop-only
// application of infrastructure migration 0001. It never targets main.
type BootstrapDevelopExtensionsRequest struct {
	Config     config.Values
	Operation  command.OperationRequest
	SourceRoot string
	Runner     command.Runner
}

// BootstrapDevelopExtensionsResult contains only the explicit target and
// canonical migration identity. It never includes client output or secrets.
type BootstrapDevelopExtensionsResult struct {
	Target          config.Target
	Status          string
	MigrationID     string
	MigrationSHA256 string
}

// BootstrapDevelopExtensions applies the canonical infrastructure 0001
// migration only to mindforge_develop. It creates the initial infra ledger as
// the non-superuser develop owner, then applies extensions and the matching
// ledger record atomically through the explicit cluster-admin connection.
func BootstrapDevelopExtensions(ctx context.Context, request BootstrapDevelopExtensionsRequest) (BootstrapDevelopExtensionsResult, error) {
	result := BootstrapDevelopExtensionsResult{Target: config.TargetDevelop, Status: "failed"}
	if request.Runner == nil {
		return result, fmt.Errorf("a command runner is required")
	}
	if request.Operation.Target != config.TargetDevelop {
		return result, fmt.Errorf("extension bootstrap requires explicit target mindforge_develop")
	}
	if _, err := command.NewOperationRequest(request.Operation.Target, request.Operation.OperatorID, request.Operation.AuthorizationRef); err != nil {
		return result, err
	}
	if strings.TrimSpace(request.SourceRoot) == "" {
		return result, fmt.Errorf("an explicit repository root is required")
	}

	developSettings, err := request.Config.Settings(config.TargetDevelop)
	if err != nil {
		return result, fmt.Errorf("develop target configuration is invalid")
	}
	clusterAdmin, err := request.Config.ClusterAdmin()
	if err != nil {
		return result, fmt.Errorf("cluster-admin configuration is invalid")
	}
	if clusterAdmin.DatabaseHost != developSettings.DatabaseHost || clusterAdmin.DatabasePort != developSettings.DatabasePort {
		return result, fmt.Errorf("develop and cluster-admin client endpoints must match")
	}
	adminSettings := bootstrapAdminTargetSettings(clusterAdmin)

	repositoryRoot, err := resolveRepositoryRoot(request.SourceRoot)
	if err != nil {
		return result, fmt.Errorf("explicit repository root is unavailable")
	}
	catalog, err := migrations.LoadCatalog(repositoryRoot)
	if err != nil {
		return result, fmt.Errorf("canonical migration catalog is unavailable")
	}
	migration, err := canonicalDevelopBootstrapMigration(catalog)
	if err != nil {
		return result, err
	}

	preflight, err := bootstrapDevelopPreflight(ctx, request, repositoryRoot)
	if err != nil {
		return result, err
	}
	if preflight.Target != config.TargetDevelop || preflight.SourceRoot != catalog.Root || preflight.CatalogHash != migrations.CatalogHash(catalog) {
		return result, fmt.Errorf("develop bootstrap preflight did not match the canonical target and catalog")
	}
	developState, err := readBootstrapDevelopIdentity(ctx, request.Runner, developSettings)
	if err != nil {
		return result, err
	}
	if err := validateBootstrapDevelopIdentity(developState, developSettings.DatabaseUser); err != nil {
		return result, err
	}
	noOp, err := validateBootstrapState(preflight, developState, migration, developSettings.DatabaseUser)
	if err != nil {
		return result, err
	}
	if noOp {
		return bootstrapResult(result, migration, "noop"), nil
	}
	adminState, err := readBootstrapDevelopIdentity(ctx, request.Runner, adminSettings)
	if err != nil {
		return result, err
	}
	if err := validateBootstrapAdminIdentity(adminState, adminSettings.DatabaseUser, developSettings.DatabaseUser); err != nil {
		return result, err
	}

	// The only standalone DDL is an empty, owner-matched ledger bootstrap as
	// the develop database owner. Extension SQL is not applied in this step.
	if _, err := runPSQLTransaction(ctx, request.Runner, developSettings, ledgerBootstrapSQL); err != nil {
		return result, fmt.Errorf("develop-owned empty ledger bootstrap failed")
	}

	// Recheck after the owner bootstrap so a concurrent migration or a changed
	// owner/target stops before the cluster-admin transaction.
	preflight, err = bootstrapDevelopPreflight(ctx, request, repositoryRoot)
	if err != nil {
		return result, err
	}
	developState, err = readBootstrapDevelopIdentity(ctx, request.Runner, developSettings)
	if err != nil {
		return result, err
	}
	if err := validateBootstrapDevelopIdentity(developState, developSettings.DatabaseUser); err != nil {
		return result, err
	}
	noOp, err = validateBootstrapState(preflight, developState, migration, developSettings.DatabaseUser)
	if err != nil {
		return result, err
	}
	if noOp {
		return bootstrapResult(result, migration, "noop"), nil
	}
	adminState, err = readBootstrapDevelopIdentity(ctx, request.Runner, adminSettings)
	if err != nil {
		return result, err
	}
	if err := validateBootstrapAdminIdentity(adminState, adminSettings.DatabaseUser, developSettings.DatabaseUser); err != nil {
		return result, err
	}

	transactionPath, cleanup, err := prepareDevelopBootstrapTransaction(repositoryRoot, migration)
	if err != nil {
		return result, fmt.Errorf("canonical migration snapshot could not be prepared")
	}
	defer cleanup()
	invocation := bootstrapMigrationFileInvocation(adminSettings, transactionPath)
	commandResult, err := request.Runner.Run(ctx, invocation)
	if err != nil || commandResult.ExitCode != 0 {
		return result, fmt.Errorf("atomic develop extension transaction failed")
	}
	return bootstrapResult(result, migration, "succeeded"), nil
}

func bootstrapDevelopPreflight(ctx context.Context, request BootstrapDevelopExtensionsRequest, repositoryRoot string) (preflight evidence.Preflight, err error) {
	return Preflight(ctx, PreflightRequest{
		Config:                       request.Config,
		Operation:                    request.Operation,
		Scope:                        ScopeInfrastructure,
		SourceRoot:                   repositoryRoot,
		Runner:                       request.Runner,
		AllowInfrastructureBootstrap: true,
	})
}

func bootstrapAdminTargetSettings(admin config.ClusterAdminSettings) config.TargetSettings {
	return config.TargetSettings{
		Target:           config.TargetDevelop,
		DatabaseHost:     admin.DatabaseHost,
		DatabasePort:     admin.DatabasePort,
		DatabaseName:     string(config.TargetDevelop),
		DatabaseUser:     admin.DatabaseUser,
		DatabasePassword: admin.DatabasePassword,
	}
}

func canonicalDevelopBootstrapMigration(catalog migrations.Catalog) (migrations.Migration, error) {
	expectedHash := migrations.GlobalMigrationLedgerBaseline()["0001"]
	for _, migration := range catalog.Migrations {
		if migration.ID != "0001" {
			continue
		}
		expectedPath := filepath.Join(catalog.Root, "src", "brain-engine", "features", "infra", "postgres", "migrations", "0001_infra_init.sql")
		if migration.Kind != migrations.KindInfrastructure || migration.Path != expectedPath || migration.SHA256 != expectedHash || expectedHash == "" {
			return migrations.Migration{}, fmt.Errorf("canonical infrastructure migration 0001 changed")
		}
		return migration, nil
	}
	return migrations.Migration{}, fmt.Errorf("canonical infrastructure migration 0001 is absent")
}

func readBootstrapDevelopIdentity(ctx context.Context, runner command.Runner, settings config.TargetSettings) (bootstrapDevelopIdentity, error) {
	result, err := runPSQLTransaction(ctx, runner, settings, "SET TRANSACTION READ ONLY;\n"+bootstrapDevelopIdentityQuery)
	if err != nil {
		return bootstrapDevelopIdentity{}, fmt.Errorf("read-only develop identity preflight failed")
	}
	identity, err := parseBootstrapDevelopIdentity(result.Stdout)
	if err != nil {
		return bootstrapDevelopIdentity{}, fmt.Errorf("read-only develop identity preflight returned an invalid catalog row")
	}
	return identity, nil
}

type bootstrapDevelopIdentity struct {
	Database             string
	CurrentUser          string
	DatabaseOwner        string
	RoleSuperuser        bool
	RoleCreateDatabase   bool
	RoleCanLogin         bool
	InfraNamespace       string
	InfraNamespaceOwner  string
	LedgerRelation       string
	LedgerRelationKind   string
	LedgerRelationOwner  string
	InstalledExtensions  map[string]string
	AvailableExtensions  map[string]string
	OtherInfraRelations  int
	AuthNamespace        string
	AuthUsers            string
	AuthIdentities       string
	AuthSchemaMigrations string
}

func parseBootstrapDevelopIdentity(output string) (bootstrapDevelopIdentity, error) {
	var line string
	for _, candidate := range strings.Split(output, "\n") {
		candidate = strings.TrimSpace(candidate)
		if strings.Count(candidate, "|") == 16 {
			line = candidate
			break
		}
	}
	if line == "" {
		return bootstrapDevelopIdentity{}, fmt.Errorf("missing identity row")
	}
	fields := strings.Split(line, "|")
	roleSuperuser, err := parseBootstrapBool(fields[3])
	if err != nil {
		return bootstrapDevelopIdentity{}, err
	}
	roleCreateDatabase, err := parseBootstrapBool(fields[4])
	if err != nil {
		return bootstrapDevelopIdentity{}, err
	}
	roleCanLogin, err := parseBootstrapBool(fields[5])
	if err != nil {
		return bootstrapDevelopIdentity{}, err
	}
	otherInfraRelations, err := strconv.Atoi(fields[11])
	if err != nil || otherInfraRelations < 0 {
		return bootstrapDevelopIdentity{}, fmt.Errorf("invalid infra relation count")
	}
	installed, err := parseBootstrapExtensionVersions(fields[10])
	if err != nil {
		return bootstrapDevelopIdentity{}, err
	}
	available, err := parseBootstrapExtensionVersions(fields[16])
	if err != nil {
		return bootstrapDevelopIdentity{}, err
	}
	ledgerRelation, ledgerRelationKind, found := strings.Cut(fields[8], ":")
	if fields[8] != "" && (!found || ledgerRelation == "" || ledgerRelationKind == "") {
		return bootstrapDevelopIdentity{}, fmt.Errorf("invalid ledger relation metadata")
	}
	return bootstrapDevelopIdentity{
		Database:             fields[0],
		CurrentUser:          fields[1],
		DatabaseOwner:        fields[2],
		RoleSuperuser:        roleSuperuser,
		RoleCreateDatabase:   roleCreateDatabase,
		RoleCanLogin:         roleCanLogin,
		InfraNamespace:       fields[6],
		InfraNamespaceOwner:  fields[7],
		LedgerRelation:       ledgerRelation,
		LedgerRelationKind:   ledgerRelationKind,
		LedgerRelationOwner:  fields[9],
		InstalledExtensions:  installed,
		OtherInfraRelations:  otherInfraRelations,
		AuthNamespace:        fields[12],
		AuthUsers:            fields[13],
		AuthIdentities:       fields[14],
		AuthSchemaMigrations: fields[15],
		AvailableExtensions:  available,
	}, nil
}

func parseBootstrapBool(value string) (bool, error) {
	switch strings.TrimSpace(value) {
	case "true", "t", "1":
		return true, nil
	case "false", "f", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid role attribute")
	}
}

func parseBootstrapExtensionVersions(raw string) (map[string]string, error) {
	versions := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return versions, nil
	}
	for _, entry := range strings.Split(raw, ",") {
		name, version, ok := strings.Cut(entry, ":")
		if !ok || version == "" || (name != "uuid-ossp" && name != "vector" && name != "citext") {
			return nil, fmt.Errorf("invalid extension catalog record")
		}
		if _, exists := versions[name]; exists {
			return nil, fmt.Errorf("duplicate extension catalog record")
		}
		versions[name] = version
	}
	return versions, nil
}

func validateBootstrapDevelopIdentity(identity bootstrapDevelopIdentity, developUser string) error {
	if identity.Database != string(config.TargetDevelop) || identity.CurrentUser != developUser || identity.DatabaseOwner != developUser {
		return fmt.Errorf("develop database, login and owner must match the explicit develop target")
	}
	if identity.RoleSuperuser || identity.RoleCreateDatabase || !identity.RoleCanLogin {
		return fmt.Errorf("develop role must be a login without superuser or database-creation privilege")
	}
	return nil
}

func validateBootstrapAdminIdentity(identity bootstrapDevelopIdentity, adminUser, developUser string) error {
	if identity.Database != string(config.TargetDevelop) || identity.CurrentUser != adminUser || identity.DatabaseOwner != developUser {
		return fmt.Errorf("cluster-admin connection must target the explicit develop database and owner")
	}
	if !identity.RoleSuperuser || !identity.RoleCanLogin {
		return fmt.Errorf("cluster-admin extension actor must be a login superuser")
	}
	return nil
}

func validateBootstrapState(preflight evidence.Preflight, identity bootstrapDevelopIdentity, migration migrations.Migration, developUser string) (bool, error) {
	if preflight.Target != config.TargetDevelop {
		return false, fmt.Errorf("bootstrap preflight target does not match develop")
	}
	ledgerHash, applied := preflight.Ledger[migration.ID]
	if applied {
		if ledgerHash != migration.SHA256 || identity.InfraNamespace != "infra" || identity.LedgerRelation != "migration_ledger" || identity.LedgerRelationKind != "r" {
			return false, fmt.Errorf("existing infrastructure migration 0001 does not match the canonical ledger")
		}
		if !bootstrapRequiredExtensionsInstalled(identity.InstalledExtensions) {
			return false, fmt.Errorf("canonical migration 0001 exists with incomplete or incompatible extensions")
		}
		return true, nil
	}
	if len(preflight.Ledger) != 0 {
		return false, fmt.Errorf("develop ledger has rows but no canonical bootstrap migration 0001")
	}
	if len(identity.InstalledExtensions) != 0 {
		return false, fmt.Errorf("develop has unledgered infrastructure extensions")
	}
	if identity.AuthNamespace != "" || identity.AuthUsers != "" || identity.AuthIdentities != "" || identity.AuthSchemaMigrations != "" {
		return false, fmt.Errorf("develop Auth state exists before infrastructure migration 0001")
	}
	if identity.OtherInfraRelations != 0 {
		return false, fmt.Errorf("develop infra schema has unrecognized relations")
	}
	if identity.InfraNamespace == "" {
		if identity.InfraNamespaceOwner != "" || identity.LedgerRelation != "" || identity.LedgerRelationOwner != "" {
			return false, fmt.Errorf("develop infra ownership metadata is inconsistent")
		}
	} else if identity.InfraNamespace != "infra" || identity.InfraNamespaceOwner != developUser {
		return false, fmt.Errorf("empty develop infra state must be owned by the configured develop role")
	}
	if identity.LedgerRelation != "" && (identity.LedgerRelation != "migration_ledger" || identity.LedgerRelationKind != "r" || identity.LedgerRelationOwner != developUser) {
		return false, fmt.Errorf("empty develop ledger must be owned by the configured develop role")
	}
	if !bootstrapRequiredExtensionsAvailable(identity.AvailableExtensions) {
		return false, fmt.Errorf("required PostgreSQL extensions are unavailable or vector is outside the supported 0.8.x range")
	}
	return false, nil
}

func bootstrapRequiredExtensionsInstalled(versions map[string]string) bool {
	if versions["uuid-ossp"] == "" || versions["citext"] == "" {
		return false
	}
	return bootstrapVector08Version.MatchString(versions["vector"])
}

func bootstrapRequiredExtensionsAvailable(versions map[string]string) bool {
	if versions["uuid-ossp"] == "" || versions["citext"] == "" {
		return false
	}
	return bootstrapVector08Version.MatchString(versions["vector"])
}

func prepareDevelopBootstrapTransaction(repositoryRoot string, migration migrations.Migration) (string, func(), error) {
	if migration.ID != "0001" || migration.SHA256 != migrations.GlobalMigrationLedgerBaseline()["0001"] || migration.Kind != migrations.KindInfrastructure {
		return "", nil, fmt.Errorf("migration identity is not the canonical infrastructure bootstrap")
	}
	source, err := readHashedSource(migration.Path, migration.SHA256)
	if err != nil {
		return "", nil, fmt.Errorf("canonical infrastructure source changed")
	}
	transaction := append([]byte(nil), source...)
	if len(transaction) > 0 && transaction[len(transaction)-1] != '\n' {
		transaction = append(transaction, '\n')
	}
	transaction = append(transaction, []byte(fmt.Sprintf("\nINSERT INTO infra.migration_ledger (id, sha256) VALUES ('%s', '%s');\n", migration.ID, migration.SHA256))...)
	file, err := os.CreateTemp("", "mindforge-develop-infra-bootstrap-*.sql")
	if err != nil {
		return "", nil, fmt.Errorf("create private migration snapshot")
	}
	path := file.Name()
	cleanup := func() { _ = os.Remove(path) }
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("protect private migration snapshot")
	}
	if _, err := file.Write(transaction); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("write private migration snapshot")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("flush private migration snapshot")
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close private migration snapshot")
	}
	canonicalPath, err := externalFile(repositoryRoot, path)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("private migration snapshot must be outside the checkout")
	}
	return canonicalPath, cleanup, nil
}

func bootstrapMigrationFileInvocation(settings config.TargetSettings, path string) command.Invocation {
	return command.Invocation{
		Name: "psql",
		Args: []string{
			"--no-psqlrc",
			"--set=ON_ERROR_STOP=1",
			"--single-transaction",
			"--host=" + settings.DatabaseHost,
			"--port=" + settings.DatabasePort,
			"--username=" + settings.DatabaseUser,
			"--dbname=" + settings.DatabaseName,
			"--file=" + path,
		},
		Env: []string{"PGPASSWORD=" + settings.DatabasePassword},
	}
}

func bootstrapResult(result BootstrapDevelopExtensionsResult, migration migrations.Migration, status string) BootstrapDevelopExtensionsResult {
	result.Status = status
	result.MigrationID = migration.ID
	result.MigrationSHA256 = migration.SHA256
	return result
}

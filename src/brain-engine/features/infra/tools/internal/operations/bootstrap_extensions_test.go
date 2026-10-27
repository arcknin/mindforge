package operations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
)

func TestBootstrapDevelopExtensionsAppliesCanonicalMigrationAsAdminToDevelop(t *testing.T) {
	t.Parallel()

	privateConfig := writeBootstrapPrivateConfig(t, true)
	settings, err := config.ParseFile(privateConfig)
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-bootstrap-2026-10-08")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	root := bootstrapRepositoryRoot(t)
	canonicalHash := migrations.GlobalMigrationLedgerBaseline()["0001"]
	runner := newBootstrapExtensionsRunner()
	runner.metadataRows = []string{
		bootstrapMetadataRow("", "", "", "", "", "", "", "", "0", ""),
		bootstrapMetadataRow("infra", "infra.migration_ledger", "", "", "", "", "", "", "0", ""),
	}
	runner.identityRows["mf_develop"] = []string{
		bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "", "", "", "", "", 0, "", "", "", ""),
		bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "", 0, "", "", "", ""),
	}
	runner.identityRows["mf_main"] = []string{
		bootstrapIdentityRow("mindforge_develop", "mf_main", "mf_develop", true, true, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "", 0, "", "", "", ""),
		bootstrapIdentityRow("mindforge_develop", "mf_main", "mf_develop", true, true, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "", 0, "", "", "", ""),
	}

	result, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: runner,
	})
	if err != nil {
		t.Fatalf("BootstrapDevelopExtensions(): %v", err)
	}
	if result.Target != config.TargetDevelop || result.Status != "succeeded" || result.MigrationID != "0001" || result.MigrationSHA256 != canonicalHash {
		t.Fatalf("BootstrapDevelopExtensions() = %#v, want sanitized canonical develop success", result)
	}
	if runner.ledgerBootstrapCount != 1 || runner.transactionCount != 1 {
		t.Fatalf("ledger bootstrap calls=%d migration transaction calls=%d, want one each", runner.ledgerBootstrapCount, runner.transactionCount)
	}
	ledgerBootstrap, ok := invocationWithCommandContains(runner.calls, "CREATE TABLE IF NOT EXISTS infra.migration_ledger")
	if !ok {
		t.Fatal("owner-ledger bootstrap invocation was not recorded")
	}
	assertConnectionTarget(t, ledgerBootstrap, "mf_develop", "mindforge_develop")
	if !containsEnv(ledgerBootstrap, "PGPASSWORD=synthetic-develop-password-123456") {
		t.Fatal("empty ledger bootstrap did not use the configured develop owner")
	}
	if runner.transactionPath == "" || runner.transactionSQL == "" || runner.transactionMode.Perm() != 0o600 {
		t.Fatalf("migration snapshot was not private and captured: path=%q mode=%#o", runner.transactionPath, runner.transactionMode.Perm())
	}
	if _, err := os.Stat(runner.transactionPath); !os.IsNotExist(err) {
		t.Fatalf("private migration snapshot still exists after operation: err=%v", err)
	}
	if !strings.Contains(runner.transactionSQL, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`) ||
		!strings.Contains(runner.transactionSQL, `CREATE EXTENSION IF NOT EXISTS "vector";`) ||
		!strings.Contains(runner.transactionSQL, `CREATE EXTENSION IF NOT EXISTS "citext";`) ||
		!strings.Contains(runner.transactionSQL, "INSERT INTO infra.migration_ledger (id, sha256) VALUES ('0001', '"+canonicalHash+"');") {
		t.Fatalf("transaction snapshot omitted canonical extensions or the matching global ledger insert")
	}
	for _, forbidden := range []string{"CREATE ROLE", "ALTER ROLE", "GRANT ", "SUPERUSER", "brain_core", "CREATE TABLE users"} {
		if strings.Contains(strings.ToUpper(runner.transactionSQL), strings.ToUpper(forbidden)) {
			t.Errorf("transaction snapshot contains forbidden operation %q", forbidden)
		}
	}
	if !strings.Contains(strings.ToLower(strings.Join(runner.transactionArgs, " ")), "--single-transaction") {
		t.Fatalf("migration snapshot was not applied in one transaction: %v", runner.transactionArgs)
	}
	assertBootstrapConnectionsOnlyDevelop(t, runner.calls)
	adminCall, ok := invocationForUser(runner.calls, "mf_main", "--file=")
	if !ok {
		t.Fatalf("aliased cluster admin did not apply the transaction: %#v", runner.calls)
	}
	assertConnectionTarget(t, adminCall, "mf_main", "mindforge_develop")
	if !containsEnv(adminCall, "PGPASSWORD=synthetic-main-password-123456") {
		t.Fatalf("cluster-admin alias did not use its validated matching credential")
	}
	if len(runner.metadataRows) != 0 || len(runner.identityRows["mf_develop"]) != 0 {
		t.Fatalf("expected both read-only preflights and develop-owner checks to be consumed")
	}
}

func TestBootstrapDevelopExtensionsIsNoopOnlyWithCanonicalLedgerAndExtensions(t *testing.T) {
	t.Parallel()

	settings, request, root := bootstrapRequest(t, config.TargetDevelop)
	hash := migrations.GlobalMigrationLedgerBaseline()["0001"]
	runner := newBootstrapExtensionsRunner()
	runner.metadataRows = []string{bootstrapMetadataRowWithVector(
		"infra", "infra.migration_ledger", "", "", "", "", "", "", "0",
		"0001:"+hash, "0.8.6",
	)}
	runner.identityRows["mf_develop"] = []string{
		bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "citext:1.6,uuid-ossp:1.1,vector:0.8.6", 0, "", "", "", ""),
	}

	result, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: runner,
	})
	if err != nil {
		t.Fatalf("BootstrapDevelopExtensions(): %v", err)
	}
	if result.Status != "noop" || result.Target != config.TargetDevelop || result.MigrationID != "0001" || result.MigrationSHA256 != hash {
		t.Fatalf("BootstrapDevelopExtensions() = %#v, want matching existing migration no-op", result)
	}
	if runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
		t.Fatalf("matching bootstrap performed writes: ledger=%d migration=%d", runner.ledgerBootstrapCount, runner.transactionCount)
	}
	assertBootstrapConnectionsOnlyDevelop(t, runner.calls)
}

func TestBootstrapDevelopExtensionsRejectsIncompatibleVectorOnExistingMigration(t *testing.T) {
	t.Parallel()

	settings, request, root := bootstrapRequest(t, config.TargetDevelop)
	hash := migrations.GlobalMigrationLedgerBaseline()["0001"]
	runner := newBootstrapExtensionsRunner()
	runner.metadataRows = []string{bootstrapMetadataRowWithVector(
		"infra", "infra.migration_ledger", "", "", "", "", "", "", "0",
		"0001:"+hash, "0.9.0",
	)}
	runner.identityRows["mf_develop"] = []string{
		bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "citext:1.6,uuid-ossp:1.1,vector:0.9.0", 0, "", "", "", ""),
	}
	_, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: runner,
	})
	if err == nil {
		t.Fatal("BootstrapDevelopExtensions() error = nil for an existing migration with incompatible vector version")
	}
	if runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
		t.Fatalf("incompatible vector state reached DDL: ledger=%d transaction=%d", runner.ledgerBootstrapCount, runner.transactionCount)
	}
}

func TestBootstrapDevelopExtensionsRejectsMainTargetBeforeAnyClientCall(t *testing.T) {
	t.Parallel()

	settings, request, root := bootstrapRequest(t, config.TargetMain)
	runner := newBootstrapExtensionsRunner()
	_, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: runner,
	})
	if err == nil {
		t.Fatal("BootstrapDevelopExtensions() error = nil for main target")
	}
	if len(runner.calls) != 0 || runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
		t.Fatalf("main target reached a client or mutation: calls=%#v", runner.calls)
	}
}

func TestBootstrapDevelopExtensionsRejectsChangedCanonicalSourceBeforeClient(t *testing.T) {
	t.Parallel()

	settings, request, _ := bootstrapRequest(t, config.TargetDevelop)
	fakeRoot := writeBootstrapCatalog(t)
	runner := newBootstrapExtensionsRunner()
	_, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: fakeRoot, Runner: runner,
	})
	if err == nil {
		t.Fatal("BootstrapDevelopExtensions() error = nil for noncanonical 0001 source hash")
	}
	if len(runner.calls) != 0 || runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
		t.Fatalf("changed canonical source reached a client or mutation: calls=%#v", runner.calls)
	}
}

func TestBootstrapDevelopExtensionsRejectsChangedLedgerHashBeforeAnyMutation(t *testing.T) {
	t.Parallel()

	settings, request, root := bootstrapRequest(t, config.TargetDevelop)
	runner := newBootstrapExtensionsRunner()
	runner.metadataRows = []string{bootstrapMetadataRow(
		"infra", "infra.migration_ledger", "", "", "", "", "", "", "0",
		"0001:"+strings.Repeat("a", 64),
	)}
	_, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: runner,
	})
	if err == nil {
		t.Fatal("BootstrapDevelopExtensions() error = nil for changed applied hash")
	}
	if runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
		t.Fatalf("changed ledger hash reached a mutation: ledger=%d transaction=%d", runner.ledgerBootstrapCount, runner.transactionCount)
	}
}

func TestBootstrapDevelopExtensionsRejectsWrongOwnerAndSuperuserDevelopRoleBeforeDDL(t *testing.T) {
	t.Parallel()

	for _, state := range []struct {
		name string
		row  string
	}{
		{name: "database owner mismatch", row: bootstrapIdentityRow("mindforge_develop", "mf_develop", "unexpected_owner", false, false, true, "", "", "", "", "", 0, "", "", "", "")},
		{name: "develop superuser", row: bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", true, false, true, "", "", "", "", "", 0, "", "", "", "")},
		{name: "develop createdb privilege", row: bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, true, true, "", "", "", "", "", 0, "", "", "", "")},
		{name: "unledgered partial extension", row: bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "", "", "", "", "vector:0.8.6", 0, "", "", "", "")},
		{name: "unledgered infra object", row: bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "infra", "mf_develop", "", "", "", 1, "", "", "", "")},
		{name: "Auth migrated before infrastructure 0001", row: bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "", "", "", "", "", 0, "auth", "", "", "auth.schema_migrations")},
	} {
		t.Run(state.name, func(t *testing.T) {
			t.Parallel()
			settings, request, root := bootstrapRequest(t, config.TargetDevelop)
			runner := newBootstrapExtensionsRunner()
			runner.metadataRows = []string{bootstrapMetadataRow("", "", "", "", "", "", "", "", "0", "")}
			runner.identityRows["mf_develop"] = []string{state.row}
			_, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
				Config: settings, Operation: request, SourceRoot: root, Runner: runner,
			})
			if err == nil {
				t.Fatal("BootstrapDevelopExtensions() error = nil for unsafe identity state")
			}
			if runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
				t.Fatalf("unsafe identity reached DDL: ledger=%d transaction=%d", runner.ledgerBootstrapCount, runner.transactionCount)
			}
		})
	}
}

func TestBootstrapDevelopExtensionsRejectsInvalidAdminBeforeDDL(t *testing.T) {
	t.Parallel()

	for _, state := range []struct {
		name string
		row  string
	}{
		{name: "non-superuser", row: bootstrapIdentityRow("mindforge_develop", "mf_cluster_admin", "mf_develop", false, true, true, "", "", "", "", "", 0, "", "", "", "")},
		{name: "admin connected outside develop", row: bootstrapIdentityRow("mindforge", "mf_cluster_admin", "mindforge", true, true, true, "", "", "", "", "", 0, "", "", "", "")},
	} {
		t.Run(state.name, func(t *testing.T) {
			t.Parallel()
			settings, request, root := bootstrapRequest(t, config.TargetDevelop)
			runner := newBootstrapExtensionsRunner()
			runner.metadataRows = []string{bootstrapMetadataRow("", "", "", "", "", "", "", "", "0", "")}
			runner.identityRows["mf_develop"] = []string{bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "", "", "", "", "", 0, "", "", "", "")}
			runner.identityRows["mf_cluster_admin"] = []string{state.row}
			_, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
				Config: settings, Operation: request, SourceRoot: root, Runner: runner,
			})
			if err == nil {
				t.Fatal("BootstrapDevelopExtensions() error = nil for invalid cluster-admin identity")
			}
			if runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
				t.Fatalf("invalid admin reached DDL: ledger=%d transaction=%d", runner.ledgerBootstrapCount, runner.transactionCount)
			}
		})
	}
}

func TestBootstrapDevelopExtensionsFailsAdminTransactionAtomicallyAndCleansSnapshot(t *testing.T) {
	t.Parallel()

	settings, request, root := bootstrapRequest(t, config.TargetDevelop)
	runner := newBootstrapExtensionsRunner()
	runner.metadataRows = []string{
		bootstrapMetadataRow("", "", "", "", "", "", "", "", "0", ""),
		bootstrapMetadataRow("infra", "infra.migration_ledger", "", "", "", "", "", "", "0", ""),
	}
	runner.identityRows["mf_develop"] = []string{
		bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "", "", "", "", "", 0, "", "", "", ""),
		bootstrapIdentityRow("mindforge_develop", "mf_develop", "mf_develop", false, false, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "", 0, "", "", "", ""),
	}
	runner.identityRows["mf_cluster_admin"] = []string{
		bootstrapIdentityRow("mindforge_develop", "mf_cluster_admin", "mf_develop", true, true, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "", 0, "", "", "", ""),
		bootstrapIdentityRow("mindforge_develop", "mf_cluster_admin", "mf_develop", true, true, true, "infra", "mf_develop", "migration_ledger", "mf_develop", "", 0, "", "", "", ""),
	}
	runner.failMigrationTransaction = true

	result, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: runner,
	})
	if err == nil || result.Status == "succeeded" {
		t.Fatalf("failed admin migration transaction returned success: result=%#v err=%v", result, err)
	}
	if runner.ledgerBootstrapCount != 1 || runner.transactionCount != 1 {
		t.Fatalf("ledger bootstrap calls=%d migration transaction calls=%d, want owner-matched empty ledger bootstrap and one atomic extension transaction", runner.ledgerBootstrapCount, runner.transactionCount)
	}
	if _, err := os.Stat(runner.transactionPath); !os.IsNotExist(err) {
		t.Fatalf("private migration snapshot still exists after failed transaction: err=%v", err)
	}
	if !strings.Contains(strings.Join(runner.transactionArgs, " "), "--single-transaction") || !containsArgPrefix(runner.transactionArgs, "--file=") {
		t.Fatalf("failed extension migration was not one file-backed transaction: %v", runner.transactionArgs)
	}
	for _, call := range runner.calls {
		if strings.Contains(strings.Join(call.Args, " "), "INSERT INTO infra.migration_ledger") {
			t.Fatalf("ledger insert ran outside the atomic migration file: %#v", call.Args)
		}
	}
}

func TestBootstrapDevelopExtensionsRejectsPSQLMajorMismatchBeforeDatabaseQueries(t *testing.T) {
	t.Parallel()

	settings, request, root := bootstrapRequest(t, config.TargetDevelop)
	runner := newBootstrapExtensionsRunner()
	runner.psqlVersion = "psql (PostgreSQL) 18.4\n"
	_, err := BootstrapDevelopExtensions(context.Background(), BootstrapDevelopExtensionsRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: runner,
	})
	if err == nil {
		t.Fatal("BootstrapDevelopExtensions() error = nil for mismatched psql major")
	}
	if len(runner.calls) != 1 || runner.ledgerBootstrapCount != 0 || runner.transactionCount != 0 {
		t.Fatalf("psql major mismatch reached a database query or mutation: %#v", runner.calls)
	}
}

type bootstrapExtensionsRunner struct {
	psqlVersion              string
	metadataRows             []string
	identityRows             map[string][]string
	calls                    []command.Invocation
	ledgerBootstrapCount     int
	transactionCount         int
	transactionPath          string
	transactionSQL           string
	transactionMode          os.FileMode
	transactionArgs          []string
	failMigrationTransaction bool
}

func newBootstrapExtensionsRunner() *bootstrapExtensionsRunner {
	return &bootstrapExtensionsRunner{
		psqlVersion:  "psql (PostgreSQL) 16.15\n",
		identityRows: make(map[string][]string),
	}
}

func (runner *bootstrapExtensionsRunner) Run(_ context.Context, invocation command.Invocation) (command.Result, error) {
	invocation.Args = append([]string(nil), invocation.Args...)
	invocation.Env = append([]string(nil), invocation.Env...)
	runner.calls = append(runner.calls, invocation)
	if invocation.Shell {
		return command.Result{}, fmt.Errorf("fake runner refuses shell invocation")
	}
	if containsArgPrefix(invocation.Args, "--version") {
		return command.Result{Stdout: runner.psqlVersion}, nil
	}
	query, hasQuery := invocationArg(invocation, "--command=")
	if hasQuery && query == metadataQuery {
		if len(runner.metadataRows) == 0 {
			return command.Result{}, fmt.Errorf("missing metadata fixture")
		}
		row := runner.metadataRows[0]
		runner.metadataRows = runner.metadataRows[1:]
		return command.Result{Stdout: row}, nil
	}
	if hasQuery && strings.Contains(query, "pg_get_userbyid") {
		username, _ := invocationArg(invocation, "--username=")
		rows := runner.identityRows[username]
		if len(rows) == 0 {
			return command.Result{}, fmt.Errorf("missing identity fixture for configured role")
		}
		row := rows[0]
		runner.identityRows[username] = rows[1:]
		return command.Result{Stdout: row + "\n"}, nil
	}
	if hasQuery && strings.Contains(query, "CREATE TABLE IF NOT EXISTS infra.migration_ledger") {
		runner.ledgerBootstrapCount++
		return command.Result{}, nil
	}
	if containsArgPrefix(invocation.Args, "--file=") {
		runner.transactionCount++
		runner.transactionArgs = append([]string(nil), invocation.Args...)
		path, _ := invocationArg(invocation, "--file=")
		runner.transactionPath = path
		contents, err := os.ReadFile(path)
		if err != nil {
			return command.Result{}, err
		}
		runner.transactionSQL = string(contents)
		info, err := os.Stat(path)
		if err != nil {
			return command.Result{}, err
		}
		runner.transactionMode = info.Mode()
		if runner.failMigrationTransaction {
			return command.Result{ExitCode: 1, Stderr: "synthetic private failure"}, fmt.Errorf("fake psql transaction failed")
		}
		return command.Result{}, nil
	}
	return command.Result{}, fmt.Errorf("unexpected fake psql invocation")
}

func bootstrapRequest(t *testing.T, target config.Target) (config.Values, command.OperationRequest, string) {
	t.Helper()
	values, err := config.ParseFile(writeBootstrapPrivateConfig(t, false))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(target, "operator-1", "owner-bootstrap-2026-10-08")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	return values, request, bootstrapRepositoryRoot(t)
}

func writeBootstrapPrivateConfig(t *testing.T, adminAliasesMain bool) string {
	t.Helper()
	adminUser := "mf_cluster_admin"
	adminPassword := "synthetic-cluster-password-123456"
	if adminAliasesMain {
		adminUser = "mf_main"
		adminPassword = "synthetic-main-password-123456"
	}
	contents := "POSTGRES_ADMIN_USER=" + adminUser + "\n" +
		"POSTGRES_ADMIN_PASSWORD=" + adminPassword + "\n" +
		"POSTGRES_HOST=db\nPOSTGRES_CLIENT_HOST=127.0.0.1\nPOSTGRES_PORT=15432\nDB_BIND_ADDR=127.0.0.1\n" +
		"DEVELOP_DB_NAME=mindforge_develop\nDEVELOP_DB_USER=mf_develop\nDEVELOP_DB_PASSWORD=synthetic-develop-password-123456\n" +
		"MAIN_DB_NAME=mindforge\nMAIN_DB_USER=mf_main\nMAIN_DB_PASSWORD=synthetic-main-password-123456\n" +
		"DEVELOP_AUTH_DB_URL=postgres://mf_develop:synthetic-develop-password-123456@db:5432/mindforge_develop?sslmode=disable&search_path=auth\n" +
		"MAIN_AUTH_DB_URL=postgres://mf_main:synthetic-main-password-123456@db:5432/mindforge?sslmode=disable&search_path=auth\n" +
		"DEVELOP_AUTH_PORT=19999\nMAIN_AUTH_PORT=19998\nDEVELOP_AUTH_BIND_ADDR=127.0.0.1\nMAIN_AUTH_BIND_ADDR=127.0.0.1\nALLOW_LAN_BINDING=no\n" +
		"DEVELOP_AUTH_SITE_URL=http://localhost:19999\nMAIN_AUTH_SITE_URL=http://localhost:19998\n" +
		"DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS=http://localhost:19999/callback\nMAIN_AUTH_ADDITIONAL_REDIRECT_URLS=http://localhost:19998/callback\n" +
		"DEVELOP_AUTH_URI_ALLOW_LIST=http://localhost:19999/*\nMAIN_AUTH_URI_ALLOW_LIST=http://localhost:19998/*\n" +
		"DEVELOP_AUTH_API_EXTERNAL_URL=http://127.0.0.1:19999\nMAIN_AUTH_API_EXTERNAL_URL=http://127.0.0.1:19998\n" +
		"DEVELOP_AUTH_JWT_SECRET=synthetic-develop-jwt-secret-123456789\nMAIN_AUTH_JWT_SECRET=synthetic-main-jwt-secret-123456789\n" +
		"DEVELOP_AUTH_OPERATOR_TOKEN=synthetic-develop-operator-token-123456789\nMAIN_AUTH_OPERATOR_TOKEN=synthetic-main-operator-token-123456789\n" +
		"GOTRUE_DB_MAX_POOL_SIZE=10\nGOTRUE_MAILER_AUTOCONFIRM=true\nTZ=UTC\n"
	path := filepath.Join(t.TempDir(), "bootstrap-private.env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write private fixture: %v", err)
	}
	return path
}

func writeBootstrapCatalog(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	infrastructure := filepath.Join(root, "src/brain-engine/features/infra/postgres/migrations")
	features := filepath.Join(root, "src/brain-engine/features/infra/postgres/feature-migrations")
	if err := os.MkdirAll(infrastructure, 0o700); err != nil {
		t.Fatalf("mkdir infrastructure migrations: %v", err)
	}
	if err := os.MkdirAll(features, 0o700); err != nil {
		t.Fatalf("mkdir feature catalog: %v", err)
	}
	for id := 1; id <= 5; id++ {
		path := filepath.Join(infrastructure, fmt.Sprintf("%04d_infra_%d.sql", id, id))
		if err := os.WriteFile(path, []byte(fmt.Sprintf("-- changed infrastructure source %d\n", id)), 0o600); err != nil {
			t.Fatalf("write infrastructure migration: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(features, "catalog.json"), []byte("{\"schema_version\":1,\"migrations\":[]}\n"), 0o600); err != nil {
		t.Fatalf("write feature descriptor: %v", err)
	}
	return root
}

func bootstrapRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not identify test source")
	}
	directory, err := filepath.Abs(filepath.Dir(source))
	if err != nil {
		t.Fatalf("filepath.Abs(): %v", err)
	}
	for {
		candidate := filepath.Join(directory, "src/brain-engine/features/infra/postgres/migrations/0001_infra_init.sql")
		if _, err := os.Stat(candidate); err == nil {
			canonical, err := filepath.EvalSymlinks(directory)
			if err != nil {
				t.Fatalf("filepath.EvalSymlinks(): %v", err)
			}
			return canonical
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("could not locate repository root from test source")
		}
		directory = parent
	}
}

func bootstrapMetadataRow(infra, ledger, evidenceTable, attemptsTable, auth, users, identities, schemaMigrations, schemaCount, ledgerRows string) string {
	return bootstrapMetadataRowWithVector(infra, ledger, evidenceTable, attemptsTable, auth, users, identities, schemaMigrations, schemaCount, ledgerRows, "")
}

func bootstrapMetadataRowWithVector(infra, ledger, evidenceTable, attemptsTable, auth, users, identities, schemaMigrations, schemaCount, ledgerRows, vectorVersion string) string {
	return strings.Join([]string{
		string(config.TargetDevelop), "mindforge,mindforge_develop,nsrdb", vectorVersion, infra, ledger,
		evidenceTable, attemptsTable, auth, users, identities, schemaMigrations, schemaCount, ledgerRows,
	}, "|") + "\n"
}

func bootstrapIdentityRow(database, currentUser, databaseOwner string, superuser, createdb, canLogin bool, infra, infraOwner, ledger, ledgerOwner, extensions string, otherInfraRelations int, auth, authUsers, authIdentities, authMigrations string) string {
	if ledger == "migration_ledger" {
		ledger += ":r"
	}
	fields := []string{
		database, currentUser, databaseOwner, fmt.Sprint(superuser), fmt.Sprint(createdb), fmt.Sprint(canLogin),
		infra, infraOwner, ledger, ledgerOwner, extensions, fmt.Sprint(otherInfraRelations),
		auth, authUsers, authIdentities, authMigrations, "citext:1.6,uuid-ossp:1.1,vector:0.8.6",
	}
	return strings.Join(fields, "|")
}

func invocationArg(invocation command.Invocation, prefix string) (string, bool) {
	for _, argument := range invocation.Args {
		if strings.HasPrefix(argument, prefix) {
			return strings.TrimPrefix(argument, prefix), true
		}
	}
	return "", false
}

func containsArgPrefix(arguments []string, prefix string) bool {
	for _, argument := range arguments {
		if argument == prefix || strings.HasPrefix(argument, prefix) {
			return true
		}
	}
	return false
}

func assertConnectionTarget(t *testing.T, invocation command.Invocation, expectedUser, expectedDatabase string) {
	t.Helper()
	user, userOK := invocationArg(invocation, "--username=")
	database, databaseOK := invocationArg(invocation, "--dbname=")
	host, hostOK := invocationArg(invocation, "--host=")
	if !userOK || !databaseOK || !hostOK || user != expectedUser || database != expectedDatabase || host != "127.0.0.1" {
		t.Fatalf("database invocation target mismatch: user=%q database=%q host=%q", user, database, host)
	}
}

func assertBootstrapConnectionsOnlyDevelop(t *testing.T, calls []command.Invocation) {
	t.Helper()
	for _, call := range calls {
		if call.Name != "psql" || containsArgPrefix(call.Args, "--version") {
			continue
		}
		database, ok := invocationArg(call, "--dbname=")
		if !ok || database != string(config.TargetDevelop) {
			t.Fatalf("bootstrap connected outside explicit develop target: %#v", call.Args)
		}
	}
}

func invocationForUser(calls []command.Invocation, user, requiredArgument string) (command.Invocation, bool) {
	for _, call := range calls {
		candidate, ok := invocationArg(call, "--username=")
		if ok && candidate == user && containsArgPrefix(call.Args, requiredArgument) {
			return call, true
		}
	}
	return command.Invocation{}, false
}

func invocationWithCommandContains(calls []command.Invocation, fragment string) (command.Invocation, bool) {
	for _, call := range calls {
		query, ok := invocationArg(call, "--command=")
		if ok && strings.Contains(query, fragment) {
			return call, true
		}
	}
	return command.Invocation{}, false
}

func containsEnv(invocation command.Invocation, expected string) bool {
	for _, value := range invocation.Env {
		if value == expected {
			return true
		}
	}
	return false
}

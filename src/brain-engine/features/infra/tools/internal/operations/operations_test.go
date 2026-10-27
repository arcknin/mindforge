package operations_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/evidence"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/operations"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/testsupport"
)

func TestPreflightIsReadOnlyAndProducesTargetSpecificReference(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	values := writePrivateConfig(t)
	settings, err := config.ParseFile(values)
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	ledger := makeLedger(hashes)
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: fmt.Sprintf(
		"mindforge_develop|mindforge,mindforge_develop|0.8.6|infra|infra.migration_ledger|infra.migration_target_evidence|infra.migration_attempts|auth|auth.users|auth.identities|auth.schema_migrations|1|%s\n",
		ledger,
	)}, nil)

	result, err := operations.Preflight(context.Background(), operations.PreflightRequest{
		Config:     settings,
		Operation:  request,
		SourceRoot: root,
		Runner:     fake,
	})
	if err != nil {
		t.Fatalf("Preflight(): %v", err)
	}
	if result.Target != config.TargetDevelop || result.Reference == "" {
		t.Fatalf("Preflight() = %#v, want target and sanitized reference", result)
	}
	if !strings.HasPrefix(result.Reference, "preflight/mindforge_develop/") {
		t.Fatalf("preflight reference = %q", result.Reference)
	}
	calls := fake.Calls()
	if len(calls) != 2 {
		t.Fatalf("runner calls = %#v, want version check and one read-only query", calls)
	}
	if calls[0].Name != "psql" || strings.Join(calls[0].Args, " ") != "--version" {
		t.Fatalf("first call = %#v, want psql --version", calls[0])
	}
	query := strings.Join(calls[1].Args, " ")
	if !strings.Contains(query, "SELECT") || strings.Contains(query, "CREATE ") || strings.Contains(query, "INSERT ") || strings.Contains(query, "UPDATE ") {
		t.Fatalf("preflight query is not read-only: %q", query)
	}
	if strings.Contains(query, "synthetic-") || strings.Contains(strings.Join(calls[1].Env, " "), "synthetic-") == false {
		t.Fatalf("preflight did not keep the database password in the child environment: %#v", calls[1])
	}
}

func TestFeaturePreflightRequiresSchemaShapeToMatchGlobalLedger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		applied bool
		state   string
		wantOK  bool
	}{
		{name: "absent eligible without ledger", state: "absent_eligible", wantOK: true},
		{name: "complete with matching ledger", applied: true, state: "complete", wantOK: true},
		{name: "partial schema", state: "partial_divergent"},
		{name: "complete schema without ledger", state: "complete"},
		{name: "ledger without complete schema", applied: true, state: "absent_eligible"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, hashes := writeCatalog(t, true)
			settings, err := config.ParseFile(writePrivateConfig(t))
			if err != nil {
				t.Fatalf("ParseFile(): %v", err)
			}
			request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-feature-preflight-2026")
			if err != nil {
				t.Fatalf("NewOperationRequest(): %v", err)
			}
			ledger := makeLedger(hashes)
			if test.applied {
				ledger = makeLedgerWithFeature(hashes)
			}
			fake := testsupport.NewFakeRunner()
			fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
			fake.Enqueue(command.Result{Stdout: fmt.Sprintf(
				"mindforge_develop|mindforge,mindforge_develop|0.8.6|infra|infra.migration_ledger|infra.migration_target_evidence|infra.migration_attempts|auth|auth.users|auth.identities|auth.schema_migrations|1|%s\n",
				ledger,
			)}, nil)
			fake.Enqueue(command.Result{Stdout: test.state + "\n"}, nil)

			result, err := operations.Preflight(context.Background(), operations.PreflightRequest{
				Config: settings, Operation: request, Scope: operations.ScopeFeature, SourceRoot: root, Runner: fake,
			})
			if (err == nil) != test.wantOK {
				t.Fatalf("Preflight() error = %v, want success=%v", err, test.wantOK)
			}
			calls := fake.Calls()
			if !test.wantOK {
				return
			}
			if len(calls) != 3 || !strings.Contains(strings.Join(calls[2].Args, " "), "--file=") || strings.Contains(strings.Join(calls[2].Args, " "), "--single-transaction") {
				t.Fatalf("feature preflight calls = %#v, want a read-only feature check against the explicit target", calls)
			}
			if !strings.Contains(result.ObservedState, "feature-state:0006="+test.state) || result.CatalogHash == "" {
				t.Fatalf("feature preflight snapshot = %#v, want schema state bound into reference", result)
			}
		})
	}
}

func TestPreflightIgnoresUnrelatedDatabaseNamesAndReferences(t *testing.T) {
	t.Parallel()

	withNSRDB := preflightForDatabaseInventory(t, "mindforge,mindforge_develop,nsrdb")
	withAnotherProject := preflightForDatabaseInventory(t, "mindforge,mindforge_develop,another_project,nsrdb")
	if withNSRDB.Reference != withAnotherProject.Reference {
		t.Fatalf("preflight references changed with unrelated cluster names: %q != %q", withNSRDB.Reference, withAnotherProject.Reference)
	}
	if strings.Contains(withNSRDB.ObservedState, "nsrdb") || strings.Contains(withNSRDB.ObservedState, "another_project") {
		t.Fatalf("preflight observed state includes unrelated database names: %q", withNSRDB.ObservedState)
	}
}

func preflightForDatabaseInventory(t *testing.T, databases string) evidence.Preflight {
	t.Helper()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-30")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: fmt.Sprintf(
		"mindforge_develop|%s|0.8.6|infra|infra.migration_ledger|infra.migration_target_evidence|infra.migration_attempts|auth|auth.users|auth.identities|auth.schema_migrations|1|%s\n",
		databases,
		makeLedger(hashes),
	)}, nil)

	result, err := operations.Preflight(context.Background(), operations.PreflightRequest{
		Config: settings, Operation: request, SourceRoot: root, Runner: fake,
	})
	if err != nil {
		t.Fatalf("Preflight(%q): %v", databases, err)
	}
	for _, call := range fake.Calls() {
		if strings.Contains(strings.Join(call.Args, " "), "--dbname=nsrdb") {
			t.Fatalf("preflight connected to unrelated database nsrdb: %#v", call)
		}
	}
	return result
}

func TestPreflightRejectsTargetDatabaseMismatchBeforeAnyDDL(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, false)
	values := writePrivateConfig(t)
	settings, err := config.ParseFile(values)
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: fmt.Sprintf(
		"mindforge|mindforge,mindforge_develop|0.8.6|infra|infra.migration_ledger|infra.migration_target_evidence|infra.migration_attempts|auth|auth.users|auth.identities|auth.schema_migrations|1|%s\n",
		makeLedger(hashes),
	)}, nil)

	_, err = operations.Preflight(context.Background(), operations.PreflightRequest{
		Config:     settings,
		Operation:  request,
		SourceRoot: root,
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Preflight() error = nil for a target/database mismatch")
	}
	if len(fake.Calls()) != 2 {
		t.Fatalf("runner calls = %#v, want no mutation after read-only mismatch", fake.Calls())
	}
}

func TestPreflightRejectsSwappedAuthDatabaseURLBeforeClientInvocation(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	configPath := writePrivateConfig(t)
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(): %v", err)
	}
	contents = []byte(strings.Replace(string(contents),
		"DEVELOP_AUTH_DB_URL=postgres://mf_develop:synthetic-develop-password-123456@db:5432/mindforge_develop?sslmode=disable&search_path=auth",
		"DEVELOP_AUTH_DB_URL=postgres://mf_main:synthetic-main-password-123456@db:5432/mindforge?sslmode=disable&search_path=auth",
		1))
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	settings, err := config.ParseFile(configPath)
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Preflight(context.Background(), operations.PreflightRequest{
		Config:     settings,
		Operation:  request,
		SourceRoot: root,
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Preflight() error = nil for a swapped Auth URL")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("runner calls = %#v, want configuration rejection before client invocation", fake.Calls())
	}
}

func TestProvisionDevelopRechecksExactClusterBeforeOneAdditiveFileOperation(t *testing.T) {
	t.Parallel()

	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-provision-2026-09-23")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge|true|mf_main|true|false|false|true\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge|true|mf_main|true|false|false|true\n"}, nil)
	fake.Enqueue(command.Result{}, nil)
	runner := &provisioningRunner{fake: fake}

	result, err := operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
		Config:    settings,
		Operation: request,
		Runner:    runner,
	})
	if err != nil {
		t.Fatalf("ProvisionDevelop(): %v", err)
	}
	if result.Target != config.TargetDevelop || result.Status != "succeeded" {
		t.Fatalf("ProvisionDevelop() = %#v, want sanitized develop success", result)
	}
	calls := fake.Calls()
	if len(calls) != 4 || calls[0].Name != "psql" || strings.Join(calls[0].Args, " ") != "--version" {
		t.Fatalf("runner calls = %#v, want client check, two inventories, one file operation", calls)
	}
	for index := 1; index <= 2; index++ {
		joined := strings.Join(calls[index].Args, " ")
		if !strings.Contains(joined, "--dbname=postgres") || !strings.Contains(joined, "--username=mf_cluster_admin") || !strings.Contains(joined, "SELECT") || strings.Contains(joined, "CREATE ") {
			t.Fatalf("inventory call %d is not read-only cluster-admin inspection: %#v", index, calls[index])
		}
		if !strings.Contains(joined, "string_agg(datname, ',' ORDER BY datname)") || strings.Contains(joined, "datallowconn") {
			t.Fatalf("inventory call %d does not inspect a sorted complete database list: %#v", index, calls[index])
		}
	}
	if strings.Contains(strings.Join(calls[3].Args, " "), "synthetic-") || len(runner.fileContents) != 1 {
		t.Fatalf("provision operation leaked secrets or omitted private SQL file: calls=%#v files=%#v", calls, runner.fileContents)
	}
	ddl := runner.fileContents[0]
	if !strings.Contains(ddl, `CREATE ROLE "mf_develop" LOGIN PASSWORD 'synthetic-develop-password-123456';`) || !strings.Contains(ddl, `CREATE DATABASE "mindforge_develop" OWNER "mf_develop";`) {
		t.Fatalf("provision SQL = %q, want only configured develop role and database", ddl)
	}
	if strings.Contains(ddl, `CREATE DATABASE "mindforge"`) || strings.Contains(ddl, `ALTER `) || strings.Contains(ddl, `DROP `) || strings.Contains(ddl, `mindforge_main`) {
		t.Fatalf("provision SQL contains a forbidden main mutation: %q", ddl)
	}
	if len(runner.filePaths) != 1 || runner.fileModes[0].Perm() != 0o600 {
		t.Fatalf("temporary SQL permissions = %#v, want mode 0600", runner.fileModes)
	}
	if _, err := os.Stat(runner.filePaths[0]); !os.IsNotExist(err) {
		t.Fatalf("temporary SQL file still exists after operation: stat error = %v", err)
	}
	for _, call := range calls {
		if strings.Contains(strings.Join(call.Args, " "), "synthetic-") {
			t.Fatalf("provision call leaked a configured secret in argv: %#v", call)
		}
	}
}

func TestProvisionDevelopAcceptsPassiveUnrelatedDatabase(t *testing.T) {
	t.Parallel()

	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-provision-2026-09-30")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge,nsrdb|true|mf_main|true|false|false|true\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "another_project,mindforge,nsrdb|true|mf_main|true|false|false|true\n"}, nil)
	fake.Enqueue(command.Result{}, nil)
	runner := &provisioningRunner{fake: fake}

	result, err := operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
		Config: settings, Operation: request, Runner: runner,
	})
	if err != nil {
		t.Fatalf("ProvisionDevelop() with passive nsrdb: %v", err)
	}
	if result.Status != "succeeded" || result.Target != config.TargetDevelop {
		t.Fatalf("ProvisionDevelop() = %#v, want explicit develop success", result)
	}
	calls := fake.Calls()
	if len(calls) != 4 {
		t.Fatalf("runner calls = %#v, want version check, two inventories, one additive operation", calls)
	}
	for _, call := range calls {
		joined := strings.Join(call.Args, " ")
		if strings.Contains(joined, "--dbname=nsrdb") || strings.Contains(joined, "CREATE DATABASE \"nsrdb\"") {
			t.Fatalf("unrelated database became an operation target: %#v", call)
		}
	}
	if len(runner.fileContents) != 1 || !strings.Contains(runner.fileContents[0], `CREATE DATABASE "mindforge_develop" OWNER "mf_develop";`) {
		t.Fatalf("provision SQL = %#v, want only the configured develop database", runner.fileContents)
	}
}

func TestPreflightAndProvisionRejectNSRDBAsExplicitTarget(t *testing.T) {
	t.Parallel()

	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	for _, operationName := range []string{"preflight", "provision"} {
		t.Run(operationName, func(t *testing.T) {
			fake := testsupport.NewFakeRunner()
			request := command.OperationRequest{Target: config.Target("nsrdb"), OperatorID: "operator-1", AuthorizationRef: "owner-2026-09-30"}
			var err error
			switch operationName {
			case "preflight":
				root, _ := writeCatalog(t, true)
				_, err = operations.Preflight(context.Background(), operations.PreflightRequest{
					Config: settings, Operation: request, SourceRoot: root, Runner: fake,
				})
			case "provision":
				_, err = operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
					Config: settings, Operation: request, Runner: fake,
				})
			}
			if err == nil || len(fake.Calls()) != 0 {
				t.Fatalf("%s accepted nsrdb or used the runner: error=%v calls=%#v", operationName, err, fake.Calls())
			}
		})
	}
}

func TestProvisionDevelopRefusesExistingPartialAndUnexpectedClusterState(t *testing.T) {
	t.Parallel()

	for name, inventory := range map[string]string{
		"existing develop database":              "mindforge,mindforge_develop|true|mf_main|true|true|true|true\n",
		"partial develop role":                   "mindforge|true|mf_main|true|true|false|true\n",
		"main missing beside unrelated database": "nsrdb|false||false|false|false|true\n",
		"main owned by another role":             "mindforge|true|unexpected_owner|true|false|false|true\n",
		"admin is not superuser":                 "mindforge|true|mf_main|true|false|false|false\n",
		"main missing":                           "|false||false|false|false|true\n",
	} {
		name, inventory := name, inventory
		t.Run(name, func(t *testing.T) {
			settings, err := config.ParseFile(writePrivateConfig(t))
			if err != nil {
				t.Fatalf("ParseFile(): %v", err)
			}
			request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-provision-2026-09-23")
			if err != nil {
				t.Fatalf("NewOperationRequest(): %v", err)
			}
			fake := testsupport.NewFakeRunner()
			fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
			fake.Enqueue(command.Result{Stdout: inventory}, nil)
			runner := &provisioningRunner{fake: fake}
			_, err = operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
				Config: settings, Operation: request, Runner: runner,
			})
			if err == nil {
				t.Fatal("ProvisionDevelop() error = nil for unsafe starting topology")
			}
			if len(fake.Calls()) != 2 || len(runner.fileContents) != 0 {
				t.Fatalf("unsafe topology reached a write: calls=%#v files=%#v", fake.Calls(), runner.fileContents)
			}
			if strings.Contains(err.Error(), "synthetic-") {
				t.Fatalf("ProvisionDevelop() leaked configuration secret: %v", err)
			}
		})
	}
}

func TestProvisionDevelopRejectsWrongTargetBeforeClientInvocation(t *testing.T) {
	t.Parallel()

	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-provision-2026-09-23")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
		Config: settings, Operation: request, Runner: fake,
	})
	if err == nil || len(fake.Calls()) != 0 {
		t.Fatalf("ProvisionDevelop() = error %v, calls %#v; want wrong target rejected before client use", err, fake.Calls())
	}
}

func TestProvisionDevelopRechecksAndStopsIfTopologyChanges(t *testing.T) {
	t.Parallel()

	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-provision-2026-09-23")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge|true|mf_main|true|false|false|true\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge,mindforge_develop|true|mf_main|true|true|true|true\n"}, nil)
	runner := &provisioningRunner{fake: fake}
	_, err = operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
		Config: settings, Operation: request, Runner: runner,
	})
	if err == nil {
		t.Fatal("ProvisionDevelop() error = nil after develop appeared during recheck")
	}
	if len(fake.Calls()) != 3 || len(runner.fileContents) != 0 {
		t.Fatalf("topology drift reached a write: calls=%#v files=%#v", fake.Calls(), runner.fileContents)
	}
}

func TestProvisionDevelopReportsSanitizedIncompleteResultWhenCreateFails(t *testing.T) {
	t.Parallel()

	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-provision-2026-09-23")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge|true|mf_main|true|false|false|true\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge|true|mf_main|true|false|false|true\n"}, nil)
	fake.Enqueue(command.Result{Stderr: "synthetic-develop-password-123456 rejected"}, fmt.Errorf("synthetic-develop-password-123456"))
	runner := &provisioningRunner{fake: fake}
	result, err := operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
		Config: settings, Operation: request, Runner: runner,
	})
	if err == nil || result.Status != "incomplete" {
		t.Fatalf("ProvisionDevelop() = %#v, error %v; want sanitized incomplete result", result, err)
	}
	if strings.Contains(err.Error(), "synthetic-") || len(runner.filePaths) != 1 {
		t.Fatalf("provisioning failure leaked a secret or omitted the private file: result=%#v error=%v paths=%v", result, err, runner.filePaths)
	}
	if _, err := os.Stat(runner.filePaths[0]); !os.IsNotExist(err) {
		t.Fatalf("failed operation left temporary SQL file: stat error = %v", err)
	}
}

type provisioningRunner struct {
	fake         *testsupport.FakeRunner
	fileContents []string
	filePaths    []string
	fileModes    []os.FileMode
}

func (runner *provisioningRunner) Run(ctx context.Context, invocation command.Invocation) (command.Result, error) {
	for _, argument := range invocation.Args {
		if strings.HasPrefix(argument, "--file=") {
			path := strings.TrimPrefix(argument, "--file=")
			contents, err := os.ReadFile(path)
			if err != nil {
				return command.Result{}, fmt.Errorf("read private SQL fixture: %w", err)
			}
			runner.fileContents = append(runner.fileContents, string(contents))
			info, err := os.Stat(path)
			if err != nil {
				return command.Result{}, fmt.Errorf("stat private SQL fixture: %w", err)
			}
			runner.filePaths = append(runner.filePaths, path)
			runner.fileModes = append(runner.fileModes, info.Mode())
		}
	}
	return runner.fake.Run(ctx, invocation)
}

func TestApplyRejectsFeatureScopeOnMainWithoutInvokingDatabase(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Apply(context.Background(), operations.ApplyRequest{
		Config:                  settings,
		Operation:               request,
		Scope:                   operations.ScopeFeature,
		SourceRoot:              root,
		Runner:                  fake,
		Preflight:               evidence.Preflight{Target: config.TargetMain, Reference: "preflight/mindforge/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		BehaviorVerificationRef: "verification/feature-2026-09-19",
	})
	if err == nil {
		t.Fatal("Apply() error = nil for feature scope on main")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("runner calls = %#v, want no database operation", fake.Calls())
	}
}

func TestApplyRecordsEvidenceOnlyAfterReadOnlyPreflightAndUsesTargetPasswordEnv(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	preflight := evidence.Preflight{
		Target:              config.TargetDevelop,
		Reference:           "preflight/mindforge_develop/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SourceRoot:          canonicalSourceRoot(t, root),
		CatalogHash:         catalogHash(t, root),
		ObservedState:       "mindforge_develop|mindforge,mindforge_develop|0.8.6|ready|feature-state:0006=absent_eligible",
		Ledger:              parseLedger(t, makeLedger(hashes)),
		InfrastructureReady: true,
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "11111111-1111-4111-8111-111111111111\n"}, nil)
	fake.Enqueue(command.Result{}, nil)
	fake.Enqueue(command.Result{}, nil)
	capture := &featureTransactionCapture{runner: fake}

	result, err := operations.Apply(context.Background(), operations.ApplyRequest{
		Config:                  settings,
		Operation:               request,
		Scope:                   operations.ScopeFeature,
		SourceRoot:              root,
		Runner:                  capture,
		Preflight:               preflight,
		BehaviorVerificationRef: "verification/feature-2026-09-19",
	})
	if err != nil {
		t.Fatalf("Apply(): %v", err)
	}
	if len(result.Applied) != 1 || result.Applied[0].Target != config.TargetDevelop {
		t.Fatalf("Apply() result = %#v, want one develop record", result)
	}
	calls := fake.Calls()
	if len(calls) != 4 {
		t.Fatalf("runner calls = %#v, want version, attempt, DDL, evidence update", calls)
	}
	for index, call := range calls {
		if call.Shell {
			t.Fatalf("call %d enables shell evaluation: %#v", index, call)
		}
		if strings.Contains(strings.Join(call.Args, " "), "synthetic-develop-password-123456") {
			t.Fatalf("call %d leaked a password in arguments: %#v", index, call)
		}
	}
	if !strings.Contains(strings.Join(calls[1].Args, " "), "migration_attempts") {
		t.Fatalf("second apply call = %#v, want incomplete attempt evidence", calls[1])
	}
	thirdCall := strings.Join(calls[2].Args, " ")
	if !strings.Contains(thirdCall, "--file=") || !strings.Contains(thirdCall, "--single-transaction") || strings.Contains(thirdCall, "--command=") {
		t.Fatalf("third apply call = %#v, want one atomic SQL file transaction", calls[2])
	}
	if !strings.Contains(capture.transactionSQL, "CREATE SCHEMA IF NOT EXISTS brain_core;") || !strings.Contains(capture.transactionSQL, "DO $$ BEGIN PERFORM 1; END $$;") || strings.Count(capture.transactionSQL, "INSERT INTO infra.migration_ledger") != 1 {
		t.Fatalf("atomic feature transaction = %q, want canonical payload, verify check, and one ledger insert", capture.transactionSQL)
	}
	if strings.Index(capture.transactionSQL, "CREATE SCHEMA IF NOT EXISTS brain_core;") > strings.Index(capture.transactionSQL, "DO $$ BEGIN PERFORM 1; END $$;") || strings.Index(capture.transactionSQL, "DO $$ BEGIN PERFORM 1; END $$;") > strings.Index(capture.transactionSQL, "INSERT INTO infra.migration_ledger") {
		t.Fatalf("atomic feature transaction is out of order: %q", capture.transactionSQL)
	}
	if !strings.Contains(strings.Join(calls[3].Args, " "), "migration_target_evidence") {
		t.Fatalf("fourth apply call = %#v, want target evidence completion", calls[3])
	}
	if !strings.Contains(strings.Join(calls[2].Env, " "), "PGPASSWORD=synthetic-develop-password-123456") {
		t.Fatalf("target password was not restricted to child environment: %#v", calls[2])
	}
}

func TestApplyMarksFailureAfterTransactionalMigrationRed(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "11111111-1111-4111-8111-111111111111\n"}, nil)
	fake.Enqueue(command.Result{ExitCode: 1}, fmt.Errorf("synthetic migration failure"))
	fake.Enqueue(command.Result{}, nil)
	capture := &featureTransactionCapture{runner: fake}

	_, err = operations.Apply(context.Background(), operations.ApplyRequest{
		Config:     settings,
		Operation:  request,
		Scope:      operations.ScopeFeature,
		SourceRoot: root,
		Runner:     capture,
		Preflight: evidence.Preflight{
			Target:              config.TargetDevelop,
			Reference:           "preflight/mindforge_develop/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SourceRoot:          canonicalSourceRoot(t, root),
			CatalogHash:         catalogHash(t, root),
			ObservedState:       "ready|feature-state:0006=absent_eligible",
			Ledger:              parseLedger(t, makeLedger(hashes)),
			InfrastructureReady: true,
		},
		BehaviorVerificationRef: "verification/feature-2026-09-19",
	})
	if err == nil {
		t.Fatal("Apply() error = nil after migration failure")
	}
	calls := fake.Calls()
	if len(calls) != 4 {
		t.Fatalf("runner calls = %#v, want version, attempt, failed DDL and failure evidence", calls)
	}
	if !strings.Contains(strings.Join(calls[2].Args, " "), "--single-transaction") || strings.Count(capture.transactionSQL, "INSERT INTO infra.migration_ledger") != 1 {
		t.Fatalf("failed feature verification was not inside the atomic payload/ledger transaction: call=%#v sql=%q", calls[2], capture.transactionSQL)
	}
	if !strings.Contains(strings.Join(calls[3].Args, " "), "sanitized_outcome='failed'") {
		t.Fatalf("failure evidence call = %#v", calls[3])
	}
}

func TestApplyOnlyTreatsMatchingLedgerAndCompleteShapeAsNoOp(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		state  string
		wantOK bool
	}{
		{name: "matching ledger and complete schema", state: "complete", wantOK: true},
		{name: "matching ledger but absent schema", state: "absent_eligible"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, hashes := writeCatalog(t, true)
			settings, err := config.ParseFile(writePrivateConfig(t))
			if err != nil {
				t.Fatalf("ParseFile(): %v", err)
			}
			request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-feature-noop-2026")
			if err != nil {
				t.Fatalf("NewOperationRequest(): %v", err)
			}
			fake := testsupport.NewFakeRunner()
			fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
			result, err := operations.Apply(context.Background(), operations.ApplyRequest{
				Config: settings, Operation: request, Scope: operations.ScopeFeature, SourceRoot: root, Runner: fake,
				Preflight: evidence.Preflight{
					Target: request.Target, Reference: "preflight/mindforge_develop/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					SourceRoot: canonicalSourceRoot(t, root), CatalogHash: catalogHash(t, root),
					ObservedState: "ready|feature-state:0006=" + test.state, Ledger: parseLedger(t, makeLedgerWithFeature(hashes)), InfrastructureReady: true,
				},
				BehaviorVerificationRef: "verification/feature-noop-2026",
			})
			if (err == nil) != test.wantOK {
				t.Fatalf("Apply() error = %v, want success=%v", err, test.wantOK)
			}
			if test.wantOK && len(result.Applied) != 0 {
				t.Fatalf("Apply() result = %#v, want no-op", result)
			}
			if len(fake.Calls()) != 1 {
				t.Fatalf("Apply() calls = %#v, want only client version check and no migration/evidence mutation", fake.Calls())
			}
		})
	}
}

func TestApplyBootstrapsEmptyInfrastructureLedgerBeforeVersionedMigrations(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)

	result, err := operations.Apply(context.Background(), operations.ApplyRequest{
		Config:     settings,
		Operation:  request,
		Scope:      operations.ScopeInfrastructure,
		SourceRoot: root,
		Runner:     fake,
		Preflight: evidence.Preflight{
			Target:              config.TargetDevelop,
			Reference:           "preflight/mindforge_develop/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SourceRoot:          canonicalSourceRoot(t, root),
			CatalogHash:         catalogHash(t, root),
			ObservedState:       "empty-cluster",
			Ledger:              migrations.Ledger{},
			InfrastructureReady: false,
		},
		BehaviorVerificationRef: "verification/infra-bootstrap-2026-09-19",
	})
	if err != nil {
		t.Fatalf("Apply(): %v", err)
	}
	if len(result.Applied) != 5 {
		t.Fatalf("Apply() applied %d migrations, want 5", len(result.Applied))
	}
	calls := fake.Calls()
	if len(calls) != 7 {
		t.Fatalf("runner calls = %d, want version, bootstrap and five migrations", len(calls))
	}
	if !strings.Contains(strings.Join(calls[1].Args, " "), "--single-transaction") || !strings.Contains(strings.Join(calls[1].Args, " "), "CREATE TABLE IF NOT EXISTS infra.migration_ledger") {
		t.Fatalf("bootstrap call = %#v, want transactional ledger bootstrap", calls[1])
	}
	if !strings.Contains(strings.Join(calls[2].Args, " "), "--file=") || !strings.Contains(strings.Join(calls[2].Args, " "), "--port=15432") {
		t.Fatalf("first migration call = %#v, want configured target connection", calls[2])
	}
}

func writeCatalog(t *testing.T, withFeature bool) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	infra := filepath.Join(root, "src/brain-engine/features/infra/postgres/migrations")
	features := filepath.Join(root, "src/brain-engine/features/infra/postgres/feature-migrations")
	brain := filepath.Join(root, "src/brain-engine/features/brain/migrations")
	if err := os.MkdirAll(infra, 0o700); err != nil {
		t.Fatalf("mkdir infra: %v", err)
	}
	if err := os.MkdirAll(features, 0o700); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}
	if err := os.MkdirAll(brain, 0o700); err != nil {
		t.Fatalf("mkdir Brain Core feature: %v", err)
	}
	for index := 1; index <= 5; index++ {
		path := filepath.Join(infra, fmt.Sprintf("%04d_infra_%d.sql", index, index))
		if err := os.WriteFile(path, []byte(fmt.Sprintf("-- infrastructure %d\n", index)), 0o600); err != nil {
			t.Fatalf("write migration: %v", err)
		}
	}
	mappings := make([]map[string]string, 0, 1)
	if withFeature {
		sources := map[string]string{
			"src/brain-engine/features/brain/migrations/0001_brain_core.sql":           "CREATE SCHEMA IF NOT EXISTS brain_core;\n",
			"src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql": "SELECT 'absent_eligible';\n",
			"src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql":    "DO $$ BEGIN PERFORM 1; END $$;\n",
		}
		for relative, contents := range sources {
			path := filepath.Join(root, filepath.FromSlash(relative))
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatalf("write feature source %s: %v", relative, err)
			}
		}
		mappings = append(mappings, map[string]string{
			"id": "0006", "name": "brain_core", "feature": "brain",
			"source":           "src/brain-engine/features/brain/migrations/0001_brain_core.sql",
			"sha256":           testHash(sources["src/brain-engine/features/brain/migrations/0001_brain_core.sql"]),
			"preflight":        "src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql",
			"preflight_sha256": testHash(sources["src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql"]),
			"verify":           "src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql",
			"verify_sha256":    testHash(sources["src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql"]),
		})
	}
	descriptor, err := json.MarshalIndent(struct {
		SchemaVersion int                 `json:"schema_version"`
		Migrations    []map[string]string `json:"migrations"`
	}{SchemaVersion: 1, Migrations: mappings}, "", "  ")
	if err != nil {
		t.Fatalf("marshal feature descriptor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(features, "catalog.json"), append(descriptor, '\n'), 0o600); err != nil {
		t.Fatalf("write feature descriptor: %v", err)
	}
	catalog, err := migrations.LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(): %v", err)
	}
	hashes := make(map[string]string, len(catalog.Migrations))
	for _, migration := range catalog.Migrations {
		hashes[migration.ID] = migration.SHA256
	}
	return root, hashes
}

func testHash(contents string) string {
	digest := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(digest[:])
}

func writePrivateConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "private.env")
	contents := "POSTGRES_ADMIN_USER=mf_cluster_admin\n" +
		"POSTGRES_ADMIN_PASSWORD=synthetic-cluster-password-123456\n" +
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
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func makeLedger(hashes map[string]string) string {
	parts := make([]string, 0, 5)
	for index := 1; index <= 5; index++ {
		id := fmt.Sprintf("%04d", index)
		parts = append(parts, id+":"+hashes[id])
	}
	return strings.Join(parts, ",")
}

func parseLedger(t *testing.T, raw string) migrations.Ledger {
	t.Helper()
	ledger := make(migrations.Ledger)
	for _, row := range strings.Split(raw, ",") {
		parts := strings.SplitN(row, ":", 2)
		if len(parts) != 2 {
			t.Fatalf("invalid ledger fixture %q", raw)
		}
		ledger[parts[0]] = parts[1]
	}
	return ledger
}

func canonicalSourceRoot(t *testing.T, root string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(): %v", err)
	}
	return canonical
}

func catalogHash(t *testing.T, root string) string {
	t.Helper()
	catalog, err := migrations.LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(): %v", err)
	}
	return migrations.CatalogHash(catalog)
}

type featureTransactionCapture struct {
	runner          *testsupport.FakeRunner
	transactionSQL  string
	transactionCall command.Invocation
}

func (capture *featureTransactionCapture) Run(ctx context.Context, invocation command.Invocation) (command.Result, error) {
	joined := strings.Join(invocation.Args, " ")
	if invocation.Name == "psql" && strings.Contains(joined, "--single-transaction") {
		for _, argument := range invocation.Args {
			if !strings.HasPrefix(argument, "--file=") {
				continue
			}
			path := strings.TrimPrefix(argument, "--file=")
			contents, err := os.ReadFile(path)
			if err != nil {
				return command.Result{}, err
			}
			capture.transactionSQL = string(contents)
			capture.transactionCall = invocation
			break
		}
	}
	return capture.runner.Run(ctx, invocation)
}

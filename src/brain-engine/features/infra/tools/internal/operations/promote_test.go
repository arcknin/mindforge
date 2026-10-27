package operations_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/operations"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/testsupport"
)

func TestPromoteRequiresTheExplicitMainTarget(t *testing.T) {
	root, _ := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	request, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	_, err = operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               request,
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  testsupport.NewFakeRunner(),
	})
	if err == nil {
		t.Fatal("Promote() error = nil for a non-main target")
	}
}

func TestPromoteRejectsNSRDBAsExplicitTargetBeforeClientInvocation(t *testing.T) {
	root, _ := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               command.OperationRequest{Target: config.Target("nsrdb"), OperatorID: "operator-1", AuthorizationRef: "owner-promote-2026-09-30"},
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-30",
		Runner:                  fake,
	})
	if err == nil || len(fake.Calls()) != 0 {
		t.Fatalf("Promote() accepted nsrdb or used the runner: error=%v calls=%#v", err, fake.Calls())
	}
}

func TestPromoteOrdersDevelopEvidenceBackupMainRecheckAndIdenticalApply(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	outputDir := t.TempDir()
	fake := testsupport.NewFakeRunner()
	enqueueDevelopEvidenceWithDatabases(fake, hashes, verifiedDevelopEvidence(hashes), "mindforge,mindforge_develop,nsrdb")
	enqueueMainPreflightWithDatabases(fake, makeLedger(hashes), "mindforge,mindforge_develop,nsrdb")
	fake.Enqueue(command.Result{Stdout: "pg_dump (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "synthetic-main-backup"}, nil)
	fake.Enqueue(command.Result{Stdout: "List of archive contents\n"}, nil)
	enqueueMainPreflightWithDatabases(fake, makeLedger(hashes), "another_project,mindforge,mindforge_develop,nsrdb")
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "11111111-1111-4111-8111-111111111111\n"}, nil)
	fake.Enqueue(command.Result{}, nil)
	fake.Enqueue(command.Result{}, nil)

	result, err := operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               outputDir,
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err != nil {
		t.Fatalf("Promote(): %v", err)
	}
	if result.Status != "succeeded" || len(result.Applied) != 1 || result.Applied[0].ID != "0006" || result.Applied[0].Target != config.TargetMain {
		t.Fatalf("Promote() result = %#v, want one main promotion", result)
	}
	if result.MainPreflight.Reference != result.MainRecheck.Reference {
		t.Fatalf("unrelated database inventory changed the main preflight reference: %q != %q", result.MainPreflight.Reference, result.MainRecheck.Reference)
	}
	if result.Backup.Manifest.Target != string(config.TargetMain) || result.EvidencePath == "" {
		t.Fatalf("Promote() backup/evidence = %#v, want main-bound records", result)
	}
	evidenceBytes, err := os.ReadFile(result.EvidencePath)
	if err != nil {
		t.Fatalf("ReadFile(promotion evidence): %v", err)
	}
	if !strings.Contains(string(evidenceBytes), `"status": "succeeded"`) || !strings.Contains(string(evidenceBytes), `"backup_reference": "backup/mindforge/`) || !strings.Contains(string(evidenceBytes), `"migration_id": "0006"`) {
		t.Fatalf("promotion evidence = %s, want successful linked backup and migration", evidenceBytes)
	}
	calls := fake.Calls()
	if len(calls) != 18 {
		t.Fatalf("promotion calls = %d, want develop/main feature checks plus backup/recheck/atomic apply: %#v", len(calls), calls)
	}
	if calls[0].Name != "psql" || calls[2].Name != "psql" || calls[7].Name != "pg_dump" || calls[8].Name != "pg_restore" || calls[11].Name != "psql" || calls[14].Name != "psql" {
		t.Fatalf("promotion call order = %#v", calls)
	}
	if !strings.Contains(strings.Join(calls[16].Args, " "), "--file=") || !strings.Contains(strings.Join(calls[16].Args, " "), "--single-transaction") || strings.Contains(strings.Join(calls[16].Args, " "), "restore") {
		t.Fatalf("promotion did not apply the checked feature transaction: %#v", calls[16])
	}
	for index, call := range calls {
		joined := strings.Join(call.Args, " ")
		if call.Shell || strings.Contains(joined, "synthetic-main-password-123456") || strings.Contains(joined, "synthetic-develop-password-123456") {
			t.Fatalf("promotion call %d leaked shell/password: %#v", index, call)
		}
		if strings.Contains(joined, "--dbname=nsrdb") {
			t.Fatalf("promotion connected to unrelated database nsrdb: %#v", call)
		}
	}
	if _, err := os.Stat(filepath.Join(outputDir, filepath.Base(result.Backup.ManifestPath))); err != nil {
		t.Fatalf("backup manifest missing: %v", err)
	}
}

func TestPromoteRejectsMissingDevelopEvidenceBeforeMainPreflight(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	enqueueDevelopEvidence(fake, hashes, "")
	result, err := operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Promote() error = nil without verified develop evidence")
	}
	if result.Status != "failed" || result.EvidencePath == "" {
		t.Fatalf("Promote() result = %#v, want diagnosable failed evidence", result)
	}
	if len(fake.Calls()) != 4 {
		t.Fatalf("promote calls = %#v, want develop checks only", fake.Calls())
	}
	evidenceBytes, readErr := os.ReadFile(result.EvidencePath)
	if readErr != nil {
		t.Fatalf("ReadFile(promotion evidence): %v", readErr)
	}
	if !strings.Contains(string(evidenceBytes), `"error_class": "develop_evidence_invalid"`) {
		t.Fatalf("promotion evidence = %s, want sanitized develop failure", evidenceBytes)
	}
}

func TestPromoteRejectsChangedDevelopHashBeforeMainPreflight(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: promotionMetadata("mindforge_develop", makeLedger(hashes)+",0006:"+strings.Repeat("b", 64))}, nil)
	_, err = operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Promote() error = nil for changed develop source hash")
	}
	if len(fake.Calls()) != 2 {
		t.Fatalf("promote calls = %#v, want develop preflight only", fake.Calls())
	}
}

func TestPromoteRejectsMainDriftAfterBackupBeforeMutation(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	enqueueDevelopEvidence(fake, hashes, verifiedDevelopEvidence(hashes))
	enqueueMainPreflight(fake, makeLedger(hashes))
	fake.Enqueue(command.Result{Stdout: "pg_dump (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "synthetic-main-backup"}, nil)
	fake.Enqueue(command.Result{Stdout: "List of archive contents\n"}, nil)
	enqueueMainPreflight(fake, makeLedgerWithFeature(hashes))
	result, err := operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Promote() error = nil after main drift")
	}
	if result.Status != "failed" {
		t.Fatalf("Promote() result = %#v, want failed drift evidence", result)
	}
	for _, call := range fake.Calls() {
		joined := strings.Join(call.Args, " ")
		if strings.Contains(joined, "--single-transaction") || strings.Contains(joined, "INSERT INTO infra.migration_attempts") {
			t.Fatalf("main mutation started after recheck drift: %#v", call)
		}
	}
	evidenceBytes, readErr := os.ReadFile(result.EvidencePath)
	if readErr != nil {
		t.Fatalf("ReadFile(promotion evidence): %v", readErr)
	}
	if !strings.Contains(string(evidenceBytes), `"error_class": "main_drift"`) || !strings.Contains(string(evidenceBytes), `"backup_reference": "backup/mindforge/`) {
		t.Fatalf("promotion evidence = %s, want drift and backup references", evidenceBytes)
	}
}

func TestPromoteRejectsMissingVerificationReferenceBeforeClient(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Promote(context.Background(), operations.PromoteRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  t.TempDir(),
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Promote() error = nil without behavior/access verification reference")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("promote calls = %#v, want validation before client invocation", fake.Calls())
	}
}

func TestPromoteRejectsMissingExternalBackupDirectoryBeforeClient(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               filepath.Join(t.TempDir(), "missing"),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Promote() error = nil without an external backup directory")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("promote calls = %#v, want destination validation before client invocation", fake.Calls())
	}
}

func TestPromoteRejectsMainPreflightBeforeBackup(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	enqueueDevelopEvidence(fake, hashes, verifiedDevelopEvidence(hashes))
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "not-a-target|mindforge,mindforge_develop|0.8.6|infra|infra.migration_ledger|infra.migration_target_evidence|infra.migration_attempts|auth|auth.users|auth.identities|auth.schema_migrations|1|" + makeLedger(hashes) + "\n"}, nil)
	result, err := operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Promote() error = nil for failed main preflight")
	}
	if result.Status != "failed" {
		t.Fatalf("Promote() result = %#v, want failed evidence", result)
	}
	for _, call := range fake.Calls() {
		if call.Name == "pg_dump" || call.Name == "pg_restore" {
			t.Fatalf("backup started after failed main preflight: %#v", call)
		}
	}
}

func TestPromoteRejectsBackupFailureBeforeMainMutation(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	enqueueDevelopEvidence(fake, hashes, verifiedDevelopEvidence(hashes))
	enqueueMainPreflight(fake, makeLedger(hashes))
	fake.Enqueue(command.Result{ExitCode: 127}, os.ErrNotExist)
	result, err := operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Promote() error = nil when main backup utility is unavailable")
	}
	if result.Status != "failed" {
		t.Fatalf("Promote() result = %#v, want failed backup evidence", result)
	}
	for _, call := range fake.Calls() {
		joined := strings.Join(call.Args, " ")
		if strings.Contains(joined, "--single-transaction") || strings.Contains(joined, "INSERT INTO infra.migration_attempts") {
			t.Fatalf("main mutation started after backup failure: %#v", call)
		}
	}
}

func TestPromoteAcceptsIdenticalAlreadyAppliedHashIdempotently(t *testing.T) {
	t.Parallel()

	root, hashes := writeCatalog(t, true)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-promote-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	enqueueDevelopEvidence(fake, hashes, verifiedDevelopEvidence(hashes))
	enqueueMainPreflight(fake, makeLedgerWithFeature(hashes))
	fake.Enqueue(command.Result{Stdout: "pg_dump (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "synthetic-main-backup"}, nil)
	fake.Enqueue(command.Result{Stdout: "List of archive contents\n"}, nil)
	enqueueMainPreflight(fake, makeLedgerWithFeature(hashes))
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	result, err := operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		OutputDir:               t.TempDir(),
		BehaviorVerificationRef: "verification/feature-2026-09-19",
		Runner:                  fake,
	})
	if err != nil {
		t.Fatalf("Promote(): %v", err)
	}
	if result.Status != "succeeded" || len(result.Applied) != 0 {
		t.Fatalf("Promote() result = %#v, want idempotent no-op success", result)
	}
	evidenceBytes, err := os.ReadFile(result.EvidencePath)
	if err != nil {
		t.Fatalf("ReadFile(promotion evidence): %v", err)
	}
	if !strings.Contains(string(evidenceBytes), `"status": "already_applied"`) {
		t.Fatalf("promotion evidence = %s, want identical-hash replay marker", evidenceBytes)
	}
	for _, call := range fake.Calls() {
		joined := strings.Join(call.Args, " ")
		if strings.Contains(joined, "--single-transaction") || strings.Contains(joined, "INSERT INTO infra.migration_attempts") {
			t.Fatalf("idempotent promotion mutated main: %#v", call)
		}
	}
}

func promotionMetadata(target, ledger string) string {
	return promotionMetadataWithDatabases(target, ledger, "mindforge,mindforge_develop")
}

func promotionMetadataWithDatabases(target, ledger, databases string) string {
	return fmt.Sprintf("%s|%s|0.8.6|infra|infra.migration_ledger|infra.migration_target_evidence|infra.migration_attempts|auth|auth.users|auth.identities|auth.schema_migrations|1|%s\n", target, databases, ledger)
}

func makeLedgerWithFeature(hashes map[string]string) string {
	return makeLedger(hashes) + ",0006:" + hashes["0006"]
}

func verifiedDevelopEvidence(hashes map[string]string) string {
	return fmt.Sprintf("0006|%s|mindforge_develop|preflight/mindforge_develop/%s|verification/feature-2026-09-19|succeeded|succeeded\n", hashes["0006"], strings.Repeat("a", 64))
}

func enqueueDevelopEvidence(fake *testsupport.FakeRunner, hashes map[string]string, evidenceOutput string) {
	enqueueDevelopEvidenceWithDatabases(fake, hashes, evidenceOutput, "mindforge,mindforge_develop")
}

func enqueueDevelopEvidenceWithDatabases(fake *testsupport.FakeRunner, hashes map[string]string, evidenceOutput, databases string) {
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: promotionMetadataWithDatabases("mindforge_develop", makeLedgerWithFeature(hashes), databases)}, nil)
	fake.Enqueue(command.Result{Stdout: "complete\n"}, nil)
	fake.Enqueue(command.Result{Stdout: evidenceOutput}, nil)
}

func enqueueMainPreflight(fake *testsupport.FakeRunner, ledger string) {
	enqueueMainPreflightWithDatabases(fake, ledger, "mindforge,mindforge_develop")
}

func enqueueMainPreflightWithDatabases(fake *testsupport.FakeRunner, ledger, databases string) {
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: promotionMetadataWithDatabases("mindforge", ledger, databases)}, nil)
	state := "absent_eligible\n"
	if strings.Contains(ledger, "0006:") {
		state = "complete\n"
	}
	fake.Enqueue(command.Result{Stdout: state}, nil)
}

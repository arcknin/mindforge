package operations_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/evidence"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/operations"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/testsupport"
)

func TestBackupPublishesPrivateArtifactAndEnvironmentManifestAfterChecks(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	outputDir := t.TempDir()
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "pg_dump (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "synthetic-custom-archive"}, nil)
	fake.Enqueue(command.Result{Stdout: "List of archive contents\n"}, nil)

	result, err := operations.Backup(context.Background(), operations.BackupRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  outputDir,
		Runner:     fake,
	})
	if err != nil {
		t.Fatalf("Backup(): %v", err)
	}
	if result.Target != config.TargetDevelop || result.ArtifactPath == "" || result.ManifestPath == "" {
		t.Fatalf("Backup() result = %#v, want explicit target and two artifacts", result)
	}
	for _, path := range []string{result.ArtifactPath, result.ManifestPath} {
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("Stat(%s): %v", path, statErr)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o, want 600", path, info.Mode().Perm())
		}
	}
	manifest, err := evidence.ReadBackupManifest(result.ManifestPath)
	if err != nil {
		t.Fatalf("ReadBackupManifest(): %v", err)
	}
	if manifest.Target != string(config.TargetDevelop) || manifest.Database != string(config.TargetDevelop) || !manifest.ArchiveListingVerified || manifest.RestoreProof != "not_verified" {
		t.Fatalf("backup manifest = %#v, want target-bound validated backup", manifest)
	}
	if manifest.ArtifactName != filepath.Base(result.ArtifactPath) || manifest.ArtifactSHA256 != result.SHA256 || manifest.OperatorID != "operator-1" {
		t.Fatalf("backup manifest provenance = %#v, want sanitized target evidence", manifest)
	}
	calls := fake.Calls()
	if len(calls) != 4 || calls[0].Name != "pg_dump" || calls[2].Name != "pg_dump" || calls[3].Name != "pg_restore" {
		t.Fatalf("backup calls = %#v, want dump version, restore version, dump, list", calls)
	}
	for index, call := range calls {
		if call.Shell || strings.Contains(strings.Join(call.Args, " "), "synthetic-develop-password-123456") {
			t.Fatalf("backup call %d leaked shell/password: %#v", index, call)
		}
	}
	if !strings.Contains(strings.Join(calls[2].Env, " "), "PGPASSWORD=synthetic-develop-password-123456") {
		t.Fatalf("pg_dump did not receive password through child environment: %#v", calls[2])
	}
}

func TestBackupLeavesNoPublishedArtifactAfterArchiveValidationFailure(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	outputDir := t.TempDir()
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-backup-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "pg_dump (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "not-an-archive"}, nil)
	fake.Enqueue(command.Result{ExitCode: 1}, os.ErrInvalid)

	_, err = operations.Backup(context.Background(), operations.BackupRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  outputDir,
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Backup() error = nil after archive listing failure")
	}
	if files, globErr := filepath.Glob(filepath.Join(outputDir, "*")); globErr != nil || len(files) != 0 {
		t.Fatalf("published backup files = %#v, want none", files)
	}
}

func TestBackupFailsBeforeClientWhenOutputDirectoryIsMissing(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Backup(context.Background(), operations.BackupRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  filepath.Join(t.TempDir(), "missing"),
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Backup() error = nil for missing output directory")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("backup calls = %#v, want output validation before client invocation", fake.Calls())
	}
}

func TestBackupRejectsOutputInsideCheckoutBeforeClient(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Backup(context.Background(), operations.BackupRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  root,
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Backup() error = nil for an output directory inside the checkout")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("backup calls = %#v, want checkout-bound output rejection before client invocation", fake.Calls())
	}
}

func TestBackupRejectsMissingClientBeforeCreatingOutput(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	outputDir := t.TempDir()
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-1", "owner-backup-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{ExitCode: 127}, os.ErrNotExist)
	_, err = operations.Backup(context.Background(), operations.BackupRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  outputDir,
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Backup() error = nil when pg_dump is unavailable")
	}
	if files, globErr := filepath.Glob(filepath.Join(outputDir, "*")); globErr != nil || len(files) != 0 {
		t.Fatalf("published backup files = %#v, want none", files)
	}
}

func TestBackupRejectsMissingArchiveValidatorBeforeDump(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	outputDir := t.TempDir()
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "pg_dump (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{ExitCode: 127}, os.ErrNotExist)
	_, err = operations.Backup(context.Background(), operations.BackupRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  outputDir,
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Backup() error = nil when pg_restore is unavailable")
	}
	if len(fake.Calls()) != 2 {
		t.Fatalf("backup calls = %#v, want only utility checks before dump", fake.Calls())
	}
	if files, globErr := filepath.Glob(filepath.Join(outputDir, "*")); globErr != nil || len(files) != 0 {
		t.Fatalf("published backup files = %#v, want none", files)
	}
}

func TestBackupLeavesNoPartialArtifactAfterDumpFailure(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	outputDir := t.TempDir()
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "pg_dump (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "partial-archive"}, os.ErrInvalid)
	_, err = operations.Backup(context.Background(), operations.BackupRequest{
		Config:     settings,
		Operation:  operation,
		SourceRoot: root,
		OutputDir:  outputDir,
		Runner:     fake,
	})
	if err == nil {
		t.Fatal("Backup() error = nil after a partial dump failure")
	}
	if files, globErr := filepath.Glob(filepath.Join(outputDir, "*")); globErr != nil || len(files) != 0 {
		t.Fatalf("published backup files = %#v, want none", files)
	}
}

func TestRestoreRejectsHashMismatchBeforeAnyClientMutation(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	backupDir := t.TempDir()
	artifactPath := filepath.Join(backupDir, "mindforge-infra-mindforge_develop-20260919T000000000Z.dump")
	manifestPath := artifactPath + ".manifest.json"
	if err := os.WriteFile(artifactPath, []byte("archive-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile(artifact): %v", err)
	}
	digest := sha256.Sum256([]byte("different-bytes"))
	manifest, err := evidence.NewBackupManifest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19", filepath.Base(artifactPath), hex.EncodeToString(digest[:]), int64(len("archive-bytes")), "2026-09-19T00:00:00Z")
	if err != nil {
		t.Fatalf("NewBackupManifest(): %v", err)
	}
	if err := evidence.WriteBackupManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteBackupManifest(): %v", err)
	}
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-restore", "owner-restore-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Restore(context.Background(), operations.RestoreRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		BackupFile:              artifactPath,
		ManifestFile:            manifestPath,
		ReplacementConfirmation: "replace/mindforge_develop/owner-confirmation-2026",
		EvidenceDir:             backupDir,
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Restore() error = nil for manifest hash mismatch")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("restore calls = %#v, want no client invocation before hash validation", fake.Calls())
	}
}

func TestRestoreRejectsWrongEnvironmentManifestBeforeAnyClientMutation(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	backupDir := t.TempDir()
	artifactPath := filepath.Join(backupDir, "mindforge-infra-mindforge-20260919T000000000Z.dump")
	manifestPath := artifactPath + ".manifest.json"
	archive := []byte("archive-bytes")
	if err := os.WriteFile(artifactPath, archive, 0o600); err != nil {
		t.Fatalf("WriteFile(artifact): %v", err)
	}
	digest := sha256.Sum256(archive)
	manifest, err := evidence.NewBackupManifest(config.TargetMain, "operator-1", "owner-backup-2026-09-19", filepath.Base(artifactPath), hex.EncodeToString(digest[:]), int64(len(archive)), "2026-09-19T00:00:00Z")
	if err != nil {
		t.Fatalf("NewBackupManifest(): %v", err)
	}
	if err := evidence.WriteBackupManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteBackupManifest(): %v", err)
	}
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-restore", "owner-restore-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Restore(context.Background(), operations.RestoreRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		BackupFile:              artifactPath,
		ManifestFile:            manifestPath,
		ReplacementConfirmation: "replace/mindforge_develop/owner-confirmation-2026",
		EvidenceDir:             backupDir,
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Restore() error = nil for a manifest from another environment")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("restore calls = %#v, want no client invocation before target compatibility validation", fake.Calls())
	}
}

func TestRestoreRejectsUnapprovedReplacementBeforeClientMutation(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	backupDir := t.TempDir()
	artifactPath := filepath.Join(backupDir, "mindforge-infra-mindforge_develop-20260919T000000000Z.dump")
	manifestPath := artifactPath + ".manifest.json"
	archive := []byte("archive-bytes")
	if err := os.WriteFile(artifactPath, archive, 0o600); err != nil {
		t.Fatalf("WriteFile(artifact): %v", err)
	}
	digest := sha256.Sum256(archive)
	manifest, err := evidence.NewBackupManifest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19", filepath.Base(artifactPath), hex.EncodeToString(digest[:]), int64(len(archive)), "2026-09-19T00:00:00Z")
	if err != nil {
		t.Fatalf("NewBackupManifest(): %v", err)
	}
	if err := evidence.WriteBackupManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteBackupManifest(): %v", err)
	}
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-restore", "owner-restore-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	_, err = operations.Restore(context.Background(), operations.RestoreRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		BackupFile:              artifactPath,
		ManifestFile:            manifestPath,
		ReplacementConfirmation: "replace/mindforge/owner-confirmation-2026",
		EvidenceDir:             backupDir,
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Restore() error = nil for an unapproved replacement")
	}
	if len(fake.Calls()) != 0 {
		t.Fatalf("restore calls = %#v, want no client invocation before replacement authorization", fake.Calls())
	}
}

func TestRestorePreflightsDestinationBeforeMutableRestoreAndRecordsFailure(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	backupDir := t.TempDir()
	artifactPath := filepath.Join(backupDir, "mindforge-infra-mindforge_develop-20260919T000000000Z.dump")
	manifestPath := artifactPath + ".manifest.json"
	archive := []byte("archive-bytes")
	if err := os.WriteFile(artifactPath, archive, 0o600); err != nil {
		t.Fatalf("WriteFile(artifact): %v", err)
	}
	digest := sha256.Sum256(archive)
	manifest, err := evidence.NewBackupManifest(config.TargetDevelop, "operator-1", "owner-backup-2026-09-19", filepath.Base(artifactPath), hex.EncodeToString(digest[:]), int64(len(archive)), "2026-09-19T00:00:00Z")
	if err != nil {
		t.Fatalf("NewBackupManifest(): %v", err)
	}
	if err := evidence.WriteBackupManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteBackupManifest(): %v", err)
	}
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetDevelop, "operator-restore", "owner-restore-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "List of archive contents\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge\n"}, nil)

	result, err := operations.Restore(context.Background(), operations.RestoreRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		BackupFile:              artifactPath,
		ManifestFile:            manifestPath,
		ReplacementConfirmation: "replace/mindforge_develop/owner-confirmation-2026",
		EvidenceDir:             backupDir,
		Runner:                  fake,
	})
	if err == nil {
		t.Fatal("Restore() error = nil when destination identity mismatched")
	}
	if result.EvidencePath == "" {
		t.Fatalf("Restore() result = %#v, want failed evidence path", result)
	}
	evidenceBytes, readErr := os.ReadFile(result.EvidencePath)
	if readErr != nil {
		t.Fatalf("ReadFile(restore evidence): %v", readErr)
	}
	if !strings.Contains(string(evidenceBytes), `"status": "failed"`) || !strings.Contains(string(evidenceBytes), `"destination_verified": false`) {
		t.Fatalf("restore evidence = %s, want sanitized failed preflight", evidenceBytes)
	}
	for _, call := range fake.Calls() {
		if strings.Contains(strings.Join(call.Args, " "), "--single-transaction") {
			t.Fatalf("mutable restore was invoked before destination preflight: %#v", call)
		}
	}
}

func TestRestorePublishesSucceededOutcomeOnlyAfterDestinationPreflight(t *testing.T) {
	t.Parallel()

	root, _ := writeCatalog(t, false)
	backupDir := t.TempDir()
	artifactPath := filepath.Join(backupDir, "mindforge-infra-mindforge-20260919T000000000Z.dump")
	manifestPath := artifactPath + ".manifest.json"
	archive := []byte("archive-bytes")
	if err := os.WriteFile(artifactPath, archive, 0o600); err != nil {
		t.Fatalf("WriteFile(artifact): %v", err)
	}
	digest := sha256.Sum256(archive)
	manifest, err := evidence.NewBackupManifest(config.TargetMain, "operator-1", "owner-backup-2026-09-19", filepath.Base(artifactPath), hex.EncodeToString(digest[:]), int64(len(archive)), "2026-09-19T00:00:00Z")
	if err != nil {
		t.Fatalf("NewBackupManifest(): %v", err)
	}
	if err := evidence.WriteBackupManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteBackupManifest(): %v", err)
	}
	settings, err := config.ParseFile(writePrivateConfig(t))
	if err != nil {
		t.Fatalf("ParseFile(): %v", err)
	}
	operation, err := command.NewOperationRequest(config.TargetMain, "operator-restore", "owner-restore-2026-09-19")
	if err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "pg_restore (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "List of archive contents\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "psql (PostgreSQL) 16.15\n"}, nil)
	fake.Enqueue(command.Result{Stdout: "mindforge\n"}, nil)
	fake.Enqueue(command.Result{}, nil)

	result, err := operations.Restore(context.Background(), operations.RestoreRequest{
		Config:                  settings,
		Operation:               operation,
		SourceRoot:              root,
		BackupFile:              artifactPath,
		ManifestFile:            manifestPath,
		ReplacementConfirmation: "replace/mindforge/owner-confirmation-2026",
		EvidenceDir:             backupDir,
		Runner:                  fake,
	})
	if err != nil {
		t.Fatalf("Restore(): %v", err)
	}
	if result.Status != "succeeded" || result.EvidencePath == "" {
		t.Fatalf("Restore() result = %#v, want succeeded evidence", result)
	}
	evidenceBytes, readErr := os.ReadFile(result.EvidencePath)
	if readErr != nil {
		t.Fatalf("ReadFile(restore evidence): %v", readErr)
	}
	if !strings.Contains(string(evidenceBytes), `"status": "succeeded"`) || !strings.Contains(string(evidenceBytes), `"destination_verified": true`) || !strings.Contains(string(evidenceBytes), `"restore_proof": "not_claimed"`) {
		t.Fatalf("restore evidence = %s, want succeeded outcome without proof claim", evidenceBytes)
	}
	calls := fake.Calls()
	if len(calls) != 5 || calls[4].Name != "pg_restore" || !strings.Contains(strings.Join(calls[4].Args, " "), "--single-transaction") || !strings.Contains(strings.Join(calls[4].Args, " "), "--clean") {
		t.Fatalf("restore calls = %#v, want read-only checks then explicit transactional restore", calls)
	}
	for _, call := range calls {
		if strings.Contains(strings.Join(call.Args, " "), "synthetic-main-password-123456") {
			t.Fatalf("restore leaked password in arguments: %#v", call)
		}
	}
}

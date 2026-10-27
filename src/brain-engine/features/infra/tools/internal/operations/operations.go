// Package operations implements the guarded, fake-testable database
// operations for the infrastructure operator. It never invokes a shell and
// it never chooses a target implicitly.
package operations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/evidence"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
)

const supportedPostgresMajor = "16"

// Scope chooses an immutable source catalog. Infrastructure migrations may be
// applied to either contracted target; feature migrations are develop-only.
type Scope string

const (
	ScopeInfrastructure Scope = "infrastructure"
	ScopeFeature        Scope = "feature"
)

// PreflightRequest describes a read-only target inspection.
type PreflightRequest struct {
	Config     config.Values
	Operation  command.OperationRequest
	Scope      Scope
	SourceRoot string
	Runner     command.Runner

	// AllowInfrastructureBootstrap permits a first infrastructure application
	// when the ledger/Auth/vector state is not present yet. It is only valid for
	// ScopeInfrastructure and does not weaken feature application preflight.
	AllowInfrastructureBootstrap bool
}

// ProvisionDevelopRequest describes the one explicit additive operation that
// may create the isolated develop role/database on an existing cluster.
type ProvisionDevelopRequest struct {
	Config    config.Values
	Operation command.OperationRequest
	Runner    command.Runner
}

// ProvisionDevelopResult contains only a sanitized target and outcome.
type ProvisionDevelopResult struct {
	Target config.Target
	Status string
}

// ApplyRequest describes one target-specific migration application.
type ApplyRequest struct {
	Config                  config.Values
	Operation               command.OperationRequest
	Scope                   Scope
	SourceRoot              string
	Runner                  command.Runner
	Preflight               evidence.Preflight
	BehaviorVerificationRef string
}

// ApplyResult contains only sanitized target and migration identity.
type ApplyResult struct {
	Applied []evidence.AppliedMigration
}

// BackupRequest describes one external, target-bound backup check.
type BackupRequest struct {
	Config     config.Values
	Operation  command.OperationRequest
	SourceRoot string
	OutputDir  string
	Runner     command.Runner
}

// BackupResult contains only portable artifact identity and sanitized
// target-bound evidence.
type BackupResult struct {
	Target       config.Target
	ArtifactPath string
	ManifestPath string
	SHA256       string
	Manifest     evidence.BackupManifest
}

// RestoreRequest describes one explicitly identified recovery destination.
type RestoreRequest struct {
	Config                  config.Values
	Operation               command.OperationRequest
	SourceRoot              string
	BackupFile              string
	ManifestFile            string
	ReplacementConfirmation string
	EvidenceDir             string
	Runner                  command.Runner
}

// RestoreResult contains the sanitized result record. RestoreProof remains
// deliberately unclaimed in the evidence manifest for this revision.
type RestoreResult struct {
	Target       config.Target
	ArtifactPath string
	ManifestPath string
	EvidencePath string
	Status       string
}

// PromoteRequest describes one immutable develop-first promotion into main.
// The operation reads develop evidence, creates a fresh main backup, rechecks
// main, and only then applies the same local feature migration bytes.
type PromoteRequest struct {
	Config                  config.Values
	Operation               command.OperationRequest
	SourceRoot              string
	OutputDir               string
	BehaviorVerificationRef string
	Runner                  command.Runner
}

// PromoteResult contains sanitized promotion references and applied IDs.
type PromoteResult struct {
	Target           config.Target
	DevelopPreflight evidence.Preflight
	MainPreflight    evidence.Preflight
	MainRecheck      evidence.Preflight
	Backup           BackupResult
	Applied          []evidence.AppliedMigration
	EvidencePath     string
	Status           string
}

// Preflight performs a client-version check and one read-only PostgreSQL
// catalog query. It emits a deterministic sanitized reference and never sends
// DDL or evidence mutations.
func Preflight(ctx context.Context, request PreflightRequest) (evidence.Preflight, error) {
	if request.Runner == nil {
		return evidence.Preflight{}, fmt.Errorf("a command runner is required")
	}
	if _, err := command.NewOperationRequest(request.Operation.Target, request.Operation.OperatorID, request.Operation.AuthorizationRef); err != nil {
		return evidence.Preflight{}, err
	}
	if request.AllowInfrastructureBootstrap && request.Scope != ScopeInfrastructure {
		return evidence.Preflight{}, fmt.Errorf("infrastructure bootstrap is limited to infrastructure scope")
	}
	settings, err := request.Config.Settings(request.Operation.Target)
	if err != nil {
		return evidence.Preflight{}, err
	}
	repositoryRoot, err := resolveRepositoryRoot(request.SourceRoot)
	if err != nil {
		return evidence.Preflight{}, err
	}
	catalog, err := migrations.LoadCatalog(repositoryRoot)
	if err != nil {
		return evidence.Preflight{}, fmt.Errorf("load migration catalog: %w", err)
	}
	if err := requirePostgresClient(ctx, request.Runner); err != nil {
		return evidence.Preflight{}, err
	}
	result, err := runPSQL(ctx, request.Runner, settings, metadataQuery)
	if err != nil {
		return evidence.Preflight{}, fmt.Errorf("read-only preflight query failed")
	}
	metadata, err := parseMetadata(result.Stdout)
	if err != nil {
		return evidence.Preflight{}, err
	}
	if metadata.Database != string(request.Operation.Target) {
		return evidence.Preflight{}, fmt.Errorf("preflight database does not match explicit target")
	}
	if !databaseInventoryContains(metadata.Databases, string(request.Operation.Target)) {
		return evidence.Preflight{}, fmt.Errorf("preflight target is absent from the cluster inventory")
	}
	ledger, err := parseLedger(metadata.Ledger)
	if err != nil {
		return evidence.Preflight{}, err
	}
	if err := migrations.VerifyLedger(catalog, ledger); err != nil {
		return evidence.Preflight{}, fmt.Errorf("preflight migration ledger is not canonical")
	}
	infrastructureReady, err := verifyInfrastructureState(catalog, metadata, ledger)
	if err != nil && !request.AllowInfrastructureBootstrap {
		return evidence.Preflight{}, err
	}
	if request.AllowInfrastructureBootstrap && !infrastructureReady {
		// A bootstrap operation may start from an empty or partial infrastructure
		// catalog, but it still requires the exact cluster and target database.
		if len(ledger) > 0 {
			if err := verifyKnownInfrastructureRows(catalog, ledger); err != nil {
				return evidence.Preflight{}, err
			}
		}
	}
	observedState := metadata.canonicalState()
	if request.Scope == ScopeFeature {
		states, err := verifyFeaturePreflight(ctx, request.Runner, settings, catalog, ledger)
		if err != nil {
			return evidence.Preflight{}, err
		}
		for _, migration := range selectMigrations(catalog, ScopeFeature) {
			observedState += "|feature-state:" + migration.ID + "=" + states[migration.ID]
		}
	}
	reference, err := evidence.NewReference(request.Operation.Target, migrations.CatalogHash(catalog), observedState)
	if err != nil {
		return evidence.Preflight{}, err
	}
	return evidence.Preflight{
		Target:              request.Operation.Target,
		Reference:           reference,
		SourceRoot:          catalog.Root,
		CatalogHash:         migrations.CatalogHash(catalog),
		ObservedState:       observedState,
		Ledger:              ledger,
		InfrastructureReady: infrastructureReady,
	}, nil
}

// ProvisionDevelop performs a read-only cluster inventory twice before it
// issues the only permitted initialized-cluster database additions: creation
// of the configured develop login role and mindforge_develop database. It
// never mutates or copies the existing mindforge database.
func ProvisionDevelop(ctx context.Context, request ProvisionDevelopRequest) (ProvisionDevelopResult, error) {
	result := ProvisionDevelopResult{Target: config.TargetDevelop, Status: "failed"}
	if request.Runner == nil {
		return result, fmt.Errorf("a command runner is required")
	}
	if request.Operation.Target != config.TargetDevelop {
		return result, fmt.Errorf("develop provisioning requires explicit target mindforge_develop")
	}
	if _, err := command.NewOperationRequest(request.Operation.Target, request.Operation.OperatorID, request.Operation.AuthorizationRef); err != nil {
		return result, err
	}
	admin, err := request.Config.ClusterAdmin()
	if err != nil {
		return result, err
	}
	develop, err := request.Config.Settings(config.TargetDevelop)
	if err != nil {
		return result, err
	}
	main, err := request.Config.Settings(config.TargetMain)
	if err != nil {
		return result, err
	}
	if err := requirePostgresClient(ctx, request.Runner); err != nil {
		return result, err
	}
	query := provisionInventoryQuery(main.DatabaseUser, develop.DatabaseUser)
	initial, err := readProvisionInventory(ctx, request.Runner, admin, query)
	if err != nil {
		return result, err
	}
	if err := validateProvisionInventory(initial, main.DatabaseUser); err != nil {
		return result, err
	}
	rechecked, err := readProvisionInventory(ctx, request.Runner, admin, query)
	if err != nil {
		return result, err
	}
	if err := validateProvisionInventory(rechecked, main.DatabaseUser); err != nil {
		return result, err
	}
	if !sameMindforgeProvisionState(initial, rechecked) {
		return result, fmt.Errorf("cluster topology changed during read-only develop provisioning checks")
	}
	filePath, err := writeProvisionSQL(develop.DatabaseUser, develop.DatabasePassword)
	if err != nil {
		return result, fmt.Errorf("prepare private develop provisioning statement")
	}
	ddlResult, ddlErr := runClusterPSQLFile(ctx, request.Runner, admin, filePath)
	removeErr := os.Remove(filePath)
	if ddlErr != nil || ddlResult.ExitCode != 0 {
		result.Status = "incomplete"
		return result, fmt.Errorf("develop provisioning is incomplete; inspect the cluster before retrying")
	}
	if removeErr != nil {
		result.Status = "incomplete"
		return result, fmt.Errorf("develop provisioning completed but private temporary-file cleanup failed")
	}
	result.Status = "succeeded"
	return result, nil
}

// Apply executes a target-specific migration sequence after a previously
// completed read-only preflight. Every attempt is recorded before its DDL and
// evidence is updated independently from the migration transaction.
func Apply(ctx context.Context, request ApplyRequest) (ApplyResult, error) {
	return applyMigrations(ctx, request, false)
}

// applyMigrations contains the shared immutable migration/evidence sequence.
// The promotion path is the only caller allowed to opt into feature SQL on
// main; the public apply operation remains develop-only for feature scope.
func applyMigrations(ctx context.Context, request ApplyRequest, allowPromotion bool) (ApplyResult, error) {
	if request.Runner == nil {
		return ApplyResult{}, fmt.Errorf("a command runner is required")
	}
	if request.Scope != ScopeInfrastructure && request.Scope != ScopeFeature {
		return ApplyResult{}, fmt.Errorf("an explicit migration scope is required")
	}
	if request.Scope == ScopeFeature && request.Operation.Target != config.TargetDevelop && !(allowPromotion && request.Operation.Target == config.TargetMain) {
		return ApplyResult{}, fmt.Errorf("feature migrations may target only mindforge_develop")
	}
	if _, err := command.NewOperationRequest(request.Operation.Target, request.Operation.OperatorID, request.Operation.AuthorizationRef); err != nil {
		return ApplyResult{}, err
	}
	if err := evidence.ValidateReference(request.Preflight.Reference); err != nil {
		return ApplyResult{}, err
	}
	if request.Preflight.SourceRoot == "" || request.Preflight.CatalogHash == "" || request.Preflight.ObservedState == "" {
		return ApplyResult{}, fmt.Errorf("complete read-only preflight evidence is required")
	}
	if request.Preflight.Target != request.Operation.Target {
		return ApplyResult{}, fmt.Errorf("preflight reference target does not match explicit operation target")
	}
	if err := evidence.ValidateVerificationReference(request.BehaviorVerificationRef); err != nil {
		return ApplyResult{}, err
	}
	settings, err := request.Config.Settings(request.Operation.Target)
	if err != nil {
		return ApplyResult{}, err
	}
	repositoryRoot, err := resolveRepositoryRoot(request.SourceRoot)
	if err != nil {
		return ApplyResult{}, err
	}
	catalog, err := migrations.LoadCatalog(repositoryRoot)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("load migration catalog: %w", err)
	}
	if request.Preflight.SourceRoot != "" && request.Preflight.SourceRoot != catalog.Root {
		return ApplyResult{}, fmt.Errorf("preflight source root does not match migration source")
	}
	if request.Preflight.CatalogHash != "" && request.Preflight.CatalogHash != migrations.CatalogHash(catalog) {
		return ApplyResult{}, fmt.Errorf("migration source changed after preflight")
	}
	if err := verifyKnownInfrastructureRows(catalog, request.Preflight.Ledger); err != nil {
		return ApplyResult{}, err
	}
	if request.Scope == ScopeFeature && !request.Preflight.InfrastructureReady {
		return ApplyResult{}, fmt.Errorf("feature application requires complete infrastructure preflight")
	}
	if err := requirePostgresClient(ctx, request.Runner); err != nil {
		return ApplyResult{}, err
	}
	if request.Scope == ScopeInfrastructure && !request.Preflight.InfrastructureReady && len(request.Preflight.Ledger) == 0 {
		if _, err := runPSQLTransaction(ctx, request.Runner, settings, ledgerBootstrapSQL); err != nil {
			return ApplyResult{}, fmt.Errorf("infrastructure ledger bootstrap failed")
		}
	}

	selected := selectMigrations(catalog, request.Scope)
	if len(selected) == 0 {
		return ApplyResult{}, fmt.Errorf("migration source is empty for the requested scope")
	}
	result := ApplyResult{Applied: make([]evidence.AppliedMigration, 0, len(selected))}
	recordEvidence := request.Preflight.InfrastructureReady || request.Scope == ScopeFeature
	for _, migration := range selected {
		if request.Scope == ScopeFeature {
			state, err := preflightStateFor(request.Preflight.ObservedState, migration.ID)
			if err != nil {
				return result, err
			}
			if existing, applied := request.Preflight.Ledger[migration.ID]; applied {
				if existing != migration.SHA256 || state != "complete" {
					return result, fmt.Errorf("applied feature migration %s does not have its complete canonical schema", migration.ID)
				}
			} else if state != "absent_eligible" {
				return result, fmt.Errorf("feature migration %s requires an absent eligible schema", migration.ID)
			}
		}
		if existing, ok := request.Preflight.Ledger[migration.ID]; ok {
			if existing != migration.SHA256 {
				return result, fmt.Errorf("applied migration %s has changed bytes", migration.ID)
			}
			continue
		}
		if !recordEvidence {
			migrationResult, err := request.Runner.Run(ctx, migrationInvocation(settings, migration))
			if err != nil || migrationResult.ExitCode != 0 {
				return result, fmt.Errorf("migration %s failed during infrastructure bootstrap", migration.ID)
			}
			result.Applied = append(result.Applied, evidence.AppliedMigration{
				ID:     migration.ID,
				SHA256: migration.SHA256,
				Target: request.Operation.Target,
			})
			continue
		}
		attemptSQL, err := evidence.InsertAttemptSQL(
			migration.ID,
			migration.SHA256,
			request.Operation.Target,
			request.Operation.OperatorID,
			request.Operation.AuthorizationRef,
			request.Preflight.Reference,
			request.BehaviorVerificationRef,
		)

		if err != nil {
			return result, err
		}
		attemptResult, err := runPSQL(ctx, request.Runner, settings, attemptSQL)
		if err != nil {
			return result, fmt.Errorf("failed to record migration attempt")
		}
		attemptID, err := parseAttemptID(attemptResult.Stdout)
		if err != nil {
			return result, err
		}
		migrationSQL := migrationInvocation(settings, migration)
		cleanupMigrationSQL := func() error { return nil }
		if migration.Kind == migrations.KindFeature {
			transactionPath, cleanup, prepareErr := prepareFeatureMigrationTransaction(migration)
			if prepareErr != nil {
				if failureErr := markFailed(ctx, request.Runner, settings, attemptID); failureErr != nil {
					return result, fmt.Errorf("migration %s source was unsafe and failure evidence could not be recorded", migration.ID)
				}
				return result, fmt.Errorf("migration %s source or checks changed after preflight; failure evidence was recorded", migration.ID)
			}
			migrationSQL = migrationFileInvocation(settings, migration, transactionPath)
			cleanupMigrationSQL = cleanup
		}
		migrationResult, err := request.Runner.Run(ctx, migrationSQL)
		cleanupErr := cleanupMigrationSQL()
		if err != nil || migrationResult.ExitCode != 0 {
			if failureErr := markFailed(ctx, request.Runner, settings, attemptID); failureErr != nil {
				return result, fmt.Errorf("migration %s failed and failure evidence could not be recorded", migration.ID)
			}
			return result, fmt.Errorf("migration %s failed; failure evidence was recorded", migration.ID)
		}
		if cleanupErr != nil {
			return result, fmt.Errorf("migration %s completed but its private transaction file could not be removed", migration.ID)
		}
		completionSQL, err := evidence.CompleteAttemptSQL(attemptID)
		if err != nil {
			return result, err
		}
		if _, err := runPSQL(ctx, request.Runner, settings, completionSQL); err != nil {
			return result, fmt.Errorf("migration %s completed without target evidence", migration.ID)
		}
		result.Applied = append(result.Applied, evidence.AppliedMigration{
			ID:      migration.ID,
			SHA256:  migration.SHA256,
			Target:  request.Operation.Target,
			Attempt: attemptID,
		})
	}
	return result, nil
}

// Backup creates and validates a custom-format dump before atomically
// publishing its private artifact and environment-bound manifest.
func Backup(ctx context.Context, request BackupRequest) (BackupResult, error) {
	if request.Runner == nil {
		return BackupResult{}, fmt.Errorf("a command runner is required")
	}
	operation, err := command.NewOperationRequest(request.Operation.Target, request.Operation.OperatorID, request.Operation.AuthorizationRef)
	if err != nil {
		return BackupResult{}, err
	}
	settings, err := request.Config.Settings(operation.Target)
	if err != nil {
		return BackupResult{}, err
	}
	sourceRoot, err := canonicalDirectory(request.SourceRoot)
	if err != nil {
		return BackupResult{}, err
	}
	outputDir, err := externalDirectory(sourceRoot, request.OutputDir)
	if err != nil {
		return BackupResult{}, err
	}
	if err := requirePostgresUtility(ctx, request.Runner, "pg_dump"); err != nil {
		return BackupResult{}, err
	}
	if err := requirePostgresUtility(ctx, request.Runner, "pg_restore"); err != nil {
		return BackupResult{}, err
	}
	tempDump, err := os.CreateTemp(outputDir, ".mindforge-infra-*.dump.partial")
	if err != nil {
		return BackupResult{}, fmt.Errorf("create private backup temporary file")
	}
	tempDumpPath := tempDump.Name()
	cleanupDump := true
	defer func() {
		if cleanupDump {
			_ = os.Remove(tempDumpPath)
		}
	}()
	if err := tempDump.Chmod(0o600); err != nil {
		_ = tempDump.Close()
		return BackupResult{}, fmt.Errorf("protect backup temporary file")
	}
	if err := tempDump.Close(); err != nil {
		return BackupResult{}, fmt.Errorf("close backup temporary file")
	}

	dumpResult, err := request.Runner.Run(ctx, command.Invocation{
		Name: "pg_dump",
		Args: []string{
			"--format=custom",
			"--no-owner",
			"--no-acl",
			"--host=" + settings.DatabaseHost,
			"--port=" + settings.DatabasePort,
			"--username=" + settings.DatabaseUser,
			"--dbname=" + settings.DatabaseName,
		},
		Env: []string{"PGPASSWORD=" + settings.DatabasePassword},
	})
	if err != nil || dumpResult.ExitCode != 0 {
		return BackupResult{}, fmt.Errorf("pg_dump failed")
	}
	if len(dumpResult.Stdout) == 0 {
		return BackupResult{}, fmt.Errorf("pg_dump produced an empty archive")
	}
	tempDump, err = os.OpenFile(tempDumpPath, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return BackupResult{}, fmt.Errorf("open backup temporary file")
	}
	if _, err := tempDump.WriteString(dumpResult.Stdout); err != nil {
		_ = tempDump.Close()
		return BackupResult{}, fmt.Errorf("write backup temporary file")
	}
	if err := tempDump.Close(); err != nil {
		return BackupResult{}, fmt.Errorf("close backup temporary file")
	}

	if err := runArchiveListing(ctx, request.Runner, tempDumpPath); err != nil {
		return BackupResult{}, err
	}
	digest, size, err := hashFile(tempDumpPath)
	if err != nil || size == 0 {
		return BackupResult{}, fmt.Errorf("hash backup archive")
	}

	now := time.Now().UTC()
	stamp := now.Format("20060102T150405.000000000Z")
	createdAt := now.Format(time.RFC3339)
	artifactName := fmt.Sprintf("mindforge-infra-%s-%s.dump", operation.Target, stamp)
	manifestName := artifactName + ".manifest.json"
	artifactPath := filepath.Join(outputDir, artifactName)
	manifestPath := filepath.Join(outputDir, manifestName)
	if _, err := os.Stat(artifactPath); err == nil {
		return BackupResult{}, fmt.Errorf("backup artifact already exists")
	}
	if _, err := os.Stat(manifestPath); err == nil {
		return BackupResult{}, fmt.Errorf("backup manifest already exists")
	}
	manifest, err := evidence.NewBackupManifest(
		operation.Target,
		operation.OperatorID,
		operation.AuthorizationRef,
		artifactName,
		digest,
		size,
		createdAt,
	)
	if err != nil {
		return BackupResult{}, err
	}
	tempManifest, err := os.CreateTemp(outputDir, ".mindforge-infra-*.manifest.partial")
	if err != nil {
		return BackupResult{}, fmt.Errorf("create private manifest temporary file")
	}
	tempManifestPath := tempManifest.Name()
	_ = tempManifest.Close()
	defer os.Remove(tempManifestPath)
	if err := evidence.WriteBackupManifest(tempManifestPath, manifest); err != nil {
		return BackupResult{}, err
	}
	if err := os.Rename(tempDumpPath, artifactPath); err != nil {
		return BackupResult{}, fmt.Errorf("publish backup artifact")
	}
	cleanupDump = false
	if err := os.Rename(tempManifestPath, manifestPath); err != nil {
		_ = os.Remove(artifactPath)
		return BackupResult{}, fmt.Errorf("publish backup manifest")
	}
	return BackupResult{
		Target:       operation.Target,
		ArtifactPath: artifactPath,
		ManifestPath: manifestPath,
		SHA256:       digest,
		Manifest:     manifest,
	}, nil
}

// Restore validates the external artifact, target-bound manifest and named
// destination before issuing one explicit transactional pg_restore command.
func Restore(ctx context.Context, request RestoreRequest) (RestoreResult, error) {
	if request.Runner == nil {
		return RestoreResult{}, fmt.Errorf("a command runner is required")
	}
	operation, err := command.NewOperationRequest(request.Operation.Target, request.Operation.OperatorID, request.Operation.AuthorizationRef)
	if err != nil {
		return RestoreResult{}, err
	}
	settings, err := request.Config.Settings(operation.Target)
	if err != nil {
		return RestoreResult{}, err
	}
	sourceRoot, err := canonicalDirectory(request.SourceRoot)
	if err != nil {
		return RestoreResult{}, err
	}
	artifactPath, err := externalFile(sourceRoot, request.BackupFile)
	if err != nil {
		return RestoreResult{}, err
	}
	manifestPath, err := externalFile(sourceRoot, request.ManifestFile)
	if err != nil {
		return RestoreResult{}, err
	}
	manifest, err := evidence.ReadBackupManifest(manifestPath)
	if err != nil {
		return RestoreResult{}, err
	}
	if manifest.ArtifactName != filepath.Base(artifactPath) || manifest.Target != string(operation.Target) || manifest.Database != settings.DatabaseName {
		return RestoreResult{}, fmt.Errorf("backup manifest does not match the explicit restore target")
	}
	digest, size, err := hashFile(artifactPath)
	if err != nil || digest != manifest.ArtifactSHA256 || size != manifest.ArtifactBytes {
		return RestoreResult{}, fmt.Errorf("backup archive hash or size does not match its manifest")
	}
	if err := evidence.ValidateReplacementConfirmation(operation.Target, request.ReplacementConfirmation); err != nil {
		return RestoreResult{}, err
	}
	if err := requirePostgresUtility(ctx, request.Runner, "pg_restore"); err != nil {
		return RestoreResult{}, err
	}
	if err := runArchiveListing(ctx, request.Runner, artifactPath); err != nil {
		return RestoreResult{}, err
	}
	evidenceDir := request.EvidenceDir
	if evidenceDir == "" {
		evidenceDir = filepath.Dir(artifactPath)
	}
	evidenceDir, err = externalDirectory(sourceRoot, evidenceDir)
	if err != nil {
		return RestoreResult{}, err
	}
	evidencePath := filepath.Join(evidenceDir, fmt.Sprintf("mindforge-restore-%s-%s.json", operation.Target, time.Now().UTC().Format("20060102T150405.000000000Z")))
	attempted, err := evidence.NewRestoreEvidence(operation.Target, operation.OperatorID, operation.AuthorizationRef, manifest.ArtifactName, manifest.ArtifactSHA256, "attempted", true, false, "", time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return RestoreResult{}, err
	}
	if err := writeRestoreEvidenceAtomic(evidencePath, attempted); err != nil {
		return RestoreResult{}, err
	}
	result := RestoreResult{Target: operation.Target, ArtifactPath: artifactPath, ManifestPath: manifestPath, EvidencePath: evidencePath, Status: "attempted"}
	finish := func(status string, destinationVerified bool, errorClass string) error {
		finalEvidence, evidenceErr := evidence.NewRestoreEvidence(operation.Target, operation.OperatorID, operation.AuthorizationRef, manifest.ArtifactName, manifest.ArtifactSHA256, status, true, destinationVerified, errorClass, time.Now().UTC().Format(time.RFC3339))
		if evidenceErr != nil {
			return evidenceErr
		}
		if evidenceErr = writeRestoreEvidenceAtomic(evidencePath, finalEvidence); evidenceErr != nil {
			return fmt.Errorf("restore outcome evidence could not be recorded")
		}
		result.Status = status
		return nil
	}
	if err := requirePostgresUtility(ctx, request.Runner, "psql"); err != nil {
		if evidenceErr := finish("failed", false, "client_unavailable"); evidenceErr != nil {
			return result, evidenceErr
		}
		return result, err
	}
	destinationResult, err := runPSQL(ctx, request.Runner, settings, "SELECT current_database();")
	if err != nil || firstOutputLine(destinationResult.Stdout) != settings.DatabaseName {
		if evidenceErr := finish("failed", false, "destination_mismatch"); evidenceErr != nil {
			return result, evidenceErr
		}
		return result, fmt.Errorf("restore destination identity did not match the explicit target")
	}
	restoreResult, err := request.Runner.Run(ctx, command.Invocation{
		Name: "pg_restore",
		Args: []string{
			"--single-transaction",
			"--exit-on-error",
			"--clean",
			"--if-exists",
			"--no-owner",
			"--no-acl",
			"--host=" + settings.DatabaseHost,
			"--port=" + settings.DatabasePort,
			"--username=" + settings.DatabaseUser,
			"--dbname=" + settings.DatabaseName,
			artifactPath,
		},
		Env: []string{"PGPASSWORD=" + settings.DatabasePassword},
	})
	if err != nil || restoreResult.ExitCode != 0 {
		if evidenceErr := finish("failed", true, "restore_command_failed"); evidenceErr != nil {
			return result, evidenceErr
		}
		return result, fmt.Errorf("pg_restore failed; restore evidence was recorded")
	}
	if err := finish("succeeded", true, ""); err != nil {
		return result, err
	}
	return result, nil
}

// Promote performs the only supported data-changing workflow for feature
// migrations: verify develop, preflight main, create a fresh main backup,
// recheck main, and apply the same catalog bytes to main.
func Promote(ctx context.Context, request PromoteRequest) (PromoteResult, error) {
	result := PromoteResult{Target: request.Operation.Target}
	if request.Runner == nil {
		return result, fmt.Errorf("a command runner is required")
	}
	if request.Operation.Target != config.TargetMain {
		return result, fmt.Errorf("promotion requires mindforge")
	}
	if err := evidence.ValidateVerificationReference(request.BehaviorVerificationRef); err != nil {
		return result, err
	}
	operation, err := command.NewOperationRequest(request.Operation.Target, request.Operation.OperatorID, request.Operation.AuthorizationRef)
	if err != nil {
		return result, err
	}
	if _, err := request.Config.Settings(config.TargetMain); err != nil {
		return result, err
	}
	if _, err := request.Config.Settings(config.TargetDevelop); err != nil {
		return result, err
	}
	sourceRoot, err := canonicalDirectory(request.SourceRoot)
	if err != nil {
		return result, err
	}
	outputDir, err := externalDirectory(sourceRoot, request.OutputDir)
	if err != nil {
		return result, err
	}
	evidencePath := filepath.Join(outputDir, fmt.Sprintf("mindforge-promotion-%s-%s.json", operation.Target, time.Now().UTC().Format("20060102T150405.000000000Z")))
	result.EvidencePath = evidencePath

	repositoryRoot, err := resolveRepositoryRoot(sourceRoot)
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "source_invalid", "", "", "", nil, nil, "not_started")
	}
	catalog, err := migrations.LoadCatalog(repositoryRoot)
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "source_invalid", "", "", "", nil, nil, "not_started")
	}
	featureMigrations := selectMigrations(catalog, ScopeFeature)
	if len(featureMigrations) == 0 {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "source_invalid", "", "", "", nil, nil, "not_started")
	}
	initialStatuses := promotionStatuses(featureMigrations, nil)
	attempted, err := newPromotionEvidence(operation, request.BehaviorVerificationRef, "attempted", "", "", "", nil, initialStatuses, "not_started", "")
	if err != nil {
		return result, err
	}
	if err := writePromotionEvidenceAtomic(evidencePath, attempted); err != nil {
		return result, err
	}
	result.Status = "attempted"

	developOperation, err := command.NewOperationRequest(config.TargetDevelop, operation.OperatorID, operation.AuthorizationRef)
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "develop_authority_invalid", "", "", "", nil, initialStatuses, "not_started")
	}
	developPreflight, err := Preflight(ctx, PreflightRequest{
		Config:     request.Config,
		Operation:  developOperation,
		Scope:      ScopeFeature,
		SourceRoot: sourceRoot,
		Runner:     request.Runner,
	})
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "develop_preflight_failed", "", "", "", nil, initialStatuses, "not_started")
	}
	result.DevelopPreflight = developPreflight
	developSettings, err := request.Config.Settings(config.TargetDevelop)
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "develop_configuration_invalid", developPreflight.Reference, "", "", nil, initialStatuses, "not_started")
	}
	developEvidence, err := readVerifiedDevelopEvidence(ctx, request.Runner, developSettings)
	if err != nil || verifyDevelopPromotionEvidence(catalog, developPreflight, developEvidence, request.BehaviorVerificationRef) != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "develop_evidence_invalid", developPreflight.Reference, "", "", nil, initialStatuses, "not_started")
	}

	mainPreflight, err := Preflight(ctx, PreflightRequest{
		Config:     request.Config,
		Operation:  operation,
		Scope:      ScopeFeature,
		SourceRoot: sourceRoot,
		Runner:     request.Runner,
	})
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "main_preflight_failed", developPreflight.Reference, "", "", nil, initialStatuses, "not_started")
	}
	result.MainPreflight = mainPreflight
	mainStatuses := promotionStatuses(featureMigrations, mainPreflight.Ledger)
	backup, err := Backup(ctx, BackupRequest{
		Config:     request.Config,
		Operation:  operation,
		SourceRoot: sourceRoot,
		OutputDir:  outputDir,
		Runner:     request.Runner,
	})
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "main_backup_failed", developPreflight.Reference, mainPreflight.Reference, "", nil, mainStatuses, "not_started")
	}
	result.Backup = backup
	mainRecheck, err := Preflight(ctx, PreflightRequest{
		Config:     request.Config,
		Operation:  operation,
		Scope:      ScopeFeature,
		SourceRoot: sourceRoot,
		Runner:     request.Runner,
	})
	if err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "main_recheck_failed", developPreflight.Reference, mainPreflight.Reference, "", &backup, mainStatuses, "not_started")
	}
	result.MainRecheck = mainRecheck
	if mainPreflight.ObservedState != mainRecheck.ObservedState || mainPreflight.CatalogHash != mainRecheck.CatalogHash || mainPreflight.SourceRoot != mainRecheck.SourceRoot {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "main_drift", developPreflight.Reference, mainPreflight.Reference, mainRecheck.Reference, &backup, mainStatuses, "not_started")
	}
	latestCatalog, err := migrations.LoadCatalog(repositoryRoot)
	if err != nil || migrations.CatalogHash(latestCatalog) != migrations.CatalogHash(catalog) {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "source_drift", developPreflight.Reference, mainPreflight.Reference, mainRecheck.Reference, &backup, mainStatuses, "not_started")
	}

	applyResult, err := applyMigrations(ctx, ApplyRequest{
		Config:                  request.Config,
		Operation:               operation,
		Scope:                   ScopeFeature,
		SourceRoot:              sourceRoot,
		Runner:                  request.Runner,
		Preflight:               mainRecheck,
		BehaviorVerificationRef: request.BehaviorVerificationRef,
	}, true)
	if err != nil {
		mainStatuses = markPromotionApplied(mainStatuses, applyResult.Applied, true)
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "promotion_apply_failed", developPreflight.Reference, mainPreflight.Reference, mainRecheck.Reference, &backup, mainStatuses, "failed")
	}
	result.Applied = applyResult.Applied
	mainStatuses = markPromotionApplied(mainStatuses, applyResult.Applied, false)
	if err := ensurePromotionComplete(mainStatuses); err != nil {
		return finishPromotionFailure(result, evidencePath, operation, request.BehaviorVerificationRef, "promotion_evidence_incomplete", developPreflight.Reference, mainPreflight.Reference, mainRecheck.Reference, &backup, mainStatuses, "inconclusive")
	}
	succeeded, err := newPromotionEvidence(operation, request.BehaviorVerificationRef, "succeeded", developPreflight.Reference, mainPreflight.Reference, mainRecheck.Reference, &backup, mainStatuses, "succeeded", "")
	if err != nil {
		return result, err
	}
	if err := writePromotionEvidenceAtomic(evidencePath, succeeded); err != nil {
		return result, err
	}
	result.Status = "succeeded"
	return result, nil
}

func readVerifiedDevelopEvidence(ctx context.Context, runner command.Runner, settings config.TargetSettings) ([]developEvidenceRecord, error) {
	result, err := runPSQL(ctx, runner, settings, verifiedDevelopEvidenceQuery)
	if err != nil {
		return nil, fmt.Errorf("read develop migration evidence")
	}
	return parseDevelopEvidence(result.Stdout)
}

type developEvidenceRecord struct {
	ID                 string
	SHA256             string
	Target             config.Target
	PreflightReference string
	VerificationRef    string
	PostResult         string
	Outcome            string
}

func parseDevelopEvidence(output string) ([]developEvidenceRecord, error) {
	records := make([]developEvidenceRecord, 0)
	seen := make(map[string]struct{})
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 7 || !regexp.MustCompile(`^[0-9]{4}$`).MatchString(parts[0]) || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(parts[1]) {
			return nil, fmt.Errorf("develop migration evidence is malformed")
		}
		target, err := config.ParseTarget(parts[2])
		if err != nil || target != config.TargetDevelop || evidence.ValidateReference(parts[3]) != nil || evidence.ValidateVerificationReference(parts[4]) != nil || parts[5] != "succeeded" || parts[6] != "succeeded" {
			return nil, fmt.Errorf("develop migration evidence is not verified")
		}
		if _, exists := seen[parts[0]]; exists {
			return nil, fmt.Errorf("develop migration evidence is duplicated")
		}
		seen[parts[0]] = struct{}{}
		records = append(records, developEvidenceRecord{ID: parts[0], SHA256: parts[1], Target: target, PreflightReference: parts[3], VerificationRef: parts[4], PostResult: parts[5], Outcome: parts[6]})
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("develop migration evidence is missing")
	}
	return records, nil
}

func verifyDevelopPromotionEvidence(catalog migrations.Catalog, preflight evidence.Preflight, records []developEvidenceRecord, verificationReference string) error {
	if preflight.Target != config.TargetDevelop || !preflight.InfrastructureReady || preflight.SourceRoot != catalog.Root || preflight.CatalogHash != migrations.CatalogHash(catalog) {
		return fmt.Errorf("develop preflight is stale or incomplete")
	}
	byID := make(map[string]developEvidenceRecord, len(records))
	for _, record := range records {
		byID[record.ID] = record
	}
	features := selectMigrations(catalog, ScopeFeature)
	for _, migration := range features {
		if preflight.Ledger[migration.ID] != migration.SHA256 {
			return fmt.Errorf("develop migration %s is not applied with its immutable hash", migration.ID)
		}
		record, ok := byID[migration.ID]
		if !ok || record.SHA256 != migration.SHA256 || record.VerificationRef != verificationReference || record.PostResult != "succeeded" || record.Outcome != "succeeded" {
			return fmt.Errorf("develop migration %s lacks verified evidence", migration.ID)
		}
	}
	return nil
}

func promotionStatuses(featureMigrations []migrations.Migration, ledger migrations.Ledger) []evidence.PromotionMigration {
	statuses := make([]evidence.PromotionMigration, 0, len(featureMigrations))
	for _, migration := range featureMigrations {
		status := "pending"
		if ledger != nil && ledger[migration.ID] == migration.SHA256 {
			status = "already_applied"
		}
		statuses = append(statuses, evidence.PromotionMigration{ID: migration.ID, SHA256: migration.SHA256, Status: status})
	}
	return statuses
}

func markPromotionApplied(statuses []evidence.PromotionMigration, applied []evidence.AppliedMigration, failed bool) []evidence.PromotionMigration {
	appliedByID := make(map[string]struct{}, len(applied))
	for _, migration := range applied {
		appliedByID[migration.ID] = struct{}{}
	}
	for index := range statuses {
		if statuses[index].Status != "pending" {
			continue
		}
		if _, ok := appliedByID[statuses[index].ID]; ok {
			statuses[index].Status = "applied"
		} else if failed {
			statuses[index].Status = "failed"
		}
	}
	return statuses
}

func ensurePromotionComplete(statuses []evidence.PromotionMigration) error {
	for _, status := range statuses {
		if status.Status != "applied" && status.Status != "already_applied" {
			return fmt.Errorf("promotion migration evidence is incomplete")
		}
	}
	return nil
}

func newPromotionEvidence(operation command.OperationRequest, verificationReference, status, developReference, mainReference, recheckReference string, backup *BackupResult, migrations []evidence.PromotionMigration, postResult, errorClass string) (evidence.PromotionEvidence, error) {
	record := evidence.PromotionEvidence{
		SchemaVersion:                 1,
		Operation:                     "promote",
		Status:                        status,
		Target:                        string(operation.Target),
		Database:                      string(operation.Target),
		OperatorID:                    operation.OperatorID,
		AuthorizationReference:        operation.AuthorizationRef,
		DevelopPreflightReference:     developReference,
		MainPreflightReference:        mainReference,
		MainRecheckReference:          recheckReference,
		BehaviorAccessVerificationRef: verificationReference,
		PostApplicationResult:         postResult,
		Migrations:                    append([]evidence.PromotionMigration(nil), migrations...),
		ErrorClass:                    errorClass,
		CreatedAt:                     time.Now().UTC().Format(time.RFC3339),
	}
	if backup != nil {
		backupTarget, err := config.ParseTarget(backup.Manifest.Target)
		if err != nil {
			return evidence.PromotionEvidence{}, err
		}
		reference, err := evidence.NewBackupReference(backupTarget, backup.SHA256)
		if err != nil {
			return evidence.PromotionEvidence{}, err
		}
		record.BackupReference = reference
		record.BackupArtifactName = backup.Manifest.ArtifactName
		record.BackupManifestName = filepath.Base(backup.ManifestPath)
		record.BackupSHA256 = backup.SHA256
	}
	if err := evidence.ValidatePromotionEvidence(record); err != nil {
		return evidence.PromotionEvidence{}, err
	}
	return record, nil
}

func finishPromotionFailure(result PromoteResult, evidencePath string, operation command.OperationRequest, verificationReference, errorClass, developReference, mainReference, recheckReference string, backup *BackupResult, migrations []evidence.PromotionMigration, postResult string) (PromoteResult, error) {
	record, err := newPromotionEvidence(operation, verificationReference, "failed", developReference, mainReference, recheckReference, backup, migrations, postResult, errorClass)
	if err != nil {
		return result, err
	}
	if err := writePromotionEvidenceAtomic(evidencePath, record); err != nil {
		return result, err
	}
	result.Status = "failed"
	return result, fmt.Errorf("promotion failed: %s", errorClass)
}

type metadata struct {
	Database       string
	Databases      string
	VectorVersion  string
	InfraNamespace string
	LedgerTable    string
	EvidenceTable  string
	AttemptsTable  string
	AuthNamespace  string
	UsersTable     string
	Identities     string
	SchemaMigrate  string
	SchemaCount    string
	Ledger         string
}

type provisionInventory struct {
	Databases       string
	MainDatabase    bool
	MainOwner       string
	MainRole        bool
	DevelopRole     bool
	DevelopDatabase bool
	AdminSuperuser  bool
}

// Optional relations must not appear in a direct FROM: PostgreSQL binds those
// names before CASE can skip an absent relation. Fixed query text is evaluated
// only after the corresponding catalog lookup succeeds. No DDL or user-data
// query is introduced; XML functions are PostgreSQL built-ins.
const metadataQuery = `SELECT current_database(),
COALESCE((SELECT string_agg(datname, ',' ORDER BY datname) FROM pg_database WHERE datistemplate = false AND datname <> 'postgres'), ''),
COALESCE((SELECT extversion FROM pg_extension WHERE extname = 'vector'), ''),
COALESCE(to_regnamespace('infra')::text, ''),
COALESCE(to_regclass('infra.migration_ledger')::text, ''),
COALESCE(to_regclass('infra.migration_target_evidence')::text, ''),
COALESCE(to_regclass('infra.migration_attempts')::text, ''),
COALESCE(to_regnamespace('auth')::text, ''),
COALESCE(to_regclass('auth.users')::text, ''),
COALESCE(to_regclass('auth.identities')::text, ''),
COALESCE(to_regclass('auth.schema_migrations')::text, ''),
CASE WHEN to_regclass('auth.schema_migrations') IS NULL THEN '0'
ELSE COALESCE((xpath('/table/row/applied/text()', query_to_xml('SELECT count(*) AS applied FROM auth.schema_migrations', false, false, '')))[1]::text, '0') END,
CASE WHEN to_regclass('infra.migration_ledger') IS NULL THEN ''
ELSE COALESCE((xpath('/table/row/ledger/text()', query_to_xml('SELECT string_agg(id || '':'' || sha256, '','' ORDER BY id) AS ledger FROM infra.migration_ledger', false, false, '')))[1]::text, '') END` // #nosec G101 -- fixed read-only metadata queries, no caller-supplied SQL.

const verifiedDevelopEvidenceQuery = `SELECT DISTINCT ON (migration_id) migration_id, migration_sha256, target_name, COALESCE(preflight_reference, ''), COALESCE(behavior_access_verification_reference, ''), post_application_result, sanitized_outcome FROM infra.migration_target_evidence WHERE target_name = 'mindforge_develop' ORDER BY migration_id, completed_at DESC NULLS LAST, evidence_id DESC` // #nosec G101 -- SQL is a fixed read-only evidence query.

const ledgerBootstrapSQL = `CREATE SCHEMA IF NOT EXISTS infra; CREATE TABLE IF NOT EXISTS infra.migration_ledger (id text PRIMARY KEY, sha256 text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now(), CHECK (id ~ '^[0-9]{4}$'), CHECK (sha256 ~ '^[0-9a-f]{64}$'));` // #nosec G101 -- fixed infrastructure bookkeeping prerequisite.

func provisionInventoryQuery(mainRole, developRole string) string {
	mainRoleLiteral := quoteSQLLiteral(mainRole)
	developRoleLiteral := quoteSQLLiteral(developRole)
	return `SELECT COALESCE((SELECT string_agg(datname, ',' ORDER BY datname) FROM pg_database WHERE datistemplate = false AND datname <> 'postgres'), ''), (EXISTS (SELECT 1 FROM pg_database WHERE datname = 'mindforge'))::text, COALESCE((SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = 'mindforge'), ''), (EXISTS (SELECT 1 FROM pg_roles WHERE rolname = ` + mainRoleLiteral + `))::text, (EXISTS (SELECT 1 FROM pg_roles WHERE rolname = ` + developRoleLiteral + `))::text, (EXISTS (SELECT 1 FROM pg_database WHERE datname = 'mindforge_develop'))::text, (COALESCE((SELECT rolsuper FROM pg_roles WHERE rolname = current_user), false))::text`
}

func readProvisionInventory(ctx context.Context, runner command.Runner, admin config.ClusterAdminSettings, query string) (provisionInventory, error) {
	result, err := runner.Run(ctx, command.Invocation{
		Name: "psql",
		Args: []string{
			"--no-psqlrc",
			"--set=ON_ERROR_STOP=1",
			"--tuples-only",
			"--no-align",
			"--field-separator=|",
			"--host=" + admin.DatabaseHost,
			"--port=" + admin.DatabasePort,
			"--username=" + admin.DatabaseUser,
			"--dbname=" + admin.DatabaseName,
			"--command=" + query,
		},
		Env: []string{"PGPASSWORD=" + admin.DatabasePassword},
	})
	if err != nil || result.ExitCode != 0 {
		return provisionInventory{}, fmt.Errorf("read-only cluster inventory failed")
	}
	return parseProvisionInventory(result.Stdout)
}

func parseProvisionInventory(output string) (provisionInventory, error) {
	line := firstOutputLine(output)
	parts := strings.Split(line, "|")
	if len(parts) != 7 {
		return provisionInventory{}, fmt.Errorf("read-only cluster inventory is incomplete")
	}
	booleans := make([]bool, 0, 6)
	for _, part := range []string{parts[1], parts[3], parts[4], parts[5], parts[6]} {
		switch strings.TrimSpace(part) {
		case "true":
			booleans = append(booleans, true)
		case "false":
			booleans = append(booleans, false)
		default:
			return provisionInventory{}, fmt.Errorf("read-only cluster inventory has an invalid state")
		}
	}
	return provisionInventory{
		Databases:       strings.TrimSpace(parts[0]),
		MainDatabase:    booleans[0],
		MainOwner:       strings.TrimSpace(parts[2]),
		MainRole:        booleans[1],
		DevelopRole:     booleans[2],
		DevelopDatabase: booleans[3],
		AdminSuperuser:  booleans[4],
	}, nil
}

func validateProvisionInventory(inventory provisionInventory, mainRole string) error {
	listedMain := databaseInventoryContains(inventory.Databases, string(config.TargetMain))
	listedDevelop := databaseInventoryContains(inventory.Databases, string(config.TargetDevelop))
	if listedMain != inventory.MainDatabase || listedDevelop != inventory.DevelopDatabase {
		return fmt.Errorf("cluster inventory target facts are inconsistent")
	}
	if !inventory.MainDatabase || inventory.MainOwner != mainRole || !inventory.MainRole || inventory.DevelopRole || inventory.DevelopDatabase || !inventory.AdminSuperuser {
		return fmt.Errorf("cluster is not in the safe state for additive develop provisioning")
	}
	return nil
}

func sameMindforgeProvisionState(left, right provisionInventory) bool {
	return left.MainDatabase == right.MainDatabase &&
		left.MainOwner == right.MainOwner &&
		left.MainRole == right.MainRole &&
		left.DevelopRole == right.DevelopRole &&
		left.DevelopDatabase == right.DevelopDatabase &&
		left.AdminSuperuser == right.AdminSuperuser
}

func databaseInventoryContains(databaseList, target string) bool {
	for _, database := range strings.Split(databaseList, ",") {
		if strings.TrimSpace(database) == target {
			return true
		}
	}
	return false
}

func canonicalMindforgeDatabaseInventory(databaseList string) string {
	var targets []string
	if databaseInventoryContains(databaseList, string(config.TargetMain)) {
		targets = append(targets, string(config.TargetMain))
	}
	if databaseInventoryContains(databaseList, string(config.TargetDevelop)) {
		targets = append(targets, string(config.TargetDevelop))
	}
	return strings.Join(targets, ",")
}

func writeProvisionSQL(developRole, password string) (string, error) {
	file, err := os.CreateTemp("", "mindforge-infra-provision-*.sql")
	if err != nil {
		return "", err
	}
	path := file.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return "", err
	}
	statement := "CREATE ROLE " + quoteSQLIdentifier(developRole) + " LOGIN PASSWORD " + quoteSQLLiteral(password) + ";\n" +
		"CREATE DATABASE " + quoteSQLIdentifier(string(config.TargetDevelop)) + " OWNER " + quoteSQLIdentifier(developRole) + ";\n"
	if _, err := io.WriteString(file, statement); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	keep = true
	return path, nil
}

func runClusterPSQLFile(ctx context.Context, runner command.Runner, admin config.ClusterAdminSettings, path string) (command.Result, error) {
	result, err := runner.Run(ctx, command.Invocation{
		Name: "psql",
		Args: []string{
			"--no-psqlrc",
			"--set=ON_ERROR_STOP=1",
			"--host=" + admin.DatabaseHost,
			"--port=" + admin.DatabasePort,
			"--username=" + admin.DatabaseUser,
			"--dbname=" + admin.DatabaseName,
			"--file=" + path,
		},
		Env: []string{"PGPASSWORD=" + admin.DatabasePassword},
	})
	if err != nil || result.ExitCode != 0 {
		return command.Result{}, fmt.Errorf("PostgreSQL provisioning command failed")
	}
	return result, nil
}

func quoteSQLIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteSQLLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func parseMetadata(output string) (metadata, error) {
	var line string
	for _, candidate := range strings.Split(output, "\n") {
		if strings.TrimSpace(candidate) != "" {
			line = strings.TrimSpace(candidate)
			break
		}
	}
	parts := strings.Split(line, "|")
	if len(parts) != 13 {
		return metadata{}, fmt.Errorf("preflight returned an incomplete catalog record")
	}
	values := make([]string, len(parts))
	for index, part := range parts {
		values[index] = strings.TrimSpace(part)
	}
	return metadata{
		Database:       values[0],
		Databases:      values[1],
		VectorVersion:  values[2],
		InfraNamespace: values[3],
		LedgerTable:    values[4],
		EvidenceTable:  values[5],
		AttemptsTable:  values[6],
		AuthNamespace:  values[7],
		UsersTable:     values[8],
		Identities:     values[9],
		SchemaMigrate:  values[10],
		SchemaCount:    values[11],
		Ledger:         values[12],
	}, nil
}

func (value metadata) canonicalState() string {
	return strings.Join([]string{value.Database, canonicalMindforgeDatabaseInventory(value.Databases), value.VectorVersion, value.InfraNamespace, value.LedgerTable, value.EvidenceTable, value.AttemptsTable, value.AuthNamespace, value.UsersTable, value.Identities, value.SchemaMigrate, value.SchemaCount, value.Ledger}, "|")
}

func verifyInfrastructureState(catalog migrations.Catalog, observed metadata, ledger migrations.Ledger) (bool, error) {
	if observed.InfraNamespace != "infra" || observed.LedgerTable != "infra.migration_ledger" || observed.EvidenceTable != "infra.migration_target_evidence" || observed.AttemptsTable != "infra.migration_attempts" {
		return false, fmt.Errorf("preflight infrastructure ledger state is incomplete")
	}
	schemaCount, err := strconv.Atoi(observed.SchemaCount)
	if err != nil || schemaCount < 1 {
		return false, fmt.Errorf("preflight Auth migration state is incomplete")
	}
	if observed.AuthNamespace != "auth" || observed.UsersTable != "auth.users" || observed.Identities != "auth.identities" || observed.SchemaMigrate != "auth.schema_migrations" {
		return false, fmt.Errorf("preflight Auth catalog state is incomplete")
	}
	if !vectorVersionCompatible(observed.VectorVersion) {
		return false, fmt.Errorf("preflight vector extension is missing or incompatible")
	}
	for _, migration := range catalog.Migrations {
		if migration.Kind != migrations.KindInfrastructure {
			continue
		}
		if ledger[migration.ID] != migration.SHA256 {
			return false, fmt.Errorf("preflight infrastructure migration %s is missing or changed", migration.ID)
		}
	}
	return true, nil
}

func verifyKnownInfrastructureRows(catalog migrations.Catalog, ledger migrations.Ledger) error {
	byID := make(map[string]migrations.Migration)
	for _, migration := range catalog.Migrations {
		if migration.Kind == migrations.KindInfrastructure {
			byID[migration.ID] = migration
		}
	}
	for id, hash := range ledger {
		migration, ok := byID[id]
		if !ok {
			continue
		}
		if migration.SHA256 != hash {
			return fmt.Errorf("applied migration %s has changed bytes", id)
		}
	}
	return nil
}

func verifyFeaturePreflight(
	ctx context.Context,
	runner command.Runner,
	settings config.TargetSettings,
	catalog migrations.Catalog,
	ledger migrations.Ledger,
) (map[string]string, error) {
	features := selectMigrations(catalog, ScopeFeature)
	if len(features) == 0 {
		return nil, fmt.Errorf("feature preflight requires a registered feature migration")
	}
	states := make(map[string]string, len(features))
	for _, migration := range features {
		result, err := runPSQLFile(ctx, runner, settings, migration.PreflightPath)
		if err != nil {
			return nil, fmt.Errorf("feature migration %s read-only preflight failed", migration.ID)
		}
		state, err := parseFeaturePreflightState(result.Stdout)
		if err != nil {
			return nil, fmt.Errorf("feature migration %s returned an invalid preflight state", migration.ID)
		}
		want := "absent_eligible"
		if appliedHash, ok := ledger[migration.ID]; ok {
			if appliedHash != migration.SHA256 {
				return nil, fmt.Errorf("feature migration %s ledger hash is inconsistent", migration.ID)
			}
			want = "complete"
		}
		if state != want {
			return nil, fmt.Errorf("feature migration %s schema is %s; expected %s", migration.ID, state, want)
		}
		states[migration.ID] = state
	}
	return states, nil
}

func parseFeaturePreflightState(output string) (string, error) {
	var state string
	for _, line := range strings.Split(output, "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" {
			continue
		}
		if state != "" {
			return "", fmt.Errorf("feature preflight returned more than one row")
		}
		state = candidate
	}
	switch state {
	case "absent_eligible", "complete", "partial_divergent":
		return state, nil
	default:
		return "", fmt.Errorf("feature preflight state is unknown")
	}
}

func preflightStateFor(observedState, id string) (string, error) {
	marker := "feature-state:" + id + "="
	var state string
	for _, field := range strings.Split(observedState, "|") {
		if !strings.HasPrefix(field, marker) {
			continue
		}
		if state != "" {
			return "", fmt.Errorf("preflight contains duplicate feature state for migration %s", id)
		}
		state = strings.TrimPrefix(field, marker)
	}
	if state != "absent_eligible" && state != "complete" {
		return "", fmt.Errorf("preflight is missing an eligible feature state for migration %s", id)
	}
	return state, nil
}

func prepareFeatureMigrationTransaction(migration migrations.Migration) (string, func() error, error) {
	if migration.Kind != migrations.KindFeature || migration.ID == "" || migration.SHA256 == "" || migration.Path == "" || migration.VerificationPath == "" || migration.VerificationSHA256 == "" {
		return "", nil, fmt.Errorf("feature migration source or verification check is incomplete")
	}
	source, err := readHashedSource(migration.Path, migration.SHA256)
	if err != nil {
		return "", nil, fmt.Errorf("canonical feature migration source changed")
	}
	verification, err := readHashedSource(migration.VerificationPath, migration.VerificationSHA256)
	if err != nil {
		return "", nil, fmt.Errorf("feature verification check changed")
	}
	if !regexp.MustCompile(`^[0-9]{4}$`).MatchString(migration.ID) || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(migration.SHA256) {
		return "", nil, fmt.Errorf("feature migration identity is unsafe")
	}
	transaction := make([]byte, 0, len(source)+len(verification)+256)
	transaction = append(transaction, source...)
	if len(transaction) > 0 && transaction[len(transaction)-1] != '\n' {
		transaction = append(transaction, '\n')
	}
	transaction = append(transaction, []byte("\n-- Verify the feature-owned schema before recording the global migration.\n")...)
	transaction = append(transaction, verification...)
	if len(transaction) > 0 && transaction[len(transaction)-1] != '\n' {
		transaction = append(transaction, '\n')
	}
	transaction = append(transaction, []byte(fmt.Sprintf(
		"\nINSERT INTO infra.migration_ledger (id, sha256) VALUES ('%s', '%s');\n",
		migration.ID,
		migration.SHA256,
	))...)
	file, err := os.CreateTemp("", "mindforge-feature-migration-*.sql")
	if err != nil {
		return "", nil, fmt.Errorf("create private feature migration transaction")
	}
	path := file.Name()
	cleanup := func() error { return os.Remove(path) }
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		_ = cleanup()
		return "", nil, fmt.Errorf("protect private feature migration transaction")
	}
	if _, err := file.Write(transaction); err != nil {
		_ = file.Close()
		_ = cleanup()
		return "", nil, fmt.Errorf("write private feature migration transaction")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = cleanup()
		return "", nil, fmt.Errorf("flush private feature migration transaction")
	}
	if err := file.Close(); err != nil {
		_ = cleanup()
		return "", nil, fmt.Errorf("close private feature migration transaction")
	}
	return path, cleanup, nil
}

func readHashedSource(path, expectedHash string) ([]byte, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(contents)
	if hex.EncodeToString(digest[:]) != expectedHash {
		return nil, fmt.Errorf("source bytes changed")
	}
	return contents, nil
}

func selectMigrations(catalog migrations.Catalog, scope Scope) []migrations.Migration {
	selected := make([]migrations.Migration, 0)
	for _, migration := range catalog.Migrations {
		if scope == ScopeInfrastructure && migration.Kind == migrations.KindInfrastructure {
			selected = append(selected, migration)
		}
		if scope == ScopeFeature && migration.Kind == migrations.KindFeature {
			selected = append(selected, migration)
		}
	}
	return selected
}

func requirePostgresClient(ctx context.Context, runner command.Runner) error {
	return requirePostgresUtility(ctx, runner, "psql")
}

func requirePostgresUtility(ctx context.Context, runner command.Runner, name string) error {
	result, err := runner.Run(ctx, command.Invocation{Name: name, Args: []string{"--version"}})
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("compatible PostgreSQL client utilities are required")
	}
	versionPattern := regexp.MustCompile(`(?i)PostgreSQL\)?\s+([0-9]+)\.`)
	matches := versionPattern.FindStringSubmatch(result.Stdout)
	if len(matches) != 2 || matches[1] != supportedPostgresMajor {
		return fmt.Errorf("PostgreSQL client major version must match the pinned server")
	}
	return nil
}

func runArchiveListing(ctx context.Context, runner command.Runner, archivePath string) error {
	result, err := runner.Run(ctx, command.Invocation{
		Name: "pg_restore",
		Args: []string{"--list", archivePath},
	})
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("backup archive listing validation failed")
	}
	return nil
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open backup artifact")
	}
	defer file.Close()
	digest := sha256.New()
	size, err := io.Copy(digest, file)
	if err != nil {
		return "", 0, fmt.Errorf("hash backup artifact")
	}
	return hex.EncodeToString(digest.Sum(nil)), size, nil
}

func canonicalDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("an explicit directory is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve directory")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("inspect directory")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("directory is unavailable")
	}
	return resolved, nil
}

func externalDirectory(sourceRoot, path string) (string, error) {
	root, err := canonicalDirectory(sourceRoot)
	if err != nil {
		return "", err
	}
	directory, err := canonicalDirectory(path)
	if err != nil {
		return "", err
	}
	if pathInside(root, directory) {
		return "", fmt.Errorf("backup evidence must be outside the checkout")
	}
	return directory, nil
}

func externalFile(sourceRoot, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("an explicit backup artifact is required")
	}
	root, err := canonicalDirectory(sourceRoot)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve backup artifact")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("inspect backup artifact")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("backup artifact is unavailable")
	}
	if pathInside(root, resolved) {
		return "", fmt.Errorf("backup artifact must be outside the checkout")
	}
	return resolved, nil
}

func pathInside(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return true
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func writeRestoreEvidenceAtomic(path string, record evidence.RestoreEvidence) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".mindforge-restore-*.json.partial")
	if err != nil {
		return fmt.Errorf("create restore evidence temporary file")
	}
	temporaryPath := temporary.Name()
	_ = temporary.Close()
	defer os.Remove(temporaryPath)
	if err := evidence.WriteRestoreEvidence(temporaryPath, record); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish restore evidence")
	}
	return nil
}

func writePromotionEvidenceAtomic(path string, record evidence.PromotionEvidence) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".mindforge-promotion-*.json.partial")
	if err != nil {
		return fmt.Errorf("create promotion evidence temporary file")
	}
	temporaryPath := temporary.Name()
	_ = temporary.Close()
	defer os.Remove(temporaryPath)
	if err := evidence.WritePromotionEvidence(temporaryPath, record); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish promotion evidence")
	}
	return nil
}

func firstOutputLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if candidate := strings.TrimSpace(line); candidate != "" {
			return candidate
		}
	}
	return ""
}

func runPSQL(ctx context.Context, runner command.Runner, settings config.TargetSettings, sql string) (command.Result, error) {
	return runPSQLWithOptions(ctx, runner, settings, sql, false)
}

func runPSQLFile(ctx context.Context, runner command.Runner, settings config.TargetSettings, path string) (command.Result, error) {
	if strings.TrimSpace(path) == "" {
		return command.Result{}, fmt.Errorf("an explicit SQL check file is required")
	}
	result, err := runner.Run(ctx, psqlFileInvocation(settings, path, false))
	if err != nil || result.ExitCode != 0 {
		return command.Result{}, fmt.Errorf("PostgreSQL client check failed")
	}
	return result, nil
}

func runPSQLTransaction(ctx context.Context, runner command.Runner, settings config.TargetSettings, sql string) (command.Result, error) {
	return runPSQLWithOptions(ctx, runner, settings, sql, true)
}

func runPSQLWithOptions(ctx context.Context, runner command.Runner, settings config.TargetSettings, sql string, singleTransaction bool) (command.Result, error) {
	args := []string{
		"--no-psqlrc",
		"--set=ON_ERROR_STOP=1",
		"--tuples-only",
		"--no-align",
		"--field-separator=|",
		"--host=" + settings.DatabaseHost,
		"--port=" + settings.DatabasePort,
		"--username=" + settings.DatabaseUser,
		"--dbname=" + settings.DatabaseName,
		"--command=" + sql,
	}
	if singleTransaction {
		args = append([]string{args[0], args[1], "--single-transaction"}, args[2:]...)
	}
	result, err := runner.Run(ctx, command.Invocation{
		Name: "psql",
		Args: args,
		Env:  []string{"PGPASSWORD=" + settings.DatabasePassword},
	})
	if err != nil || result.ExitCode != 0 {
		return command.Result{}, fmt.Errorf("PostgreSQL client operation failed")
	}
	return result, nil
}

func migrationInvocation(settings config.TargetSettings, migration migrations.Migration) command.Invocation {
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
			"--file=" + migration.Path,
			"--command=INSERT INTO infra.migration_ledger (id, sha256) VALUES ('" + migration.ID + "', '" + migration.SHA256 + "');",
		},
		Env: []string{"PGPASSWORD=" + settings.DatabasePassword},
	}
}

func migrationFileInvocation(settings config.TargetSettings, migration migrations.Migration, path string) command.Invocation {
	return psqlFileInvocation(settings, path, true)
}

func psqlFileInvocation(settings config.TargetSettings, path string, singleTransaction bool) command.Invocation {
	args := []string{
		"--no-psqlrc",
		"--set=ON_ERROR_STOP=1",
		"--tuples-only",
		"--no-align",
		"--field-separator=|",
		"--host=" + settings.DatabaseHost,
		"--port=" + settings.DatabasePort,
		"--username=" + settings.DatabaseUser,
		"--dbname=" + settings.DatabaseName,
		"--file=" + path,
	}
	if singleTransaction {
		args = append([]string{args[0], args[1], "--single-transaction"}, args[2:]...)
	}
	return command.Invocation{
		Name: "psql",
		Args: args,
		Env:  []string{"PGPASSWORD=" + settings.DatabasePassword},
	}
}

func parseAttemptID(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" {
			continue
		}
		if evidence.UUIDPattern.MatchString(candidate) {
			return candidate, nil
		}
		return "", fmt.Errorf("database returned an invalid migration attempt ID")
	}
	return "", fmt.Errorf("database did not return a migration attempt ID")
}

func markFailed(ctx context.Context, runner command.Runner, settings config.TargetSettings, attemptID string) error {
	sql, err := evidence.FailAttemptSQL(attemptID)
	if err != nil {
		return err
	}
	_, err = runPSQL(ctx, runner, settings, sql)
	return err
}

func resolveRepositoryRoot(sourceRoot string) (string, error) {
	if strings.TrimSpace(sourceRoot) == "" {
		return "", fmt.Errorf("an explicit source root is required")
	}
	abs, err := filepath.Abs(sourceRoot)
	if err != nil {
		return "", fmt.Errorf("resolve source root")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve repository source root")
	}
	infraMigrations := filepath.Join(resolved, "src", "brain-engine", "features", "infra", "postgres", "migrations")
	info, err := os.Stat(infraMigrations)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("source root must be the repository root containing infra and feature migration catalogs")
	}
	return resolved, nil
}

func parseLedger(raw string) (migrations.Ledger, error) {
	ledger := make(migrations.Ledger)
	if strings.TrimSpace(raw) == "" {
		return ledger, nil
	}
	for _, row := range strings.Split(raw, ",") {
		parts := strings.Split(row, ":")
		if len(parts) != 2 || !regexp.MustCompile(`^[0-9]{4}$`).MatchString(parts[0]) || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(parts[1]) {
			return nil, fmt.Errorf("preflight returned an invalid migration ledger record")
		}
		if _, exists := ledger[parts[0]]; exists {
			return nil, fmt.Errorf("preflight returned a duplicate migration ledger record")
		}
		ledger[parts[0]] = parts[1]
	}
	return ledger, nil
}

func vectorVersionCompatible(version string) bool {
	match := regexp.MustCompile(`^0\.([0-9]+)\.([0-9]+)$`).FindStringSubmatch(strings.TrimSpace(version))
	if len(match) != 3 {
		return false
	}
	minor, err := strconv.Atoi(match[1])
	return err == nil && minor >= 8
}

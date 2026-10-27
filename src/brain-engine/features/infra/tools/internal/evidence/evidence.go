// Package evidence defines the sanitized, target-specific records shared by
// preflight and migration application. It never stores credentials or raw
// child-process output.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
)

var (
	referencePattern       = regexp.MustCompile(`^preflight/(mindforge_develop|mindforge)/[0-9a-f]{64}$`)
	verificationPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{2,255}$`)
	UUIDPattern            = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	artifactNamePattern    = regexp.MustCompile(`^mindforge-infra-(mindforge_develop|mindforge)-[A-Za-z0-9][A-Za-z0-9._-]{0,127}\.dump$`)
	sha256Pattern          = regexp.MustCompile(`^[0-9a-f]{64}$`)
	errorClassPattern      = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	backupReferencePattern = regexp.MustCompile(`^backup/(mindforge_develop|mindforge)/[0-9a-f]{64}$`)
)

const backupEvidenceSchemaVersion = 1

// BackupManifest is the portable, target-bound evidence shipped next to a
// validated custom-format dump. It deliberately contains no connection
// password, process output, or restore claim.
type BackupManifest struct {
	SchemaVersion          int    `json:"schema_version"`
	Operation              string `json:"operation"`
	Target                 string `json:"target"`
	Database               string `json:"database"`
	OperatorID             string `json:"operator_id"`
	AuthorizationReference string `json:"authorization_reference"`
	ArtifactName           string `json:"artifact_name"`
	ArtifactSHA256         string `json:"artifact_sha256"`
	ArtifactBytes          int64  `json:"artifact_bytes"`
	ArchiveListingVerified bool   `json:"archive_listing_verified"`
	RestoreProof           string `json:"restore_proof"`
	CreatedAt              string `json:"created_at"`
}

// RestoreEvidence records the sanitized outcome of one guarded restore
// attempt. A successful client command is not represented as fresh restore
// proof for this revision.
type RestoreEvidence struct {
	SchemaVersion          int    `json:"schema_version"`
	Operation              string `json:"operation"`
	Status                 string `json:"status"`
	Target                 string `json:"target"`
	Database               string `json:"database"`
	OperatorID             string `json:"operator_id"`
	AuthorizationReference string `json:"authorization_reference"`
	ArtifactName           string `json:"artifact_name"`
	ArtifactSHA256         string `json:"artifact_sha256"`
	ArchiveListingVerified bool   `json:"archive_listing_verified"`
	DestinationVerified    bool   `json:"destination_verified"`
	RestoreProof           string `json:"restore_proof"`
	ErrorClass             string `json:"error_class,omitempty"`
	CreatedAt              string `json:"created_at"`
}

// PromotionMigration is the immutable source identity and terminal status
// recorded for one migration considered by a promotion attempt.
type PromotionMigration struct {
	ID     string `json:"migration_id"`
	SHA256 string `json:"migration_sha256"`
	Status string `json:"status"`
}

// PromotionEvidence links the develop verification, main preflights, fresh
// main backup and target-specific application outcome. It is written outside
// the migration transaction so a failed promotion remains diagnosable.
type PromotionEvidence struct {
	SchemaVersion                 int                  `json:"schema_version"`
	Operation                     string               `json:"operation"`
	Status                        string               `json:"status"`
	Target                        string               `json:"target"`
	Database                      string               `json:"database"`
	OperatorID                    string               `json:"operator_id"`
	AuthorizationReference        string               `json:"authorization_reference"`
	DevelopPreflightReference     string               `json:"develop_preflight_reference,omitempty"`
	MainPreflightReference        string               `json:"main_preflight_reference,omitempty"`
	MainRecheckReference          string               `json:"main_recheck_reference,omitempty"`
	BackupReference               string               `json:"backup_reference,omitempty"`
	BackupArtifactName            string               `json:"backup_artifact_name,omitempty"`
	BackupManifestName            string               `json:"backup_manifest_name,omitempty"`
	BackupSHA256                  string               `json:"backup_sha256,omitempty"`
	BehaviorAccessVerificationRef string               `json:"behavior_access_verification_reference"`
	PostApplicationResult         string               `json:"post_application_result"`
	Migrations                    []PromotionMigration `json:"migrations"`
	ErrorClass                    string               `json:"error_class,omitempty"`
	CreatedAt                     string               `json:"created_at"`
}

// Preflight is a read-only snapshot. SourceRoot and CatalogHash bind an
// application to the exact local migration source inspected before DDL.
type Preflight struct {
	Target              config.Target
	Reference           string
	SourceRoot          string
	CatalogHash         string
	ObservedState       string
	Ledger              migrations.Ledger
	InfrastructureReady bool
}

// AppliedMigration is the sanitized outcome of one migration attempt.
type AppliedMigration struct {
	ID      string
	SHA256  string
	Target  config.Target
	Attempt string
}

// NewReference binds target, source catalog and read-only observed state to a
// deterministic reference. The state is hashed before it is exposed.
func NewReference(target config.Target, catalogHash, observedState string) (string, error) {
	if _, err := config.ParseTarget(string(target)); err != nil {
		return "", err
	}
	if catalogHash == "" {
		return "", fmt.Errorf("catalog hash is required")
	}
	digest := sha256.Sum256([]byte(string(target) + "|" + catalogHash + "|" + observedState))
	return "preflight/" + string(target) + "/" + hex.EncodeToString(digest[:]), nil
}

// ValidateReference accepts only a reference produced by NewReference.
func ValidateReference(reference string) error {
	if !referencePattern.MatchString(reference) {
		return fmt.Errorf("a valid preflight reference is required")
	}
	return nil
}

// ValidateVerificationReference rejects values that could carry secrets,
// shell syntax or an ambiguous empty marker into evidence.
func ValidateVerificationReference(reference string) error {
	if !verificationPattern.MatchString(reference) {
		return fmt.Errorf("a sanitized behavior/access verification reference is required")
	}
	return nil
}

// ValidateReplacementConfirmation requires a target-bound confirmation token
// separate from the owner authorization reference.
func ValidateReplacementConfirmation(target config.Target, confirmation string) error {
	if _, err := config.ParseTarget(string(target)); err != nil {
		return err
	}
	prefix := "replace/" + string(target) + "/"
	if !strings.HasPrefix(confirmation, prefix) || !verificationPattern.MatchString(confirmation) {
		return fmt.Errorf("an explicit target-bound replacement confirmation is required")
	}
	return nil
}

// NewBackupManifest builds sanitized evidence for a completed backup check.
func NewBackupManifest(target config.Target, operatorID, authorizationReference, artifactName, artifactSHA256 string, artifactBytes int64, createdAt string) (BackupManifest, error) {
	manifest := BackupManifest{
		SchemaVersion:          backupEvidenceSchemaVersion,
		Operation:              "backup-check",
		Target:                 string(target),
		Database:               string(target),
		OperatorID:             operatorID,
		AuthorizationReference: authorizationReference,
		ArtifactName:           artifactName,
		ArtifactSHA256:         artifactSHA256,
		ArtifactBytes:          artifactBytes,
		ArchiveListingVerified: true,
		RestoreProof:           "not_verified",
		CreatedAt:              createdAt,
	}
	if err := ValidateBackupManifest(manifest); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

// ValidateBackupManifest rejects a manifest that can bind a dump to another
// target, a path, an altered hash, or a claimed restore proof.
func ValidateBackupManifest(manifest BackupManifest) error {
	if manifest.SchemaVersion != backupEvidenceSchemaVersion || manifest.Operation != "backup-check" {
		return fmt.Errorf("backup manifest schema or operation is invalid")
	}
	if _, err := config.ParseTarget(manifest.Target); err != nil {
		return err
	}
	if manifest.Database != manifest.Target {
		return fmt.Errorf("backup manifest database does not match target")
	}
	if !verificationPattern.MatchString(manifest.OperatorID) || !verificationPattern.MatchString(manifest.AuthorizationReference) {
		return fmt.Errorf("backup manifest authority is invalid")
	}
	if !artifactNamePattern.MatchString(manifest.ArtifactName) {
		return fmt.Errorf("backup manifest artifact name is invalid")
	}
	if !strings.HasPrefix(manifest.ArtifactName, "mindforge-infra-"+manifest.Target+"-") {
		return fmt.Errorf("backup manifest artifact target does not match manifest target")
	}
	if !sha256Pattern.MatchString(manifest.ArtifactSHA256) || manifest.ArtifactBytes <= 0 {
		return fmt.Errorf("backup manifest artifact hash or size is invalid")
	}
	if !manifest.ArchiveListingVerified || manifest.RestoreProof != "not_verified" {
		return fmt.Errorf("backup manifest does not contain a completed archive check")
	}
	if _, err := time.Parse(time.RFC3339, manifest.CreatedAt); err != nil {
		return fmt.Errorf("backup manifest timestamp is invalid")
	}
	return nil
}

// ReadBackupManifest loads and validates one small JSON evidence file.
func ReadBackupManifest(path string) (BackupManifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return BackupManifest{}, fmt.Errorf("read backup manifest")
	}
	if len(contents) > 64*1024 {
		return BackupManifest{}, fmt.Errorf("backup manifest is too large")
	}
	var manifest BackupManifest
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return BackupManifest{}, fmt.Errorf("decode backup manifest")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return BackupManifest{}, fmt.Errorf("backup manifest contains trailing data")
	}
	if err := ValidateBackupManifest(manifest); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

// WriteBackupManifest writes private JSON evidence. Callers that need an
// atomic publication should write a temporary path and rename it themselves.
func WriteBackupManifest(path string, manifest BackupManifest) error {
	if err := ValidateBackupManifest(manifest); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode backup manifest")
	}
	contents = append(contents, '\n')
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return fmt.Errorf("write backup manifest")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect backup manifest")
	}
	return nil
}

// NewRestoreEvidence creates an outcome record without persisting a raw
// confirmation token or child-process output.
func NewRestoreEvidence(target config.Target, operatorID, authorizationReference, artifactName, artifactSHA256, status string, archiveListingVerified, destinationVerified bool, errorClass, createdAt string) (RestoreEvidence, error) {
	evidence := RestoreEvidence{
		SchemaVersion:          backupEvidenceSchemaVersion,
		Operation:              "restore",
		Status:                 status,
		Target:                 string(target),
		Database:               string(target),
		OperatorID:             operatorID,
		AuthorizationReference: authorizationReference,
		ArtifactName:           artifactName,
		ArtifactSHA256:         artifactSHA256,
		ArchiveListingVerified: archiveListingVerified,
		DestinationVerified:    destinationVerified,
		RestoreProof:           "not_claimed",
		ErrorClass:             errorClass,
		CreatedAt:              createdAt,
	}
	if err := ValidateRestoreEvidence(evidence); err != nil {
		return RestoreEvidence{}, err
	}
	return evidence, nil
}

// ValidateRestoreEvidence accepts only sanitized terminal or in-progress
// restore states.
func ValidateRestoreEvidence(evidence RestoreEvidence) error {
	if evidence.SchemaVersion != backupEvidenceSchemaVersion || evidence.Operation != "restore" {
		return fmt.Errorf("restore evidence schema or operation is invalid")
	}
	if evidence.Status != "attempted" && evidence.Status != "succeeded" && evidence.Status != "failed" {
		return fmt.Errorf("restore evidence status is invalid")
	}
	if _, err := config.ParseTarget(evidence.Target); err != nil {
		return err
	}
	if evidence.Database != evidence.Target || !verificationPattern.MatchString(evidence.OperatorID) || !verificationPattern.MatchString(evidence.AuthorizationReference) {
		return fmt.Errorf("restore evidence target or authority is invalid")
	}
	if !artifactNamePattern.MatchString(evidence.ArtifactName) || !sha256Pattern.MatchString(evidence.ArtifactSHA256) {
		return fmt.Errorf("restore evidence artifact identity is invalid")
	}
	if evidence.ErrorClass != "" && !errorClassPattern.MatchString(evidence.ErrorClass) {
		return fmt.Errorf("restore evidence error class is invalid")
	}
	if evidence.Status == "failed" && evidence.ErrorClass == "" {
		return fmt.Errorf("failed restore evidence requires an error class")
	}
	if evidence.Status != "failed" && evidence.ErrorClass != "" {
		return fmt.Errorf("non-failed restore evidence cannot contain an error class")
	}
	if evidence.RestoreProof != "not_claimed" || (evidence.Status == "succeeded" && !evidence.DestinationVerified) {
		return fmt.Errorf("restore evidence proof boundary is invalid")
	}
	if _, err := time.Parse(time.RFC3339, evidence.CreatedAt); err != nil {
		return fmt.Errorf("restore evidence timestamp is invalid")
	}
	return nil
}

// WriteRestoreEvidence writes private JSON restore outcome evidence.
func WriteRestoreEvidence(path string, evidence RestoreEvidence) error {
	if err := ValidateRestoreEvidence(evidence); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fmt.Errorf("encode restore evidence")
	}
	contents = append(contents, '\n')
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return fmt.Errorf("write restore evidence")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect restore evidence")
	}
	return nil
}

// NewBackupReference binds a validated backup to its explicit target without
// persisting a machine-specific path or a credential.
func NewBackupReference(target config.Target, artifactSHA256 string) (string, error) {
	if _, err := config.ParseTarget(string(target)); err != nil {
		return "", err
	}
	if !sha256Pattern.MatchString(artifactSHA256) {
		return "", fmt.Errorf("backup hash must be SHA-256")
	}
	return "backup/" + string(target) + "/" + artifactSHA256, nil
}

// ValidateBackupReference accepts only a target-bound backup identity.
func ValidateBackupReference(reference string) error {
	if !backupReferencePattern.MatchString(reference) {
		return fmt.Errorf("a valid target-bound backup reference is required")
	}
	return nil
}

// ValidatePromotionEvidence rejects promotion records that omit the checks
// needed to understand whether main was changed or left untouched.
func ValidatePromotionEvidence(record PromotionEvidence) error {
	if record.SchemaVersion != backupEvidenceSchemaVersion || record.Operation != "promote" {
		return fmt.Errorf("promotion evidence schema or operation is invalid")
	}
	if record.Status != "attempted" && record.Status != "succeeded" && record.Status != "failed" {
		return fmt.Errorf("promotion evidence status is invalid")
	}
	if record.Target != string(config.TargetMain) || record.Database != string(config.TargetMain) {
		return fmt.Errorf("promotion evidence must target mindforge")
	}
	if !verificationPattern.MatchString(record.OperatorID) || !verificationPattern.MatchString(record.AuthorizationReference) || !verificationPattern.MatchString(record.BehaviorAccessVerificationRef) {
		return fmt.Errorf("promotion evidence authority or verification reference is invalid")
	}
	for _, reference := range []string{record.DevelopPreflightReference, record.MainPreflightReference, record.MainRecheckReference} {
		if reference != "" {
			if err := ValidateReference(reference); err != nil {
				return err
			}
		}
	}
	if record.BackupReference == "" {
		if record.BackupArtifactName != "" || record.BackupManifestName != "" || record.BackupSHA256 != "" {
			return fmt.Errorf("promotion backup identity is incomplete")
		}
	} else {
		if err := ValidateBackupReference(record.BackupReference); err != nil {
			return err
		}
		if !artifactNamePattern.MatchString(record.BackupArtifactName) || !strings.HasPrefix(record.BackupArtifactName, "mindforge-infra-mindforge-") {
			return fmt.Errorf("promotion backup artifact is invalid")
		}
		if record.BackupManifestName != record.BackupArtifactName+".manifest.json" || !sha256Pattern.MatchString(record.BackupSHA256) {
			return fmt.Errorf("promotion backup manifest identity is invalid")
		}
		if !strings.HasSuffix(record.BackupReference, "/"+record.BackupSHA256) {
			return fmt.Errorf("promotion backup reference does not match its hash")
		}
	}
	if record.PostApplicationResult != "not_started" && record.PostApplicationResult != "succeeded" && record.PostApplicationResult != "failed" && record.PostApplicationResult != "inconclusive" {
		return fmt.Errorf("promotion application result is invalid")
	}
	seen := make(map[string]struct{}, len(record.Migrations))
	for _, migration := range record.Migrations {
		if !regexp.MustCompile(`^[0-9]{4}$`).MatchString(migration.ID) || !sha256Pattern.MatchString(migration.SHA256) {
			return fmt.Errorf("promotion migration identity is invalid")
		}
		if migration.Status != "pending" && migration.Status != "applied" && migration.Status != "already_applied" && migration.Status != "failed" {
			return fmt.Errorf("promotion migration status is invalid")
		}
		if _, exists := seen[migration.ID]; exists {
			return fmt.Errorf("promotion migration identity is duplicated")
		}
		seen[migration.ID] = struct{}{}
	}
	if record.Status == "failed" {
		if !errorClassPattern.MatchString(record.ErrorClass) {
			return fmt.Errorf("failed promotion evidence requires an error class")
		}
	} else if record.ErrorClass != "" {
		return fmt.Errorf("non-failed promotion evidence cannot contain an error class")
	}
	if record.Status == "succeeded" {
		if record.DevelopPreflightReference == "" || record.MainPreflightReference == "" || record.MainRecheckReference == "" || record.BackupReference == "" || record.PostApplicationResult != "succeeded" {
			return fmt.Errorf("successful promotion evidence is incomplete")
		}
		for _, migration := range record.Migrations {
			if migration.Status != "applied" && migration.Status != "already_applied" {
				return fmt.Errorf("successful promotion evidence contains an incomplete migration")
			}
		}
	}
	if _, err := time.Parse(time.RFC3339, record.CreatedAt); err != nil {
		return fmt.Errorf("promotion evidence timestamp is invalid")
	}
	return nil
}

// WritePromotionEvidence writes private promotion outcome evidence. The
// caller may publish a temporary path atomically before replacing the final
// record, preserving the last diagnosable state on failure.
func WritePromotionEvidence(path string, record PromotionEvidence) error {
	if err := ValidatePromotionEvidence(record); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode promotion evidence")
	}
	contents = append(contents, '\n')
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return fmt.Errorf("write promotion evidence")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect promotion evidence")
	}
	return nil
}

// InsertAttemptSQL creates an incomplete evidence record before migration DDL.
func InsertAttemptSQL(migrationID, migrationHash string, target config.Target, operatorID, authorizationReference, preflightReference, verificationReference string) (string, error) {
	if err := validateMigrationFields(migrationID, migrationHash, target, operatorID, authorizationReference, preflightReference, verificationReference); err != nil {
		return "", err
	}
	return "INSERT INTO infra.migration_attempts (migration_id, migration_sha256, target_name, operator_id, authorization_reference, preflight_reference, post_application_result, behavior_access_verification_reference, sanitized_outcome) VALUES (" +
		quote(migrationID) + "," + quote(migrationHash) + "," + quote(string(target)) + "," + quote(operatorID) + "," + quote(authorizationReference) + "," + quote(preflightReference) + ",'not_started'," + quote(verificationReference) + ",'incomplete') RETURNING attempt_id::text;", nil
}

// CompleteAttemptSQL closes an attempt and copies its final state to the
// target evidence relation. The SQL contains only allowlisted identifiers and
// sanitized metadata; it does not include a password or command output.
func CompleteAttemptSQL(attemptID string) (string, error) {
	if !UUIDPattern.MatchString(attemptID) {
		return "", fmt.Errorf("database returned an invalid migration attempt ID")
	}
	quoted := quote(attemptID)
	return "UPDATE infra.migration_attempts SET post_application_result='succeeded', sanitized_outcome='succeeded', completed_at=now() WHERE attempt_id=" + quoted + "; INSERT INTO infra.migration_target_evidence (migration_id, migration_sha256, target_name, operator_id, authorization_reference, preflight_reference, post_application_result, behavior_access_verification_reference, sanitized_outcome, started_at, completed_at) SELECT migration_id, migration_sha256, target_name, operator_id, authorization_reference, preflight_reference, post_application_result, behavior_access_verification_reference, sanitized_outcome, started_at, completed_at FROM infra.migration_attempts WHERE attempt_id=" + quoted + ";", nil
}

// FailAttemptSQL preserves failure evidence outside the migration transaction.
func FailAttemptSQL(attemptID string) (string, error) {
	if !UUIDPattern.MatchString(attemptID) {
		return "", fmt.Errorf("database returned an invalid migration attempt ID")
	}
	return "UPDATE infra.migration_attempts SET post_application_result='failed', sanitized_outcome='failed', completed_at=now() WHERE attempt_id=" + quote(attemptID) + ";", nil
}

func validateMigrationFields(migrationID, migrationHash string, target config.Target, operatorID, authorizationReference, preflightReference, verificationReference string) error {
	if !regexp.MustCompile(`^[0-9]{4}$`).MatchString(migrationID) {
		return fmt.Errorf("migration ID must be canonical")
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(migrationHash) {
		return fmt.Errorf("migration hash must be SHA-256")
	}
	if _, err := config.ParseTarget(string(target)); err != nil {
		return err
	}
	if !verificationPattern.MatchString(operatorID) || !verificationPattern.MatchString(authorizationReference) {
		return fmt.Errorf("operator and authorization references must be sanitized")
	}
	if err := ValidateReference(preflightReference); err != nil {
		return err
	}
	return ValidateVerificationReference(verificationReference)
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

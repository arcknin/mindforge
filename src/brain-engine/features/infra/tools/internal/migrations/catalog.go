// Package migrations owns local, static inspection of infrastructure and
// approved feature migration catalogs. It never connects to PostgreSQL.
package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Kind separates infra-owned bookkeeping from feature-owned SQL.
type Kind string

const (
	KindInfrastructure Kind = "infrastructure"
	KindFeature        Kind = "feature"
)

var (
	filenamePattern = regexp.MustCompile(`^([0-9]{4})_([a-z0-9][a-z0-9_-]*)\.sql$`)
	featurePattern  = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	digestPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

const (
	infrastructureMigrationsPath = "src/brain-engine/features/infra/postgres/migrations"
	featureCatalogPath           = "src/brain-engine/features/infra/postgres/feature-migrations"
	featureDescriptorName        = "catalog.json"
)

// Filename is the parsed canonical identity of one migration file.
type Filename struct {
	ID   string
	Name string
}

// Migration identifies immutable source bytes in the global catalog. Feature
// checks are feature-owned and their hashes bind the operator preflight and
// transactional verification to this catalog entry.
type Migration struct {
	ID                 string
	Name               string
	Path               string
	RelativePath       string
	Kind               Kind
	SHA256             string
	Feature            string
	PreflightPath      string
	PreflightRelative  string
	PreflightSHA256    string
	VerificationPath   string
	VerificationRel    string
	VerificationSHA256 string
}

// Catalog is the globally ordered union of infrastructure and feature SQL.
// Root is the canonical repository root used to resolve every source.
type Catalog struct {
	Root       string
	Migrations []Migration
}

// Ledger stores applied migration hashes. A later database-backed ledger can
// satisfy the same invariant without changing local catalog parsing.
type Ledger map[string]string

type featureDescriptor struct {
	SchemaVersion int              `json:"schema_version"`
	Migrations    []featureMapping `json:"migrations"`
}

type featureMapping struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Feature         string `json:"feature"`
	Source          string `json:"source"`
	SHA256          string `json:"sha256"`
	Preflight       string `json:"preflight"`
	PreflightSHA256 string `json:"preflight_sha256"`
	Verify          string `json:"verify"`
	VerifySHA256    string `json:"verify_sha256"`
}

// ParseFilename accepts one zero-padded numeric migration ID and safe slug.
func ParseFilename(name string) (Filename, error) {
	matches := filenamePattern.FindStringSubmatch(name)
	if matches == nil {
		return Filename{}, fmt.Errorf("noncanonical migration filename %q", name)
	}
	return Filename{ID: matches[1], Name: matches[2]}, nil
}

// LoadCatalog resolves all SQL paths from an explicit repository root, checks
// the feature descriptor and hashes, and rejects paths escaping their owner
// feature subtree or the repository through symlinks.
func LoadCatalog(repositoryRoot string) (Catalog, error) {
	canonicalRoot, err := canonicalDirectory(repositoryRoot)
	if err != nil {
		return Catalog{}, err
	}
	infra, err := loadInfrastructureDirectory(canonicalRoot)
	if err != nil {
		return Catalog{}, err
	}
	feature, err := loadFeatureDescriptor(canonicalRoot)
	if err != nil {
		return Catalog{}, err
	}
	migrations := append(infra, feature...)
	sort.Slice(migrations, func(left, right int) bool {
		return migrations[left].ID < migrations[right].ID
	})
	for index, migration := range migrations {
		expected := index + 1
		actual, _ := strconv.Atoi(migration.ID)
		if actual != expected {
			return Catalog{}, fmt.Errorf("migration IDs must be globally contiguous from 0001; found %s where %04d is required", migration.ID, expected)
		}
		if index > 0 && migrations[index-1].ID == migration.ID {
			return Catalog{}, fmt.Errorf("duplicate global migration ID %s", migration.ID)
		}
	}
	return Catalog{Root: canonicalRoot, Migrations: migrations}, nil
}

// NewStaticLedger captures hashes for a fake-backed or local static check.
func NewStaticLedger(catalog Catalog) Ledger {
	ledger := make(Ledger, len(catalog.Migrations))
	for _, migration := range catalog.Migrations {
		ledger[migration.ID] = migration.SHA256
	}
	return ledger
}

// CatalogHash returns a stable digest of global identities, canonical source
// bytes and feature-owned check identities. It is safe to keep in a preflight
// reference because it contains no SQL content or secrets.
func CatalogHash(catalog Catalog) string {
	hash := sha256.New()
	for _, migration := range catalog.Migrations {
		fmt.Fprintf(hash, "%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n",
			migration.ID,
			migration.Name,
			migration.Kind,
			migration.SHA256,
			migration.Feature,
			migration.RelativePath,
			migration.PreflightRelative,
			migration.PreflightSHA256,
			migration.VerificationRel,
			migration.VerificationSHA256,
		)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// VerifyLedger rejects a missing canonical source file or changed bytes for
// every applied ledger entry. It permits pending files not present in ledger.
func VerifyLedger(catalog Catalog, ledger Ledger) error {
	byID := make(map[string]Migration, len(catalog.Migrations))
	for _, migration := range catalog.Migrations {
		byID[migration.ID] = migration
	}
	for id, expectedHash := range ledger {
		migration, ok := byID[id]
		if !ok {
			return fmt.Errorf("applied migration %s has no canonical source file", id)
		}
		if expectedHash != migration.SHA256 {
			return fmt.Errorf("applied migration %s has changed bytes", id)
		}
	}
	return nil
}

// VerifyExactCatalog rejects extra, absent or changed entries against an
// immutable local ledger baseline.
func VerifyExactCatalog(catalog Catalog, ledger Ledger) error {
	if err := VerifyLedger(catalog, ledger); err != nil {
		return err
	}
	if len(catalog.Migrations) != len(ledger) {
		return fmt.Errorf("catalog does not match the immutable local ledger")
	}
	return nil
}

func canonicalDirectory(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("inspect repository root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root must be a directory")
	}
	return resolved, nil
}

func loadInfrastructureDirectory(root string) ([]Migration, error) {
	directory, err := resolveUnderRoot(root, filepath.Join(root, filepath.FromSlash(infrastructureMigrationsPath)))
	if err != nil {
		return nil, fmt.Errorf("resolve infrastructure migration catalog: %w", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read infrastructure migration catalog: %w", err)
	}
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			return nil, fmt.Errorf("infrastructure migration catalog contains a nested directory")
		}
		filename, err := ParseFilename(name)
		if err != nil {
			return nil, err
		}
		relative := filepath.ToSlash(filepath.Join(infrastructureMigrationsPath, name))
		path, digest, err := readPinnedFile(root, relative, "")
		if err != nil {
			return nil, fmt.Errorf("inspect infrastructure migration %s: %w", name, err)
		}
		migrations = append(migrations, Migration{
			ID:           filename.ID,
			Name:         filename.Name,
			Path:         path,
			RelativePath: relative,
			Kind:         KindInfrastructure,
			SHA256:       digest,
		})
	}
	return migrations, nil
}

func loadFeatureDescriptor(root string) ([]Migration, error) {
	directoryRelative := featureCatalogPath
	directory, err := resolveUnderRoot(root, filepath.Join(root, filepath.FromSlash(directoryRelative)))
	if err != nil {
		return nil, fmt.Errorf("resolve feature migration catalog: %w", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read feature migration catalog: %w", err)
	}
	descriptorSeen := false
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("feature migration catalog contains a nested directory")
		}
		switch entry.Name() {
		case ".gitkeep", "README.md":
		case featureDescriptorName:
			descriptorSeen = true
		default:
			return nil, fmt.Errorf("feature migration catalog contains unexpected file %q", entry.Name())
		}
	}
	if !descriptorSeen {
		return nil, fmt.Errorf("feature migration catalog descriptor is missing")
	}
	descriptorRelative := filepath.ToSlash(filepath.Join(directoryRelative, featureDescriptorName))
	descriptorPath, _, err := readPinnedFile(root, descriptorRelative, "")
	if err != nil {
		return nil, fmt.Errorf("inspect feature migration descriptor: %w", err)
	}
	file, err := os.Open(descriptorPath)
	if err != nil {
		return nil, fmt.Errorf("open feature migration descriptor: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var descriptor featureDescriptor
	if err := decoder.Decode(&descriptor); err != nil {
		return nil, fmt.Errorf("decode feature migration descriptor: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("feature migration descriptor has trailing data")
	}
	if descriptor.SchemaVersion != 1 || descriptor.Migrations == nil {
		return nil, fmt.Errorf("feature migration descriptor schema is invalid")
	}
	migrations := make([]Migration, 0, len(descriptor.Migrations))
	for _, mapping := range descriptor.Migrations {
		migration, err := loadFeatureMapping(root, mapping)
		if err != nil {
			return nil, fmt.Errorf("feature migration %s: %w", mapping.ID, err)
		}
		migrations = append(migrations, migration)
	}
	return migrations, nil
}

func loadFeatureMapping(root string, mapping featureMapping) (Migration, error) {
	if !regexp.MustCompile(`^[0-9]{4}$`).MatchString(mapping.ID) || mapping.Name == "" || !featurePattern.MatchString(mapping.Feature) {
		return Migration{}, fmt.Errorf("global identity or feature name is invalid")
	}
	sourceRelative, err := validateFeatureRelative(mapping.Source, mapping.Feature)
	if err != nil {
		return Migration{}, fmt.Errorf("source path is unsafe: %w", err)
	}
	filename, err := ParseFilename(filepath.Base(sourceRelative))
	if err != nil || filename.Name != mapping.Name {
		return Migration{}, fmt.Errorf("canonical feature source name does not match the descriptor")
	}
	base := strings.TrimSuffix(sourceRelative, ".sql")
	preflightRelative, err := validateFeatureRelative(mapping.Preflight, mapping.Feature)
	if err != nil || preflightRelative != base+".preflight.sql" {
		return Migration{}, fmt.Errorf("read-only preflight check path is missing or unsafe")
	}
	verifyRelative, err := validateFeatureRelative(mapping.Verify, mapping.Feature)
	if err != nil || verifyRelative != base+".verify.sql" {
		return Migration{}, fmt.Errorf("schema verification check path is missing or unsafe")
	}
	sourcePath, sourceHash, err := readPinnedFile(root, sourceRelative, mapping.SHA256)
	if err != nil {
		return Migration{}, fmt.Errorf("canonical source is missing or changed: %w", err)
	}
	if err := requireWithinFeature(root, mapping.Feature, sourcePath); err != nil {
		return Migration{}, fmt.Errorf("canonical source escapes its registered feature: %w", err)
	}
	preflightPath, preflightHash, err := readPinnedFile(root, preflightRelative, mapping.PreflightSHA256)
	if err != nil {
		return Migration{}, fmt.Errorf("preflight check is missing or changed: %w", err)
	}
	if err := requireWithinFeature(root, mapping.Feature, preflightPath); err != nil {
		return Migration{}, fmt.Errorf("preflight check escapes its registered feature: %w", err)
	}
	verifyPath, verifyHash, err := readPinnedFile(root, verifyRelative, mapping.VerifySHA256)
	if err != nil {
		return Migration{}, fmt.Errorf("verification check is missing or changed: %w", err)
	}
	if err := requireWithinFeature(root, mapping.Feature, verifyPath); err != nil {
		return Migration{}, fmt.Errorf("verification check escapes its registered feature: %w", err)
	}
	return Migration{
		ID:                 mapping.ID,
		Name:               mapping.Name,
		Path:               sourcePath,
		RelativePath:       sourceRelative,
		Kind:               KindFeature,
		SHA256:             sourceHash,
		Feature:            mapping.Feature,
		PreflightPath:      preflightPath,
		PreflightRelative:  preflightRelative,
		PreflightSHA256:    preflightHash,
		VerificationPath:   verifyPath,
		VerificationRel:    verifyRelative,
		VerificationSHA256: verifyHash,
	}, nil
}

func requireWithinFeature(root, feature, path string) error {
	featureDirectory := filepath.Join(root, "src", "brain-engine", "features", feature, "migrations")
	resolvedFeatureDirectory, err := filepath.EvalSymlinks(featureDirectory)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(resolvedFeatureDirectory, path)
	if err != nil {
		return err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("resolved source is outside the feature migration directory")
	}
	return nil
}

func validateFeatureRelative(path, feature string) (string, error) {
	if path == "" || strings.Contains(path, `\`) || filepath.IsAbs(path) || filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) != path {
		return "", fmt.Errorf("path must be a normalized repository-relative path")
	}
	prefix := "src/brain-engine/features/" + feature + "/migrations/"
	if !strings.HasPrefix(path, prefix) || strings.Contains(strings.TrimPrefix(path, prefix), "/") || strings.HasSuffix(path, "/") {
		return "", fmt.Errorf("path must be a direct child of the registered feature migration directory")
	}
	if strings.HasPrefix(filepath.Base(path), ".") {
		return "", fmt.Errorf("hidden feature source paths are not allowed")
	}
	return path, nil
}

func readPinnedFile(root, relative, expectedHash string) (string, string, error) {
	path, err := resolveUnderRoot(root, filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("path is not a regular file")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(bytes)
	actualHash := hex.EncodeToString(digest[:])
	if expectedHash != "" {
		if !digestPattern.MatchString(expectedHash) || actualHash != expectedHash {
			return "", "", fmt.Errorf("bytes do not match the descriptor SHA-256")
		}
	}
	return path, actualHash, nil
}

func resolveUnderRoot(root, path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("path is outside repository root")
	}
	return resolved, nil
}

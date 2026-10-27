// mindforge-infra is the distributable infrastructure operator tool.
// Local audit remains database-free; B03 and B04 add guarded target-specific
// migration, backup and restore operations through explicit PostgreSQL clients.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/audit"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/migrations"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/operations"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		printUsage(stderr)
		return 2
	}
	switch arguments[0] {
	case "config":
		return runConfig(arguments[1:], stdout, stderr)
	case "audit":
		return runAudit(arguments[1:], stdout, stderr)
	case "preflight":
		return runPreflight(arguments[1:], stdout, stderr)
	case "apply":
		return runApply(arguments[1:], stdout, stderr)
	case "backup":
		return runBackup(arguments[1:], stdout, stderr)
	case "restore":
		return runRestore(arguments[1:], stdout, stderr)
	case "promote":
		return runPromote(arguments[1:], stdout, stderr)
	case "provision-develop":
		return runProvisionDevelop(arguments[1:], stdout, stderr)
	case "bootstrap-develop-extensions":
		return runBootstrapDevelopExtensions(arguments[1:], stdout, stderr)
	default:
		printUsage(stderr)
		return 2
	}
}

func runConfig(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || arguments[0] != "validate" {
		printUsage(stderr)
		return 2
	}
	flags := flag.NewFlagSet("config validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("config", "", "explicit private configuration path")
	if err := flags.Parse(arguments[1:]); err != nil || *path == "" || flags.NArg() != 0 {
		return 2
	}
	values, err := config.ParseFile(*path)
	if err == nil {
		err = values.ValidateOperationalShape()
	}
	if err != nil {
		fmt.Fprintln(stderr, "configuration validation failed")
		return 1
	}
	fmt.Fprintln(stdout, "configuration shape valid")
	return 0
}

func runAudit(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("audit", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourceRoot := flags.String("source-root", "", "repository source root")
	if err := flags.Parse(arguments); err != nil || *sourceRoot == "" || flags.NArg() != 0 {
		return 2
	}
	if err := audit.AuditSource(*sourceRoot, migrations.GlobalMigrationLedgerBaseline(), migrations.GlobalCatalogHashBaseline()); err != nil {
		fmt.Fprintln(stderr, "local static audit failed")
		return 1
	}
	fmt.Fprintln(stdout, "local static audit passed")
	return 0
}

type operationFlags struct {
	configPath       string
	sourceRoot       string
	target           string
	operatorID       string
	authorizationRef string
	scope            string
	preflightRef     string
	verificationRef  string
	allowBootstrap   bool
	outputDir        string
	backupFile       string
	manifestFile     string
	replacement      string
	evidenceDir      string
}

func addOperationFlags(flags *flag.FlagSet, values *operationFlags) {
	flags.StringVar(&values.configPath, "config", "", "explicit private configuration path")
	flags.StringVar(&values.sourceRoot, "source-root", "", "explicit repository root")
	flags.StringVar(&values.target, "target", "", "explicit contracted database target")
	flags.StringVar(&values.operatorID, "operator-id", "", "explicit operator identifier")
	flags.StringVar(&values.authorizationRef, "authorization-ref", "", "non-secret owner authorization reference")
	flags.StringVar(&values.scope, "scope", "", "migration scope")
	flags.StringVar(&values.preflightRef, "preflight-ref", "", "read-only preflight reference")
	flags.StringVar(&values.verificationRef, "verification-ref", "", "sanitized behavior/access verification reference")
	flags.BoolVar(&values.allowBootstrap, "allow-infrastructure-bootstrap", false, "allow incomplete infrastructure state for infrastructure setup")
	flags.StringVar(&values.outputDir, "output-dir", "", "existing external backup directory")
	flags.StringVar(&values.backupFile, "backup-file", "", "external backup artifact")
	flags.StringVar(&values.manifestFile, "manifest-file", "", "external environment-bound backup manifest")
	flags.StringVar(&values.replacement, "replacement-confirmation", "", "explicit target-bound replacement confirmation")
	flags.StringVar(&values.evidenceDir, "evidence-dir", "", "existing external restore evidence directory")
}

func parseOperation(values operationFlags) (config.Values, command.OperationRequest, error) {
	if values.configPath == "" || values.sourceRoot == "" || values.target == "" || values.operatorID == "" || values.authorizationRef == "" {
		return config.Values{}, command.OperationRequest{}, fmt.Errorf("config, source-root, target, operator-id and authorization-ref are required")
	}
	privateConfig, err := config.ParseFile(values.configPath)
	if err != nil {
		return config.Values{}, command.OperationRequest{}, fmt.Errorf("configuration validation failed")
	}
	if err := privateConfig.ValidateShape(); err != nil {
		return config.Values{}, command.OperationRequest{}, fmt.Errorf("configuration validation failed")
	}
	target, err := config.ParseTarget(values.target)
	if err != nil {
		return config.Values{}, command.OperationRequest{}, err
	}
	request, err := command.NewOperationRequest(target, values.operatorID, values.authorizationRef)
	if err != nil {
		return config.Values{}, command.OperationRequest{}, err
	}
	return privateConfig, request, nil
}

func runPreflight(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("preflight", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var values operationFlags
	addOperationFlags(flags, &values)
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	privateConfig, operationRequest, err := parseOperation(values)
	if err != nil || (values.scope != "" && values.scope != string(operations.ScopeInfrastructure) && values.scope != string(operations.ScopeFeature)) || (values.allowBootstrap && values.scope != string(operations.ScopeInfrastructure)) {
		fmt.Fprintln(stderr, "preflight arguments are invalid")
		return 2
	}
	result, err := operations.Preflight(context.Background(), operations.PreflightRequest{
		Config:                       privateConfig,
		Operation:                    operationRequest,
		Scope:                        operations.Scope(values.scope),
		SourceRoot:                   values.sourceRoot,
		Runner:                       command.ExecRunner{},
		AllowInfrastructureBootstrap: values.allowBootstrap,
	})
	if err != nil {
		fmt.Fprintln(stderr, "preflight failed")
		return 1
	}
	fmt.Fprintf(stdout, "preflight-reference=%s\n", result.Reference)
	return 0
}

func runApply(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("apply", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var values operationFlags
	addOperationFlags(flags, &values)
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	if values.scope != string(operations.ScopeInfrastructure) && values.scope != string(operations.ScopeFeature) || values.preflightRef == "" || values.verificationRef == "" {
		fmt.Fprintln(stderr, "apply requires scope, preflight-ref and verification-ref")
		return 2
	}
	privateConfig, operationRequest, err := parseOperation(values)
	if err != nil {
		fmt.Fprintln(stderr, "apply arguments are invalid")
		return 2
	}
	scope := operations.Scope(values.scope)
	preflight, err := operations.Preflight(context.Background(), operations.PreflightRequest{
		Config:                       privateConfig,
		Operation:                    operationRequest,
		Scope:                        scope,
		SourceRoot:                   values.sourceRoot,
		Runner:                       command.ExecRunner{},
		AllowInfrastructureBootstrap: scope == operations.ScopeInfrastructure,
	})
	if err != nil || preflight.Reference != values.preflightRef {
		fmt.Fprintln(stderr, "apply preflight failed or reference is stale")
		return 1
	}
	result, err := operations.Apply(context.Background(), operations.ApplyRequest{
		Config:                  privateConfig,
		Operation:               operationRequest,
		Scope:                   scope,
		SourceRoot:              values.sourceRoot,
		Runner:                  command.ExecRunner{},
		Preflight:               preflight,
		BehaviorVerificationRef: values.verificationRef,
	})
	if err != nil {
		fmt.Fprintln(stderr, "migration application failed")
		return 1
	}
	if len(result.Applied) == 0 {
		fmt.Fprintln(stdout, "applied=none")
		return 0
	}
	for _, applied := range result.Applied {
		fmt.Fprintf(stdout, "applied=%s target=%s\n", applied.ID, applied.Target)
	}
	return 0
}

func runBackup(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("backup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var values operationFlags
	addOperationFlags(flags, &values)
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || values.outputDir == "" {
		fmt.Fprintln(stderr, "backup requires an explicit output-dir")
		return 2
	}
	privateConfig, operationRequest, err := parseOperation(values)
	if err != nil {
		fmt.Fprintln(stderr, "backup arguments are invalid")
		return 2
	}
	result, err := operations.Backup(context.Background(), operations.BackupRequest{
		Config:     privateConfig,
		Operation:  operationRequest,
		SourceRoot: values.sourceRoot,
		OutputDir:  values.outputDir,
		Runner:     command.ExecRunner{},
	})
	if err != nil {
		fmt.Fprintln(stderr, "backup failed")
		return 1
	}
	fmt.Fprintf(stdout, "backup-artifact=%s\nbackup-manifest=%s\n", result.ArtifactPath, result.ManifestPath)
	return 0
}

func runRestore(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("restore", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var values operationFlags
	addOperationFlags(flags, &values)
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || values.backupFile == "" || values.manifestFile == "" || values.replacement == "" {
		fmt.Fprintln(stderr, "restore requires backup-file, manifest-file and replacement-confirmation")
		return 2
	}
	privateConfig, operationRequest, err := parseOperation(values)
	if err != nil {
		fmt.Fprintln(stderr, "restore arguments are invalid")
		return 2
	}
	result, err := operations.Restore(context.Background(), operations.RestoreRequest{
		Config:                  privateConfig,
		Operation:               operationRequest,
		SourceRoot:              values.sourceRoot,
		BackupFile:              values.backupFile,
		ManifestFile:            values.manifestFile,
		ReplacementConfirmation: values.replacement,
		EvidenceDir:             values.evidenceDir,
		Runner:                  command.ExecRunner{},
	})
	if err != nil {
		fmt.Fprintln(stderr, "restore failed")
		return 1
	}
	fmt.Fprintf(stdout, "restore-status=%s\nrestore-evidence=%s\n", result.Status, result.EvidencePath)
	return 0
}

func runPromote(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("promote", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var values operationFlags
	addOperationFlags(flags, &values)
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || values.outputDir == "" || values.verificationRef == "" {
		fmt.Fprintln(stderr, "promote requires output-dir and verification-ref")
		return 2
	}
	privateConfig, operationRequest, err := parseOperation(values)
	if err != nil {
		fmt.Fprintln(stderr, "promote arguments are invalid")
		return 2
	}
	result, err := operations.Promote(context.Background(), operations.PromoteRequest{
		Config:                  privateConfig,
		Operation:               operationRequest,
		SourceRoot:              values.sourceRoot,
		OutputDir:               values.outputDir,
		BehaviorVerificationRef: values.verificationRef,
		Runner:                  command.ExecRunner{},
	})
	if err != nil {
		fmt.Fprintf(stderr, "promote failed; evidence=%s\n", result.EvidencePath)
		return 1
	}
	fmt.Fprintf(stdout, "promote-status=%s\npromote-evidence=%s\n", result.Status, result.EvidencePath)
	for _, applied := range result.Applied {
		fmt.Fprintf(stdout, "promoted=%s target=%s\n", applied.ID, applied.Target)
	}
	return 0
}

func runProvisionDevelop(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("provision-develop", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "explicit private configuration path")
	targetName := flags.String("target", "", "explicit develop database target")
	operatorID := flags.String("operator-id", "", "explicit operator identifier")
	authorizationRef := flags.String("authorization-ref", "", "non-secret owner authorization reference")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *configPath == "" || *targetName == "" || *operatorID == "" || *authorizationRef == "" {
		fmt.Fprintln(stderr, "provision-develop requires config, target, operator-id and authorization-ref")
		return 2
	}
	if *targetName != string(config.TargetDevelop) {
		fmt.Fprintln(stderr, "provision-develop requires explicit target mindforge_develop")
		return 2
	}
	target, err := config.ParseTarget(*targetName)
	if err != nil {
		fmt.Fprintln(stderr, "provision-develop arguments are invalid")
		return 2
	}
	operation, err := command.NewOperationRequest(target, *operatorID, *authorizationRef)
	if err != nil {
		fmt.Fprintln(stderr, "provision-develop arguments are invalid")
		return 2
	}
	privateConfig, err := config.ParseFile(*configPath)
	if err == nil {
		err = privateConfig.ValidateOperationalShape()
	}
	if err != nil {
		fmt.Fprintln(stderr, "provision-develop configuration is invalid")
		return 2
	}
	result, err := operations.ProvisionDevelop(context.Background(), operations.ProvisionDevelopRequest{
		Config: privateConfig, Operation: operation, Runner: command.ExecRunner{},
	})
	if err != nil {
		fmt.Fprintln(stderr, "develop provisioning failed")
		return 1
	}
	fmt.Fprintf(stdout, "provision-status=%s target=%s\n", result.Status, result.Target)
	return 0
}

func runBootstrapDevelopExtensions(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("bootstrap-develop-extensions", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "explicit private configuration path")
	sourceRoot := flags.String("source-root", "", "explicit repository root for canonical migration sources")
	targetName := flags.String("target", "", "explicit develop database target")
	operatorID := flags.String("operator-id", "", "explicit operator identifier")
	authorizationRef := flags.String("authorization-ref", "", "non-secret owner authorization reference")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *configPath == "" || *sourceRoot == "" || *targetName == "" || *operatorID == "" || *authorizationRef == "" {
		fmt.Fprintln(stderr, "bootstrap-develop-extensions requires config, source-root, target, operator-id and authorization-ref")
		return 2
	}
	if *targetName != string(config.TargetDevelop) {
		fmt.Fprintln(stderr, "bootstrap-develop-extensions requires explicit target mindforge_develop")
		return 2
	}
	target, err := config.ParseTarget(*targetName)
	if err != nil {
		fmt.Fprintln(stderr, "bootstrap-develop-extensions arguments are invalid")
		return 2
	}
	operation, err := command.NewOperationRequest(target, *operatorID, *authorizationRef)
	if err != nil {
		fmt.Fprintln(stderr, "bootstrap-develop-extensions arguments are invalid")
		return 2
	}
	privateConfig, err := config.ParseFile(*configPath)
	if err == nil {
		err = privateConfig.ValidateOperationalShape()
	}
	if err != nil {
		fmt.Fprintln(stderr, "bootstrap-develop-extensions configuration is invalid")
		return 2
	}
	result, err := operations.BootstrapDevelopExtensions(context.Background(), operations.BootstrapDevelopExtensionsRequest{
		Config: privateConfig, Operation: operation, SourceRoot: *sourceRoot, Runner: command.ExecRunner{},
	})
	if err != nil {
		fmt.Fprintln(stderr, "develop extension bootstrap failed")
		return 1
	}
	fmt.Fprintf(stdout, "bootstrap-status=%s target=%s migration=%s sha256=%s\n", result.Status, result.Target, result.MigrationID, result.MigrationSHA256)
	return 0
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: mindforge-infra config validate --config <explicit-private-config>")
	fmt.Fprintln(writer, "       mindforge-infra audit --source-root <repo-root>")
	fmt.Fprintln(writer, "       mindforge-infra preflight --config <file> --source-root <root> --target <name> --operator-id <id> --authorization-ref <ref>")
	fmt.Fprintln(writer, "       mindforge-infra apply --config <file> --source-root <root> --target <name> --operator-id <id> --authorization-ref <ref> --scope <infrastructure|feature> --preflight-ref <ref> --verification-ref <ref>")
	fmt.Fprintln(writer, "       mindforge-infra backup --config <file> --source-root <root> --target <name> --operator-id <id> --authorization-ref <ref> --output-dir <external-dir>")
	fmt.Fprintln(writer, "       mindforge-infra restore --config <file> --source-root <root> --target <name> --operator-id <id> --authorization-ref <ref> --backup-file <file> --manifest-file <file> --replacement-confirmation <ref> [--evidence-dir <external-dir>]")
	fmt.Fprintln(writer, "       mindforge-infra promote --config <file> --source-root <root> --target mindforge --operator-id <id> --authorization-ref <ref> --output-dir <external-dir> --verification-ref <ref>")
	fmt.Fprintln(writer, "       mindforge-infra provision-develop --config <file> --target mindforge_develop --operator-id <id> --authorization-ref <ref>")
	fmt.Fprintln(writer, "       mindforge-infra bootstrap-develop-extensions --config <file> --source-root <root> --target mindforge_develop --operator-id <id> --authorization-ref <ref>")
}

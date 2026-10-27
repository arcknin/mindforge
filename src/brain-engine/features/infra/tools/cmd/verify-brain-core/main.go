// verify-brain-core exercises Brain Core persistence behavior on the explicit
// develop database. Synthetic writes are durable and are never cleaned up.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/brainproof"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, brainproof.PostgresDependencies()))
}

func run(arguments []string, stdout, stderr io.Writer, dependencies brainproof.Dependencies) int {
	flags := flag.NewFlagSet("verify-brain-core", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "private configuration file")
	operatorID := flags.String("operator-id", "", "operator identifier")
	authorizationRef := flags.String("authorization-ref", "", "non-secret owner authorization reference")
	runID := flags.String("run-id", "", "synthetic verification run identifier")
	allowSyntheticWrites := flags.Bool("allow-synthetic-writes", false, "allow durable synthetic Brain Core writes in mindforge_develop")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		printUsage(stderr)
		return 2
	}
	if *configPath == "" || *operatorID == "" || *authorizationRef == "" || *runID == "" || !*allowSyntheticWrites {
		printUsage(stderr)
		return 2
	}
	privateConfig, err := config.ParseFile(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, "private configuration is invalid")
		return 2
	}
	settings, err := privateConfig.Settings(config.TargetDevelop)
	if err != nil || settings.Target != config.TargetDevelop || settings.DatabaseName != string(config.TargetDevelop) {
		fmt.Fprintln(stderr, "private configuration is invalid")
		return 2
	}

	fmt.Fprintln(stderr, "Synthetic Brain Core rows are durable; this command does not clean them up.")
	result, err := brainproof.Verify(context.Background(), settings, brainproof.Options{
		OperatorID:           *operatorID,
		AuthorizationRef:     *authorizationRef,
		RunID:                *runID,
		AllowSyntheticWrites: *allowSyntheticWrites,
	}, dependencies)
	if err != nil {
		fmt.Fprintln(stderr, "Brain Core verification failed; any synthetic rows are durable and are not cleaned up.")
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "Brain Core verification output failed")
		return 1
	}
	return 0
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: verify-brain-core --config <private-file> --operator-id <id> --authorization-ref <ref> --run-id <id> --allow-synthetic-writes")
	fmt.Fprintln(writer, "target is fixed to mindforge_develop; synthetic rows are durable and no cleanup is performed")
}

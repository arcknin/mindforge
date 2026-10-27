package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProvisionDevelopCommandRequiresAllExplicitAuthorityInputs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"provision-develop",
		"--config", "/missing/private.env",
		"--target", "mindforge_develop",
		"--operator-id", "operator-1",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run() code = %d, want missing authorization reference rejected as usage", code)
	}
}

func TestProvisionDevelopRejectsMainAndSupersededTargetsBeforeLoadingConfig(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"mindforge", "mindforge_main"} {
		var stdout, stderr bytes.Buffer
		code := run([]string{
			"provision-develop",
			"--config", "/missing/private.env",
			"--target", target,
			"--operator-id", "operator-1",
			"--authorization-ref", "owner-provision-2026",
		}, &stdout, &stderr)
		if code != 2 {
			t.Errorf("run(target=%q) code = %d, want target rejected before config or client access", target, code)
		}
		if strings.Contains(stdout.String()+stderr.String(), "synthetic-") {
			t.Errorf("run(target=%q) leaked a configured secret", target)
		}
	}
}

func TestUsageDocumentsOnlyExplicitDevelopProvisioning(t *testing.T) {
	t.Parallel()

	var usage bytes.Buffer
	printUsage(&usage)
	text := usage.String()
	if !strings.Contains(text, "provision-develop --config <file> --target mindforge_develop") {
		t.Fatalf("usage omits the explicit develop provisioning command: %s", text)
	}
}

func TestBootstrapDevelopExtensionsRequiresAllExplicitInputs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"bootstrap-develop-extensions",
		"--config", "/missing/private.env",
		"--source-root", "/missing/repository",
		"--target", "mindforge_develop",
		"--operator-id", "operator-1",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run() code = %d, want missing authorization reference rejected as usage", code)
	}
}

func TestBootstrapDevelopExtensionsRejectsMainAndUnrelatedTargetsBeforeConfig(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"mindforge", "mindforge_main", "nsrdb"} {
		var stdout, stderr bytes.Buffer
		code := run([]string{
			"bootstrap-develop-extensions",
			"--config", "/missing/private.env",
			"--source-root", "/missing/repository",
			"--target", target,
			"--operator-id", "operator-1",
			"--authorization-ref", "owner-bootstrap-2026-10-08",
		}, &stdout, &stderr)
		if code != 2 {
			t.Errorf("run(target=%q) code = %d, want target rejected before config/client access", target, code)
		}
	}
}

func TestUsageDocumentsExplicitDevelopExtensionBootstrap(t *testing.T) {
	t.Parallel()

	var usage bytes.Buffer
	printUsage(&usage)
	if !strings.Contains(usage.String(), "bootstrap-develop-extensions --config <file> --source-root <root> --target mindforge_develop") {
		t.Fatalf("usage omits the explicit develop extension bootstrap command: %s", usage.String())
	}
}

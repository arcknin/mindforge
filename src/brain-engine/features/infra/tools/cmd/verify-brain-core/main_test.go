package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/brainproof"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
)

func TestRunRequiresAllExplicitMetadataAndSyntheticWriteOptIn(t *testing.T) {
	t.Parallel()

	base := []string{"--config", "/private/config.env", "--operator-id", "operator-test", "--authorization-ref", "owner-auth-2026-10-08", "--run-id", "run-test"}
	for name, args := range map[string][]string{
		"missing synthetic-write opt-in": base,
		"missing operator":               {"--config", "/private/config.env", "--authorization-ref", "owner-auth-2026-10-08", "--run-id", "run-test", "--allow-synthetic-writes"},
		"missing authorization":          {"--config", "/private/config.env", "--operator-id", "operator-test", "--run-id", "run-test", "--allow-synthetic-writes"},
		"missing run ID":                 {"--config", "/private/config.env", "--operator-id", "operator-test", "--authorization-ref", "owner-auth-2026-10-08", "--allow-synthetic-writes"},
	} {
		name, args := name, args
		t.Run(name, func(t *testing.T) {
			called := false
			dependencies := brainproof.Dependencies{
				CurrentDatabase: func(context.Context, config.TargetSettings) (string, error) {
					called = true
					return string(config.TargetDevelop), nil
				},
				OpenStore: func(context.Context, config.TargetSettings) (brainproof.Store, error) {
					called = true
					return nil, errors.New("must not open store")
				},
			}
			var stdout, stderr bytes.Buffer
			if code := run(args, &stdout, &stderr, dependencies); code != 2 {
				t.Fatalf("run() code = %d, want usage rejection", code)
			}
			if called {
				t.Fatal("database dependency called before explicit arguments were complete")
			}
		})
	}
}

func TestRunHasNoSelectableMainTargetOrDSNArgument(t *testing.T) {
	t.Parallel()

	const privateDSN = "postgres://private:secret@main.invalid/mindforge"
	var stdout, stderr bytes.Buffer
	code := run([]string{"--target", "mindforge", "--dsn", privateDSN}, &stdout, &stderr, brainproof.Dependencies{})
	if code != 2 {
		t.Fatalf("run() code = %d, want usage rejection", code)
	}
	if strings.Contains(stderr.String(), privateDSN) || strings.Contains(stdout.String(), privateDSN) {
		t.Fatal("usage output leaked a rejected DSN argument")
	}
	if strings.Contains(stderr.String(), "--target") || strings.Contains(stderr.String(), "--dsn") {
		t.Fatalf("usage exposes a target or DSN option: %s", stderr.String())
	}
}

func TestRunSanitizesPrivateConfigAndDatabaseFailures(t *testing.T) {
	t.Parallel()

	const privateMarker = "synthetic-private-password-123456"
	configPath := filepath.Join(t.TempDir(), "private.env")
	if err := os.WriteFile(configPath, []byte(validConfig(privateMarker)), 0o600); err != nil {
		t.Fatalf("write private config: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--config", configPath,
		"--operator-id", "operator-test",
		"--authorization-ref", "owner-auth-2026-10-08",
		"--run-id", "run-test",
		"--allow-synthetic-writes",
	}, &stdout, &stderr, brainproof.Dependencies{
		CurrentDatabase: func(context.Context, config.TargetSettings) (string, error) {
			return string(config.TargetDevelop), nil
		},
		OpenStore: func(context.Context, config.TargetSettings) (brainproof.Store, error) {
			return nil, errors.New("driver DSN marker " + privateMarker)
		},
	})
	if code != 1 {
		t.Fatalf("run() code = %d, want sanitized verification failure", code)
	}
	if strings.Contains(stdout.String()+stderr.String(), privateMarker) {
		t.Fatal("verification output leaked a private credential")
	}
	if !strings.Contains(stderr.String(), "are durable") || !strings.Contains(stderr.String(), "verification failed") {
		t.Fatalf("failure output omitted durable-write and failure status: %s", stderr.String())
	}
}

func validConfig(mainPassword string) string {
	return strings.Join([]string{
		"POSTGRES_ADMIN_USER=mf_cluster_admin",
		"POSTGRES_ADMIN_PASSWORD=synthetic-cluster-password-123456",
		"POSTGRES_HOST=db",
		"POSTGRES_CLIENT_HOST=127.0.0.1",
		"POSTGRES_PORT=15432",
		"DB_BIND_ADDR=127.0.0.1",
		"DEVELOP_DB_NAME=mindforge_develop",
		"DEVELOP_DB_USER=mf_develop",
		"DEVELOP_DB_PASSWORD=synthetic-develop-password-123456",
		"MAIN_DB_NAME=mindforge",
		"MAIN_DB_USER=mf_main",
		"MAIN_DB_PASSWORD=" + mainPassword,
		"DEVELOP_AUTH_DB_URL=postgres://mf_develop:synthetic-develop-password-123456@db:5432/mindforge_develop?sslmode=disable&search_path=auth",
		"MAIN_AUTH_DB_URL=postgres://mf_main:" + mainPassword + "@db:5432/mindforge?sslmode=disable&search_path=auth",
		"DEVELOP_AUTH_PORT=10000",
		"MAIN_AUTH_PORT=9999",
		"DEVELOP_AUTH_BIND_ADDR=127.0.0.1",
		"MAIN_AUTH_BIND_ADDR=127.0.0.1",
		"ALLOW_LAN_BINDING=no",
		"DEVELOP_AUTH_SITE_URL=http://127.0.0.1:3000",
		"MAIN_AUTH_SITE_URL=http://127.0.0.1:3001",
		"DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS=http://127.0.0.1:3000",
		"MAIN_AUTH_ADDITIONAL_REDIRECT_URLS=http://127.0.0.1:3001",
		"DEVELOP_AUTH_URI_ALLOW_LIST=http://127.0.0.1:3000",
		"MAIN_AUTH_URI_ALLOW_LIST=http://127.0.0.1:3001",
		"DEVELOP_AUTH_API_EXTERNAL_URL=http://127.0.0.1:10000",
		"MAIN_AUTH_API_EXTERNAL_URL=http://127.0.0.1:9999",
		"DEVELOP_AUTH_JWT_SECRET=synthetic-develop-jwt-secret-123456789012",
		"MAIN_AUTH_JWT_SECRET=synthetic-main-jwt-secret-123456789012",
		"DEVELOP_AUTH_OPERATOR_TOKEN=synthetic-develop-operator-token-123456789",
		"MAIN_AUTH_OPERATOR_TOKEN=synthetic-main-operator-token-123456789",
		"GOTRUE_DB_MAX_POOL_SIZE=10",
		"GOTRUE_MAILER_AUTOCONFIRM=false",
		"TZ=UTC",
	}, "\n") + "\n"
}

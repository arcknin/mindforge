package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFileRejectsMalformedDuplicateAndUnknownPrivateConfiguration(t *testing.T) {
	t.Parallel()

	for name, contents := range map[string]string{
		"malformed": "DEVELOP_DB_NAME\n",
		"duplicate": "DEVELOP_DB_NAME=mindforge_develop\nDEVELOP_DB_NAME=mindforge_develop\n",
		"unknown":   "UNSAFE_OVERRIDE=value\n",
	} {
		name, contents := name, contents
		t.Run(name, func(t *testing.T) {
			path := writePrivateConfig(t, contents)
			_, err := ParseFile(path)
			if err == nil {
				t.Fatal("ParseFile() error = nil, want malformed private configuration rejection")
			}
			if strings.Contains(err.Error(), "value") {
				t.Fatalf("ParseFile() error leaked configuration value: %v", err)
			}
		})
	}
}

func TestValidateShapeRequiresEveryKnownKeyWithoutEchoingSecrets(t *testing.T) {
	t.Parallel()

	values := validPrivateConfig()
	delete(values, "MAIN_AUTH_OPERATOR_TOKEN")
	values["DEVELOP_AUTH_JWT_SECRET"] = "synthetic-secret-that-must-not-appear"

	err := Values{values: values}.ValidateShape()
	if err == nil {
		t.Fatal("ValidateShape() error = nil, want missing required key rejection")
	}
	if strings.Contains(err.Error(), "synthetic-secret-that-must-not-appear") {
		t.Fatalf("ValidateShape() error leaked a secret: %v", err)
	}
}

func TestValidateShapeUsesPhysicalMainDatabaseName(t *testing.T) {
	t.Parallel()

	values := validPrivateConfig()
	if err := (Values{values: values}).ValidateShape(); err != nil {
		t.Fatalf("ValidateShape() with mindforge main: %v", err)
	}
	values["MAIN_DB_NAME"] = "mindforge_main"
	if err := (Values{values: values}).ValidateShape(); err == nil {
		t.Fatal("ValidateShape() error = nil for superseded physical main database name")
	}
}

func TestSettingsUseHostClientAddressAndPreserveInternalAuthURLs(t *testing.T) {
	t.Parallel()

	values := validOperationalConfig()
	for _, target := range []Target{TargetDevelop, TargetMain} {
		settings, err := (Values{values: values}).Settings(target)
		if err != nil {
			t.Fatalf("Settings(%q): %v", target, err)
		}
		if settings.DatabaseHost != "127.0.0.1" {
			t.Errorf("Settings(%q).DatabaseHost = %q, want host-side client address", target, settings.DatabaseHost)
		}
		if settings.DatabasePort != "15432" {
			t.Errorf("Settings(%q).DatabasePort = %q, want host port", target, settings.DatabasePort)
		}
		if !strings.Contains(settings.AuthDatabaseURL, "@db:5432/") {
			t.Errorf("Settings(%q).AuthDatabaseURL lost the internal Compose endpoint", target)
		}
	}

	admin, err := (Values{values: values}).ClusterAdmin()
	if err != nil {
		t.Fatalf("ClusterAdmin(): %v", err)
	}
	if admin.DatabaseHost != "127.0.0.1" || admin.DatabasePort != "15432" {
		t.Fatalf("ClusterAdmin() host:port = %s:%s, want 127.0.0.1:15432", admin.DatabaseHost, admin.DatabasePort)
	}
}

func TestPostgresClientHostIsRequiredAndRejectsUnsafeHostnames(t *testing.T) {
	t.Parallel()

	values := validOperationalConfig()
	delete(values, "POSTGRES_CLIENT_HOST")
	if err := (Values{values: values}).ValidateShape(); err == nil {
		t.Fatal("ValidateShape() error = nil without POSTGRES_CLIENT_HOST")
	}

	for _, host := range []string{"", "db:5432", "bad..host", "-bad", "bad-", "db;select"} {
		values := validOperationalConfig()
		values["POSTGRES_CLIENT_HOST"] = host
		_, err := (Values{values: values}).Settings(TargetDevelop)
		if err == nil {
			t.Errorf("Settings() error = nil for unsafe POSTGRES_CLIENT_HOST %q", host)
			continue
		}
		if strings.Contains(err.Error(), values["DEVELOP_DB_PASSWORD"]) || strings.Contains(err.Error(), values["POSTGRES_ADMIN_PASSWORD"]) {
			t.Errorf("Settings() error leaked a credential: %v", err)
		}
	}
}

func TestClusterAdminAllowsExplicitMainRoleAliasOnlyWithMatchingCredential(t *testing.T) {
	t.Parallel()

	values := validOperationalConfig()
	values["POSTGRES_ADMIN_USER"] = values["MAIN_DB_USER"]
	values["POSTGRES_ADMIN_PASSWORD"] = values["MAIN_DB_PASSWORD"]
	admin, err := (Values{values: values}).ClusterAdmin()
	if err != nil {
		t.Fatalf("ClusterAdmin() with explicit main admin alias: %v", err)
	}
	if admin.DatabaseUser != values["MAIN_DB_USER"] || admin.DatabasePassword != values["MAIN_DB_PASSWORD"] {
		t.Fatal("ClusterAdmin() did not preserve the configured main admin credential")
	}

	values["POSTGRES_ADMIN_PASSWORD"] = "different-admin-password-123456"
	if _, err := (Values{values: values}).ClusterAdmin(); err == nil {
		t.Fatal("ClusterAdmin() error = nil when aliased role has mismatched credentials")
	}
}

func TestClusterAdminKeepsDevelopRoleDistinctFromBothAdminAndMain(t *testing.T) {
	t.Parallel()

	for name, aliasMain := range map[string]bool{"admin": false, "main": true} {
		values := validOperationalConfig()
		if aliasMain {
			values["MAIN_DB_USER"] = values["DEVELOP_DB_USER"]
			values["MAIN_AUTH_DB_URL"] = authDatabaseURL(values["MAIN_DB_USER"], values["MAIN_DB_PASSWORD"], values["MAIN_DB_NAME"])
		} else {
			values["POSTGRES_ADMIN_USER"] = values["DEVELOP_DB_USER"]
			values["POSTGRES_ADMIN_PASSWORD"] = values["DEVELOP_DB_PASSWORD"]
		}
		t.Run(name, func(t *testing.T) {
			if _, err := (Values{values: values}).ClusterAdmin(); err == nil {
				t.Fatal("ClusterAdmin() error = nil for a shared develop role")
			}
		})
	}
}

func TestParseTargetRequiresAnExplicitContractedTarget(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"mindforge_develop", "mindforge"} {
		target, err := ParseTarget(raw)
		if err != nil {
			t.Fatalf("ParseTarget(%q): %v", raw, err)
		}
		if string(target) != raw {
			t.Fatalf("ParseTarget(%q) = %q", raw, target)
		}
	}
	for _, raw := range []string{"", "develop", "mindforge_main", "mindforge "} {
		if _, err := ParseTarget(raw); err == nil {
			t.Fatalf("ParseTarget(%q) error = nil, want explicit target rejection", raw)
		}
	}
}

func writePrivateConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "private.env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write private config: %v", err)
	}
	return path
}

func validPrivateConfig() map[string]string {
	values := make(map[string]string, len(requiredKeys))
	for _, key := range requiredKeys {
		values[key] = "synthetic-value"
	}
	values["DEVELOP_DB_NAME"] = "mindforge_develop"
	values["MAIN_DB_NAME"] = "mindforge"
	return values
}

func validOperationalConfig() map[string]string {
	values := validPrivateConfig()
	values["POSTGRES_ADMIN_USER"] = "mf_cluster_admin"
	values["POSTGRES_ADMIN_PASSWORD"] = "synthetic-cluster-password-123456"
	values["POSTGRES_HOST"] = "db"
	values["POSTGRES_CLIENT_HOST"] = "127.0.0.1"
	values["POSTGRES_PORT"] = "15432"
	values["DEVELOP_DB_USER"] = "mf_develop"
	values["DEVELOP_DB_PASSWORD"] = "synthetic-develop-password-123456"
	values["MAIN_DB_USER"] = "mf_main"
	values["MAIN_DB_PASSWORD"] = "synthetic-main-password-123456"
	values["DEVELOP_AUTH_DB_URL"] = authDatabaseURL(values["DEVELOP_DB_USER"], values["DEVELOP_DB_PASSWORD"], values["DEVELOP_DB_NAME"])
	values["MAIN_AUTH_DB_URL"] = authDatabaseURL(values["MAIN_DB_USER"], values["MAIN_DB_PASSWORD"], values["MAIN_DB_NAME"])
	values["DEVELOP_AUTH_JWT_SECRET"] = "synthetic-develop-jwt-secret-123456789012"
	values["MAIN_AUTH_JWT_SECRET"] = "synthetic-main-jwt-secret-123456789012"
	values["DEVELOP_AUTH_OPERATOR_TOKEN"] = "synthetic-develop-operator-token-123456789"
	values["MAIN_AUTH_OPERATOR_TOKEN"] = "synthetic-main-operator-token-123456789"
	return values
}

func authDatabaseURL(user, password, database string) string {
	return "postgres://" + user + ":" + password + "@db:5432/" + database + "?sslmode=disable&search_path=auth"
}

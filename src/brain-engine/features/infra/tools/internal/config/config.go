// Package config parses the private, operator-supplied infrastructure
// configuration without evaluating it as shell input.
package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Target names the only database environments contracted for this stack.
type Target string

const (
	TargetDevelop Target = "mindforge_develop"
	TargetMain    Target = "mindforge"
)

var keyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var postgresIdentifierPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
var postgresClientHostPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`)

var requiredKeys = []string{
	"POSTGRES_ADMIN_USER",
	"POSTGRES_ADMIN_PASSWORD",
	"POSTGRES_HOST",
	"POSTGRES_CLIENT_HOST",
	"POSTGRES_PORT",
	"DB_BIND_ADDR",
	"DEVELOP_DB_NAME",
	"DEVELOP_DB_USER",
	"DEVELOP_DB_PASSWORD",
	"MAIN_DB_NAME",
	"MAIN_DB_USER",
	"MAIN_DB_PASSWORD",
	"DEVELOP_AUTH_DB_URL",
	"MAIN_AUTH_DB_URL",
	"DEVELOP_AUTH_PORT",
	"MAIN_AUTH_PORT",
	"DEVELOP_AUTH_BIND_ADDR",
	"MAIN_AUTH_BIND_ADDR",
	"ALLOW_LAN_BINDING",
	"DEVELOP_AUTH_SITE_URL",
	"MAIN_AUTH_SITE_URL",
	"DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS",
	"MAIN_AUTH_ADDITIONAL_REDIRECT_URLS",
	"DEVELOP_AUTH_URI_ALLOW_LIST",
	"MAIN_AUTH_URI_ALLOW_LIST",
	"DEVELOP_AUTH_API_EXTERNAL_URL",
	"MAIN_AUTH_API_EXTERNAL_URL",
	"DEVELOP_AUTH_JWT_SECRET",
	"MAIN_AUTH_JWT_SECRET",
	"DEVELOP_AUTH_OPERATOR_TOKEN",
	"MAIN_AUTH_OPERATOR_TOKEN",
	"GOTRUE_DB_MAX_POOL_SIZE",
	"GOTRUE_MAILER_AUTOCONFIRM",
	"TZ",
}

var allowedKeys = func() map[string]struct{} {
	keys := make(map[string]struct{}, len(requiredKeys))
	for _, key := range requiredKeys {
		keys[key] = struct{}{}
	}
	return keys
}()

// Values holds private configuration values. It deliberately has no String
// method so callers cannot accidentally format the complete configuration.
type Values struct {
	values map[string]string
}

// TargetSettings contains the private connection values selected for one
// explicit database target. Callers must keep DatabasePassword and the Auth
// secrets out of arguments, logs and persisted evidence.
type TargetSettings struct {
	Target             Target
	DatabaseHost       string
	DatabasePort       string
	DatabaseName       string
	DatabaseUser       string
	DatabasePassword   string
	AuthDatabaseURL    string
	AuthPort           string
	AuthBindAddress    string
	AuthSiteURL        string
	AuthAPIExternalURL string
	AuthJWTSecret      string
	AuthOperatorToken  string
}

// ClusterAdminSettings contains the private connection boundary for the
// explicit additive develop-provisioning operation. DatabaseName is always
// PostgreSQL's administrative database, never an application target.
type ClusterAdminSettings struct {
	DatabaseHost     string
	DatabasePort     string
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
}

// ParseFile parses a restricted dotenv-like file. Values are raw data: shell
// expressions, substitutions and expansions are never evaluated.
func ParseFile(path string) (Values, error) {
	file, err := os.Open(path)
	if err != nil {
		return Values{}, fmt.Errorf("open private configuration: %w", err)
	}
	defer file.Close()

	values := Values{values: make(map[string]string, len(requiredKeys))}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || !keyPattern.MatchString(key) {
			return Values{}, fmt.Errorf("private configuration line %d must use a supported KEY=VALUE form", lineNumber)
		}
		if _, ok := allowedKeys[key]; !ok {
			return Values{}, fmt.Errorf("private configuration key %s is not allowed", key)
		}
		if _, exists := values.values[key]; exists {
			return Values{}, fmt.Errorf("private configuration key %s is duplicated", key)
		}
		if value == "" {
			return Values{}, fmt.Errorf("private configuration key %s is empty", key)
		}
		values.values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return Values{}, fmt.Errorf("read private configuration: %w", err)
	}
	return values, nil
}

// ValidateShape checks required keys and the two non-secret target names. It
// deliberately does not log or return any private value.
func (v Values) ValidateShape() error {
	for _, key := range requiredKeys {
		if v.values[key] == "" {
			return fmt.Errorf("private configuration key %s is required", key)
		}
	}
	if v.values["DEVELOP_DB_NAME"] != string(TargetDevelop) {
		return fmt.Errorf("private configuration DEVELOP_DB_NAME must name the contracted develop target")
	}
	if v.values["MAIN_DB_NAME"] != string(TargetMain) {
		return fmt.Errorf("private configuration MAIN_DB_NAME must name the contracted main target")
	}
	return nil
}

// ClusterAdmin returns a validated cluster-admin connection for the one
// additive develop provisioning operation. It does not connect or mutate.
func (v Values) ClusterAdmin() (ClusterAdminSettings, error) {
	if err := v.ValidateOperationalShape(); err != nil {
		return ClusterAdminSettings{}, err
	}
	host := v.values["POSTGRES_CLIENT_HOST"]
	port := v.values["POSTGRES_PORT"]
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 || strconv.Itoa(portNumber) != port {
		return ClusterAdminSettings{}, fmt.Errorf("private configuration PostgreSQL port is invalid")
	}
	adminUser := v.values["POSTGRES_ADMIN_USER"]
	developUser := v.values["DEVELOP_DB_USER"]
	mainUser := v.values["MAIN_DB_USER"]
	if !postgresIdentifierPattern.MatchString(adminUser) || !postgresIdentifierPattern.MatchString(developUser) || !postgresIdentifierPattern.MatchString(mainUser) {
		return ClusterAdminSettings{}, fmt.Errorf("private configuration database users must be safe PostgreSQL identifiers")
	}
	if adminUser == developUser || developUser == mainUser {
		return ClusterAdminSettings{}, fmt.Errorf("private configuration develop database user must be distinct")
	}
	adminPassword := v.values["POSTGRES_ADMIN_PASSWORD"]
	developPassword := v.values["DEVELOP_DB_PASSWORD"]
	mainPassword := v.values["MAIN_DB_PASSWORD"]
	if adminPassword == developPassword || developPassword == mainPassword {
		return ClusterAdminSettings{}, fmt.Errorf("private configuration database passwords must be distinct")
	}
	if adminUser == mainUser {
		if adminPassword != mainPassword {
			return ClusterAdminSettings{}, fmt.Errorf("private configuration aliased admin and main users must have matching credentials")
		}
	} else if adminPassword == mainPassword {
		return ClusterAdminSettings{}, fmt.Errorf("private configuration distinct admin and main users must have distinct passwords")
	}
	return ClusterAdminSettings{
		DatabaseHost:     host,
		DatabasePort:     port,
		DatabaseName:     "postgres",
		DatabaseUser:     adminUser,
		DatabasePassword: v.values["POSTGRES_ADMIN_PASSWORD"],
	}, nil
}

// Value returns one private value for a later guarded operation. Callers must
// keep the value out of arguments, logs and evidence.
func (v Values) Value(key string) (string, bool) {
	value, ok := v.values[key]
	return value, ok
}

// Settings selects the complete private connection boundary for one
// contracted target. It refuses to infer a target from a default or from a
// partially populated configuration.
func (v Values) Settings(target Target) (TargetSettings, error) {
	if err := v.ValidateOperationalShape(); err != nil {
		return TargetSettings{}, err
	}
	switch target {
	case TargetDevelop:
		return TargetSettings{
			Target:             TargetDevelop,
			DatabaseHost:       v.values["POSTGRES_CLIENT_HOST"],
			DatabasePort:       v.values["POSTGRES_PORT"],
			DatabaseName:       v.values["DEVELOP_DB_NAME"],
			DatabaseUser:       v.values["DEVELOP_DB_USER"],
			DatabasePassword:   v.values["DEVELOP_DB_PASSWORD"],
			AuthDatabaseURL:    v.values["DEVELOP_AUTH_DB_URL"],
			AuthPort:           v.values["DEVELOP_AUTH_PORT"],
			AuthBindAddress:    v.values["DEVELOP_AUTH_BIND_ADDR"],
			AuthSiteURL:        v.values["DEVELOP_AUTH_SITE_URL"],
			AuthAPIExternalURL: v.values["DEVELOP_AUTH_API_EXTERNAL_URL"],
			AuthJWTSecret:      v.values["DEVELOP_AUTH_JWT_SECRET"],
			AuthOperatorToken:  v.values["DEVELOP_AUTH_OPERATOR_TOKEN"],
		}, nil
	case TargetMain:
		return TargetSettings{
			Target:             TargetMain,
			DatabaseHost:       v.values["POSTGRES_CLIENT_HOST"],
			DatabasePort:       v.values["POSTGRES_PORT"],
			DatabaseName:       v.values["MAIN_DB_NAME"],
			DatabaseUser:       v.values["MAIN_DB_USER"],
			DatabasePassword:   v.values["MAIN_DB_PASSWORD"],
			AuthDatabaseURL:    v.values["MAIN_AUTH_DB_URL"],
			AuthPort:           v.values["MAIN_AUTH_PORT"],
			AuthBindAddress:    v.values["MAIN_AUTH_BIND_ADDR"],
			AuthSiteURL:        v.values["MAIN_AUTH_SITE_URL"],
			AuthAPIExternalURL: v.values["MAIN_AUTH_API_EXTERNAL_URL"],
			AuthJWTSecret:      v.values["MAIN_AUTH_JWT_SECRET"],
			AuthOperatorToken:  v.values["MAIN_AUTH_OPERATOR_TOKEN"],
		}, nil
	default:
		return TargetSettings{}, fmt.Errorf("an explicit contracted target is required")
	}
}

// ValidateOperationalShape checks the target binding that database-facing
// operations rely on. The shell configuration validator remains responsible
// for the complete private environment grammar; this method keeps the Go
// operator from silently swapping a target URL or sharing Auth secrets.
func (v Values) ValidateOperationalShape() error {
	if err := v.ValidateShape(); err != nil {
		return err
	}
	clientHost := v.values["POSTGRES_CLIENT_HOST"]
	if len(clientHost) > 253 || !postgresClientHostPattern.MatchString(clientHost) {
		return fmt.Errorf("private configuration POSTGRES_CLIENT_HOST must be a safe hostname")
	}
	expectedDevelopURL := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable&search_path=auth", v.values["DEVELOP_DB_USER"], v.values["DEVELOP_DB_PASSWORD"], v.values["POSTGRES_HOST"], v.values["DEVELOP_DB_NAME"])
	expectedMainURL := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable&search_path=auth", v.values["MAIN_DB_USER"], v.values["MAIN_DB_PASSWORD"], v.values["POSTGRES_HOST"], v.values["MAIN_DB_NAME"])
	if v.values["DEVELOP_AUTH_DB_URL"] != expectedDevelopURL || v.values["MAIN_AUTH_DB_URL"] != expectedMainURL {
		return fmt.Errorf("private configuration Auth database URLs must match their explicit targets")
	}
	if v.values["DEVELOP_AUTH_DB_URL"] == v.values["MAIN_AUTH_DB_URL"] {
		return fmt.Errorf("private configuration Auth database URLs must be distinct")
	}
	for _, pair := range [][2]string{
		{"DEVELOP_AUTH_JWT_SECRET", "MAIN_AUTH_JWT_SECRET"},
		{"DEVELOP_AUTH_OPERATOR_TOKEN", "MAIN_AUTH_OPERATOR_TOKEN"},
		{"DEVELOP_AUTH_JWT_SECRET", "DEVELOP_AUTH_OPERATOR_TOKEN"},
		{"MAIN_AUTH_JWT_SECRET", "MAIN_AUTH_OPERATOR_TOKEN"},
	} {
		if v.values[pair[0]] == v.values[pair[1]] {
			return fmt.Errorf("private configuration Auth secrets must be distinct")
		}
	}
	return nil
}

// ParseTarget accepts only a deliberate selection of one contracted target.
func ParseTarget(raw string) (Target, error) {
	switch Target(raw) {
	case TargetDevelop:
		return TargetDevelop, nil
	case TargetMain:
		return TargetMain, nil
	default:
		return "", fmt.Errorf("an explicit contracted target is required")
	}
}

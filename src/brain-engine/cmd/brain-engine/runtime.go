package main

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"github.com/jackc/pgx/v5"
)

const developDatabase = "mindforge_develop"
const developAuthUserURL = "http://auth-develop:9999/user"
const developAuthCallbackURL = "http://127.0.0.1:11001/v1/auth/callback"
const developAuthRedirectURLs = "mindforge-dev://auth/callback,http://web-mind.home.arpa/login/callback"

type runtimeConfig struct {
	databaseURL      string
	authUserURL      string
	authCallbackURL  string
	authRedirectURLs []string
	listenAddress    string
}

// Configuration errors are fixed strings: secret values and parse errors never
// appear in logs or return values. This first service is locked to develop.
func loadRuntimeConfig(getenv func(string) string) (runtimeConfig, error) {
	var c runtimeConfig
	if getenv("MINDFORGE_ENVIRONMENT") != "develop" {
		return c, errors.New("MINDFORGE_ENVIRONMENT must explicitly select develop")
	}
	c.databaseURL = getenv("MINDFORGE_DATABASE_URL")
	u, err := url.Parse(c.databaseURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.User == nil || u.User.Username() != developDatabase || u.Path != "/"+developDatabase || u.Hostname() != "db" || u.Port() != "5432" || u.RawQuery != "sslmode=disable" || u.Fragment != "" {
		return runtimeConfig{}, errors.New("MINDFORGE_DATABASE_URL must bind the develop owner and database")
	}
	parsed, err := pgx.ParseConfig(c.databaseURL)
	if err != nil || parsed.Database != developDatabase || parsed.User != developDatabase {
		return runtimeConfig{}, errors.New("MINDFORGE_DATABASE_URL is invalid for develop")
	}
	c.authUserURL = getenv("MINDFORGE_AUTH_USER_URL")
	if c.authUserURL != developAuthUserURL {
		return runtimeConfig{}, errors.New("MINDFORGE_AUTH_USER_URL must bind the internal develop Auth")
	}
	c.authCallbackURL = getenv("MINDFORGE_AUTH_CALLBACK_URL")
	if c.authCallbackURL != developAuthCallbackURL {
		return runtimeConfig{}, errors.New("MINDFORGE_AUTH_CALLBACK_URL must bind the local API callback")
	}
	if getenv("MINDFORGE_AUTH_REDIRECT_URLS") != developAuthRedirectURLs {
		return runtimeConfig{}, errors.New("MINDFORGE_AUTH_REDIRECT_URLS must match the exact develop allowlist")
	}
	c.authRedirectURLs = strings.Split(developAuthRedirectURLs, ",")
	c.listenAddress = getenv("MINDFORGE_LISTEN_ADDR")
	host, port, err := net.SplitHostPort(c.listenAddress)
	n, e := strconv.Atoi(port)
	if err != nil || e != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port || (host != "" && host != "127.0.0.1" && host != "0.0.0.0" && host != "::1") {
		return runtimeConfig{}, errors.New("MINDFORGE_LISTEN_ADDR must be an explicit supported TCP binding")
	}
	return c, nil
}

type databaseIdentity struct {
	database, user, owner           string
	superuser, createDB, createRole bool
	brains, nodes                   bool
	documents, documentLinks        bool
}
type databaseProbe func(context.Context, string) (databaseIdentity, error)

func verifyDatabaseIdentity(ctx context.Context, c runtimeConfig, probe databaseProbe) error {
	if probe == nil {
		return errors.New("database identity probe is required")
	}
	id, err := probe(ctx, c.databaseURL)
	if err != nil {
		return errors.New("develop database identity probe failed")
	}
	if id.database != developDatabase || id.user != developDatabase || id.owner != developDatabase || id.superuser || id.createDB || id.createRole || !id.brains || !id.nodes || !id.documents || !id.documentLinks {
		return errors.New("database identity, role or Brain Core/document prerequisites do not match develop")
	}
	return nil
}

// Probe is read-only and never bootstraps a schema, runs a migration or creates
// data. The authoritative migration pipeline remains with the infra operator.
func probeDatabase(ctx context.Context, dsn string) (databaseIdentity, error) {
	var id databaseIdentity
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return id, errors.New("database connection unavailable")
	}
	defer conn.Close(context.WithoutCancel(ctx))
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return id, errors.New("database catalog transaction unavailable")
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	err = tx.QueryRow(ctx, `SELECT current_database(),current_user,
 pg_get_userbyid((SELECT datdba FROM pg_catalog.pg_database WHERE datname=current_database())),
 actor_role.rolsuper,actor_role.rolcreatedb,actor_role.rolcreaterole,
 to_regclass('brain_core.brains') IS NOT NULL,
 to_regclass('brain_core.nodes') IS NOT NULL,
 to_regclass('brain_documents.documents') IS NOT NULL,
 to_regclass('brain_documents.document_links') IS NOT NULL
 FROM pg_catalog.pg_roles AS actor_role WHERE actor_role.rolname=current_user`).Scan(&id.database, &id.user, &id.owner, &id.superuser, &id.createDB, &id.createRole, &id.brains, &id.nodes, &id.documents, &id.documentLinks)
	if err != nil {
		return databaseIdentity{}, errors.New("database identity unavailable")
	}
	if strings.TrimSpace(id.database) == "" || strings.TrimSpace(id.user) == "" {
		return databaseIdentity{}, errors.New("database identity unavailable")
	}
	if err = tx.Commit(ctx); err != nil {
		return databaseIdentity{}, errors.New("database catalog transaction failed")
	}
	return id, nil
}

type runtimeStore interface {
	brain.Store
	Close()
}
type storeFactory func(context.Context, string) (runtimeStore, error)

func initializeStore(ctx context.Context, c runtimeConfig, probe databaseProbe, open storeFactory) (runtimeStore, error) {
	if err := verifyDatabaseIdentity(ctx, c, probe); err != nil {
		return nil, err
	}
	if open == nil {
		return nil, errors.New("develop store factory is required")
	}
	store, err := open(ctx, c.databaseURL)
	if err != nil || store == nil {
		return nil, errors.New("develop persistence store could not be opened")
	}
	return store, nil
}

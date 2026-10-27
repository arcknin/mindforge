package main

import (
	"context"
	"errors"
	"github.com/arcknin/mindforge/src/brain-engine/features/brain"
	"strings"
	"testing"
)

func envFixture() map[string]string {
	return map[string]string{
		"MINDFORGE_ENVIRONMENT":        "develop",
		"MINDFORGE_DATABASE_URL":       "postgres://mindforge_develop:synthetic-password@db:5432/mindforge_develop?sslmode=disable",
		"MINDFORGE_AUTH_USER_URL":      "http://auth-develop:9999/user",
		"MINDFORGE_AUTH_CALLBACK_URL":  "http://127.0.0.1:11001/v1/auth/callback",
		"MINDFORGE_AUTH_REDIRECT_URLS": "mindforge-dev://auth/callback,http://web-mind.home.arpa/login/callback",
		"MINDFORGE_LISTEN_ADDR":        ":8080",
	}
}
func loadFixture(v map[string]string) (runtimeConfig, error) {
	return loadRuntimeConfig(func(k string) string { return v[k] })
}
func TestRuntimeRejectsWrongEnvironmentAndCredentialDestinations(t *testing.T) {
	cases := []struct{ name, key, value string }{
		{"main environment", "MINDFORGE_ENVIRONMENT", "main"},
		{"missing environment", "MINDFORGE_ENVIRONMENT", ""},
		{"main database", "MINDFORGE_DATABASE_URL", "postgres://mindforge_develop:private-marker@db:5432/mindforge?sslmode=disable"},
		{"admin role", "MINDFORGE_DATABASE_URL", "postgres://mindforge:private-marker@db:5432/mindforge_develop?sslmode=disable"},
		{"bad secret URL", "MINDFORGE_DATABASE_URL", "postgres://%private-marker"},
		{"main Auth", "MINDFORGE_AUTH_USER_URL", "http://auth:9999/user"},
		{"redirect destination", "MINDFORGE_AUTH_USER_URL", "http://auth-develop:9999/user?redirect=private-marker"},
		{"wrong callback", "MINDFORGE_AUTH_CALLBACK_URL", "http://127.0.0.1:11000/callback"},
		{"wrong redirect allowlist", "MINDFORGE_AUTH_REDIRECT_URLS", "mindforge-dev://other/callback"},
		{"missing DSN", "MINDFORGE_DATABASE_URL", ""},
		{"invalid listener", "MINDFORGE_LISTEN_ADDR", "private-marker"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := envFixture()
			v[tc.key] = tc.value
			_, err := loadFixture(v)
			if err == nil {
				t.Fatal("invalid runtime configuration accepted")
			}
			if strings.Contains(err.Error(), "private-marker") {
				t.Fatal("configuration error exposed private value")
			}
		})
	}
}
func TestRuntimeAcceptsExplicitDevelopBinding(t *testing.T) {
	c, err := loadFixture(envFixture())
	if err != nil {
		t.Fatal(err)
	}
	if c.listenAddress != ":8080" || c.authUserURL != "http://auth-develop:9999/user" {
		t.Fatal("runtime binding drifted")
	}
	if c.authCallbackURL != developAuthCallbackURL || len(c.authRedirectURLs) != 2 || c.authRedirectURLs[0] != "mindforge-dev://auth/callback" || c.authRedirectURLs[1] != "http://web-mind.home.arpa/login/callback" {
		t.Fatalf("Auth facade redirects drifted: callback=%q redirects=%v", c.authCallbackURL, c.authRedirectURLs)
	}
}
func TestRuntimeDatabaseProbeRejectsRoleAndTargetBeforeStoreOpen(t *testing.T) {
	c, err := loadFixture(envFixture())
	if err != nil {
		t.Fatal(err)
	}
	good := databaseIdentity{database: "mindforge_develop", user: "mindforge_develop", owner: "mindforge_develop", brains: true, nodes: true, documents: true, documentLinks: true}
	cases := []databaseIdentity{good, {database: "mindforge", user: "mindforge_develop", owner: "mindforge_develop", brains: true, nodes: true, documents: true, documentLinks: true}, {database: "mindforge_develop", user: "mindforge", owner: "mindforge_develop", brains: true, nodes: true, documents: true, documentLinks: true}, {database: "mindforge_develop", user: "mindforge_develop", owner: "mindforge_develop", superuser: true, brains: true, nodes: true, documents: true, documentLinks: true}, {database: "mindforge_develop", user: "mindforge_develop", owner: "mindforge_develop", createDB: true, brains: true, nodes: true, documents: true, documentLinks: true}, {database: "mindforge_develop", user: "mindforge_develop", owner: "mindforge_develop", createRole: true, brains: true, nodes: true, documents: true, documentLinks: true}, {database: "mindforge_develop", user: "mindforge_develop", owner: "mindforge_develop", brains: false, nodes: true}}
	for i, id := range cases {
		err := verifyDatabaseIdentity(context.Background(), c, func(context.Context, string) (databaseIdentity, error) { return id, nil })
		if (err == nil) != (i == 0) {
			t.Fatalf("case%d accepted=%t", i, err == nil)
		}
	}
	err = verifyDatabaseIdentity(context.Background(), c, func(context.Context, string) (databaseIdentity, error) {
		return databaseIdentity{}, errors.New("private-marker DSN")
	})
	if err == nil || strings.Contains(err.Error(), "private-marker") {
		t.Fatal("probe error not opaque")
	}
}

type startupStore struct{ closed bool }

func (s *startupStore) Close() { s.closed = true }
func (s *startupStore) EnsureBrain(context.Context, brain.Principal, brain.Brain) (brain.Brain, error) {
	return brain.Brain{}, brain.ErrInternal
}
func (s *startupStore) Observe(context.Context, brain.Principal) (brain.Hierarchy, error) {
	return brain.Hierarchy{}, brain.ErrInternal
}
func (s *startupStore) InsertLayer(context.Context, brain.Principal, brain.Layer) (brain.Layer, error) {
	return brain.Layer{}, brain.ErrInternal
}
func (s *startupStore) CompareAndSwapLayerName(context.Context, brain.Principal, brain.BrainID, brain.LayerID, brain.LayerVersion, string) (brain.Layer, error) {
	return brain.Layer{}, brain.ErrInternal
}
func TestInitializeStoreCannotOpenBeforeIdentityPasses(t *testing.T) {
	c, _ := loadFixture(envFixture())
	opened := 0
	factory := func(context.Context, string) (runtimeStore, error) { opened++; return &startupStore{}, nil }
	_, err := initializeStore(context.Background(), c, func(context.Context, string) (databaseIdentity, error) {
		return databaseIdentity{database: "mindforge", user: "mindforge"}, nil
	}, factory)
	if err == nil || opened != 0 {
		t.Fatal("store opened for wrong target")
	}
	_, err = initializeStore(context.Background(), c, func(context.Context, string) (databaseIdentity, error) {
		return databaseIdentity{database: developDatabase, user: developDatabase, owner: developDatabase, brains: true, nodes: true, documents: true, documentLinks: true}, nil
	}, factory)
	if err != nil || opened != 1 {
		t.Fatal("valid target failed to open")
	}
}

func TestProbeRejectsOwnerDrift(t *testing.T) {
	c, _ := loadFixture(envFixture())
	err := verifyDatabaseIdentity(context.Background(), c, func(context.Context, string) (databaseIdentity, error) {
		return databaseIdentity{database: developDatabase, user: developDatabase, owner: "mindforge", brains: true, nodes: true, documents: true, documentLinks: true}, nil
	})
	if err == nil {
		t.Fatal("wrong database owner accepted")
	}
}

func TestDocumentSchemaRequiredBeforeOpeningRuntimeStore(t *testing.T) {
	c, _ := loadFixture(envFixture())
	good := databaseIdentity{database: developDatabase, user: developDatabase, owner: developDatabase, brains: true, nodes: true, documents: true, documentLinks: true}
	for _, missing := range []string{"documents", "links"} {
		id := good
		if missing == "documents" {
			id.documents = false
		} else {
			id.documentLinks = false
		}
		opened := false
		_, err := initializeStore(context.Background(), c, func(context.Context, string) (databaseIdentity, error) { return id, nil }, func(context.Context, string) (runtimeStore, error) { opened = true; return &startupStore{}, nil })
		if err == nil || opened {
			t.Fatalf("missing %s schema allowed runtime store startup", missing)
		}
	}
	if err := verifyDatabaseIdentity(context.Background(), c, func(context.Context, string) (databaseIdentity, error) { return good, nil }); err != nil {
		t.Fatal(err)
	}
}

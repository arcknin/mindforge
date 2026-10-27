package command_test

import (
	"context"
	"testing"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/testsupport"
)

func TestOperationRequestFailsClosedWithoutTargetOrAuthorityReference(t *testing.T) {
	t.Parallel()

	if _, err := command.NewOperationRequest(config.TargetDevelop, "", "owner-2026-09-19"); err == nil {
		t.Fatal("NewOperationRequest() error = nil without operator ID")
	}
	if _, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", ""); err == nil {
		t.Fatal("NewOperationRequest() error = nil without authorization reference")
	}
	if _, err := command.NewOperationRequest(config.Target(""), "operator-1", "owner-2026-09-19"); err == nil {
		t.Fatal("NewOperationRequest() error = nil without an explicit target")
	}
	if _, err := command.NewOperationRequest(config.TargetDevelop, "operator-1", "owner-2026-09-19"); err != nil {
		t.Fatalf("NewOperationRequest(): %v", err)
	}
}

func TestFakeRunnerRecordsStructuredInvocationWithoutAShell(t *testing.T) {
	t.Parallel()

	fake := testsupport.NewFakeRunner()
	fake.Enqueue(command.Result{Stdout: "ok"}, nil)
	result, err := fake.Run(context.Background(), command.Invocation{
		Name: "psql",
		Args: []string{"--set", "ON_ERROR_STOP=1"},
		Env:  []string{"PGPASSWORD=synthetic-private-value"},
	})
	if err != nil {
		t.Fatalf("fake Run(): %v", err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("fake result = %#v", result)
	}
	calls := fake.Calls()
	if len(calls) != 1 || calls[0].Name != "psql" {
		t.Fatalf("fake calls = %#v", calls)
	}
	if calls[0].Shell {
		t.Fatalf("fake invocation unexpectedly permits a shell: %#v", calls[0])
	}
}

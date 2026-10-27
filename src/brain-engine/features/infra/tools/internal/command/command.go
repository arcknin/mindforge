// Package command defines shell-free command construction and the common
// authority envelope used by future database-facing operations.
package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/config"
)

var operatorIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var authorizationReferencePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{2,255}$`)

// OperationRequest contains the explicit authority metadata that every future
// VM, database or backup operation must supply.
type OperationRequest struct {
	Target           config.Target
	OperatorID       string
	AuthorizationRef string
}

// NewOperationRequest fails closed before an operation can infer a target or
// use an ambiguous authority reference.
func NewOperationRequest(target config.Target, operatorID, authorizationRef string) (OperationRequest, error) {
	parsedTarget, err := config.ParseTarget(string(target))
	if err != nil {
		return OperationRequest{}, err
	}
	if !operatorIDPattern.MatchString(operatorID) {
		return OperationRequest{}, fmt.Errorf("an explicit operator ID is required")
	}
	if !authorizationReferencePattern.MatchString(authorizationRef) {
		return OperationRequest{}, fmt.Errorf("an explicit authorization reference is required")
	}
	return OperationRequest{Target: parsedTarget, OperatorID: operatorID, AuthorizationRef: authorizationRef}, nil
}

// Invocation is an argv-based child process request. Shell is retained solely
// so every runner can reject it explicitly; no implementation may enable it.
type Invocation struct {
	Name  string
	Args  []string
	Env   []string
	Shell bool
}

// Result is captured process output for a local caller. It is never persisted
// automatically as evidence.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner abstracts PostgreSQL client invocations for fake-backed tests and
// later guarded operations.
type Runner interface {
	Run(context.Context, Invocation) (Result, error)
}

// ExecRunner executes argv directly through os/exec. It never invokes a shell.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, invocation Invocation) (Result, error) {
	if err := validateInvocation(invocation); err != nil {
		return Result{}, err
	}
	command := exec.CommandContext(ctx, invocation.Name, invocation.Args...)
	// A future operation supplies a deliberately private, minimal child
	// environment. Ambient process variables are never forwarded by default.
	command.Env = append([]string(nil), invocation.Env...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
		return result, fmt.Errorf("command failed with exit code %d", result.ExitCode)
	}
	return result, fmt.Errorf("command did not start or complete")
}

func validateInvocation(invocation Invocation) error {
	if invocation.Shell {
		return fmt.Errorf("shell command evaluation is forbidden")
	}
	if strings.TrimSpace(invocation.Name) == "" || strings.ContainsRune(invocation.Name, '\x00') {
		return fmt.Errorf("an explicit command name is required")
	}
	for _, argument := range invocation.Args {
		if strings.ContainsRune(argument, '\x00') {
			return fmt.Errorf("command arguments cannot contain a NUL byte")
		}
	}
	for _, value := range invocation.Env {
		if strings.ContainsRune(value, '\x00') || !strings.Contains(value, "=") {
			return fmt.Errorf("child environment entries must be KEY=VALUE")
		}
	}
	return nil
}

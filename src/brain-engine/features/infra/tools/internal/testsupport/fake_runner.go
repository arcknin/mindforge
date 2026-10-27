// Package testsupport provides deterministic fake infrastructure command
// execution. It is intentionally limited to unit tests and never starts a
// container, database or VM process.
package testsupport

import (
	"context"
	"fmt"
	"sync"

	"github.com/arcknin/mindforge/src/brain-engine/features/infra/tools/internal/command"
)

type queuedResult struct {
	result command.Result
	err    error
}

// FakeRunner records argv invocations and returns queued outcomes.
type FakeRunner struct {
	mu     sync.Mutex
	queued []queuedResult
	calls  []command.Invocation
}

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{}
}

func (r *FakeRunner) Enqueue(result command.Result, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queued = append(r.queued, queuedResult{result: result, err: err})
}

func (r *FakeRunner) Run(_ context.Context, invocation command.Invocation) (command.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if invocation.Shell {
		return command.Result{}, fmt.Errorf("fake runner refuses shell evaluation")
	}
	r.calls = append(r.calls, cloneInvocation(invocation))
	if len(r.queued) == 0 {
		return command.Result{}, nil
	}
	next := r.queued[0]
	r.queued = r.queued[1:]
	return next.result, next.err
}

func (r *FakeRunner) Calls() []command.Invocation {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]command.Invocation, len(r.calls))
	for index, call := range r.calls {
		calls[index] = cloneInvocation(call)
	}
	return calls
}

func cloneInvocation(invocation command.Invocation) command.Invocation {
	invocation.Args = append([]string(nil), invocation.Args...)
	invocation.Env = append([]string(nil), invocation.Env...)
	return invocation
}

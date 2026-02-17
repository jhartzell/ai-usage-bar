package recovery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

type mockExecutor struct {
	available map[string]bool
	runErr    map[string]error
	runs      []string
}

func (m *mockExecutor) LookPath(name string) error {
	if m.available[name] {
		return nil
	}
	return errors.New("not found")
}

func (m *mockExecutor) Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	m.runs = append(m.runs, fmt.Sprintf("%s %s", name, strings.Join(args, " ")))
	if err := m.runErr[name]; err != nil {
		return err
	}
	return nil
}

func TestRunAuthRecoverySuccess(t *testing.T) {
	exec := &mockExecutor{available: map[string]bool{"claude": true, "codex": true}, runErr: map[string]error{}}
	cacheCleared := false

	var out bytes.Buffer
	err := runAuthRecovery(
		context.Background(),
		strings.NewReader(""),
		&out,
		&out,
		exec,
		func() error {
			cacheCleared = true
			return nil
		},
		defaultTargets,
	)

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !cacheCleared {
		t.Fatal("expected cache clear to be called")
	}
	if len(exec.runs) != 2 {
		t.Fatalf("expected 2 login runs, got %d", len(exec.runs))
	}
	if !strings.Contains(out.String(), "Auth recovery complete") {
		t.Fatalf("expected completion message, got %q", out.String())
	}
}

func TestRunAuthRecoverySkipsMissingCommands(t *testing.T) {
	exec := &mockExecutor{available: map[string]bool{"claude": true}, runErr: map[string]error{}}

	var out bytes.Buffer
	err := runAuthRecovery(
		context.Background(),
		strings.NewReader(""),
		&out,
		&out,
		exec,
		func() error { return nil },
		defaultTargets,
	)

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(exec.runs) != 1 {
		t.Fatalf("expected one run for available command, got %d", len(exec.runs))
	}
	if !strings.Contains(out.String(), "Skipping `codex login`") {
		t.Fatalf("expected skip message, got %q", out.String())
	}
}

func TestRunAuthRecoveryReturnsFailureButStillClearsCache(t *testing.T) {
	exec := &mockExecutor{
		available: map[string]bool{"claude": true, "codex": true},
		runErr:    map[string]error{"codex": errors.New("login failed")},
	}
	cacheCleared := false

	err := runAuthRecovery(
		context.Background(),
		strings.NewReader(""),
		io.Discard,
		io.Discard,
		exec,
		func() error {
			cacheCleared = true
			return nil
		},
		defaultTargets,
	)

	if err == nil {
		t.Fatal("expected auth recovery failure")
	}
	if !strings.Contains(err.Error(), "auth recovery incomplete") {
		t.Fatalf("expected incomplete recovery error, got %v", err)
	}
	if !cacheCleared {
		t.Fatal("expected cache to be cleared even if one login fails")
	}
}

func TestRunAuthRecoveryReturnsCacheClearError(t *testing.T) {
	exec := &mockExecutor{available: map[string]bool{}, runErr: map[string]error{}}

	err := runAuthRecovery(
		context.Background(),
		strings.NewReader(""),
		io.Discard,
		io.Discard,
		exec,
		func() error { return errors.New("boom") },
		defaultTargets,
	)

	if err == nil {
		t.Fatal("expected cache clear error")
	}
	if !strings.Contains(err.Error(), "failed to clear cache") {
		t.Fatalf("unexpected error: %v", err)
	}
}

package recovery

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/jhartzell/ai-usage-bar/internal/cache"
)

type commandExecutor interface {
	LookPath(name string) error
	Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

type osCommandExecutor struct{}

func (osCommandExecutor) LookPath(name string) error {
	_, err := exec.LookPath(name)
	return err
}

func (osCommandExecutor) Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

type loginTarget struct {
	command string
	args    []string
}

var defaultTargets = []loginTarget{
	{command: "claude", args: []string{"login"}},
	{command: "codex", args: []string{"login"}},
}

// RunAuthRecovery runs provider login commands (if installed) and clears local cache.
func RunAuthRecovery(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	return runAuthRecovery(ctx, stdin, stdout, stderr, osCommandExecutor{}, cache.Clear, defaultTargets)
}

func runAuthRecovery(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	exec commandExecutor,
	clearCache func() error,
	targets []loginTarget,
) error {
	fprintf(stdout, "Starting auth recovery...\n")

	var failures []string
	for _, t := range targets {
		if err := exec.LookPath(t.command); err != nil {
			fprintf(stdout, "Skipping `%s %s` (command not found).\n", t.command, strings.Join(t.args, " "))
			continue
		}

		fprintf(stdout, "Running `%s %s`...\n", t.command, strings.Join(t.args, " "))
		if err := exec.Run(ctx, t.command, t.args, stdin, stdout, stderr); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", t.command, err))
			continue
		}

		fprintf(stdout, "Completed `%s %s`.\n", t.command, strings.Join(t.args, " "))
	}

	fprintf(stdout, "Clearing local cache...\n")
	if err := clearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	if len(failures) > 0 {
		return fmt.Errorf("auth recovery incomplete: %s", strings.Join(failures, "; "))
	}

	fprintf(stdout, "Auth recovery complete. Run `ai-usage-bar` to verify.\n")
	return nil
}

func fprintf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, format, args...)
}

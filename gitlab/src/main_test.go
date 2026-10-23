package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestGlabCommandPreservesSpacedArgs guards the bug the github plugin once
// had: arguments were split with strings.Fields, which shredded any value that
// contained spaces (e.g. --title "My Title"). glabCommand takes an explicit
// []string, so each element must reach the child process verbatim.
func TestGlabCommandPreservesSpacedArgs(t *testing.T) {
	want := []string{"mr", "create", "--title", "My Title", "--description", "looks good to me"}

	out, err := helperGlabCommand(t, want...).Output()
	if err != nil {
		t.Fatalf("helper failed: %v", err)
	}

	// The helper prints each received arg on its own line.
	got := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(got) != len(want) {
		t.Fatalf("arg count: got %d %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestGlabCommandDisablesPrompts checks that the child built by glabCommand
// sees GLAB_NO_PROMPT=1 even when the user's environment says otherwise.
func TestGlabCommandDisablesPrompts(t *testing.T) {
	t.Setenv("HELPER_PRINT_ENV", "GLAB_NO_PROMPT")
	t.Setenv("GLAB_NO_PROMPT", "0")

	out, err := helperGlabCommand(t).Output()
	if err != nil {
		t.Fatalf("helper failed: %v", err)
	}
	if got := strings.TrimRight(string(out), "\n"); got != "1" {
		t.Errorf("GLAB_NO_PROMPT in child: got %q, want %q", got, "1")
	}
}

// helperGlabCommand builds the command with the production glabCommand, then
// points it at this test binary in helper mode (the os/exec self-exec pattern),
// so the tests need no glab install. Err is cleared because exec.Command
// records a LookPath failure there when glab is not on PATH.
func helperGlabCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	cmd := glabCommand(context.Background(), append([]string{"-test.run=TestHelperProcess", "--"}, args...)...)
	cmd.Path = os.Args[0]
	cmd.Err = nil
	return cmd
}

// TestHelperProcess is not a real test; it's the child process spawned by
// helperGlabCommand. It echoes the argv it received after "--", one per line,
// or the value of the variable named by HELPER_PRINT_ENV when that is set.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if name := os.Getenv("HELPER_PRINT_ENV"); name != "" {
		os.Stdout.WriteString(os.Getenv(name) + "\n")
		os.Exit(0)
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	for _, a := range args {
		os.Stdout.WriteString(a + "\n")
	}
	os.Exit(0)
}

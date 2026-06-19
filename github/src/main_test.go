package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestRunGhCommandPreservesSpacedArgs is the regression guard for the original
// bug: arguments were split with strings.Fields, which shredded any value that
// contained spaces (e.g. --title "My Title"). We now pass an explicit []string,
// so each element must reach the child process verbatim — quotes and all.
//
// We can't depend on `gh` being installed/authed in CI, so we swap the exec
// target for a tiny helper process (the os/exec self-exec pattern) and assert
// it received the exact argv we passed.
func TestRunGhCommandPreservesSpacedArgs(t *testing.T) {
	want := []string{"pr", "create", "--title", "My Title", "--body", "looks good to me"}

	cmd := helperCommand(t, want...)
	out, err := cmd.Output()
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

// helperCommand builds an exec.Cmd that re-runs this test binary in helper mode,
// passing args through after a "--" sentinel.
func helperCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

// TestHelperProcess is not a real test; it's the child process spawned by
// helperCommand. It echoes the argv it received after "--", one per line.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
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

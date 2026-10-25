package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCappedBufferKeepsShortOutput(t *testing.T) {
	b := &cappedBuffer{limit: 8}
	b.Write([]byte("abc"))
	b.Write([]byte("def"))
	if got := b.String(); got != "abcdef" {
		t.Errorf("got %q, want %q", got, "abcdef")
	}
}

// The cut must not leave half of a multi-byte character before the marker.
func TestCappedBufferCutsOnRuneBoundary(t *testing.T) {
	b := &cappedBuffer{limit: 4}
	b.Write([]byte("abc"))
	b.Write([]byte("日本"))
	want := "abc\n[output truncated: kept the first 3 of 9 bytes]\n"
	if got := b.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseTimeout(t *testing.T) {
	if d, err := parseTimeout(map[string]any{}); err != nil || d != defaultTimeout*time.Second {
		t.Errorf("default: got %v, %v", d, err)
	}
	for _, n := range []any{1, float64(maxTimeout)} {
		if _, err := parseTimeout(map[string]any{"timeout_seconds": n}); err != nil {
			t.Errorf("rejected %v: %v", n, err)
		}
	}
	for _, n := range []any{0, maxTimeout + 1, 1.5, "60", nil, -1} {
		if _, err := parseTimeout(map[string]any{"timeout_seconds": n}); err == nil {
			t.Errorf("accepted %#v", n)
		}
	}
}

func TestRunTruncatesStdoutButDrainsIt(t *testing.T) {
	out, err := run(context.Background(), time.Minute, "helper", execHelper("flood"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "\n[output truncated: kept the first 65536 of 1048576 bytes]\n"
	if !strings.HasSuffix(out, want) || len(out) != stdoutLimit+len(want) {
		t.Errorf("got %d bytes ending %q", len(out), out[max(0, len(out)-80):])
	}
}

func TestRunReportsStderrOnFailure(t *testing.T) {
	_, err := run(context.Background(), time.Minute, "helper", execHelper("fail"))
	if err == nil || !strings.Contains(err.Error(), "helper command failed") || !strings.Contains(err.Error(), "bad flag") {
		t.Errorf("got %v", err)
	}
}

// A grandchild holding stdout keeps Wait blocked until WaitDelay unless the
// whole process tree is stopped, so returning well before WaitDelay shows the
// tree was stopped, not just the direct child.
func TestRunTimeoutStopsProcessTree(t *testing.T) {
	start := time.Now()
	_, err := run(context.Background(), 300*time.Millisecond, "helper", execHelper("tree"))
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "timed out after 300ms") || !strings.Contains(err.Error(), "partial output") {
		t.Errorf("got %v", err)
	}
	if elapsed > 1500*time.Millisecond {
		t.Errorf("took %s; the process tree was not stopped", elapsed)
	}
}

func TestRunCancelStopsProcessTree(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := run(ctx, time.Minute, "helper", execHelper("tree"))
	if err == nil || !strings.Contains(err.Error(), "helper command cancelled") {
		t.Errorf("got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Errorf("took %s; the process tree was not stopped", elapsed)
	}
}

// execHelper returns a command builder that re-runs this test binary as
// TestExecHelper in the given mode.
func execHelper(mode string) func(context.Context) *exec.Cmd {
	return func(ctx context.Context) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestExecHelper$")
		cmd.Env = append(os.Environ(), "EXEC_HELPER="+mode)
		return cmd
	}
}

// TestExecHelper is not a real test; it is the child process for execHelper.
func TestExecHelper(t *testing.T) {
	switch os.Getenv("EXEC_HELPER") {
	case "":
		return
	case "flood":
		chunk := []byte(strings.Repeat("x", 1024))
		for range 1024 {
			os.Stdout.Write(chunk)
		}
	case "fail":
		os.Stderr.WriteString("bad flag\n")
		os.Exit(3)
	case "tree":
		os.Stdout.WriteString("partial output\n")
		child := exec.Command(os.Args[0], "-test.run=^TestExecHelper$")
		child.Env = append(os.Environ(), "EXEC_HELPER=sleep")
		child.Stdout = os.Stdout
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		time.Sleep(time.Minute)
	case "sleep":
		time.Sleep(time.Minute)
	}
	os.Exit(0)
}

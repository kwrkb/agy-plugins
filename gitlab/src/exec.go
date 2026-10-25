package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"
	"unicode/utf8"
)

const (
	// stdoutLimit and stderrLimit bound what one call keeps in memory and hands
	// to the model, matching test-runner's 64 KiB of returned log text.
	stdoutLimit = 64 << 10
	stderrLimit = 16 << 10

	// defaultTimeout is long enough for a paginated api call or a large diff;
	// streaming commands (gh run watch, glab ci trace) pass timeout_seconds instead.
	defaultTimeout = 300
	maxTimeout     = 3600
)

// cappedBuffer drains the pipe even after its retention limit is reached, so
// the child never blocks on a full pipe, and counts what it dropped.
type cappedBuffer struct {
	data  []byte
	limit int
	total int64
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	keep := min(n, max(0, b.limit-len(b.data)))
	b.data = append(b.data, p[:keep]...)
	b.total += int64(n)
	return n, nil
}

func (b *cappedBuffer) truncated() bool { return b.total > int64(len(b.data)) }

// String returns the kept bytes, followed by a marker line when output was cut.
// A cut never splits a UTF-8 sequence that was valid in the full output.
func (b *cappedBuffer) String() string {
	if !b.truncated() {
		return string(b.data)
	}
	kept := b.data
	for i := len(kept) - 1; i >= 0 && i >= len(kept)-utf8.UTFMax; i-- {
		if utf8.RuneStart(kept[i]) {
			if !utf8.FullRune(kept[i:]) {
				kept = kept[:i]
			}
			break
		}
	}
	return fmt.Sprintf("%s\n[output truncated: kept the first %d of %d bytes]\n", kept, len(kept), b.total)
}

// parseTimeout reads the optional timeout_seconds argument strictly, without
// the SDK's coercing getters, so "60" or 1.5 are rejected rather than guessed.
func parseTimeout(args map[string]any) (time.Duration, error) {
	v, exists := args["timeout_seconds"]
	if !exists {
		return defaultTimeout * time.Second, nil
	}
	b, err := json.Marshal(v)
	var n float64
	if err != nil || json.Unmarshal(b, &n) != nil || math.IsNaN(n) || n < 1 || n > maxTimeout || math.Trunc(n) != n {
		return 0, fmt.Errorf("timeout_seconds must be an integer from 1 to %d", maxTimeout)
	}
	return time.Duration(n) * time.Second, nil
}

// run executes the command built by newCmd under its own deadline. On timeout
// it returns what the command printed so far, since a watch that ran out of
// time still produced useful output.
func run(parent context.Context, timeout time.Duration, name string, newCmd func(context.Context) *exec.Cmd) (string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	stdout := &cappedBuffer{limit: stdoutLimit}
	stderr := &cappedBuffer{limit: stderrLimit}
	err := execute(newCmd(ctx), stdout, stderr)
	switch {
	case err == nil:
		return stdout.String(), nil
	case parent.Err() == nil && errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "", fmt.Errorf("%s command timed out after %s and was stopped\nstdout: %s\nstderr: %s", name, timeout, stdout, stderr)
	case parent.Err() != nil:
		return "", fmt.Errorf("%s command cancelled: %v\nstderr: %s", name, err, stderr)
	default:
		return "", fmt.Errorf("%s command failed: %v\nstderr: %s", name, err, stderr)
	}
}

// execute stops the whole process tree on cancellation, as test-runner does:
// gh and glab may start helpers (credential helpers, git) that would otherwise
// outlive the call and keep the pipes open.
func execute(cmd *exec.Cmd, stdout, stderr *cappedBuffer) error {
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = 2 * time.Second
	var stopErr error
	configureProcess(cmd)
	cmd.Cancel = func() error {
		err := stopProcess(cmd)
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			stopErr = err
		}
		return err
	}
	err := cmd.Run()
	// Cmd.Wait joins its cancellation goroutine before returning.
	if stopErr != nil {
		err = fmt.Errorf("%w; process-tree cleanup failed: %v", err, stopErr)
	}
	return err
}

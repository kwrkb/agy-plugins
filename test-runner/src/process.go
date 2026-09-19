package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// cappedBuffer drains the pipe even after its retention limit is reached.
type cappedBuffer struct {
	data      []byte
	limit     int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	keep := min(n, max(0, b.limit-len(b.data)))
	b.data = append(b.data, p[:keep]...)
	b.truncated = b.truncated || keep < n
	return n, nil
}

func execute(cmd *exec.Cmd, out, stderr io.Writer) (*int, error) {
	cmd.Stdout, cmd.Stderr = out, stderr
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
	var code *int
	if cmd.ProcessState != nil {
		n := cmd.ProcessState.ExitCode()
		code = &n
	}
	return code, err
}

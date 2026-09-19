package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func configureProcess(cmd *exec.Cmd) {}
func stopProcess(cmd *exec.Cmd) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Resolve the OS utility directly, rather than a taskkill in the project PATH.
	kill := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "taskkill.exe"), "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	kill.WaitDelay = time.Second
	if out, err := kill.CombinedOutput(); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("taskkill: %w: %s", err, out)
	}
	return nil
}
